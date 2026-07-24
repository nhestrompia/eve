package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nhestrompia/eve"
)

type verificationConfig struct {
	SchemaVersion       int    `json:"schemaVersion"`
	LegacyConfigVersion int    `json:"config_version"`
	SnapshotSchema      string `json:"snapshotSchema"`
	Verification        struct {
		Checks       map[string]verificationCheck `json:"checks"`
		Suites       map[string][]string          `json:"suites"`
		ProfileRules []verificationProfileRule    `json:"profileRules"`
	} `json:"verification"`
}

type verificationCheck struct {
	Argv               []string          `json:"argv"`
	WorkingDirectory   string            `json:"workingDirectory"`
	TimeoutSeconds     int               `json:"timeoutSeconds"`
	SuccessExitCodes   []int             `json:"successExitCodes"`
	OutputLimitBytes   int               `json:"outputLimitBytes"`
	InheritEnvironment []string          `json:"inheritEnvironment"`
	Environment        map[string]string `json:"environment"`
}

type verificationProfileRule struct {
	Match   *verificationRuleMatch `json:"match,omitempty"`
	Default string                 `json:"default,omitempty"`
	Profile string                 `json:"profile,omitempty"`
}

type verificationRuleMatch struct {
	Branch string `json:"branch,omitempty"`
	Tag    string `json:"tag,omitempty"`
}

type verificationRefContext struct {
	Branch          string   `json:"branch"`
	MatchingTags    []string `json:"matchingTags"`
	MatchedRule     string   `json:"matchedRule"`
	ResolvedProfile string   `json:"resolvedProfile"`
}

type verificationRun struct {
	RunID               string                       `json:"runId"`
	SchemaVersion       string                       `json:"schemaVersion,omitempty"`
	Commit              string                       `json:"commit"`
	ConfigBlobHash      string                       `json:"configBlobHash"`
	Profile             string                       `json:"profile"`
	Suite               string                       `json:"suite"`
	RefContext          verificationRefContext       `json:"refContext"`
	ExecutorFingerprint map[string]string            `json:"executorFingerprint"`
	ActorClaim          string                       `json:"actorClaim,omitempty"`
	ActorProvenance     string                       `json:"actorProvenance,omitempty"`
	ResolvedChecks      map[string]verificationCheck `json:"resolvedChecks,omitempty"`
	ResolvedSuite       []string                     `json:"resolvedSuite,omitempty"`
	Status              string                       `json:"status"`
	DriftReason         string                       `json:"driftReason,omitempty"`
	StartedAt           string                       `json:"startedAt"`
	CompletedAt         string                       `json:"completedAt,omitempty"`
	Checks              []verificationAttempt        `json:"checks"`
}

type verificationAttempt struct {
	CheckID      string `json:"checkId"`
	Status       string `json:"status"`
	ExitCode     int    `json:"exitCode"`
	StartedAt    string `json:"startedAt"`
	CompletedAt  string `json:"completedAt,omitempty"`
	Output       string `json:"output,omitempty"`
	Stdout       string `json:"stdout,omitempty"`
	Stderr       string `json:"stderr,omitempty"`
	OutputBytes  int    `json:"outputBytes,omitempty"`
	OutputDigest string `json:"outputDigest,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
	StdoutBytes  int    `json:"stdoutBytes,omitempty"`
	StderrBytes  int    `json:"stderrBytes,omitempty"`
	StdoutDigest string `json:"stdoutDigest,omitempty"`
	StderrDigest string `json:"stderrDigest,omitempty"`
}

// legacyVerificationRun preserves the exact JSON shape used before run records
// gained their own schema version. Keep the field order and tags stable: older
// snapshots bind to the compact JSON digest of this representation.
type legacyVerificationRun struct {
	RunID               string                      `json:"runId"`
	Commit              string                      `json:"commit"`
	ConfigBlobHash      string                      `json:"configBlobHash"`
	Profile             string                      `json:"profile"`
	Suite               string                      `json:"suite"`
	RefContext          verificationRefContext      `json:"refContext"`
	ExecutorFingerprint map[string]string           `json:"executorFingerprint"`
	Status              string                      `json:"status"`
	StartedAt           string                      `json:"startedAt"`
	CompletedAt         string                      `json:"completedAt,omitempty"`
	Checks              []legacyVerificationAttempt `json:"checks"`
}

type legacyVerificationAttempt struct {
	CheckID      string `json:"checkId"`
	Status       string `json:"status"`
	ExitCode     int    `json:"exitCode,omitempty"`
	StartedAt    string `json:"startedAt"`
	CompletedAt  string `json:"completedAt,omitempty"`
	Output       string `json:"output,omitempty"`
	OutputBytes  int    `json:"outputBytes,omitempty"`
	OutputDigest string `json:"outputDigest,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
}

type capturedEvidence struct {
	Excerpt   []byte
	Bytes     int
	Digest    string
	Truncated bool
}

type redactingEvidenceWriter struct {
	mu       sync.Mutex
	patterns [][]byte
	pending  []byte
	excerpt  []byte
	limit    int
	bytes    int
	digest   hash.Hash
}

func newRedactingEvidenceWriter(limit int, values []string) *redactingEvidenceWriter {
	patterns := make([][]byte, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			patterns = append(patterns, []byte(value))
		}
	}
	sort.Slice(patterns, func(i, j int) bool { return len(patterns[i]) > len(patterns[j]) })
	return &redactingEvidenceWriter{patterns: patterns, limit: limit, digest: sha256.New()}
}

func (writer *redactingEvidenceWriter) Write(data []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	writer.pending = append(writer.pending, data...)
	writer.drain(false)
	return len(data), nil
}

func (writer *redactingEvidenceWriter) finish() capturedEvidence {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	writer.drain(true)
	return capturedEvidence{Excerpt: append([]byte{}, writer.excerpt...), Bytes: writer.bytes, Digest: "sha256:" + hex.EncodeToString(writer.digest.Sum(nil)), Truncated: writer.bytes > len(writer.excerpt)}
}

func (writer *redactingEvidenceWriter) drain(final bool) {
	for len(writer.pending) > 0 {
		matched := false
		incomplete := false
		for _, pattern := range writer.patterns {
			if len(writer.pending) >= len(pattern) && bytes.Equal(writer.pending[:len(pattern)], pattern) {
				writer.emit([]byte("[REDACTED]"))
				writer.pending = writer.pending[len(pattern):]
				matched = true
				break
			}
			if len(writer.pending) < len(pattern) && bytes.Equal(writer.pending, pattern[:len(writer.pending)]) {
				incomplete = true
			}
		}
		if matched {
			continue
		}
		if incomplete && !final {
			return
		}
		nextCandidate := len(writer.pending)
		for _, pattern := range writer.patterns {
			if index := bytes.IndexByte(writer.pending[1:], pattern[0]); index >= 0 && index+1 < nextCandidate {
				nextCandidate = index + 1
			}
		}
		if nextCandidate == 0 {
			nextCandidate = 1
		}
		writer.emit(writer.pending[:nextCandidate])
		writer.pending = writer.pending[nextCandidate:]
	}
}

