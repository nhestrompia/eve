package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestRuntimeEventsAreQuiescentUntilFilesystemInput(t *testing.T) {
	root := t.TempDir()
	repo := repository{ID: "fixture", Root: root, eveDir: filepath.Join(root, ".eve")}
	if err := os.MkdirAll(filepath.Join(repo.eveDir, "snapshots"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo.eveDir, "cache"), 0o755); err != nil {
		t.Fatal(err)
	}

	events := newRuntimeEvents(20 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := events.watch(ctx, []repository{repo}); err != nil {
		t.Fatal(err)
	}
	stream, unsubscribe := events.subscribe()
	defer unsubscribe()

	select {
	case event := <-stream:
		t.Fatalf("quiescent watcher emitted %#v", event)
	case <-time.After(100 * time.Millisecond):
	}

	path := filepath.Join(repo.eveDir, "snapshots", "snap_fixture.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-stream:
		if event.Kind != runtimeEventSnapshots || event.Repository != repo.ID {
			t.Fatalf("snapshot event = %#v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("snapshot write did not emit a runtime event")
	}

	select {
	case event := <-stream:
		t.Fatalf("single snapshot write emitted an uncoalesced duplicate %#v", event)
	case <-time.After(100 * time.Millisecond):
	}

	if err := os.WriteFile(filepath.Join(repo.eveDir, "cache", "index.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-stream:
		t.Fatalf("derived cache write emitted %#v", event)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestRuntimeEventsBoundSlowSubscribersAndCleanUp(t *testing.T) {
	events := newRuntimeEvents(time.Millisecond)
	stream, unsubscribe := events.subscribe()

	for i := 0; i < runtimeEventSubscriberBuffer*4; i++ {
		events.publish(runtimeEvent{Kind: runtimeEventRepositories})
	}
	if got := events.subscriberCount(); got != 1 {
		t.Fatalf("subscriber count = %d, want 1", got)
	}

	select {
	case <-stream:
	case <-time.After(time.Second):
		t.Fatal("bounded subscriber received no event")
	}

	unsubscribe()
	if got := events.subscriberCount(); got != 0 {
		t.Fatalf("subscriber count after unsubscribe = %d, want 0", got)
	}
}

func TestRuntimeEventsIgnoreWatchedDirectorySelfNotifications(t *testing.T) {
	root := t.TempDir()
	watched := watchedRepository{
		repo:       repository{ID: "fixture", Root: root, eveDir: filepath.Join(root, ".eve")},
		root:       root,
		eveRoot:    filepath.Join(root, ".eve"),
		privateEve: filepath.Join(root, ".git", "eve"),
		gitDir:     filepath.Join(root, ".git"),
		commonDir:  filepath.Join(root, ".git"),
	}
	for _, path := range []string{watched.root, watched.eveRoot, watched.privateEve, watched.gitDir} {
		if event, ok := classifyRuntimePath(path, []watchedRepository{watched}); ok {
			t.Fatalf("directory self notification %q emitted %#v", path, event)
		}
	}
}

func TestDeriveGitFactsDoesNotMutateWatchedIndex(t *testing.T) {
	repo := setupPlanTestRepo(t)
	gitDir, err := resolveGitPath(repo, "--git-dir")
	if err != nil {
		t.Fatal(err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()
	if err := watcher.Add(gitDir); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	for {
		select {
		case <-watcher.Events:
			continue
		default:
		}
		break
	}
	if _, err := deriveGitFacts(repo); err != nil {
		t.Fatal(err)
	}
	deadline := time.NewTimer(150 * time.Millisecond)
	defer deadline.Stop()
	for {
		select {
		case event := <-watcher.Events:
			name := filepath.Base(event.Name)
			mutated := event.Has(fsnotify.Write) || event.Has(fsnotify.Create) ||
				event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename)
			if name == "index.lock" || (name == "index" && mutated) {
				t.Fatalf("read-only Git facts mutated %s with %s", name, event.Op)
			}
		case <-deadline.C:
			return
		}
	}
}

func TestRuntimeEventsBoundFilesystemWatchRegistrations(t *testing.T) {
	root := initTempGitRepo(t)
	repo := repoFromRoot(root)
	if err := os.MkdirAll(repo.eveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < maxRuntimeWatchDirectories+100; index++ {
		path := filepath.Join(root, "packages", fmt.Sprintf("%04d", index))
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "tracked.txt"), []byte("tracked\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	gitRun(t, root, "add", "packages")
	events := newRuntimeEvents(time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := events.watch(ctx, []repository{repo}); err != nil {
		t.Fatal(err)
	}
	if got := events.watchedDirectoryCount(); got > maxRuntimeWatchDirectories {
		t.Fatalf("watched directories = %d, limit = %d", got, maxRuntimeWatchDirectories)
	}
}

func TestRuntimeEventsWatchTrackedNestedDirectories(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("macOS kqueue opens every file in watched source directories")
	}
	repo := setupPlanTestRepo(t)
	nestedDir := filepath.Join(repo.Root, "cmd", "worker")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	nestedFile := filepath.Join(nestedDir, "worker.go")
	if err := os.WriteFile(nestedFile, []byte("package worker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo.Root, "add", "cmd/worker/worker.go")
	gitRun(t, repo.Root, "commit", "-m", "add nested worker")

	events := newRuntimeEvents(20 * time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := events.watch(ctx, []repository{repo}); err != nil {
		t.Fatal(err)
	}
	stream, unsubscribe := events.subscribe()
	defer unsubscribe()
	if err := os.WriteFile(nestedFile, []byte("package worker\n\nconst changed = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-stream:
		if event.Kind != runtimeEventRepositories || event.Repository != repo.ID {
			t.Fatalf("nested worktree event = %#v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("tracked nested file write did not emit a repository event")
	}
}

func TestRuntimeEventSSEDeliversPublishedInvalidation(t *testing.T) {
	repo := repository{ID: "fixture", Root: t.TempDir()}
	server := newRuntimeServer(repo, "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recorder := newLockedSSERecorder()
	done := make(chan struct{})
	go func() {
		server.routes().ServeHTTP(
			recorder,
			httptest.NewRequest(http.MethodGet, "/api/events", nil).WithContext(ctx),
		)
		close(done)
	}()
	deadline := time.Now().Add(time.Second)
	for !strings.Contains(recorder.String(), "event: ready") && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	server.events.publish(runtimeEvent{Kind: runtimeEventSnapshots, Repository: repo.ID})
	deadline = time.Now().Add(time.Second)
	for !strings.Contains(recorder.String(), "event: snapshots") && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	<-done
	if body := recorder.String(); !strings.Contains(body, "event: snapshots") ||
		!strings.Contains(body, `"repository":"fixture"`) {
		t.Fatalf("runtime event stream = %q", body)
	}
}
