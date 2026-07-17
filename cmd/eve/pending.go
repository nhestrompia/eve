package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	skipSchemaVersion    = "0.1.0"
	pendingStateVersion  = 1
	pendingIdleThreshold = 2 * time.Hour
	pendingStateFileName = "pending-state.json"
	pendingTriggerIdle   = "idle"
	pendingTriggerMerge  = "merge"
)

type gitRange struct {
	From    string   `json:"from"`
	To      string   `json:"to"`
	Commits []string `json:"commits"`
}

type skipAgent struct {
	Provider string `json:"provider,omitempty"`
	ID       string `json:"id,omitempty"`
}

type skipRecord struct {
	SchemaVersion string    `json:"schemaVersion"`
	ID            string    `json:"id"`
	RepoID        string    `json:"repoId"`
	Branch        string    `json:"branch"`
	Reason        string    `json:"reason"`
	Range         gitRange  `json:"range"`
	CreatedAt     string    `json:"createdAt"`
	Agent         skipAgent `json:"agent,omitempty"`
}

type pendingSnapshot struct {
	RepoID               string   `json:"repoId"`
	Branch               string   `json:"branch"`
	TrunkBranch          string   `json:"trunkBranch"`
	LastResolvedGitState string   `json:"lastResolvedGitState"`
	PendingSince         string   `json:"pendingSince,omitempty"`
	Trigger              string   `json:"trigger,omitempty"`
	IdleThreshold        string   `json:"idleThreshold,omitempty"`
	Range                gitRange `json:"range"`
}

type pendingSnapshotResponse struct {
	Pending         bool             `json:"pending"`
	PendingSnapshot *pendingSnapshot `json:"pendingSnapshot,omitempty"`
}

type pendingStateFile struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Branches      []pendingBranchState `json:"branches"`
}

type pendingBranchState struct {
	RepoID               string    `json:"repoId"`
	Root                 string    `json:"root"`
	Branch               string    `json:"branch"`
	LastResolvedGitState string    `json:"lastResolvedGitState"`
	PendingSince         string    `json:"pendingSince,omitempty"`
	PendingRange         *gitRange `json:"pendingRange,omitempty"`
}

type pendingOptions struct {
	Initialize bool
	Now        time.Time
}

type resolvedCandidate struct {
	GitState  string
	Branch    string
	CreatedAt string
	Distance  int
}

func (repo repository) skipsDir() string {
	return filepath.Join(repo.eveDir, "skips")
}

func (repo repository) skipPath(id string) string {
	return filepath.Join(repo.skipsDir(), id+".json")
}

func (repo repository) pendingStatePath() string {
	if path := strings.TrimSpace(os.Getenv("EVE_PENDING_STATE")); path != "" {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(repo.eveDir, "cache", pendingStateFileName)
	}
	return filepath.Join(cacheDir, "eve", pendingStateFileName)
}

func (repo repository) saveSkip(record *skipRecord) error {
	if strings.TrimSpace(record.Reason) == "" {
		return errors.New("skip reason is required")
	}
	if err := os.MkdirAll(repo.skipsDir(), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", repo.skipsDir(), err)
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal skip record: %w", err)
	}
	if err := os.WriteFile(repo.skipPath(record.ID), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write skip %s: %w", record.ID, err)
	}
	rememberRepository(repo)
	return nil
}

func (repo repository) listSkips() ([]skipRecord, error) {
	entries, err := os.ReadDir(repo.skipsDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read skips: %w", err)
	}
	records := []skipRecord{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repo.skipsDir(), entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read skip %s: %w", entry.Name(), err)
		}
		var record skipRecord
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("parse skip %s: %w", entry.Name(), err)
		}
		if strings.TrimSpace(record.SchemaVersion) == "" {
			record.SchemaVersion = skipSchemaVersion
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].CreatedAt == records[j].CreatedAt {
			return records[i].ID < records[j].ID
		}
		return records[i].CreatedAt > records[j].CreatedAt
	})
	return records, nil
}

