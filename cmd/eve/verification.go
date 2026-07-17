package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nhestrompia/eve"
)

type verificationConfig struct {
	SchemaVersion int `json:"schemaVersion"`
	Verification  struct {
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
	RunID               string                 `json:"runId"`
	Commit              string                 `json:"commit"`
	ConfigBlobHash      string                 `json:"configBlobHash"`
	Profile             string                 `json:"profile"`
	Suite               string                 `json:"suite"`
	RefContext          verificationRefContext `json:"refContext"`
	ExecutorFingerprint map[string]string      `json:"executorFingerprint"`
	Status              string                 `json:"status"`
	StartedAt           string                 `json:"startedAt"`
	CompletedAt         string                 `json:"completedAt,omitempty"`
	Checks              []verificationAttempt  `json:"checks"`
}

type verificationAttempt struct {
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

type verificationRegistry struct {
	mu      sync.RWMutex
	runs    map[string]*verificationRun
	cancels map[string]context.CancelFunc
}

func newVerificationRegistry() *verificationRegistry {
	return &verificationRegistry{runs: map[string]*verificationRun{}, cancels: map[string]context.CancelFunc{}}
}

func (repo repository) verificationRunsDir() string {
	return filepath.Join(repo.eveDir, "runs")
}

func (repo repository) verificationRunPath(id string) string {
	return filepath.Join(repo.verificationRunsDir(), id+".json")
}

func (repo repository) loadVerificationConfig() (verificationConfig, error) {
	data, err := os.ReadFile(repo.configPath())
	if err != nil {
		return verificationConfig{}, err
	}
	var config verificationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return verificationConfig{}, fmt.Errorf("parse verification config: %w", err)
	}
	return config, nil
}

func (repo repository) verificationPolicy() (verificationConfig, string, error) {
	config, err := repo.loadVerificationConfig()
	if err != nil {
		return verificationConfig{}, "", err
	}
	data, err := os.ReadFile(repo.configPath())
	if err != nil {
		return verificationConfig{}, "", err
	}
	hash := sha256.Sum256(data)
	return config, "sha256:" + hex.EncodeToString(hash[:]), nil
}

func resolveVerificationProfile(repo repository, config verificationConfig, branch string, tags []string) verificationRefContext {
	ctx := verificationRefContext{Branch: branch, MatchingTags: append([]string{}, tags...), ResolvedProfile: ""}
	for _, rule := range config.Verification.ProfileRules {
		if rule.Default != "" {
			if ctx.ResolvedProfile == "" {
				ctx.ResolvedProfile = rule.Default
				ctx.MatchedRule = "default"
			}
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
		for _, tag := range tags {
			if rule.Match.Tag != "" && globMatch(rule.Match.Tag, tag) {
				ctx.ResolvedProfile = rule.Profile
				ctx.MatchedRule = "tag:" + rule.Match.Tag
				break
			}
		}
		if ctx.ResolvedProfile != "" {
			break
		}
	}
	return ctx
}

func globMatch(pattern, value string) bool {
	matched, err := filepath.Match(pattern, value)
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
	return map[string]string{"eve": eve.CLIVersion, "os": runtime.GOOS, "arch": runtime.GOARCH}
}

func verificationRunDigest(run *verificationRun) (string, error) {
	data, err := json.Marshal(run)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}

func (repo repository) saveVerificationRun(run *verificationRun) error {
	if err := os.MkdirAll(repo.verificationRunsDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(repo.verificationRunsDir(), ".run-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(append(data, '\n')); err != nil {
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
			continue
		}
		var run verificationRun
		if json.Unmarshal(data, &run) == nil {
			runs = append(runs, &run)
		}
	}
	return runs, nil
}

func verificationStatusForRun(run *verificationRun, required []string) string {
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

func (server runtimeServer) startVerificationRun(ctx context.Context, repo repository, commit string, suiteName string) (*verificationRun, error) {
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
	status, err := gitOutput(repo.Root, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
		if strings.TrimSpace(line) != "" && !gitStatusLineIgnored(line, []string{".eve/runs"}) {
			return nil, errors.New("working tree must be clean before starting verification")
		}
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
	run := &verificationRun{
		RunID: newSnapshotID(), Commit: strings.TrimSpace(commit), ConfigBlobHash: configHash,
		Profile: ref.ResolvedProfile, Suite: suiteName, RefContext: ref,
		ExecutorFingerprint: verificationExecutorFingerprint(), Status: "running", StartedAt: nowUTC(),
		Checks: make([]verificationAttempt, 0, len(checks)),
	}
	runCtx, cancel := context.WithCancel(context.Background())
	server.verificationRegistry.mu.Lock()
	server.verificationRegistry.runs[run.RunID] = run
	server.verificationRegistry.cancels[run.RunID] = cancel
	server.verificationRegistry.mu.Unlock()
	if err := repo.saveVerificationRun(run); err != nil {
		return nil, err
	}
	go server.executeVerificationRun(runCtx, repo, run, config, checks)
	return run, nil
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
	for _, id := range checks {
		if ctx.Err() != nil {
			server.verificationRegistry.mu.Lock()
			run.Checks = append(run.Checks, verificationAttempt{CheckID: id, Status: "cancelled", StartedAt: nowUTC(), CompletedAt: nowUTC()})
			server.verificationRegistry.mu.Unlock()
			continue
		}
		check := config.Verification.Checks[id]
		attempt := verificationAttempt{CheckID: id, Status: "running", StartedAt: nowUTC()}
		server.verificationRegistry.mu.Lock()
		run.Checks = append(run.Checks, attempt)
		server.verificationRegistry.mu.Unlock()
		commandCtx := ctx
		cancel := func() {}
		if check.TimeoutSeconds > 0 {
			commandCtx, cancel = context.WithTimeout(ctx, time.Duration(check.TimeoutSeconds)*time.Second)
		}
		cmd := exec.CommandContext(commandCtx, check.Argv[0], check.Argv[1:]...)
		cmd.Dir = repo.Root
		if check.WorkingDirectory != "" {
			workingDirectory := filepath.Clean(check.WorkingDirectory)
			if filepath.IsAbs(workingDirectory) || workingDirectory == ".." || strings.HasPrefix(workingDirectory, ".."+string(filepath.Separator)) {
				server.markVerificationAttempt(run, id, "failed", -1, "working directory must stay inside repository")
				cancel()
				continue
			}
			cmd.Dir = filepath.Join(repo.Root, workingDirectory)
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
		output, err := cmd.CombinedOutput()
		contextErr := commandCtx.Err()
		cancel()
		status := "passed"
		exitCode := 0
		if contextErr == context.DeadlineExceeded {
			status = "timed_out"
			exitCode = -1
		} else if contextErr == context.Canceled {
			status = "cancelled"
			exitCode = -1
		} else if err != nil {
			status = "failed"
			exitCode = commandExitCode(err)
		}
		limit := check.OutputLimitBytes
		if limit <= 0 {
			limit = 1_000_000
		}
		truncated := len(output) > limit
		if truncated {
			output = output[:limit]
		}
		attemptOutput := string(output)
		hash := sha256.Sum256(output)
		server.verificationRegistry.mu.Lock()
		for i := range run.Checks {
			if run.Checks[i].CheckID == id && run.Checks[i].Status == "running" {
				run.Checks[i].Status, run.Checks[i].ExitCode = status, exitCode
				run.Checks[i].Output, run.Checks[i].OutputBytes = attemptOutput, len(output)
				run.Checks[i].OutputDigest, run.Checks[i].Truncated = "sha256:"+hex.EncodeToString(hash[:]), truncated
				run.Checks[i].CompletedAt = nowUTC()
				break
			}
		}
		server.verificationRegistry.mu.Unlock()
		_ = repo.saveVerificationRun(run)
	}
	if ctx.Err() != nil {
		run.Status = "cancelled"
	} else {
		run.Status = "completed"
	}
	run.CompletedAt = nowUTC()
	_ = repo.saveVerificationRun(run)
	server.verificationRegistry.mu.Lock()
	delete(server.verificationRegistry.cancels, run.RunID)
	server.verificationRegistry.mu.Unlock()
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
	server.verificationRegistry.mu.RUnlock()
	if run != nil {
		return run, nil
	}
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

func (server runtimeServer) latestVerificationRun(repo repository, commit, profile, configHash string) (*verificationRun, error) {
	runs, err := repo.loadVerificationRuns()
	if err != nil {
		return nil, err
	}
	var selected *verificationRun
	for _, run := range runs {
		if run.Commit != commit || run.Profile != profile || run.ConfigBlobHash != configHash || run.Status == "invalidated" || run.Status == "running" || run.Status == "queued" {
			continue
		}
		if selected == nil || run.CompletedAt > selected.CompletedAt {
			selected = run
		}
	}
	return selected, nil
}

func verificationFromRun(run *verificationRun, required []string) *eve.Verification {
	if run == nil {
		return nil
	}
	ran := make([]string, 0, len(run.Checks))
	for _, check := range run.Checks {
		ran = append(ran, check.CheckID)
	}
	digest, _ := verificationRunDigest(run)
	executor := map[string]string{}
	for key, value := range run.ExecutorFingerprint {
		executor[key] = value
	}
	return &eve.Verification{Status: verificationStatusForRun(run, required), Profile: run.Profile, RequiredChecks: required, RanChecks: ran, SelectedRunID: run.RunID, RunRecordDigest: digest, ConfigBlobHash: run.ConfigBlobHash, ExecutorFingerprint: executor, Integrity: "matched"}
}
