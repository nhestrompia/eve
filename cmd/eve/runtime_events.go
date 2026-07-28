package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

type runtimeEventKind string

const (
	runtimeEventAll          runtimeEventKind = "all"
	runtimeEventAgents       runtimeEventKind = "agents"
	runtimeEventConfig       runtimeEventKind = "config"
	runtimeEventPlans        runtimeEventKind = "plans"
	runtimeEventRepositories runtimeEventKind = "repositories"
	runtimeEventSnapshots    runtimeEventKind = "snapshots"
	runtimeEventVerification runtimeEventKind = "verification"

	runtimeEventSubscriberBuffer = 16
	maxRuntimeWatchDirectories   = 256
)

type runtimeEvent struct {
	Kind       runtimeEventKind `json:"kind"`
	Repository string           `json:"repository,omitempty"`
}

type runtimeEvents struct {
	mu          sync.RWMutex
	subscribers map[uint64]chan runtimeEvent
	nextID      uint64
	debounce    time.Duration
	watchedDirs int
	generation  atomic.Uint64
}

func newRuntimeEvents(debounce time.Duration) *runtimeEvents {
	if debounce <= 0 {
		debounce = 150 * time.Millisecond
	}
	return &runtimeEvents{
		subscribers: make(map[uint64]chan runtimeEvent),
		debounce:    debounce,
	}
}

func (events *runtimeEvents) subscribe() (<-chan runtimeEvent, func()) {
	channel := make(chan runtimeEvent, runtimeEventSubscriberBuffer)
	events.mu.Lock()
	id := events.nextID
	events.nextID++
	events.subscribers[id] = channel
	events.mu.Unlock()

	var once sync.Once
	return channel, func() {
		once.Do(func() {
			events.mu.Lock()
			delete(events.subscribers, id)
			events.mu.Unlock()
		})
	}
}

func (events *runtimeEvents) publish(event runtimeEvent) {
	events.generation.Add(1)
	events.mu.RLock()
	defer events.mu.RUnlock()
	for _, channel := range events.subscribers {
		select {
		case channel <- event:
		default:
			select {
			case <-channel:
			default:
			}
			select {
			case channel <- event:
			default:
			}
		}
	}
}

func (events *runtimeEvents) currentGeneration() uint64 {
	if events == nil {
		return 0
	}
	return events.generation.Load()
}

func (events *runtimeEvents) subscriberCount() int {
	if events == nil {
		return 0
	}
	events.mu.RLock()
	defer events.mu.RUnlock()
	return len(events.subscribers)
}

func (events *runtimeEvents) watchedDirectoryCount() int {
	if events == nil {
		return 0
	}
	events.mu.RLock()
	defer events.mu.RUnlock()
	return events.watchedDirs
}

func (events *runtimeEvents) runRecoveryRefresh(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			events.publish(runtimeEvent{Kind: runtimeEventAll})
		}
	}
}

type watchedRepository struct {
	repo       repository
	root       string
	eveRoot    string
	privateEve string
	gitDir     string
	commonDir  string
}