func (repo repository) createSkip(reason string, agent skipAgent) (*skipRecord, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, errors.New("skip reason is required")
	}
	pending, err := repo.detectPending(pendingOptions{Initialize: false, Now: time.Now().UTC()})
	if err != nil {
		return nil, err
	}
	facts, err := deriveGitFacts(repo)
	if err != nil {
		return nil, err
	}
	recordRange := gitRange{To: facts.GitState, Commits: normalizedCommitList(facts.Commits)}
	if pending != nil {
		recordRange = pending.Range
	} else {
		recordRange.From = facts.BaseCommit
	}
	record := &skipRecord{
		SchemaVersion: skipSchemaVersion,
		ID:            newSkipID(),
		RepoID:        repo.ID,
		Branch:        facts.Branch,
		Reason:        reason,
		Range:         recordRange,
		CreatedAt:     nowUTC(),
		Agent:         agent,
	}
	if record.Range.To == "" {
		record.Range.To = facts.GitState
	}
	if len(record.Range.Commits) == 0 {
		record.Range.Commits = []string{facts.GitState}
	}
	if err := repo.saveSkip(record); err != nil {
		return nil, err
	}
	if err := repo.resolvePendingBranch(record.Branch, record.Range.To); err != nil {
		return nil, err
	}
	return record, nil
}

