package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	agentStateRunning = "running"
	agentStateWaiting = "waiting"
	agentStateOffline = "offline"

	defaultAgentLeaseTTL     = 90 * time.Second
	defaultAgentLeaseRenewal = 30 * time.Second
)

type agentLease struct {
	SchemaVersion  int    `json:"schemaVersion"`
	AgentID        string `json:"agentId"`
	Provider       string `json:"provider"`
	ProcessID      int    `json:"processId"`
	Repository     string `json:"repository"`
	RepositoryRoot string `json:"repositoryRoot"`
	PlanRequestID  string `json:"planRequestId"`
	PlanID         string `json:"planId,omitempty"`
	Label          string `json:"label"`
	Status         string `json:"status"`
	UpdatedAt      string `json:"updatedAt"`
	ExpiresAt      string `json:"expiresAt"`
}

type agentActivity struct {
	AgentID       string `json:"agentId"`
	Provider      string `json:"provider"`
	Repository    string `json:"repository"`
	PlanRequestID string `json:"planRequestId"`
	PlanID        string `json:"planId,omitempty"`
	Label         string `json:"label"`
	Status        string `json:"status"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
	ExpiresAt     string `json:"expiresAt,omitempty"`
}

type agentLeaseOwner struct {
	mu         sync.Mutex
	repo       repository
	provider   string
	agentID    string
	ttl        time.Duration
	renewEvery time.Duration
	lease      *agentLease
	stop       chan struct{}
	done       chan struct{}
	started    bool
	closed     bool
}

func newAgentLeaseOwner(
	repo repository,
	provider string,
	agentID string,
	ttl time.Duration,
	renewEvery time.Duration,
) *agentLeaseOwner {
	if ttl <= 0 {
		ttl = defaultAgentLeaseTTL
	}
	if renewEvery <= 0 {
		renewEvery = defaultAgentLeaseRenewal
	}
	return &agentLeaseOwner{
		repo:       repo,
		provider:   fallback(strings.TrimSpace(provider), "Agent"),
		agentID:    fallback(strings.TrimSpace(agentID), newRecordID("agent")),
		ttl:        ttl,
		renewEvery: renewEvery,
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
}

func (owner *agentLeaseOwner) update(request *planRequest, status string, now time.Time) error {
	if request == nil {
		return errors.New("agent lease requires a plan request")
	}
	switch status {
	case agentStateRunning, agentStateWaiting:
	default:
		return errors.New("agent lease status must be running or waiting")
	}
	label := "Working on an approved plan"
	if revision := currentPlanRevision(request); revision != nil && strings.TrimSpace(revision.Goal) != "" {
		label = strings.TrimSpace(revision.Goal)
	}
	owner.mu.Lock()
	defer owner.mu.Unlock()
	if owner.closed {
		return errors.New("agent lease owner is closed")
	}
	owner.lease = &agentLease{
		SchemaVersion:  1,
		AgentID:        owner.agentID,
		Provider:       owner.provider,
		ProcessID:      os.Getpid(),
		Repository:     request.Repository,
		RepositoryRoot: request.RepositoryRoot,
		PlanRequestID:  request.PlanRequestID,
		PlanID:         request.PlanID,
		Label:          label,
		Status:         status,
	}
	if owner.lease.Repository == "" {
		owner.lease.Repository = owner.repo.ID
	}
	if owner.lease.RepositoryRoot == "" {
		owner.lease.RepositoryRoot = owner.repo.Root
	}
	if err := owner.writeLocked(now); err != nil {
		return err
	}
	if !owner.started {
		owner.started = true
		go owner.renew()
	}
	return nil
}

func currentPlanRevision(request *planRequest) *evePlanRevisionView {
	if request == nil || request.CurrentRevision <= 0 || request.CurrentRevision > len(request.Revisions) {
		return nil
	}
	revision := request.Revisions[request.CurrentRevision-1]
	return &evePlanRevisionView{Goal: revision.Goal}
}

type evePlanRevisionView struct {
	Goal string
}

func (owner *agentLeaseOwner) renew() {
	ticker := time.NewTicker(owner.renewEvery)
	defer ticker.Stop()
	defer close(owner.done)
	for {
		select {
		case <-owner.stop:
			return
		case now := <-ticker.C:
			owner.mu.Lock()
			if owner.lease != nil && !owner.closed {
				_ = owner.writeLocked(now.UTC())
			}
			owner.mu.Unlock()
		}
	}
}

func (owner *agentLeaseOwner) writeLocked(now time.Time) error {
	if owner.lease == nil {
		return nil
	}
	owner.lease.UpdatedAt = now.UTC().Format(time.RFC3339Nano)
	owner.lease.ExpiresAt = now.UTC().Add(owner.ttl).Format(time.RFC3339Nano)
	path, err := owner.repo.agentLeasePath(owner.agentID)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(owner.lease, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomically(path, append(data, '\n'), 0o600)
}

func (owner *agentLeaseOwner) remove() error {
	owner.mu.Lock()
	defer owner.mu.Unlock()
	owner.lease = nil
	path, err := owner.repo.agentLeasePath(owner.agentID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (owner *agentLeaseOwner) close() {
	owner.mu.Lock()
	if owner.closed {
		owner.mu.Unlock()
		return
	}
	owner.closed = true
	started := owner.started
	if started {
		close(owner.stop)
	}
	owner.lease = nil
	path, _ := owner.repo.agentLeasePath(owner.agentID)
	owner.mu.Unlock()
	if started {
		<-owner.done
	}
	if path != "" {
		_ = os.Remove(path)
	}
}

func (repo repository) agentLeasesDir() (string, error) {
	return repo.verificationPrivatePath("agents")
}

func (repo repository) agentLeasePath(agentID string) (string, error) {
	dir, err := repo.agentLeasesDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(agentID))
	return filepath.Join(dir, hex.EncodeToString(sum[:16])+".json"), nil
}

func (repo repository) agentLeases(now time.Time) []agentLease {
	dir, err := repo.agentLeasesDir()
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	result := make([]agentLease, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, entry.Name()))
		if readErr != nil {
			continue
		}
		var lease agentLease
		if json.Unmarshal(data, &lease) != nil {
			continue
		}
		expiresAt, parseErr := time.Parse(time.RFC3339Nano, lease.ExpiresAt)
		if parseErr != nil || !expiresAt.After(now) {
			continue
		}
		result = append(result, lease)
	}
	return result
}

func (repo repository) nextAgentLeaseExpiry(now time.Time) time.Time {
	var next time.Time
	for _, lease := range repo.agentLeases(now) {
		expiresAt, err := time.Parse(time.RFC3339Nano, lease.ExpiresAt)
		if err == nil && (next.IsZero() || expiresAt.Before(next)) {
			next = expiresAt
		}
	}
	return next
}

func (events *runtimeEvents) watchAgentExpirations(ctx context.Context, repositories []repository) {
	stream, unsubscribe := events.subscribe()
	defer unsubscribe()
	var timer *time.Timer
	var timerChannel <-chan time.Time
	reset := func() {
		next := time.Time{}
		now := time.Now().UTC()
		for _, repo := range repositories {
			expiry := repo.nextAgentLeaseExpiry(now)
			if !expiry.IsZero() && (next.IsZero() || expiry.Before(next)) {
				next = expiry
			}
		}
		if timer != nil && !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		timerChannel = nil
		if next.IsZero() {
			return
		}
		delay := time.Until(next)
		if delay < 0 {
			delay = 0
		}
		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			timer.Reset(delay)
		}
		timerChannel = timer.C
	}
	reset()
	for {
		select {
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			return
		case <-timerChannel:
			events.publish(runtimeEvent{Kind: runtimeEventAgents})
			reset()
		case event := <-stream:
			if event.Kind == runtimeEventAgents || event.Kind == runtimeEventAll {
				reset()
			}
		}
	}
}

func (server runtimeServer) agentActivities(now time.Time) []agentActivity {
	repositories := server.planRequestRepositories()
	requests, _ := planRequestsFromRepositories(context.Background(), repositories, "")
	leases := make(map[string]agentLease)
	for _, repo := range repositories {
		for _, lease := range repo.agentLeases(now) {
			leases[lease.PlanRequestID] = lease
		}
	}
	result := make([]agentActivity, 0)
	seen := make(map[string]bool)
	for _, request := range requests {
		if request.State != "locked" && request.State != "pending_approval" {
			continue
		}
		if seen[request.PlanRequestID] {
			continue
		}
		activity := agentActivity{
			AgentID:       "plan:" + request.PlanRequestID,
			Provider:      "Agent",
			Repository:    request.Repository,
			PlanRequestID: request.PlanRequestID,
			PlanID:        request.PlanID,
			Label:         "Working on an approved plan",
			Status:        agentStateOffline,
		}
		if revision := currentPlanRevision(request); revision != nil && strings.TrimSpace(revision.Goal) != "" {
			activity.Label = strings.TrimSpace(revision.Goal)
		}
		if lease, ok := leases[request.PlanRequestID]; ok {
			activity.AgentID = lease.AgentID
			activity.Provider = lease.Provider
			activity.Status = lease.Status
			activity.UpdatedAt = lease.UpdatedAt
			activity.ExpiresAt = lease.ExpiresAt
			delete(leases, request.PlanRequestID)
		}
		seen[request.PlanRequestID] = true
		result = append(result, activity)
	}
	for _, lease := range leases {
		if seen[lease.PlanRequestID] {
			continue
		}
		result = append(result, agentActivity{
			AgentID:       lease.AgentID,
			Provider:      lease.Provider,
			Repository:    lease.Repository,
			PlanRequestID: lease.PlanRequestID,
			PlanID:        lease.PlanID,
			Label:         lease.Label,
			Status:        lease.Status,
			UpdatedAt:     lease.UpdatedAt,
			ExpiresAt:     lease.ExpiresAt,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Status != result[j].Status {
			return agentStatusOrder(result[i].Status) < agentStatusOrder(result[j].Status)
		}
		if result[i].Repository != result[j].Repository {
			return result[i].Repository < result[j].Repository
		}
		return result[i].PlanRequestID < result[j].PlanRequestID
	})
	return result
}

func (server runtimeServer) cachedAgentActivities(now time.Time) []agentActivity {
	activities, err := cachedRuntimeValue(
		context.Background(),
		server.derivedCache,
		"agent-activities",
		server.events.currentGeneration(),
		runtimeDerivedCacheTTL,
		func() ([]agentActivity, error) {
			return server.agentActivities(now), nil
		},
	)
	if err != nil {
		return server.agentActivities(now)
	}
	return append([]agentActivity(nil), activities...)
}

func agentStatusOrder(status string) int {
	switch status {
	case agentStateRunning:
		return 0
	case agentStateWaiting:
		return 1
	default:
		return 2
	}
}

func detectedAgentIdentity() (string, string) {
	if provider := strings.TrimSpace(os.Getenv("EVE_AGENT_PROVIDER")); provider != "" {
		return provider, detectedAgentID()
	}
	switch {
	case strings.TrimSpace(os.Getenv("CODEX_THREAD_ID")) != "":
		return "Codex", detectedAgentID()
	case strings.TrimSpace(os.Getenv("CLAUDE_SESSION_ID")) != "":
		return "Claude", detectedAgentID()
	case strings.TrimSpace(os.Getenv("OPENCODE_SESSION_ID")) != "":
		return "OpenCode", detectedAgentID()
	default:
		return "Agent", detectedAgentID()
	}
}

func detectedAgentID() string {
	if value := strings.TrimSpace(os.Getenv("EVE_AGENT_ID")); value != "" {
		return value
	}
	for _, key := range []string{"CODEX_THREAD_ID", "CLAUDE_SESSION_ID", "OPENCODE_SESSION_ID"} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			sum := sha256.Sum256([]byte(value))
			return "agent_" + hex.EncodeToString(sum[:8])
		}
	}
	return newRecordID("agent")
}

func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".eve-write-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func (server runtimeServer) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, server.cachedAgentActivities(time.Now().UTC()))
}