func (events *runtimeEvents) watch(ctx context.Context, repositories []repository) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	watched := make([]watchedRepository, 0, len(repositories))
	added := make(map[string]bool)
	addDir := func(path string) {
		path = filepath.Clean(path)
		if added[path] || len(added) >= maxRuntimeWatchDirectories {
			return
		}
		if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
			if watcher.Add(path) == nil {
				added[path] = true
				events.mu.Lock()
				events.watchedDirs = len(added)
				events.mu.Unlock()
			}
		}
	}
	addTree := func(root string) {
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || !entry.IsDir() {
				return nil
			}
			addDir(path)
			return nil
		})
	}

	for _, repo := range repositories {
		entry := watchedRepository{
			repo:    repo,
			root:    filepath.Clean(repo.Root),
			eveRoot: filepath.Clean(repo.eveDir),
		}
		if path, pathErr := repo.verificationPrivatePath(); pathErr == nil {
			entry.privateEve = filepath.Clean(path)
		}
		if path, pathErr := resolveGitPath(repo, "--git-dir"); pathErr == nil {
			entry.gitDir = path
		}
		if path, pathErr := resolveGitPath(repo, "--git-common-dir"); pathErr == nil {
			entry.commonDir = path
		}
		watched = append(watched, entry)

		addDir(entry.root)
		addDir(entry.eveRoot)
		for _, name := range []string{"snapshots", "plans", "runs", "skips", "cache", "artifacts"} {
			addDir(filepath.Join(entry.eveRoot, name))
		}
		addDir(entry.privateEve)
		for _, name := range []string{"agents", "plan-requests", "cancel", "known-runs"} {
			addDir(filepath.Join(entry.privateEve, name))
		}
		addDir(entry.gitDir)
		addTree(filepath.Join(entry.gitDir, "refs"))
		if entry.commonDir != entry.gitDir {
			addDir(entry.commonDir)
			addTree(filepath.Join(entry.commonDir, "refs"))
		}
	}
	if runtime.GOOS != "darwin" && len(watched) > 0 {
		for _, dir := range trackedWorktreeDirectories(watched[0].repo) {
			addDir(dir)
		}
	}
	if len(added) == 0 {
		_ = watcher.Close()
		return errors.New("no runtime event paths were watchable")
	}

	go events.runWatcher(ctx, watcher, watched, added)
	return nil
}

func resolveGitPath(repo repository, flag string) (string, error) {
	value, err := gitOutput(repo.Root, "rev-parse", flag)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if !filepath.IsAbs(value) {
		value = filepath.Join(repo.Root, filepath.FromSlash(value))
	}
	return filepath.Clean(value), nil
}

func (events *runtimeEvents) runWatcher(
	ctx context.Context,
	watcher *fsnotify.Watcher,
	repositories []watchedRepository,
	added map[string]bool,
) {
	defer watcher.Close()
	pending := make(map[string]runtimeEvent)
	var timer *time.Timer
	var timerChannel <-chan time.Time

	flush := func() {
		keys := make([]string, 0, len(pending))
		for key := range pending {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			events.publish(pending[key])
			delete(pending, key)
		}
		timerChannel = nil
	}
	queue := func(event runtimeEvent) {
		key := string(event.Kind)
		if existing, ok := pending[key]; ok && existing.Repository != event.Repository {
			event.Repository = ""
		}
		pending[key] = event
		if timer == nil {
			timer = time.NewTimer(events.debounce)
			timerChannel = timer.C
			return
		}
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timer.Reset(events.debounce)
		timerChannel = timer.C
	}
	addNewDirectory := func(path string) {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() || runtimeWatchPathIgnored(path, repositories) {
			return
		}
		_ = filepath.WalkDir(path, func(current string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil || !entry.IsDir() || added[current] {
				return nil
			}
			if runtimeWatchPathIgnored(current, repositories) {
				return filepath.SkipDir
			}
			if len(added) < maxRuntimeWatchDirectories && watcher.Add(current) == nil {
				added[current] = true
				events.mu.Lock()
				events.watchedDirs = len(added)
				events.mu.Unlock()
			}
			return nil
		})
	}

	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case <-timerChannel:
			flush()
		case watchErr, ok := <-watcher.Errors:
			if !ok {
				return
			}
			_ = watchErr
			queue(runtimeEvent{Kind: runtimeEventAll})
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op == fsnotify.Chmod {
				continue
			}
			if event.Has(fsnotify.Create) {
				addNewDirectory(event.Name)
			}
			if mapped, ok := classifyRuntimePath(event.Name, repositories); ok {
				queue(mapped)
			}
		}
	}
}