func (repo repository) detectPending(options pendingOptions) (*pendingSnapshot, error) {
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	branch, err := currentBranch(repo)
	if err != nil {
		return nil, err
	}
	if branch == "" {
		return nil, nil
	}
	head, err := gitOutput(repo.Root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	head = strings.TrimSpace(head)
	trunkBranch := repo.trunkBranch()

	state, err := repo.loadPendingState()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	branchState, hasBranchState := state.branch(repo.ID, repo.Root, branch)

	candidate, hasCandidate, err := repo.lastResolvedCandidate(branch, head, branchState, hasBranchState)
	if err != nil {
		return nil, err
	}
	if !hasCandidate {
		if options.Initialize {
			if err := repo.resolvePendingBranch(branch, head); err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
	lastResolved := candidate.GitState
	if lastResolved == "" || lastResolved == head {
		if err := repo.resolvePendingBranch(branch, head); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if !repo.isAncestor(lastResolved, head) {
		if err := repo.resolvePendingBranch(branch, head); err != nil {
			return nil, err
		}
		return nil, nil
	}
	commits, err := implementationCommits(repo, lastResolved, head)
	if err != nil {
		if err := repo.resolvePendingBranch(branch, head); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if len(commits) == 0 {
		if err := repo.resolvePendingBranch(branch, head); err != nil {
			return nil, err
		}
		return nil, nil
	}

	trigger := pendingTriggerIdle
	pendingSince := ""
	if branch == trunkBranch {
		trigger = pendingTriggerMerge
		pendingSince = now.UTC().Format(time.RFC3339)
	} else {
		committedAt, err := headCommitterTime(repo, head)
		if err != nil {
			return nil, err
		}
		if now.Sub(committedAt) < pendingIdleThreshold {
			return nil, nil
		}
		pendingSince = committedAt.Add(pendingIdleThreshold).UTC().Format(time.RFC3339)
	}

	pendingRange := gitRange{From: lastResolved, To: head, Commits: commits}
	if err := repo.markPendingBranch(branch, lastResolved, pendingSince, pendingRange); err != nil {
		return nil, err
	}
	return &pendingSnapshot{
		RepoID:               repo.ID,
		Branch:               branch,
		TrunkBranch:          trunkBranch,
		LastResolvedGitState: lastResolved,
		PendingSince:         pendingSince,
		Trigger:              trigger,
		IdleThreshold:        pendingIdleThreshold.String(),
		Range:                pendingRange,
	}, nil
}

func (repo repository) lastResolvedCandidate(branch string, head string, branchState pendingBranchState, hasBranchState bool) (resolvedCandidate, bool, error) {
	var candidates []resolvedCandidate
	snapshots, err := repo.listSnapshots("")
	if err != nil {
		return resolvedCandidate{}, false, err
	}
	for _, snapshot := range snapshots {
		gitState := strings.TrimSpace(snapshot.Implementation.GitState)
		if gitState == "" || strings.TrimSpace(snapshot.Implementation.Branch) != branch {
			continue
		}
		if !repo.isAncestor(gitState, head) {
			continue
		}
		distance, err := repo.commitDistance(gitState, head)
		if err != nil {
			continue
		}
		candidates = append(candidates, resolvedCandidate{GitState: gitState, Branch: branch, CreatedAt: snapshot.CreatedAt, Distance: distance})
	}
	skips, err := repo.listSkips()
	if err != nil {
		return resolvedCandidate{}, false, err
	}
	for _, skip := range skips {
		gitState := strings.TrimSpace(skip.Range.To)
		if gitState == "" || strings.TrimSpace(skip.Branch) != branch {
			continue
		}
		if !repo.isAncestor(gitState, head) {
			continue
		}
		distance, err := repo.commitDistance(gitState, head)
		if err != nil {
			continue
		}
		candidates = append(candidates, resolvedCandidate{GitState: gitState, Branch: branch, CreatedAt: skip.CreatedAt, Distance: distance})
	}
	if hasBranchState {
		gitState := strings.TrimSpace(branchState.LastResolvedGitState)
		if gitState != "" && repo.isAncestor(gitState, head) {
			distance, err := repo.commitDistance(gitState, head)
			if err == nil {
				candidates = append(candidates, resolvedCandidate{GitState: gitState, Branch: branch, Distance: distance})
			}
		}
	}
	if len(candidates) == 0 {
		return resolvedCandidate{}, false, nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Distance == candidates[j].Distance {
			return candidates[i].CreatedAt > candidates[j].CreatedAt
		}
		return candidates[i].Distance < candidates[j].Distance
	})
	return candidates[0], true, nil
}

func (repo repository) loadPendingState() (pendingStateFile, error) {
	data, err := os.ReadFile(repo.pendingStatePath())
	if err != nil {
		return pendingStateFile{}, err
	}
	var state pendingStateFile
	if err := json.Unmarshal(data, &state); err != nil {
		return pendingStateFile{}, fmt.Errorf("parse pending state: %w", err)
	}
	return state, nil
}

func (repo repository) savePendingState(state pendingStateFile) error {
	state.SchemaVersion = pendingStateVersion
	if err := os.MkdirAll(filepath.Dir(repo.pendingStatePath()), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(repo.pendingStatePath()), err)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal pending state: %w", err)
	}
	return os.WriteFile(repo.pendingStatePath(), append(data, '\n'), 0o644)
}

func (state pendingStateFile) branch(repoID string, root string, branch string) (pendingBranchState, bool) {
	for _, entry := range state.Branches {
		if entry.RepoID == repoID && entry.Root == root && entry.Branch == branch {
			return entry, true
		}
	}
	return pendingBranchState{}, false
}

func (repo repository) resolvePendingBranch(branch string, gitState string) error {
	return repo.updatePendingBranch(pendingBranchState{
		RepoID:               repo.ID,
		Root:                 repo.Root,
		Branch:               branch,
		LastResolvedGitState: strings.TrimSpace(gitState),
	})
}

func (repo repository) markPendingBranch(branch string, lastResolved string, pendingSince string, pendingRange gitRange) error {
	return repo.updatePendingBranch(pendingBranchState{
		RepoID:               repo.ID,
		Root:                 repo.Root,
		Branch:               branch,
		LastResolvedGitState: strings.TrimSpace(lastResolved),
		PendingSince:         pendingSince,
		PendingRange:         &pendingRange,
	})
}

func (repo repository) updatePendingBranch(next pendingBranchState) error {
	if strings.TrimSpace(next.Branch) == "" || strings.TrimSpace(next.LastResolvedGitState) == "" {
		return nil
	}
	state, err := repo.loadPendingState()
	if errors.Is(err, os.ErrNotExist) {
		state = pendingStateFile{}
	} else if err != nil {
		return err
	}
	updated := false
	for i, entry := range state.Branches {
		if entry.RepoID == next.RepoID && entry.Root == next.Root && entry.Branch == next.Branch {
			state.Branches[i] = next
			updated = true
			break
		}
	}
	if !updated {
		state.Branches = append(state.Branches, next)
	}
	sort.Slice(state.Branches, func(i, j int) bool {
		if state.Branches[i].Root == state.Branches[j].Root {
			return state.Branches[i].Branch < state.Branches[j].Branch
		}
		return state.Branches[i].Root < state.Branches[j].Root
	})
	return repo.savePendingState(state)
}

func (repo repository) trunkBranch() string {
	if config, err := repo.loadConfig(); err == nil && strings.TrimSpace(config.TrunkBranch) != "" {
		return strings.TrimSpace(config.TrunkBranch)
	}
	if value, err := gitOutput(repo.Root, "symbolic-ref", "refs/remotes/origin/HEAD"); err == nil {
		if branch := strings.TrimPrefix(strings.TrimSpace(value), "refs/remotes/origin/"); branch != "" && branch != value {
			return branch
		}
	}
	if repo.localBranchExists("main") {
		return "main"
	}
	if repo.localBranchExists("master") {
		return "master"
	}
	return "main"
}

type eveConfig struct {
	SchemaVersion       int    `json:"schemaVersion"`
	LegacyConfigVersion int    `json:"config_version"`
	SnapshotSchema      string `json:"snapshotSchema"`
	CreatedAt           string `json:"createdAt"`
	TrunkBranch         string `json:"trunkBranch"`
}

func (config eveConfig) effectiveSchemaVersion() int {
	if config.SchemaVersion != 0 {
		return config.SchemaVersion
	}
	if config.LegacyConfigVersion == 1 {
		return 1
	}
	return 0
}

func (repo repository) loadConfig() (eveConfig, error) {
	data, err := os.ReadFile(repo.configPath())
	if err != nil {
		return eveConfig{}, err
	}
	var config eveConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return eveConfig{}, fmt.Errorf("parse config: %w", err)
	}
	return config, nil
}

func (repo repository) localBranchExists(branch string) bool {
	_, err := gitOutput(repo.Root, "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

func currentBranch(repo repository) (string, error) {
	branch, err := gitOutput(repo.Root, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(branch), nil
}

func headCommitterTime(repo repository, head string) (time.Time, error) {
	value, err := gitOutput(repo.Root, "show", "-s", "--format=%cI", head)
	if err != nil {
		return time.Time{}, err
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

func (repo repository) isAncestor(ancestor string, descendant string) bool {
	if strings.TrimSpace(ancestor) == "" || strings.TrimSpace(descendant) == "" {
		return false
	}
	_, err := gitOutput(repo.Root, "merge-base", "--is-ancestor", strings.TrimSpace(ancestor), strings.TrimSpace(descendant))
	return err == nil
}

func (repo repository) commitDistance(from string, to string) (int, error) {
	value, err := gitOutput(repo.Root, "rev-list", "--count", strings.TrimSpace(from)+".."+strings.TrimSpace(to))
	if err != nil {
		return 0, err
	}
	var count int
	if _, err := fmt.Sscanf(value, "%d", &count); err != nil {
		return 0, err
	}
	return count, nil
}

func newSkipID() string {
	return "skip_" + strings.TrimPrefix(newSnapshotID(), "snap_")
}