func (writer *redactingEvidenceWriter) emit(data []byte) {
	_, _ = writer.digest.Write(data)
	writer.bytes += len(data)
	remaining := writer.limit - len(writer.excerpt)
	if remaining > 0 {
		writer.excerpt = append(writer.excerpt, data[:min(remaining, len(data))]...)
	}
}

type verificationRegistry struct {
	mu           sync.RWMutex
	runs         map[string]*verificationRun
	cancels      map[string]context.CancelFunc
	watcherStops map[string]context.CancelFunc
	locks        map[string]*verificationLock
}

type verificationLock struct {
	file       *os.File
	markerPath string
	token      string
}

var errVerificationEvidenceInvalid = errors.New("verification evidence is invalid")
var errVerificationLockBusy = errors.New("verification lock is busy")

const verificationRunSchemaVersion = "0.1.0"

func newVerificationRegistry() *verificationRegistry {
	return &verificationRegistry{
		runs: map[string]*verificationRun{}, cancels: map[string]context.CancelFunc{},
		watcherStops: map[string]context.CancelFunc{}, locks: map[string]*verificationLock{},
	}
}

func (repo repository) verificationRunsDir() string {
	return filepath.Join(repo.eveDir, "runs")
}

func (repo repository) verificationRunPath(id string) string {
	return filepath.Join(repo.verificationRunsDir(), id+".json")
}

func (repo repository) verificationPrivatePath(parts ...string) (string, error) {
	pathParts := append([]string{"eve"}, parts...)
	path, err := gitOutput(repo.Root, "rev-parse", "--git-path", filepath.ToSlash(filepath.Join(pathParts...)))
	if err != nil {
		return "", err
	}
	path = strings.TrimSpace(path)
	if !filepath.IsAbs(path) {
		path = filepath.Join(repo.Root, filepath.FromSlash(path))
	}
	return filepath.Clean(path), nil
}