func classifyRuntimePath(path string, repositories []watchedRepository) (runtimeEvent, bool) {
	path = filepath.Clean(path)
	for _, watched := range repositories {
		if path == watched.root || path == watched.eveRoot || path == watched.privateEve ||
			path == watched.gitDir || path == watched.commonDir {
			return runtimeEvent{}, false
		}
		if relativePathWithin(watched.privateEve, path) {
			relative, _ := filepath.Rel(watched.privateEve, path)
			switch firstPathPart(relative) {
			case "agents":
				return runtimeEvent{Kind: runtimeEventAgents, Repository: watched.repo.ID}, true
			case "plan-requests":
				return runtimeEvent{Kind: runtimeEventPlans, Repository: watched.repo.ID}, true
			case "cancel", "known-runs":
				return runtimeEvent{Kind: runtimeEventVerification, Repository: watched.repo.ID}, true
			default:
				return runtimeEvent{}, false
			}
		}
		if relativePathWithin(watched.eveRoot, path) {
			relative, _ := filepath.Rel(watched.eveRoot, path)
			switch firstPathPart(relative) {
			case "cache", ".state":
				return runtimeEvent{}, false
			case "snapshots":
				return runtimeEvent{Kind: runtimeEventSnapshots, Repository: watched.repo.ID}, true
			case "plans":
				return runtimeEvent{Kind: runtimeEventPlans, Repository: watched.repo.ID}, true
			case "runs":
				return runtimeEvent{Kind: runtimeEventVerification, Repository: watched.repo.ID}, true
			case "config.json":
				return runtimeEvent{Kind: runtimeEventConfig, Repository: watched.repo.ID}, true
			default:
				return runtimeEvent{Kind: runtimeEventRepositories, Repository: watched.repo.ID}, true
			}
		}
		if relativePathWithin(watched.gitDir, path) || relativePathWithin(watched.commonDir, path) {
			return runtimeEvent{Kind: runtimeEventRepositories, Repository: watched.repo.ID}, true
		}
		if relativePathWithin(watched.root, path) {
			return runtimeEvent{Kind: runtimeEventRepositories, Repository: watched.repo.ID}, true
		}
	}
	return runtimeEvent{}, false
}

func trackedWorktreeDirectories(repo repository) []string {
	output, err := gitOutput(repo.Root, "ls-files", "-z")
	if err != nil {
		return nil
	}
	directories := make(map[string]bool)
	for _, tracked := range strings.Split(output, "\x00") {
		tracked = strings.TrimSpace(tracked)
		if tracked == "" {
			continue
		}
		dir := filepath.Dir(filepath.Join(repo.Root, filepath.FromSlash(tracked)))
		for relativePathWithin(repo.Root, dir) && dir != repo.Root {
			directories[dir] = true
			dir = filepath.Dir(dir)
		}
	}
	result := make([]string, 0, len(directories))
	for dir := range directories {
		result = append(result, dir)
	}
	sort.Strings(result)
	return result
}

func runtimeWatchPathIgnored(path string, repositories []watchedRepository) bool {
	for _, watched := range repositories {
		if relativePathWithin(watched.eveRoot, path) ||
			relativePathWithin(watched.privateEve, path) ||
			relativePathWithin(watched.gitDir, path) ||
			relativePathWithin(watched.commonDir, path) {
			return false
		}
		if relativePathWithin(watched.root, path) {
			relative, _ := filepath.Rel(watched.root, path)
			for _, part := range strings.Split(filepath.Clean(relative), string(os.PathSeparator)) {
				switch part {
				case ".git", ".eve", "node_modules", "ui_dist", "dist", "build", "vendor":
					return true
				}
			}
			return false
		}
	}
	return true
}

func relativePathWithin(root, path string) bool {
	if strings.TrimSpace(root) == "" {
		return false
	}
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator))
}

func firstPathPart(path string) string {
	path = filepath.Clean(path)
	if index := strings.IndexRune(path, os.PathSeparator); index >= 0 {
		return path[:index]
	}
	return path
}

func (server runtimeServer) handleRuntimeEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, errors.New("streaming is unavailable"))
		return
	}
	if server.events == nil {
		writeAPIError(w, http.StatusServiceUnavailable, errors.New("runtime events are unavailable"))
		return
	}
	stream, unsubscribe := server.events.subscribe()
	defer unsubscribe()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	_, _ = fmt.Fprint(w, "event: ready\ndata: {}\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-stream:
			data, _ := json.Marshal(event)
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Kind, data)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}