func (repo repository) registerVerificationRun(runID string) error {
	path, err := repo.verificationPrivatePath("known-runs", runID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.WriteString(fmt.Sprintf("pid=%d\n", os.Getpid())); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	return file.Close()
}

func (repo repository) unregisterVerificationRun(runID string) {
	path, err := repo.verificationPrivatePath("known-runs", runID)
	if err == nil {
		_ = os.Remove(path)
	}
}

func (repo repository) verificationRunRegistered(runID string) bool {
	path, err := repo.verificationPrivatePath("known-runs", runID)
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (repo repository) cancellationPath(runID string) (string, error) {
	return repo.verificationPrivatePath("cancel", runID)
}

func (repo repository) requestVerificationCancellation(runID string) error {
	path, err := repo.cancellationPath(runID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(nowUTC()+"\n"), 0o600)
}

func (repo repository) loadVerificationConfig() (verificationConfig, error) {
	data, err := os.ReadFile(repo.configPath())
	if err != nil {
		return verificationConfig{}, err
	}
	return parseVerificationConfig(data)
}

func (repo repository) verificationPolicy() (verificationConfig, string, error) {
	data, err := os.ReadFile(repo.configPath())
	if err != nil {
		return verificationConfig{}, "", err
	}
	config, err := parseVerificationConfig(data)
	if err != nil {
		return verificationConfig{}, "", err
	}
	hash := sha256.Sum256(data)
	return config, "sha256:" + hex.EncodeToString(hash[:]), nil
}

func parseVerificationConfig(data []byte) (verificationConfig, error) {
	var config verificationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return verificationConfig{}, fmt.Errorf("parse verification config: %w", err)
	}
	var root struct {
		Verification json.RawMessage `json:"verification"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return verificationConfig{}, fmt.Errorf("parse verification config: %w", err)
	}
	if len(root.Verification) > 0 && string(root.Verification) != "null" {
		var strict struct {
			Checks       map[string]verificationCheck `json:"checks"`
			Suites       map[string][]string          `json:"suites"`
			ProfileRules []verificationProfileRule    `json:"profileRules"`
		}
		decoder := json.NewDecoder(bytes.NewReader(root.Verification))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&strict); err != nil {
			return verificationConfig{}, fmt.Errorf("parse verification config: %w", err)
		}
	}
	if err := validateVerificationConfig(config); err != nil {
		return verificationConfig{}, err
	}
	return config, nil
}

func validateVerificationConfig(config verificationConfig) error {
	configured := len(config.Verification.Checks) > 0 || len(config.Verification.Suites) > 0 || len(config.Verification.ProfileRules) > 0
	if config.SchemaVersion == 0 && config.LegacyConfigVersion == 1 {
		if configured {
			return errors.New("verification requires configuration schemaVersion 3; found legacy version 1")
		}
		return nil
	}
	if config.SchemaVersion == 2 {
		if config.SnapshotSchema != "0.1.0" && config.SnapshotSchema != "0.2.0" && config.SnapshotSchema != eve.SnapshotSchemaVersion {
			return fmt.Errorf("configuration schemaVersion 2 has unsupported snapshotSchema %q", config.SnapshotSchema)
		}
		if configured {
			return errors.New("verification requires configuration schemaVersion 3; found 2")
		}
		return nil
	}
	if config.SchemaVersion != 3 {
		return fmt.Errorf("unsupported EVE configuration schemaVersion %d", config.SchemaVersion)
	}
	if config.SnapshotSchema != eve.SnapshotSchemaVersion && config.SnapshotSchema != "0.2.0" {
		return fmt.Errorf("configuration schemaVersion 3 requires snapshotSchema %q or legacy-compatible %q", eve.SnapshotSchemaVersion, "0.2.0")
	}
	if !configured {
		return nil
	}
	if len(config.Verification.ProfileRules) == 0 {
		return errors.New("verification requires at least one profile rule")
	}
	defaults := 0
	for _, rule := range config.Verification.ProfileRules {
		if rule.Default != "" {
			defaults++
			if rule.Match != nil || rule.Profile != "" {
				return errors.New("verification default profile rule cannot also define match or profile")
			}
		}
		if rule.Match != nil && rule.Profile == "" {
			return errors.New("verification profile rule has no profile")
		}
		if rule.Match != nil && rule.Match.Branch == "" && rule.Match.Tag == "" {
			return errors.New("verification profile rule match must define branch or tag")
		}
		if rule.Match != nil && rule.Match.Branch != "" {
			if _, err := path.Match(rule.Match.Branch, ""); err != nil {
				return fmt.Errorf("verification profile rule has invalid branch glob %q: %w", rule.Match.Branch, err)
			}
		}
		if rule.Match != nil && rule.Match.Tag != "" {
			if _, err := path.Match(rule.Match.Tag, ""); err != nil {
				return fmt.Errorf("verification profile rule has invalid tag glob %q: %w", rule.Match.Tag, err)
			}
		}
	}
	if defaults != 1 {
		return errors.New("verification requires exactly one default profile")
	}
	for id, check := range config.Verification.Checks {
		if strings.TrimSpace(id) == "" {
			return errors.New("verification check ID cannot be empty")
		}
		if len(check.Argv) == 0 || strings.TrimSpace(check.Argv[0]) == "" {
			return fmt.Errorf("verification check %q must define argv", id)
		}
		if check.TimeoutSeconds <= 0 {
			return fmt.Errorf("verification check %q must define a positive timeoutSeconds", id)
		}
		if check.OutputLimitBytes <= 0 {
			return fmt.Errorf("verification check %q must define a positive outputLimitBytes", id)
		}
		if len(check.SuccessExitCodes) == 0 {
			return fmt.Errorf("verification check %q must define successExitCodes", id)
		}
	}
	for suite, checks := range config.Verification.Suites {
		if len(checks) == 0 {
			return fmt.Errorf("verification suite %q cannot be empty", suite)
		}
		seen := map[string]bool{}
		for _, id := range checks {
			if seen[id] {
				return fmt.Errorf("verification suite %q contains duplicate check %q", suite, id)
			}
			seen[id] = true
			if _, ok := config.Verification.Checks[id]; !ok {
				return fmt.Errorf("verification suite %q references undefined check %q", suite, id)
			}
		}
	}
	for _, rule := range config.Verification.ProfileRules {
		profile := rule.Default
		if profile == "" {
			profile = rule.Profile
		}
		if _, ok := config.Verification.Suites[profile]; !ok {
			return fmt.Errorf("verification profile %q references undefined suite", profile)
		}
	}
	return nil
}

func resolveVerificationProfile(repo repository, config verificationConfig, branch string, tags []string) verificationRefContext {
	ctx := verificationRefContext{Branch: branch, MatchingTags: append([]string{}, tags...), ResolvedProfile: ""}
	for _, rule := range config.Verification.ProfileRules {
		if rule.Default != "" {
			ctx.ResolvedProfile = rule.Default
			ctx.MatchedRule = "default"
			break
		}
	}
	for _, rule := range config.Verification.ProfileRules {
		if rule.Default != "" {
			continue
		}
		if rule.Match == nil {
			continue
		}
		if rule.Match.Branch != "" && globMatch(rule.Match.Branch, branch) {
			ctx.ResolvedProfile = rule.Profile
			ctx.MatchedRule = "branch:" + rule.Match.Branch
			break
		}
		matchedTag := false
		for _, tag := range tags {
			if rule.Match.Tag != "" && globMatch(rule.Match.Tag, tag) {
				ctx.ResolvedProfile = rule.Profile
				ctx.MatchedRule = "tag:" + rule.Match.Tag
				matchedTag = true
				break
			}
		}
		if matchedTag {
			break
		}
	}
	return ctx
}

func globMatch(pattern, value string) bool {
	matched, err := path.Match(pattern, value)
	return err == nil && matched
}

func gitTagsAt(repo repository, commit string) []string {
	output, err := gitOutput(repo.Root, "tag", "--points-at", commit)
	if err != nil || strings.TrimSpace(output) == "" {
		return []string{}
	}
	values := strings.Fields(output)
	sort.Strings(values)
	return values
}

func verificationExecutorFingerprint() map[string]string {
	return map[string]string{
		"eve": eve.CLIVersion, "os": runtime.GOOS, "arch": runtime.GOARCH,
		"go": detectedToolVersion("go", "version"), "node": detectedToolVersion("node", "--version"),
	}
}

func detectedToolVersion(name string, args ...string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return "unavailable"
	}
	output, err := exec.Command(path, args...).CombinedOutput()
	if err != nil {
		return "detection_failed"
	}
	return strings.TrimSpace(string(output))
}

func verificationRunDigest(run *verificationRun) (string, error) {
	if isLegacyVerificationRun(run) {
		data, err := json.Marshal(asLegacyVerificationRun(run))
		if err != nil {
			return "", err
		}
		hash := sha256.Sum256(data)
		return "sha256:" + hex.EncodeToString(hash[:]), nil
	}
	data, err := verificationRunCanonicalBytes(run)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

func verificationRunCanonicalBytes(run *verificationRun) ([]byte, error) {
	var value any = run
	if isLegacyVerificationRun(run) {
		value = asLegacyVerificationRun(run)
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func isLegacyVerificationRun(run *verificationRun) bool {
	return run != nil && run.SchemaVersion == "" && strings.HasPrefix(run.RunID, "snap_")
}

func asLegacyVerificationRun(run *verificationRun) legacyVerificationRun {
	checks := make([]legacyVerificationAttempt, 0, len(run.Checks))
	for _, check := range run.Checks {
		checks = append(checks, legacyVerificationAttempt{
			CheckID: check.CheckID, Status: check.Status, ExitCode: check.ExitCode,
			StartedAt: check.StartedAt, CompletedAt: check.CompletedAt, Output: check.Output,
			OutputBytes: check.OutputBytes, OutputDigest: check.OutputDigest, Truncated: check.Truncated,
		})
	}
	return legacyVerificationRun{
		RunID: run.RunID, Commit: run.Commit, ConfigBlobHash: run.ConfigBlobHash,
		Profile: run.Profile, Suite: run.Suite, RefContext: run.RefContext,
		ExecutorFingerprint: run.ExecutorFingerprint, Status: run.Status,
		StartedAt: run.StartedAt, CompletedAt: run.CompletedAt, Checks: checks,
	}
}

func (repo repository) saveVerificationRun(run *verificationRun) error {
	if err := os.MkdirAll(repo.verificationRunsDir(), 0o755); err != nil {
		return err
	}
	data, err := verificationRunCanonicalBytes(run)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(repo.verificationRunsDir(), ".run-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, repo.verificationRunPath(run.RunID))
}

func (repo repository) loadVerificationRuns() ([]*verificationRun, error) {
	entries, err := os.ReadDir(repo.verificationRunsDir())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var runs []*verificationRun
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repo.verificationRunsDir(), entry.Name()))
		if err != nil {
			return nil, err
		}
		var run verificationRun
		if err := json.Unmarshal(data, &run); err != nil {
			return nil, fmt.Errorf("%w: %s", errVerificationEvidenceInvalid, entry.Name())
		}
		canonical, canonicalErr := verificationRunCanonicalBytes(&run)
		if canonicalErr != nil || !bytes.Equal(data, canonical) || entry.Name() != run.RunID+".json" {
			return nil, fmt.Errorf("%w: %s", errVerificationEvidenceInvalid, entry.Name())
		}
		runs = append(runs, &run)
	}
	return runs, nil
}

func verificationStatusForRun(run *verificationRun, required []string) string {
	if run.Status == "cancelled" {
		return "failed"
	}
	if run.Status != "completed" {
		return "incomplete"
	}
	byID := map[string]verificationAttempt{}
	for _, attempt := range run.Checks {
		byID[attempt.CheckID] = attempt
	}
	missing := false
	failed := false
	passed := 0
	for _, id := range required {
		attempt, ok := byID[id]
		if !ok || attempt.Status == "pending" {
			missing = true
			continue
		}
		if attempt.Status != "passed" {
			failed = true
			continue
		}
		passed++
	}
	if failed {
		return "failed"
	}
	if missing {
		if passed == 0 {
			return "not_run"
		}
		return "incomplete"
	}
	return "required_checks_passed"
}

func (server runtimeServer) startVerificationRun(ctx context.Context, repo repository, commit string, suiteName string, actorClaim string) (*verificationRun, error) {
	_ = ctx
	config, configHash, err := repo.verificationPolicy()
	if err != nil {
		return nil, err
	}
	if len(config.Verification.Suites) == 0 {
		return nil, errors.New("verification is not configured")
	}
	branch, err := currentBranch(repo)
	if err != nil {
		return nil, err
	}
	head, err := gitOutput(repo.Root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(commit) == "" {
		commit = head
	}
	if strings.TrimSpace(commit) != strings.TrimSpace(head) {
		return nil, fmt.Errorf("requested commit %s is not current HEAD %s", commit, head)
	}
	clean, err := verificationTreeClean(repo, "")
	if err != nil {
		return nil, err
	}
	if !clean {
		if paths, inspectErr := unregisteredVerificationEvidencePaths(repo); inspectErr == nil && len(paths) > 0 {
			return nil, fmt.Errorf("working tree must be clean before starting verification; unregistered run evidence from an older or external EVE process requires review: %s (commit trusted evidence or remove it, then rerun the suite)", strings.Join(paths, ", "))
		}
		return nil, errors.New("working tree must be clean before starting verification")
	}
	ref := resolveVerificationProfile(repo, config, branch, gitTagsAt(repo, commit))
	if ref.ResolvedProfile == "" {
		return nil, errors.New("verification profile is not configured")
	}
	if suiteName == "" {
		suiteName = ref.ResolvedProfile
	}
	checks, ok := config.Verification.Suites[suiteName]
	if !ok {
		return nil, fmt.Errorf("verification suite %q is not configured", suiteName)
	}
	required := config.Verification.Suites[ref.ResolvedProfile]
	if !containsAll(checks, required) {
		return nil, fmt.Errorf("suite %q does not include all checks required by profile %q", suiteName, ref.ResolvedProfile)
	}
	for _, id := range checks {
		check, ok := config.Verification.Checks[id]
		if !ok || len(check.Argv) == 0 {
			return nil, fmt.Errorf("verification check %q is not defined", id)
		}
	}
	if err := os.MkdirAll(repo.verificationRunsDir(), 0o755); err != nil {
		return nil, err
	}
	runID := newRecordID("run")
	if err := repo.registerVerificationRun(runID); err != nil {
		return nil, fmt.Errorf("register verification operation: %w", err)
	}
	lock, lockErr := acquireVerificationLock(repo, runID)
	if lockErr != nil && !errors.Is(lockErr, errVerificationLockBusy) {
		repo.unregisterVerificationRun(runID)
		return nil, lockErr
	}
	resolvedChecks := make(map[string]verificationCheck, len(checks))
	pendingChecks := make([]verificationAttempt, 0, len(checks))
	for _, id := range checks {
		resolvedChecks[id] = config.Verification.Checks[id]
		pendingChecks = append(pendingChecks, verificationAttempt{CheckID: id, Status: "pending"})
	}
	run := &verificationRun{
		RunID: runID, SchemaVersion: verificationRunSchemaVersion, Commit: strings.TrimSpace(commit), ConfigBlobHash: configHash,
		Profile: ref.ResolvedProfile, Suite: suiteName, RefContext: ref,
		ExecutorFingerprint: verificationExecutorFingerprint(), ActorClaim: strings.TrimSpace(actorClaim), ActorProvenance: "unauthenticated", ResolvedChecks: resolvedChecks, ResolvedSuite: append([]string{}, checks...), Status: "running", StartedAt: nowUTC(),
		Checks: pendingChecks,
	}
	if errors.Is(lockErr, errVerificationLockBusy) {
		run.Status = "queued"
	}
	runCtx, cancel := context.WithCancel(context.Background())
	watcherCtx, stopWatcher := context.WithCancel(context.Background())
	server.verificationRegistry.mu.Lock()
	server.verificationRegistry.runs[run.RunID] = run
	server.verificationRegistry.cancels[run.RunID] = cancel
	server.verificationRegistry.watcherStops[run.RunID] = stopWatcher
	if lock != nil {
		server.verificationRegistry.locks[run.RunID] = lock
	}
	server.verificationRegistry.mu.Unlock()
	go watchVerificationCancellation(watcherCtx, repo, run.RunID, cancel)
	if err := repo.saveVerificationRun(run); err != nil {
		server.verificationRegistry.mu.Lock()
		delete(server.verificationRegistry.runs, run.RunID)
		delete(server.verificationRegistry.cancels, run.RunID)
		delete(server.verificationRegistry.watcherStops, run.RunID)
		delete(server.verificationRegistry.locks, run.RunID)
		server.verificationRegistry.mu.Unlock()
		stopWatcher()
		repo.unregisterVerificationRun(run.RunID)
		if lock != nil {
			releaseVerificationLock(lock)
		}
		return nil, err
	}
	result, err := cloneVerificationRun(run)
	if err != nil {
		server.verificationRegistry.mu.Lock()
		delete(server.verificationRegistry.runs, run.RunID)
		delete(server.verificationRegistry.cancels, run.RunID)
		delete(server.verificationRegistry.watcherStops, run.RunID)
		delete(server.verificationRegistry.locks, run.RunID)
		server.verificationRegistry.mu.Unlock()
		stopWatcher()
		repo.unregisterVerificationRun(run.RunID)
		if lock != nil {
			releaseVerificationLock(lock)
		}
		return nil, err
	}
	if run.Status == "queued" {
		go server.awaitVerificationLock(runCtx, repo, run, config, checks)
	} else {
		go server.executeVerificationRun(runCtx, repo, run, config, checks)
	}
	return result, nil
}

func acquireVerificationLock(repo repository, runID string) (*verificationLock, error) {
	lockPath, err := repo.verificationPrivatePath("verification.lock")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return nil, err
	}
	lockFile, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open verification lock: %w", err)
	}
	if err := tryVerificationFileLock(lockFile); err != nil {
		_ = lockFile.Close()
		if errors.Is(err, errVerificationLockBusy) {
			return nil, err
		}
		return nil, fmt.Errorf("acquire verification lock: %w", err)
	}
	token := newRecordID("lock")
	marker := fmt.Sprintf("pid=%d\nrunId=%s\ntoken=%s\n", os.Getpid(), runID, token)
	markerPath := filepath.Join(repo.verificationRunsDir(), ".active.lock")
	markerFile, err := createVerificationMarker(markerPath)
	if err != nil {
		_ = unlockVerificationFile(lockFile)
		_ = lockFile.Close()
		return nil, err
	}
	if _, err := markerFile.WriteString(marker); err != nil {
		_ = markerFile.Close()
		_ = os.Remove(markerPath)
		_ = unlockVerificationFile(lockFile)
		_ = lockFile.Close()
		return nil, err
	}
	if err := markerFile.Close(); err != nil {
		_ = os.Remove(markerPath)
		_ = unlockVerificationFile(lockFile)
		_ = lockFile.Close()
		return nil, err
	}
	return &verificationLock{file: lockFile, markerPath: markerPath, token: token}, nil
}

func createVerificationMarker(path string) (*os.File, error) {
	for attempt := 0; attempt < 2; attempt++ {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return file, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		data, readErr := os.ReadFile(path)
		pid := 0
		if readErr == nil {
			_, _ = fmt.Sscanf(strings.TrimSpace(string(data)), "pid=%d", &pid)
		}
		if pid > 0 && verificationProcessActive(pid) {
			return nil, errVerificationLockBusy
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, errVerificationLockBusy
		}
	}
	return nil, errVerificationLockBusy
}

func releaseVerificationLock(lock *verificationLock) {
	if lock == nil {
		return
	}
	if data, err := os.ReadFile(lock.markerPath); err == nil && strings.Contains(string(data), "token="+lock.token+"\n") {
		_ = os.Remove(lock.markerPath)
	}
	_ = unlockVerificationFile(lock.file)
	_ = lock.file.Close()
}

func watchVerificationCancellation(ctx context.Context, repo repository, runID string, cancel context.CancelFunc) {
	path, err := repo.cancellationPath(runID)
	if err != nil {
		return
	}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(path); err == nil {
			cancel()
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (server runtimeServer) awaitVerificationLock(ctx context.Context, repo repository, run *verificationRun, config verificationConfig, checks []string) {
	for {
		if ctx.Err() != nil {
			server.verificationRegistry.mu.Lock()
			markVerificationRunCancelled(run)
			server.verificationRegistry.mu.Unlock()
			_ = server.persistVerificationRun(repo, run)
			server.cleanupVerificationRun(repo, run.RunID)
			return
		}
		lock, err := acquireVerificationLock(repo, run.RunID)
		if err == nil {
			server.verificationRegistry.mu.Lock()
			server.verificationRegistry.locks[run.RunID] = lock
			run.Status = "running"
			run.StartedAt = nowUTC()
			server.verificationRegistry.mu.Unlock()
			if reason := verificationDriftReason(repo, run); reason != "" {
				server.verificationRegistry.mu.Lock()
				run.Status, run.DriftReason, run.CompletedAt = "invalidated", reason, nowUTC()
				server.verificationRegistry.mu.Unlock()
				_ = server.persistVerificationRun(repo, run)
				server.cleanupVerificationRun(repo, run.RunID)
				return
			}
			_ = server.persistVerificationRun(repo, run)
			server.executeVerificationRun(ctx, repo, run, config, checks)
			return
		}
		if !errors.Is(err, errVerificationLockBusy) {
			server.verificationRegistry.mu.Lock()
			run.Status, run.DriftReason, run.CompletedAt = "invalidated", err.Error(), nowUTC()
			server.verificationRegistry.mu.Unlock()
			_ = server.persistVerificationRun(repo, run)
			server.cleanupVerificationRun(repo, run.RunID)
			return
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}

func containsAll(have, required []string) bool {
	set := map[string]bool{}
	for _, value := range have {
		set[value] = true
	}
	for _, value := range required {
		if !set[value] {
			return false
		}
	}
	return true
}

func (server runtimeServer) executeVerificationRun(ctx context.Context, repo repository, run *verificationRun, config verificationConfig, checks []string) {
	persistenceFailure := ""
	for _, id := range checks {
		if ctx.Err() != nil {
			server.verificationRegistry.mu.Lock()
			for i := range run.Checks {
				if run.Checks[i].CheckID == id && run.Checks[i].Status == "pending" {
					now := nowUTC()
					run.Checks[i].Status, run.Checks[i].StartedAt, run.Checks[i].CompletedAt = "cancelled", now, now
				}
			}
			server.verificationRegistry.mu.Unlock()
			continue
		}
		check := config.Verification.Checks[id]
		server.verificationRegistry.mu.Lock()
		for i := range run.Checks {
			if run.Checks[i].CheckID == id && run.Checks[i].Status == "pending" {
				run.Checks[i].Status = "running"
				run.Checks[i].StartedAt = nowUTC()
			}
		}
		server.verificationRegistry.mu.Unlock()
		commandCtx := ctx
		cancel := func() {}
		if check.TimeoutSeconds > 0 {
			commandCtx, cancel = context.WithTimeout(ctx, time.Duration(check.TimeoutSeconds)*time.Second)
		}
		cmd := exec.CommandContext(commandCtx, check.Argv[0], check.Argv[1:]...)
		configureVerificationProcess(cmd)
		cmd.Dir = repo.Root
		if check.WorkingDirectory != "" {
			workingDirectory := filepath.Clean(check.WorkingDirectory)
			if filepath.IsAbs(workingDirectory) || workingDirectory == ".." || strings.HasPrefix(workingDirectory, ".."+string(filepath.Separator)) {
				server.markVerificationAttempt(run, id, "execution_error", -1, "working directory must stay inside repository")
				cancel()
				continue
			}
			cmd.Dir = filepath.Join(repo.Root, workingDirectory)
			resolvedRoot, rootErr := filepath.EvalSymlinks(repo.Root)
			resolved, resolveErr := filepath.EvalSymlinks(cmd.Dir)
			if rootErr != nil || resolveErr != nil || !pathWithinRoot(resolvedRoot, resolved) {
				server.markVerificationAttempt(run, id, "execution_error", -1, "working directory must stay inside repository")
				cancel()
				continue
			}
		}
		allowed := map[string]bool{}
		for _, key := range check.InheritEnvironment {
			allowed[key] = true
		}
		env := filterEnvironment(os.Environ(), allowed)
		for key, value := range check.Environment {
			env = append(env, key+"="+value)
		}
		cmd.Env = env
		redactionValues := configuredEnvironmentValues(check, os.Environ())
		stdoutCapture := newRedactingEvidenceWriter(check.OutputLimitBytes, redactionValues)
		stderrCapture := newRedactingEvidenceWriter(check.OutputLimitBytes, redactionValues)
		cmd.Stdout, cmd.Stderr = stdoutCapture, stderrCapture
		err := cmd.Run()
		contextErr := commandCtx.Err()
		cancel()
		exitCode := commandExitCode(err)
		if err == nil {
			exitCode = 0
		}
		status := "failed"
		if contextErr == context.DeadlineExceeded {
			status = "timed_out"
			exitCode = -1
		} else if contextErr == context.Canceled {
			status = "cancelled"
			exitCode = -1
		} else if isExecutionError(err) {
			status = "execution_error"
		} else if acceptedExitCode(exitCode, check.SuccessExitCodes) {
			status = "passed"
		}
		limit := check.OutputLimitBytes
		stdoutEvidence, stderrEvidence := stdoutCapture.finish(), stderrCapture.finish()
		fullOutputBytes := stdoutEvidence.Bytes + stderrEvidence.Bytes
		truncated := fullOutputBytes > limit
		stdoutExcerpt := boundedPrefix(stdoutEvidence.Excerpt, limit)
		stderrExcerpt := boundedPrefix(stderrEvidence.Excerpt, limit-len(stdoutExcerpt))
		output := append(append([]byte{}, stdoutExcerpt...), stderrExcerpt...)
		attemptOutput := string(output)
		combinedHash := sha256.Sum256([]byte(stdoutEvidence.Digest + "\n" + stderrEvidence.Digest))
		server.verificationRegistry.mu.Lock()
		for i := range run.Checks {
			if run.Checks[i].CheckID == id && run.Checks[i].Status == "running" {
				run.Checks[i].Status, run.Checks[i].ExitCode = status, exitCode
				run.Checks[i].Output, run.Checks[i].OutputBytes = attemptOutput, fullOutputBytes
				run.Checks[i].Stdout, run.Checks[i].Stderr = string(stdoutExcerpt), string(stderrExcerpt)
				run.Checks[i].StdoutBytes, run.Checks[i].StderrBytes = stdoutEvidence.Bytes, stderrEvidence.Bytes
				run.Checks[i].StdoutDigest, run.Checks[i].StderrDigest = stdoutEvidence.Digest, stderrEvidence.Digest
				run.Checks[i].OutputDigest, run.Checks[i].Truncated = "sha256:"+hex.EncodeToString(combinedHash[:]), truncated
				run.Checks[i].CompletedAt = nowUTC()
				break
			}
		}
		server.verificationRegistry.mu.Unlock()
		if err := server.persistVerificationRun(repo, run); err != nil {
			persistenceFailure = "persist verification evidence: " + err.Error()
			break
		}
	}
	driftReason := verificationDriftReason(repo, run)
	server.verificationRegistry.mu.Lock()
	if persistenceFailure != "" {
		run.Status = "invalidated"
		run.DriftReason = persistenceFailure
	} else if driftReason != "" {
		run.Status = "invalidated"
		run.DriftReason = driftReason
		for i := range run.Checks {
			if run.Checks[i].Status == "running" || run.Checks[i].Status == "pending" {
				run.Checks[i].Status = "invalidated"
			}
		}
	} else if ctx.Err() != nil {
		run.Status = "cancelled"
	} else {
		run.Status = "completed"
	}
	run.CompletedAt = nowUTC()
	server.verificationRegistry.mu.Unlock()
	if err := server.persistVerificationRun(repo, run); err != nil {
		server.verificationRegistry.mu.Lock()
		run.Status = "invalidated"
		run.DriftReason = "persist terminal verification evidence: " + err.Error()
		server.verificationRegistry.mu.Unlock()
		_ = server.persistVerificationRun(repo, run)
	}
	server.cleanupVerificationRun(repo, run.RunID)
}

func (server runtimeServer) cleanupVerificationRun(repo repository, runID string) {
	server.verificationRegistry.mu.Lock()
	lock := server.verificationRegistry.locks[runID]
	stopWatcher := server.verificationRegistry.watcherStops[runID]
	delete(server.verificationRegistry.cancels, runID)
	delete(server.verificationRegistry.watcherStops, runID)
	delete(server.verificationRegistry.locks, runID)
	server.verificationRegistry.mu.Unlock()
	if stopWatcher != nil {
		stopWatcher()
	}
	if path, err := repo.cancellationPath(runID); err == nil {
		_ = os.Remove(path)
	}
	if lock != nil {
		releaseVerificationLock(lock)
	}
}

func isExecutionError(err error) bool {
	if err == nil {
		return false
	}
	var execErr *exec.Error
	var pathErr *os.PathError
	return errors.As(err, &execErr) || errors.As(err, &pathErr)
}

func (server runtimeServer) persistVerificationRun(repo repository, run *verificationRun) error {
	server.verificationRegistry.mu.RLock()
	copy, err := cloneVerificationRun(run)
	server.verificationRegistry.mu.RUnlock()
	if err != nil {
		return err
	}
	return repo.saveVerificationRun(copy)
}

func cloneVerificationRun(run *verificationRun) (*verificationRun, error) {
	data, err := json.Marshal(run)
	if err != nil {
		return nil, err
	}
	var copy verificationRun
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil, err
	}
	return &copy, nil
}

func markVerificationRunCancelled(run *verificationRun) {
	now := nowUTC()
	seen := map[string]bool{}
	for i := range run.Checks {
		seen[run.Checks[i].CheckID] = true
		if run.Checks[i].Status == "running" || run.Checks[i].Status == "pending" {
			run.Checks[i].Status = "cancelled"
			run.Checks[i].CompletedAt = now
		}
	}
	for _, id := range run.ResolvedSuite {
		if !seen[id] {
			run.Checks = append(run.Checks, verificationAttempt{CheckID: id, Status: "cancelled", StartedAt: now, CompletedAt: now})
		}
	}
	run.Status = "cancelled"
	run.CompletedAt = now
}

func boundedPrefix(value []byte, limit int) []byte {
	if limit <= 0 {
		return nil
	}
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func acceptedExitCode(exitCode int, accepted []int) bool {
	if len(accepted) == 0 {
		return exitCode == 0
	}
	for _, code := range accepted {
		if exitCode == code {
			return true
		}
	}
	return false
}

func pathWithinRoot(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func configuredEnvironmentValues(check verificationCheck, parent []string) []string {
	values := make([]string, 0, len(check.Environment)+len(check.InheritEnvironment))
	for _, key := range check.InheritEnvironment {
		for _, entry := range parent {
			if strings.HasPrefix(entry, key+"=") {
				values = append(values, strings.TrimPrefix(entry, key+"="))
			}
		}
	}
	for _, value := range check.Environment {
		values = append(values, value)
	}
	return values
}

func verificationDriftReason(repo repository, run *verificationRun) string {
	head, err := gitOutput(repo.Root, "rev-parse", "HEAD")
	if err != nil || strings.TrimSpace(head) != run.Commit {
		return "HEAD changed during verification"
	}
	clean, err := verificationTreeClean(repo, run.RunID)
	if err != nil {
		return "unable to inspect working tree after verification"
	}
	if !clean {
		return "tracked or untracked files changed during verification"
	}
	config, configHash, err := repo.verificationPolicy()
	if err != nil || configHash != run.ConfigBlobHash {
		return "verification policy changed during execution"
	}
	branch, err := currentBranch(repo)
	if err != nil {
		return "unable to resolve verification branch after execution"
	}
	ref := resolveVerificationProfile(repo, config, branch, gitTagsAt(repo, head))
	if ref.Branch != run.RefContext.Branch || ref.ResolvedProfile != run.RefContext.ResolvedProfile || ref.MatchedRule != run.RefContext.MatchedRule || !equalStrings(ref.MatchingTags, run.RefContext.MatchingTags) {
		return "verification reference context changed during execution"
	}
	return ""
}

func verificationTreeClean(repo repository, activeRunID string) (bool, error) {
	status, err := gitOutput(repo.Root, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if verificationStatusLineAllowed(repo, line, activeRunID) {
			continue
		}
		return false, nil
	}
	return true, nil
}

func verificationNewEvidencePaths(repo repository) ([]string, error) {
	status, err := gitOutput(repo.Root, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
		if verificationStatusLineAllowed(repo, line, "") {
			path := filepath.ToSlash(strings.Trim(strings.TrimSpace(line[3:]), `"`))
			if filepath.Ext(path) == ".json" {
				paths = append(paths, path)
			}
		}
	}
	return paths, nil
}

func unregisteredVerificationEvidencePaths(repo repository) ([]string, error) {
	status, err := gitOutput(repo.Root, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	prefix := filepath.ToSlash(filepath.Join(".eve", "runs")) + "/"
	var paths []string
	for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
		if len(line) < 4 || line[:2] != "??" {
			continue
		}
		path := filepath.ToSlash(strings.Trim(strings.TrimSpace(line[3:]), `"`))
		if !strings.HasPrefix(path, prefix) || filepath.Ext(path) != ".json" {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(repo.Root, filepath.FromSlash(path)))
		if readErr != nil {
			continue
		}
		var run verificationRun
		if json.Unmarshal(data, &run) != nil || filepath.Base(path) != run.RunID+".json" {
			continue
		}
		current := run.SchemaVersion == verificationRunSchemaVersion && strings.HasPrefix(run.RunID, "run_")
		if !current && !isLegacyVerificationRun(&run) {
			continue
		}
		canonical, canonicalErr := verificationRunCanonicalBytes(&run)
		if canonicalErr == nil && bytes.Equal(data, canonical) && !repo.verificationRunRegistered(run.RunID) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func verificationPolicyChange(repo repository, baseCommit string, currentRef verificationRefContext, currentRequired []string, currentHash string) (*eve.PolicyChange, error) {
	change := &eve.PolicyChange{CurrentConfigHash: currentHash, AddedChecks: []string{}, RemovedChecks: []string{}}
	if strings.TrimSpace(baseCommit) == "" {
		change.Changed = true
		change.PolicyIntroduced = true
		change.ProfileIntroduced = true
		change.AddedChecks = append(change.AddedChecks, currentRequired...)
		return change, nil
	}
	data, exists, err := gitFileAtCommit(repo, baseCommit, ".eve/config.json")
	if err != nil {
		return nil, err
	}
	if !exists {
		change.Changed = true
		change.PolicyIntroduced = true
		change.ProfileIntroduced = true
		change.AddedChecks = append(change.AddedChecks, currentRequired...)
		return change, nil
	}
	hash := sha256.Sum256(data)
	change.PreviousConfigHash = "sha256:" + hex.EncodeToString(hash[:])
	change.Changed = change.PreviousConfigHash != currentHash
	previous, err := parseVerificationConfig(data)
	if err != nil {
		return nil, fmt.Errorf("validate verification policy at base commit %s: %w", baseCommit, err)
	}
	previousRef := resolveVerificationProfile(repo, previous, currentRef.Branch, gitTagsAt(repo, baseCommit))
	previousRequired := previous.Verification.Suites[previousRef.ResolvedProfile]
	if len(previousRequired) == 0 && len(currentRequired) > 0 {
		change.ProfileIntroduced = true
	}
	if len(previousRequired) > 0 && len(currentRequired) == 0 {
		change.ProfileRemoved = true
		change.RequirementsReduced = true
	}
	change.AddedChecks, change.RemovedChecks = checkSetDifference(previousRequired, currentRequired)
	change.RequirementsReduced = change.RequirementsReduced || len(change.RemovedChecks) > 0
	return change, nil
}

func gitFileAtCommit(repo repository, commit, path string) ([]byte, bool, error) {
	spec := commit + ":" + path
	check := exec.Command("git", "cat-file", "-e", spec)
	check.Dir = repo.Root
	if err := check.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, false, nil
		}
		return nil, false, err
	}
	cmd := exec.Command("git", "show", spec)
	cmd.Dir = repo.Root
	data, err := cmd.Output()
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func checkSetDifference(previous, current []string) (added, removed []string) {
	previousSet, currentSet := map[string]bool{}, map[string]bool{}
	for _, id := range previous {
		previousSet[id] = true
	}
	for _, id := range current {
		currentSet[id] = true
	}
	for _, id := range current {
		if !previousSet[id] {
			added = append(added, id)
		}
	}
	for _, id := range previous {
		if !currentSet[id] {
			removed = append(removed, id)
		}
	}
	return added, removed
}

func verificationStatusLineAllowed(repo repository, line, _ string) bool {
	if len(line) < 4 || line[:2] != "??" {
		return false
	}
	path := filepath.ToSlash(strings.Trim(strings.TrimSpace(line[3:]), `"`))
	if path == filepath.ToSlash(filepath.Join(".eve", "runs", ".active.lock")) {
		data, err := os.ReadFile(filepath.Join(repo.Root, filepath.FromSlash(path)))
		if err != nil {
			return false
		}
		pid := 0
		_, scanErr := fmt.Sscanf(strings.TrimSpace(string(data)), "pid=%d", &pid)
		return scanErr == nil && pid > 0 && verificationProcessActive(pid)
	}
	prefix := filepath.ToSlash(filepath.Join(".eve", "runs")) + "/"
	if !strings.HasPrefix(path, prefix) || filepath.Ext(path) != ".json" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(repo.Root, filepath.FromSlash(path)))
	if err != nil {
		return false
	}
	var run verificationRun
	if json.Unmarshal(data, &run) != nil || filepath.Base(path) != run.RunID+".json" {
		return false
	}
	current := run.SchemaVersion == verificationRunSchemaVersion && strings.HasPrefix(run.RunID, "run_") && run.ActorProvenance == "unauthenticated" && len(run.ResolvedSuite) > 0 && len(run.ResolvedChecks) > 0
	if !current || !repo.verificationRunRegistered(run.RunID) {
		return false
	}
	canonical, err := verificationRunCanonicalBytes(&run)
	if err != nil || !bytes.Equal(data, canonical) {
		return false
	}
	return true
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func filterEnvironment(values []string, allowed map[string]bool) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		key := value
		if index := strings.IndexByte(value, '='); index >= 0 {
			key = value[:index]
		}
		if allowed[key] {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func (server runtimeServer) markVerificationAttempt(run *verificationRun, id, status string, exitCode int, output string) {
	server.verificationRegistry.mu.Lock()
	defer server.verificationRegistry.mu.Unlock()
	for i := range run.Checks {
		if run.Checks[i].CheckID == id && run.Checks[i].Status == "running" {
			run.Checks[i].Status, run.Checks[i].ExitCode = status, exitCode
			run.Checks[i].Output, run.Checks[i].OutputBytes = output, len(output)
			run.Checks[i].CompletedAt = nowUTC()
			return
		}
	}
}

func commandExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func (server runtimeServer) verificationRun(repo repository, id string) (*verificationRun, error) {
	server.verificationRegistry.mu.RLock()
	run := server.verificationRegistry.runs[id]
	if run != nil {
		copy, err := cloneVerificationRun(run)
		server.verificationRegistry.mu.RUnlock()
		return copy, err
	}
	server.verificationRegistry.mu.RUnlock()
	data, err := os.ReadFile(repo.verificationRunPath(id))
	if err != nil {
		return nil, err
	}
	var loaded verificationRun
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil, err
	}
	return &loaded, nil
}

func (server runtimeServer) latestVerificationRun(repo repository, commit, profile, configHash string, ref verificationRefContext) (*verificationRun, error) {
	runs, err := repo.loadVerificationRuns()
	if err != nil {
		return nil, err
	}
	var selected *verificationRun
	for _, run := range runs {
		if run.Commit != commit || run.Profile != profile || run.ConfigBlobHash != configHash || !equalVerificationRefContext(run.RefContext, ref) || run.Status == "invalidated" || run.Status == "running" || run.Status == "queued" {
			continue
		}
		if selected == nil || run.CompletedAt > selected.CompletedAt || (run.CompletedAt == selected.CompletedAt && run.RunID > selected.RunID) {
			selected = run
		}
	}
	return selected, nil
}

func equalVerificationRefContext(left, right verificationRefContext) bool {
	return left.Branch == right.Branch &&
		left.MatchedRule == right.MatchedRule &&
		left.ResolvedProfile == right.ResolvedProfile &&
		equalStrings(left.MatchingTags, right.MatchingTags)
}

func verifySnapshotRunEvidence(repo repository, snapshot *eve.Snapshot) {
	if snapshot == nil || snapshot.Verification == nil {
		return
	}
	missingRunID := snapshot.Verification.SelectedRunID == ""
	missingDigest := snapshot.Verification.RunRecordDigest == ""
	if missingRunID && missingDigest && snapshot.Verification.Status != "required_checks_passed" {
		return
	}
	if missingRunID || missingDigest {
		snapshot.Verification.Status = "failed"
		snapshot.Verification.Integrity = "evidence_binding_missing"
		return
	}
	data, err := os.ReadFile(repo.verificationRunPath(snapshot.Verification.SelectedRunID))
	if err != nil {
		snapshot.Verification.Status = "failed"
		snapshot.Verification.Integrity = "evidence_missing"
		return
	}
	var run verificationRun
	if json.Unmarshal(data, &run) != nil || run.RunID != snapshot.Verification.SelectedRunID {
		snapshot.Verification.Status = "failed"
		snapshot.Verification.Integrity = "evidence_invalid"
		return
	}
	canonical, canonicalErr := verificationRunCanonicalBytes(&run)
	if canonicalErr != nil || !bytes.Equal(data, canonical) {
		snapshot.Verification.Status = "failed"
		snapshot.Verification.Integrity = "evidence_digest_mismatch"
		return
	}
	digest, err := verificationRunDigest(&run)
	if err != nil || digest != snapshot.Verification.RunRecordDigest {
		snapshot.Verification.Status = "failed"
		snapshot.Verification.Integrity = "evidence_digest_mismatch"
		return
	}
	if run.Commit != snapshot.Implementation.GitState || run.Profile != snapshot.Verification.Profile || run.ConfigBlobHash != snapshot.Verification.ConfigBlobHash || verificationStatusForRun(&run, snapshot.Verification.RequiredChecks) != snapshot.Verification.Status {
		snapshot.Verification.Status = "failed"
		snapshot.Verification.Integrity = "evidence_context_mismatch"
		return
	}
	snapshot.Verification.Integrity = "matched"
}

func verificationFromRun(run *verificationRun, required []string) *eve.Verification {
	if run == nil {
		return nil
	}
	ran := make([]string, 0, len(run.Checks))
	results := make([]eve.VerificationCheckResult, 0, len(run.Checks))
	for _, check := range run.Checks {
		ran = append(ran, check.CheckID)
		results = append(results, eve.VerificationCheckResult{CheckID: check.CheckID, Status: check.Status, ExitCode: check.ExitCode, StartedAt: check.StartedAt, CompletedAt: check.CompletedAt, Output: check.Output, OutputBytes: check.OutputBytes, OutputDigest: check.OutputDigest, Truncated: check.Truncated})
	}
	digest, _ := verificationRunDigest(run)
	executor := map[string]string{}
	for key, value := range run.ExecutorFingerprint {
		executor[key] = value
	}
	return &eve.Verification{Status: verificationStatusForRun(run, required), Profile: run.Profile, Suite: run.Suite, RequiredChecks: required, RanChecks: ran, CheckResults: results, SelectedRunID: run.RunID, RunStartedAt: run.StartedAt, RunCompletedAt: run.CompletedAt, RunRecordDigest: digest, ConfigBlobHash: run.ConfigBlobHash, ExecutorFingerprint: executor, Integrity: "matched"}
}
