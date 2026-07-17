package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nhestrompia/eve"
)

func TestMain(m *testing.M) {
	registryDir, err := os.MkdirTemp("", "eve-test-registry-*")
	if err == nil {
		_ = os.Setenv("EVE_REPOSITORY_REGISTRY", filepath.Join(registryDir, "repositories.json"))
	}
	pendingDir, pendingErr := os.MkdirTemp("", "eve-test-pending-*")
	if pendingErr == nil {
		_ = os.Setenv("EVE_PENDING_STATE", filepath.Join(pendingDir, "pending-state.json"))
	}
	code := m.Run()
	if registryDir != "" {
		_ = os.RemoveAll(registryDir)
	}
	if pendingDir != "" {
		_ = os.RemoveAll(pendingDir)
	}
	os.Exit(code)
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "snapshot schema "+eve.SnapshotSchemaVersion) {
		t.Fatalf("stdout = %q, want snapshot schema", stdout.String())
	}
}

func TestInstallMCPWritesClientConfigs(t *testing.T) {
	home := t.TempDir()
	eveBin := filepath.Join(home, "bin", "eve")
	if err := os.MkdirAll(filepath.Dir(eveBin), 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	if err := os.WriteFile(eveBin, []byte("fake eve"), 0o755); err != nil {
		t.Fatalf("write eve bin: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"install-mcp", "--home", home, "--eve-bin", eveBin}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("install-mcp exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, want := range []string{"Configured codex", "Configured claude", "Configured opencode"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout = %q, want %q", stdout.String(), want)
		}
	}

	codexConfig := readTextFile(t, filepath.Join(home, ".codex", "config.toml"))
	for _, want := range []string{
		`[mcp_servers.eve]`,
		`command = "` + eveBin + `"`,
		`args = ["mcp-stdio"]`,
	} {
		if !strings.Contains(codexConfig, want) {
			t.Fatalf("codex config = %q, want %q", codexConfig, want)
		}
	}

	var claude map[string]any
	readJSONFile(t, filepath.Join(home, ".claude.json"), &claude)
	claudeEve := claude["mcpServers"].(map[string]any)["eve"].(map[string]any)
	if claudeEve["command"] != eveBin {
		t.Fatalf("claude eve command = %#v, want %s", claudeEve["command"], eveBin)
	}
	if got := strings.Join(jsonStringSlice(t, claudeEve["args"]), " "); got != "mcp-stdio --cwd ${CLAUDE_PROJECT_DIR:-.}" {
		t.Fatalf("claude args = %q", got)
	}

	var opencode map[string]any
	readJSONFile(t, filepath.Join(home, ".config", "opencode", "opencode.json"), &opencode)
	opencodeEve := opencode["mcp"].(map[string]any)["eve"].(map[string]any)
	if opencodeEve["type"] != "local" || opencodeEve["enabled"] != true {
		t.Fatalf("opencode eve = %#v, want enabled local server", opencodeEve)
	}
	if got := strings.Join(jsonStringSlice(t, opencodeEve["command"]), " "); got != eveBin+" mcp-stdio" {
		t.Fatalf("opencode command = %q", got)
	}
	if _, ok := opencodeEve["cwd"]; ok {
		t.Fatalf("opencode eve = %#v, should not set cwd by default", opencodeEve)
	}
}

func TestInstallMCPUpdatesExistingCodexTable(t *testing.T) {
	home := t.TempDir()
	eveBin := filepath.Join(home, "eve")
	if err := os.WriteFile(eveBin, []byte("fake eve"), 0o755); err != nil {
		t.Fatalf("write eve bin: %v", err)
	}
	configPath := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir codex config dir: %v", err)
	}
	existing := `model = "gpt-5"

[mcp_servers.eve]
command = "eve"
args = ["mcp-stdio"]

[mcp_servers.other]
command = "other"
`
	if err := os.WriteFile(configPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"install-mcp", "--clients", "codex", "--home", home, "--eve-bin", eveBin, "--cwd", "/repo"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("install-mcp exit code = %d, stderr = %s", code, stderr.String())
	}

	config := readTextFile(t, configPath)
	for _, want := range []string{
		`model = "gpt-5"`,
		`command = "` + eveBin + `"`,
		`args = ["mcp-stdio", "--cwd", "/repo"]`,
		`[mcp_servers.other]`,
	} {
		if !strings.Contains(config, want) {
			t.Fatalf("codex config = %q, want %q", config, want)
		}
	}
	if strings.Contains(config, `command = "eve"`) {
		t.Fatalf("codex config still contains old eve command: %q", config)
	}
}

func TestInitCreatesSnapshotStructure(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	code := run([]string{"init"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, path := range []string{
		filepath.Join(repo, ".eve", "config.json"),
		filepath.Join(repo, ".eve", "snapshots"),
		filepath.Join(repo, ".eve", "skips"),
		filepath.Join(repo, ".eve", "artifacts"),
		filepath.Join(repo, ".eve", "cache"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".eve", "evolutions")); !os.IsNotExist(err) {
		t.Fatalf(".eve/evolutions should not be created, err = %v", err)
	}
	before := map[string]string{}
	for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
		path := filepath.Join(repo, name)
		before[name] = readTextFile(t, path)
		if strings.Count(before[name], "<!-- eve:instructions:start") != 1 || !strings.Contains(before[name], "`complete_snapshot`") || !strings.Contains(before[name], "`skip_snapshot`") {
			t.Fatalf("%s = %q, want one canonical EVE block", name, before[name])
		}
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"init"}, &stdout, &stderr); code != 0 {
		t.Fatalf("second init exit code = %d, stderr = %s", code, stderr.String())
	}
	for name, want := range before {
		if got := readTextFile(t, filepath.Join(repo, name)); got != want {
			t.Fatalf("second init changed %s", name)
		}
	}
}

func TestSnapshotValidateCanonicalizeAndCleanBreakList(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	snapshot := sampleSnapshot("snap_001", "Snapshot Runtime", gitOutputForTest(t, repo, "rev-parse", "HEAD"))
	writeSnapshot(t, repo, snapshot)
	writeLegacyEvolution(t, repo)

	assertCommandContains(t, []string{"snapshot", "snap_001"}, []string{"Snapshot Runtime", "Repository:", "Commit:"})
	assertCommandContains(t, []string{"validate", filepath.Join(repo, ".eve", "snapshots", "snap_001.json")}, []string{"is valid"})
	assertCommandContains(t, []string{"canonicalize", filepath.Join(repo, ".eve", "snapshots", "snap_001.json")}, []string{`"id":"snap_001"`})

	handler := newRuntimeServer(repoFromRoot(repo), "localhost:0").routes()
	var rows []snapshotSummary
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots", nil, &rows)
	if len(rows) != 1 || rows[0].ID != "snap_001" {
		t.Fatalf("rows = %#v, want only snap_001", rows)
	}
}

func TestChangelogCLISelectsRangesAndGroupsSnapshots(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	head := gitOutputForTest(t, repo, "rev-parse", "HEAD")

	first := sampleSnapshot("snap_001", "Password login", head)
	first.CreatedAt = "2026-07-01T09:00:00Z"
	first.Type = "feature"
	first.UserVisibleChange = "Added password login."
	second := sampleSnapshot("snap_002", "Redirect loop", head)
	second.CreatedAt = "2026-07-02T09:00:00Z"
	second.Type = "bugfix"
	second.UserVisibleChange = "Fixed redirect loop."
	third := sampleSnapshot("snap_003", "Repository indexing", head)
	third.CreatedAt = "2026-07-03T09:00:00Z"
	third.Type = "refactor"
	third.UserVisibleChange = "Improved repository indexing."
	fourth := sampleSnapshot("snap_004", "Release prep", head)
	fourth.CreatedAt = "2026-07-04T09:00:00Z"
	fourth.Type = "release"
	for _, snapshot := range []*eve.Snapshot{first, second, third, fourth} {
		writeSnapshot(t, repo, snapshot)
	}

	assertCommandContains(t, []string{"changelog", "--since", "snap_001", "--markdown"}, []string{
		"# Release Notes",
		"## Improvements",
		"- Improved repository indexing.",
		"## Fixes",
		"- Fixed redirect loop.",
		"## Other",
		"- Release prep",
	})
	assertCommandOmits(t, []string{"changelog", "--since", "snap_001", "--markdown"}, []string{"Added password login."})
	assertCommandContains(t, []string{"changelog", "--since", "2026-07-03", "--markdown"}, []string{"Improved repository indexing.", "Release prep"})
	assertCommandOmits(t, []string{"changelog", "--since", "2026-07-03", "--markdown"}, []string{"Fixed redirect loop."})
	assertCommandContains(t, []string{"changelog", "--from", "snap_001", "--to", "snap_003"}, []string{"Fixed redirect loop.", "Improved repository indexing."})
	assertCommandContains(t, []string{"changelog", "--since", "snap_004"}, []string{"No snapshot changes found."})

	assertCommandFails(t, []string{"changelog", "--since", "snap_001", "--from", "snap_002", "--to", "snap_003"}, "--since cannot be combined")
	assertCommandFails(t, []string{"changelog", "--from", "snap_003", "--to", "snap_001"}, "snapshot range is reversed")
}

func TestCompareCLIAndRuntimeAPIAggregateProductHistory(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	head := gitOutputForTest(t, repo, "rev-parse", "HEAD")

	base := sampleSnapshot("snap_base", "Base state", head)
	base.CreatedAt = "2026-07-01T09:00:00Z"
	feature := sampleSnapshot("snap_feature", "OAuth login", head)
	feature.CreatedAt = "2026-07-02T09:00:00Z"
	feature.Type = "feature"
	feature.UserVisibleChange = "Added OAuth login."
	feature.Decisions = []eve.Decision{{Title: "Use OAuth", Rationale: "External providers reduce password handling."}}
	feature.Risks = []eve.Risk{{Title: "Provider outage affects login.", Severity: "medium", Mitigation: "Keep sessions alive."}}
	feature.Timeline = []eve.TimelineEntry{{Phase: "validation", Title: "Browser validation", Summary: "Login flow passed.", OccurredAt: "2026-07-02T10:00:00Z"}}
	fix := sampleSnapshot("snap_fix", "Redirect fix", head)
	fix.CreatedAt = "2026-07-03T09:00:00Z"
	fix.Type = "bugfix"
	fix.UserVisibleChange = "Fixed OAuth redirect loop."
	for _, snapshot := range []*eve.Snapshot{base, feature, fix} {
		writeSnapshot(t, repo, snapshot)
	}

	assertCommandContains(t, []string{"compare", "snap_base", "snap_fix", "--markdown"}, []string{
		"# Snapshot Comparison",
		"## Added",
		"Added OAuth login.",
		"## Fixed",
		"Fixed OAuth redirect loop.",
		"## Decisions",
		"Use OAuth",
		"## Risks",
		"Provider outage affects login.",
		"## Validation",
		"go test ./...",
		"## Timeline",
		"snap_feature",
	})

	handler := newRuntimeServer(repoFromRoot(repo), "localhost:0").routes()
	var comparison comparisonResponse
	requestJSON(t, handler, http.MethodGet, "/api/compare?from=snap_base&to=snap_fix", nil, &comparison)
	if comparison.Repository != filepath.Base(repo) || len(comparison.Range) != 2 || len(comparison.Added) != 1 || len(comparison.Fixed) != 1 {
		t.Fatalf("comparison = %#v, want aggregated feature and fix range", comparison)
	}
	if comparison.Changed == nil {
		t.Fatalf("changed = nil, want empty array")
	}
	if len(comparison.Decisions) != 1 || comparison.Decisions[0].Title != "Use OAuth" {
		t.Fatalf("decisions = %#v, want OAuth decision", comparison.Decisions)
	}
	assertRequestStatus(t, handler, http.MethodGet, "/api/compare?from=snap_fix&to=snap_base", http.StatusBadRequest, "snapshot range is reversed")
	assertRequestStatus(t, handler, http.MethodGet, "/api/compare?from=missing&to=snap_base", http.StatusNotFound, "snapshot missing not found")
}

func TestCompareAPIRejectsCrossRepositorySnapshots(t *testing.T) {
	parent := t.TempDir()
	primary := initTempGitRepoAt(t, filepath.Join(parent, "primary"))
	secondary := initTempGitRepoAt(t, filepath.Join(parent, "secondary"))
	primaryHead := gitOutputForTest(t, primary, "rev-parse", "HEAD")
	secondaryHead := gitOutputForTest(t, secondary, "rev-parse", "HEAD")
	mustRunInRepo(t, primary, []string{"init"})
	mustRunInRepo(t, secondary, []string{"init"})
	writeSnapshot(t, primary, sampleSnapshot("snap_primary", "Primary Snapshot", primaryHead))
	writeSnapshot(t, secondary, sampleSnapshot("snap_secondary", "Secondary Snapshot", secondaryHead))

	handler := newRuntimeServer(repoFromRoot(primary), "localhost:0").routes()
	assertRequestStatus(t, handler, http.MethodGet, "/api/compare?from=snap_primary&to=snap_secondary", http.StatusBadRequest, "same repository")
}

func TestRuntimeAPIAndMCP(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	head := gitOutputForTest(t, repo, "rev-parse", "HEAD")
	gitRun(t, repo, "remote", "add", "origin", "git@github.com:nhestrompia/eve.git")
	writeSnapshot(t, repo, sampleSnapshot("snap_api", "API Snapshot", head))

	handler := newRuntimeServer(repoFromRoot(repo), "localhost:0").routes()
	var repos []repoSummary
	requestJSON(t, handler, http.MethodGet, "/api/repos", nil, &repos)
	if len(repos) != 1 || repos[0].SnapshotCount != 1 {
		t.Fatalf("repos = %#v, want one repo with one snapshot", repos)
	}
	if repos[0].Head != head || repos[0].RemoteURL != "https://github.com/nhestrompia/eve" {
		t.Fatalf("repo metadata = %#v, want head and normalized GitHub remote", repos[0])
	}

	var detail snapshotDetailResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots/snap_api", nil, &detail)
	if detail.Snapshot.Title != "API Snapshot" || len(detail.RawJSON) == 0 {
		t.Fatalf("detail = %#v, want snapshot detail", detail)
	}

	response := mcpCall(t, handler, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	if !strings.Contains(response, "complete_snapshot") {
		t.Fatalf("tools/list response = %s, want complete_snapshot", response)
	}
	response = mcpCall(t, handler, `{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"eve://repos/`+filepath.Base(repo)+`/snapshots/snap_api"}}`)
	if !strings.Contains(response, "API Snapshot") {
		t.Fatalf("resources/read response = %s, want snapshot", response)
	}
	response = mcpCall(t, handler, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"complete_snapshot","arguments":{"title":"Completed by MCP","type":"feature","summary":"MCP writes snapshots.","validation":[{"command":"go test ./...","status":"passed"}],"allowDirty":true}}}`)
	if !strings.Contains(response, "Completed by MCP") {
		t.Fatalf("complete_snapshot response = %s, want created snapshot", response)
	}
	snapshots, err := repoFromRoot(repo).listSnapshots("")
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("snapshot count = %d, want 2", len(snapshots))
	}
}

func TestVerificationRunAndSnapshotAggregation(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init", "--no-agent-instructions"})
	config := `{"schemaVersion":3,"snapshotSchema":"0.2.0","verification":{"checks":{"unit":{"argv":["go","version"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":2000}},"suites":{"change":["unit"]},"profileRules":[{"default":"change"}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(repo, ".eve", "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "configure verification")
	head := gitOutputForTest(t, repo, "rev-parse", "HEAD")
	server := newRuntimeServer(repoFromRoot(repo), "localhost:0")
	run, err := server.startVerificationRun(context.Background(), repoFromRoot(repo), head, "", "")
	if err != nil {
		t.Fatalf("start verification: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		current, readErr := server.verificationRun(repoFromRoot(repo), run.RunID)
		if readErr == nil && current.Status != "running" && current.Status != "queued" {
			run = current
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if run.Status != "completed" || len(run.Checks) != 1 || run.Checks[0].Status != "passed" {
		t.Fatalf("run = %#v, want completed passing run", run)
	}
	input := completeSnapshotInput{Title: "Verified snapshot", Type: "feature", Summary: "Recorded execution evidence."}
	snapshot, err := completeSnapshot(repoFromRoot(repo), input, []string{".eve/runs"})
	if err != nil {
		t.Fatalf("complete snapshot: %v", err)
	}
	if snapshot.Verification == nil || snapshot.Verification.Status != "required_checks_passed" {
		t.Fatalf("verification = %#v, want required_checks_passed", snapshot.Verification)
	}
	if snapshot.Verification.SelectedRunID != run.RunID || snapshot.Verification.RunRecordDigest == "" {
		t.Fatalf("verification = %#v, want selected run and digest", snapshot.Verification)
	}
}

func TestLegacyVerificationEvidenceRemainsValidWithoutRewrite(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init", "--no-agent-instructions"})
	head := gitOutputForTest(t, repo, "rev-parse", "HEAD")
	store := repoFromRoot(repo)
	run := &verificationRun{
		RunID: "snap_legacy_run", Commit: head, ConfigBlobHash: "sha256:legacy-policy",
		Profile: "change", Suite: "change", Status: "completed",
		RefContext:          verificationRefContext{Branch: "master", MatchingTags: []string{}, MatchedRule: "default", ResolvedProfile: "change"},
		ExecutorFingerprint: map[string]string{"eve": "0.1.0", "os": "test", "arch": "test"},
		StartedAt:           "2026-07-01T10:00:00Z", CompletedAt: "2026-07-01T10:00:01Z",
		Checks: []verificationAttempt{{
			CheckID: "unit", Status: "passed", ExitCode: 0,
			StartedAt: "2026-07-01T10:00:00Z", CompletedAt: "2026-07-01T10:00:01Z",
			Output: "ok\n", OutputBytes: 3, OutputDigest: "sha256:legacy-output",
		}},
	}
	legacyBytes, err := verificationRunCanonicalBytes(run)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(legacyBytes, []byte(`"exitCode": 0`)) || bytes.Contains(legacyBytes, []byte(`"schemaVersion"`)) {
		t.Fatalf("legacy evidence changed shape: %s", legacyBytes)
	}
	if err := os.MkdirAll(store.verificationRunsDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.verificationRunPath(run.RunID), legacyBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	digest, err := verificationRunDigest(run)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := sampleSnapshot("snap_legacy_evidence", "Legacy evidence", head)
	snapshot.Verification = &eve.Verification{
		Status: "required_checks_passed", Profile: "change", Suite: "change",
		RequiredChecks: []string{"unit"}, RanChecks: []string{"unit"},
		SelectedRunID: run.RunID, RunRecordDigest: digest, ConfigBlobHash: run.ConfigBlobHash,
	}
	writeSnapshot(t, repo, snapshot)

	loadedRuns, err := store.loadVerificationRuns()
	if err != nil || len(loadedRuns) != 1 {
		t.Fatalf("load legacy runs = %#v, %v", loadedRuns, err)
	}
	loaded, err := store.loadSnapshot(snapshot.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Verification.Status != "required_checks_passed" || loaded.Verification.Integrity != "matched" {
		t.Fatalf("legacy verification = %#v, want matched passing evidence", loaded.Verification)
	}
	after, err := os.ReadFile(store.verificationRunPath(run.RunID))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, legacyBytes) {
		t.Fatal("loading legacy evidence rewrote the run record")
	}
}

func TestVerificationConfigMigrationPreservesOlderRepositories(t *testing.T) {
	for name, config := range map[string]string{
		"legacy snake case": `{"config_version":1,"created_at":"2026-07-01T00:00:00Z","eve":{"version":1}}`,
		"schema version 2":  `{"schemaVersion":2,"snapshotSchema":"0.1.0","createdAt":"2026-07-01T00:00:00Z"}`,
	} {
		t.Run(name, func(t *testing.T) {
			repo := initTempGitRepo(t)
			t.Chdir(repo)
			mustRun(t, []string{"init", "--no-agent-instructions"})
			if err := os.WriteFile(filepath.Join(repo, ".eve", "config.json"), []byte(config+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := validateDoctorRepository(repoFromRoot(repo)); err != nil {
				t.Fatalf("older configuration should remain readable: %v", err)
			}
		})
	}
}

func TestInitWritesCurrentConfigWithoutRewritingOlderConfig(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init", "--no-agent-instructions"})
	var current map[string]any
	if err := json.Unmarshal([]byte(readTextFile(t, filepath.Join(repo, ".eve", "config.json"))), &current); err != nil {
		t.Fatal(err)
	}
	if current["schemaVersion"] != float64(3) {
		t.Fatalf("new config schemaVersion = %#v, want 3", current["schemaVersion"])
	}
	legacy := `{"schemaVersion":2,"snapshotSchema":"0.1.0","createdAt":"2026-07-01T00:00:00Z"}` + "\n"
	if err := os.WriteFile(filepath.Join(repo, ".eve", "config.json"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, []string{"init", "--no-agent-instructions"})
	if got := readTextFile(t, filepath.Join(repo, ".eve", "config.json")); got != legacy {
		t.Fatalf("init rewrote older config:\n%s", got)
	}
}

func TestVerificationRejectsUnsupportedOrMalformedConfiguredPolicy(t *testing.T) {
	for name, config := range map[string]string{
		"unsupported version":        `{"schemaVersion":999,"verification":{}}`,
		"version 2 verification":     `{"schemaVersion":2,"verification":{"checks":{"unit":{"argv":["go","version"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":1000}},"suites":{"change":["unit"]},"profileRules":[{"default":"change"}]}}`,
		"undefined check":            `{"schemaVersion":3,"verification":{"checks":{},"suites":{"change":["missing"]},"profileRules":[{"default":"change"}]}}`,
		"unknown verification field": `{"schemaVersion":3,"verification":{"checkz":{},"suites":{},"profileRules":[]}}`,
		"missing snapshot schema":    `{"schemaVersion":3,"verification":{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			repo := initTempGitRepo(t)
			t.Chdir(repo)
			mustRun(t, []string{"init", "--no-agent-instructions"})
			if err := os.WriteFile(filepath.Join(repo, ".eve", "config.json"), []byte(config+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			gitRun(t, repo, "add", ".")
			gitRun(t, repo, "commit", "-m", "set verification policy")
			_, err := completeSnapshot(repoFromRoot(repo), completeSnapshotInput{Title: "Config", Type: "feature", Summary: "Policy"}, nil)
			if err == nil {
				t.Fatal("completeSnapshot succeeded with unsupported verification policy")
			}
		})
	}
}

func TestVerificationHonorsConfiguredSuccessExitCodes(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init", "--no-agent-instructions"})
	config := `{"schemaVersion":3,"snapshotSchema":"0.2.0","verification":{"checks":{"unit":{"argv":["sh","-c","exit 7"],"timeoutSeconds":10,"successExitCodes":[7],"outputLimitBytes":2000}},"suites":{"change":["unit"]},"profileRules":[{"default":"change"}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(repo, ".eve", "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "configure verification")
	server := newRuntimeServer(repoFromRoot(repo), "localhost:0")
	run, err := server.startVerificationRun(context.Background(), repoFromRoot(repo), "", "", "")
	if err != nil {
		t.Fatalf("start verification: %v", err)
	}
	run = waitForVerificationRun(t, server, repoFromRoot(repo), run.RunID)
	if run.Status != "completed" || run.Checks[0].Status != "passed" {
		t.Fatalf("run = %#v, want accepted exit code", run)
	}
}

func TestVerificationRedactsAndBoundsPersistedOutput(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	t.Setenv("EVE_TEST_SECRET", "secret-value-123")
	mustRun(t, []string{"init", "--no-agent-instructions"})
	config := `{"schemaVersion":3,"snapshotSchema":"0.2.0","verification":{"checks":{"unit":{"argv":["sh","-c","printf \"$EVE_TEST_SECRET-stdout-long\"; printf \"$EVE_TEST_SECRET-stderr-long\" >&2"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":24,"inheritEnvironment":["EVE_TEST_SECRET"]}},"suites":{"change":["unit"]},"profileRules":[{"default":"change"}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(repo, ".eve", "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "configure verification")
	server := newRuntimeServer(repoFromRoot(repo), "localhost:0")
	run, err := server.startVerificationRun(context.Background(), repoFromRoot(repo), "", "", "test actor")
	if err != nil {
		t.Fatalf("start verification: %v", err)
	}
	run = waitForVerificationRun(t, server, repoFromRoot(repo), run.RunID)
	if run.Status != "completed" || len(run.Checks) != 1 {
		t.Fatalf("run = %#v, want completed output evidence", run)
	}
	attempt := run.Checks[0]
	encoded, _ := json.Marshal(run)
	if strings.Contains(string(encoded), "secret-value-123") {
		t.Fatalf("persisted run leaked inherited secret: %s", encoded)
	}
	if !attempt.Truncated || len(attempt.Output) > 24 || len(attempt.Stdout)+len(attempt.Stderr) > 24 {
		t.Fatalf("attempt output was not bounded: %#v", attempt)
	}
	if attempt.StdoutBytes == 0 || attempt.StderrBytes == 0 || attempt.StdoutDigest == "" || attempt.StderrDigest == "" {
		t.Fatalf("attempt lacks separate stdout/stderr evidence: %#v", attempt)
	}
	if run.ActorClaim != "test actor" || run.ActorProvenance != "unauthenticated" {
		t.Fatalf("actor evidence = %#v", run)
	}
}

func TestVerificationCancellationTerminatesProcessGroup(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init", "--no-agent-instructions"})
	config := `{"schemaVersion":3,"snapshotSchema":"0.2.0","verification":{"checks":{"slow":{"argv":["sh","-c","sleep 30"],"timeoutSeconds":60,"successExitCodes":[0],"outputLimitBytes":1000}},"suites":{"change":["slow"]},"profileRules":[{"default":"change"}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(repo, ".eve", "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "configure slow verification")
	server := newRuntimeServer(repoFromRoot(repo), "localhost:0")
	run, err := server.startVerificationRun(context.Background(), repoFromRoot(repo), "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		current, readErr := server.verificationRun(repoFromRoot(repo), run.RunID)
		if readErr == nil && len(current.Checks) == 1 && current.Checks[0].Status == "running" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	otherProcess := newRuntimeServer(repoFromRoot(repo), "localhost:0")
	queued, secondErr := otherProcess.startVerificationRun(context.Background(), repoFromRoot(repo), "", "", "")
	if secondErr != nil || queued.Status != "queued" {
		t.Fatalf("second runtime run = %#v err = %v, want queued serialization", queued, secondErr)
	}
	otherProcess.verificationRegistry.mu.RLock()
	cancelQueued := otherProcess.verificationRegistry.cancels[queued.RunID]
	otherProcess.verificationRegistry.mu.RUnlock()
	if cancelQueued == nil {
		t.Fatal("queued suite has no cancellation handle")
	}
	cancelQueued()
	queued = waitForVerificationRun(t, otherProcess, repoFromRoot(repo), queued.RunID)
	if queued.Status != "cancelled" {
		t.Fatalf("queued cancellation = %#v", queued)
	}
	server.verificationRegistry.mu.RLock()
	cancel := server.verificationRegistry.cancels[run.RunID]
	server.verificationRegistry.mu.RUnlock()
	if cancel == nil {
		t.Fatal("running suite has no cancellation handle")
	}
	started := time.Now()
	cancel()
	run = waitForVerificationRun(t, server, repoFromRoot(repo), run.RunID)
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("cancellation took %s; child process likely survived", elapsed)
	}
	if run.Status != "cancelled" || run.Checks[0].Status != "cancelled" {
		t.Fatalf("cancelled run = %#v", run)
	}
}

func TestVerificationInvalidatesTrackedDriftAndTamperedEvidence(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init", "--no-agent-instructions"})
	config := `{"schemaVersion":3,"snapshotSchema":"0.2.0","verification":{"checks":{"unit":{"argv":["sh","-c","echo changed > tracked.txt"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":2000}},"suites":{"change":["unit"]},"profileRules":[{"default":"change"}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(repo, ".eve", "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "configure verification")
	server := newRuntimeServer(repoFromRoot(repo), "localhost:0")
	run, err := server.startVerificationRun(context.Background(), repoFromRoot(repo), "", "", "")
	if err != nil {
		t.Fatalf("start verification: %v", err)
	}
	run = waitForVerificationRun(t, server, repoFromRoot(repo), run.RunID)
	if run.Status != "invalidated" {
		t.Fatalf("run status = %q, want invalidated", run.Status)
	}

	config = strings.Replace(config, `"sh","-c","echo changed > tracked.txt"`, `"go","version"`, 1)
	if err := os.WriteFile(filepath.Join(repo, ".eve", "config.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "use passing check")
	run, err = server.startVerificationRun(context.Background(), repoFromRoot(repo), "", "", "")
	if err != nil {
		t.Fatalf("start passing verification: %v", err)
	}
	run = waitForVerificationRun(t, server, repoFromRoot(repo), run.RunID)
	if run.Status != "completed" {
		t.Fatalf("passing run status = %q", run.Status)
	}
	snapshot, err := completeSnapshot(repoFromRoot(repo), completeSnapshotInput{Title: "Verified", Type: "feature", Summary: "Evidence"}, []string{".eve/runs"})
	if err != nil {
		t.Fatalf("complete snapshot: %v", err)
	}
	runData, err := os.ReadFile(repoFromRoot(repo).verificationRunPath(snapshot.Verification.SelectedRunID))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(repoFromRoot(repo).verificationRunPath(snapshot.Verification.SelectedRunID), append(runData, []byte(" ")...), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := repoFromRoot(repo).loadSnapshot(snapshot.ID)
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if loaded.Verification.Integrity == "matched" || loaded.Verification.Status == "required_checks_passed" {
		t.Fatalf("tampered verification = %#v, want integrity failure", loaded.Verification)
	}
	listed, err := repoFromRoot(repo).listSnapshots("")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Verification.Status == "required_checks_passed" {
		t.Fatalf("snapshot list retained green tampered evidence: %#v", listed)
	}
}

func TestSnapshotDisclosesVerificationPolicyReduction(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init", "--no-agent-instructions"})
	fullPolicy := `{"schemaVersion":3,"snapshotSchema":"0.2.0","verification":{"checks":{"unit":{"argv":["go","version"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":2000},"e2e":{"argv":["go","version"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":2000}},"suites":{"release":["unit","e2e"]},"profileRules":[{"default":"release"}]}}` + "\n"
	if err := os.WriteFile(filepath.Join(repo, ".eve", "config.json"), []byte(fullPolicy), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".")
	gitRun(t, repo, "commit", "-m", "configure full verification")
	_, err := completeSnapshot(repoFromRoot(repo), completeSnapshotInput{Title: "Baseline", Type: "release", Summary: "Full policy baseline"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".eve")
	gitRun(t, repo, "commit", "-m", "record baseline snapshot")

	reducedPolicy := strings.Replace(fullPolicy, `"release":["unit","e2e"]`, `"release":["unit"]`, 1)
	if err := os.WriteFile(filepath.Join(repo, ".eve", "config.json"), []byte(reducedPolicy), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "add", ".eve/config.json")
	gitRun(t, repo, "commit", "-m", "reduce verification requirements")
	server := newRuntimeServer(repoFromRoot(repo), "localhost:0")
	run, err := server.startVerificationRun(context.Background(), repoFromRoot(repo), "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	waitForVerificationRun(t, server, repoFromRoot(repo), run.RunID)
	snapshot, err := completeSnapshot(repoFromRoot(repo), completeSnapshotInput{Title: "Reduced", Type: "release", Summary: "Reduced policy"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	change := snapshot.Verification.PolicyChange
	if change == nil || !change.Changed || !change.RequirementsReduced || !slices.Contains(change.RemovedChecks, "e2e") {
		t.Fatalf("policy change = %#v, want disclosed e2e reduction", change)
	}
}

func waitForVerificationRun(t *testing.T, server runtimeServer, repo repository, id string) *verificationRun {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		run, err := server.verificationRun(repo, id)
		if err == nil && run.Status != "running" && run.Status != "queued" {
			return run
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("verification run %s did not finish", id)
	return nil
}

func TestPendingSnapshotIdleTriggerAndSkipResolution(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	gitRun(t, repo, "add", ".eve/config.json", "AGENTS.md", "CLAUDE.md")
	gitRun(t, repo, "commit", "-m", "initialize eve")
	gitRun(t, repo, "checkout", "-b", "feature")

	handler := newRuntimeServer(repoFromRoot(repo), "localhost:0").routes()
	var initial pendingSnapshotResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/pending", nil, &initial)
	if initial.Pending {
		t.Fatalf("initial pending = %#v, want no pending on first observation", initial)
	}

	commitProductChangeAt(t, repo, "product.txt", "product\nidle feature\n", "idle feature", time.Now().UTC().Add(-3*time.Hour))

	var pending pendingSnapshotResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/pending", nil, &pending)
	if !pending.Pending || pending.PendingSnapshot == nil || pending.PendingSnapshot.Trigger != pendingTriggerIdle {
		t.Fatalf("pending = %#v, want idle pending snapshot", pending)
	}
	if got := len(pending.PendingSnapshot.Range.Commits); got != 1 {
		t.Fatalf("pending commits = %d, want 1", got)
	}

	response := mcpCall(t, handler, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"skip_snapshot","arguments":{"reason":""}}}`)
	if !strings.Contains(response, "isError") || !strings.Contains(response, "skip reason is required") {
		t.Fatalf("skip empty reason response = %s, want error", response)
	}

	response = mcpCall(t, handler, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"skip_snapshot","arguments":{"reason":"Internal-only cleanup."}}}`)
	if !strings.Contains(response, "Internal-only cleanup.") || !strings.Contains(response, "skip_") {
		t.Fatalf("skip response = %s, want durable skip record", response)
	}
	skips, err := repoFromRoot(repo).listSkips()
	if err != nil {
		t.Fatalf("list skips: %v", err)
	}
	if len(skips) != 1 || skips[0].Reason != "Internal-only cleanup." {
		t.Fatalf("skips = %#v, want one skip record", skips)
	}
	snapshots, err := repoFromRoot(repo).listSnapshots("")
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snapshots) != 0 {
		t.Fatalf("snapshot count = %d, want skip excluded from product history", len(snapshots))
	}
	var resolved pendingSnapshotResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/pending", nil, &resolved)
	if resolved.Pending {
		t.Fatalf("resolved pending = %#v, want no pending after skip", resolved)
	}
}

func TestPendingSnapshotMergeTriggerAndCompleteResolution(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	gitRun(t, repo, "add", ".eve/config.json", "AGENTS.md", "CLAUDE.md")
	gitRun(t, repo, "commit", "-m", "initialize eve")

	handler := newRuntimeServer(repoFromRoot(repo), "localhost:0").routes()
	var initial pendingSnapshotResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/pending", nil, &initial)
	if initial.Pending {
		t.Fatalf("initial pending = %#v, want no pending on first observation", initial)
	}

	commitProductChangeAt(t, repo, "product.txt", "product\ntrunk merge\n", "trunk merge", time.Now().UTC())
	var config configResponse
	requestJSON(t, handler, http.MethodGet, "/api/config", nil, &config)
	if config.PendingSnapshot == nil || config.PendingSnapshot.Trigger != pendingTriggerMerge {
		t.Fatalf("config pending = %#v, want merge-trigger pending", config.PendingSnapshot)
	}

	response := mcpCall(t, handler, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"pending_snapshot","arguments":{}}}`)
	if !strings.Contains(response, `"pending":true`) || !strings.Contains(response, pendingTriggerMerge) {
		t.Fatalf("pending_snapshot response = %s, want merge pending", response)
	}
	response = mcpCall(t, handler, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"complete_snapshot","arguments":{"title":"Trunk Snapshot","type":"feature","summary":"Completed trunk work.","validation":[{"command":"go test ./...","status":"passed"}]}}}`)
	if !strings.Contains(response, "Trunk Snapshot") {
		t.Fatalf("complete response = %s, want snapshot", response)
	}
	var resolved pendingSnapshotResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/pending", nil, &resolved)
	if resolved.Pending {
		t.Fatalf("resolved pending = %#v, want no pending after complete snapshot", resolved)
	}
}

func TestTrunkBranchDetection(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	store := repoFromRoot(repo)
	config := readTextFile(t, store.configPath())
	config = strings.TrimSuffix(config, "\n")
	config = strings.TrimSuffix(config, "}") + `,` + "\n" + `  "trunkBranch": "release"` + "\n" + `}` + "\n"
	if err := os.WriteFile(store.configPath(), []byte(config), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if got := store.trunkBranch(); got != "release" {
		t.Fatalf("configured trunk = %q, want release", got)
	}

	mustRun(t, []string{"init"})
	if err := os.WriteFile(store.configPath(), []byte(`{"schemaVersion":2,"snapshotSchema":"0.1.0","createdAt":"2026-07-01T00:00:00Z"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	gitRun(t, repo, "update-ref", "refs/remotes/origin/release", "HEAD")
	gitRun(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/release")
	if got := store.trunkBranch(); got != "release" {
		t.Fatalf("origin trunk = %q, want release", got)
	}
}

func TestRuntimeAPIDerivesDisplayCommitsFromSnapshotBoundary(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	gitRun(t, repo, "add", ".eve/config.json", "AGENTS.md", "CLAUDE.md")
	gitRun(t, repo, "commit", "-m", "initialize eve")

	productPath := filepath.Join(repo, "product.txt")
	if err := os.WriteFile(productPath, []byte("product\nfocused change\n"), 0o644); err != nil {
		t.Fatalf("write product: %v", err)
	}
	gitRun(t, repo, "add", "product.txt")
	gitRun(t, repo, "commit", "-m", "focused implementation")
	head := gitOutputForTest(t, repo, "rev-parse", "HEAD")

	pollutedCommits := strings.Split(gitOutputForTest(t, repo, "log", "--format=%H", "-n", "50"), "\n")
	snapshot := sampleSnapshot("snap_polluted", "Polluted Commit Snapshot", head)
	snapshot.Implementation.Commits = pollutedCommits
	writeSnapshot(t, repo, snapshot)

	handler := newRuntimeServer(repoFromRoot(repo), "localhost:0").routes()
	var rows []snapshotSummary
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots", nil, &rows)
	if len(rows) != 1 || rows[0].CommitCount != 1 {
		t.Fatalf("rows = %#v, want one display commit", rows)
	}

	var detail snapshotDetailResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots/snap_polluted", nil, &detail)
	if len(detail.Commits) != 1 || detail.Commits[0].Hash != head {
		t.Fatalf("detail commits = %#v, want only %s", detail.Commits, head)
	}
	if detail.Summary.CommitCount != 1 {
		t.Fatalf("detail summary commit count = %d, want 1", detail.Summary.CommitCount)
	}
}

func TestSnapshotCodeAPIListsAndLoadsFiles(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	gitRun(t, repo, "add", ".eve/config.json", "AGENTS.md", "CLAUDE.md")
	gitRun(t, repo, "commit", "-m", "initialize eve")
	base := gitOutputForTest(t, repo, "rev-parse", "HEAD")

	if err := os.WriteFile(filepath.Join(repo, "auth.go"), []byte("package main\n\nfunc login() string {\n\treturn \"github\"\n}\n"), 0o644); err != nil {
		t.Fatalf("write auth.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "login.tsx"), []byte("export function Login() {\n  return <button>Sign in</button>;\n}\n"), 0o644); err != nil {
		t.Fatalf("write login.tsx: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "unmentioned.css"), []byte(".login { color: blue; }\n"), 0o644); err != nil {
		t.Fatalf("write css: %v", err)
	}
	gitRun(t, repo, "add", "auth.go", "login.tsx", "unmentioned.css")
	gitRun(t, repo, "commit", "-m", "add github login")
	head := gitOutputForTest(t, repo, "rev-parse", "HEAD")

	snapshot := sampleSnapshot("snap_code", "GitHub authentication added", head)
	snapshot.Summary = "GitHub authentication added in auth.go."
	snapshot.UserVisibleChange = "GitHub authentication added"
	snapshot.Validation = []eve.Validation{{Command: "npm test login.tsx", Status: "passed"}}
	snapshot.Implementation.BaseCommit = base
	snapshot.Implementation.Commits = []string{head}
	writeSnapshot(t, repo, snapshot)

	handler := newRuntimeServer(repoFromRoot(repo), "localhost:0").routes()
	var files snapshotCodeFilesResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots/snap_code/code/files", nil, &files)
	if files.Repository != filepath.Base(repo) || files.SnapshotID != "snap_code" || files.Base != base || files.Head != head {
		t.Fatalf("files metadata = %#v, want repo snapshot range", files)
	}
	if len(files.Files) != 3 {
		t.Fatalf("files = %#v, want three changed files", files.Files)
	}
	if files.Files[0].Path != "auth.go" || !files.Files[0].Curated || files.Files[0].Language != "go" {
		t.Fatalf("first file = %#v, want curated auth.go", files.Files[0])
	}
	if files.Files[1].Path != "login.tsx" || !files.Files[1].Curated {
		t.Fatalf("second file = %#v, want curated login.tsx", files.Files[1])
	}
	if files.Files[2].Path != "unmentioned.css" || files.Files[2].Curated {
		t.Fatalf("third file = %#v, want uncurated css", files.Files[2])
	}

	var diff snapshotCodeFileResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots/snap_code/code/file?path=auth.go&mode=diff", nil, &diff)
	if diff.Mode != "diff" || diff.Language != "go" || !strings.Contains(diff.Content, "@@") || !strings.Contains(diff.Content, "+func login() string") {
		t.Fatalf("diff response = %#v, want highlighted hunk source", diff)
	}
	if strings.Contains(diff.Content, "diff --git") {
		t.Fatalf("diff content includes file header: %q", diff.Content)
	}

	var full snapshotCodeFileResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots/snap_code/code/file?path=auth.go&mode=full", nil, &full)
	if full.Mode != "full" || !strings.Contains(full.Content, "package main") || !strings.Contains(full.Content, "return \"github\"") {
		t.Fatalf("full response = %#v, want file at snapshot git state", full)
	}

	assertRequestStatus(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots/snap_code/code/file?path=auth.go&mode=review", http.StatusBadRequest, "mode must be diff or full")
	assertRequestStatus(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots/snap_code/code/file?path=../auth.go&mode=diff", http.StatusBadRequest, "invalid path")
	assertRequestStatus(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots/snap_code/code/file?path=missing.go&mode=diff", http.StatusNotFound, "not changed")
}

func TestSnapshotCodeAPIHandlesLargeBinaryAndDeletedFiles(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	gitRun(t, repo, "add", ".eve/config.json", "AGENTS.md", "CLAUDE.md")
	gitRun(t, repo, "commit", "-m", "initialize eve")
	base := gitOutputForTest(t, repo, "rev-parse", "HEAD")

	large := strings.Repeat("a", snapshotCodePreviewLimit+1)
	if err := os.WriteFile(filepath.Join(repo, "large.txt"), []byte(large), 0o644); err != nil {
		t.Fatalf("write large file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "binary.dat"), []byte{0x00, 0x01, 0x02}, 0o644); err != nil {
		t.Fatalf("write binary file: %v", err)
	}
	if err := os.Remove(filepath.Join(repo, "product.txt")); err != nil {
		t.Fatalf("remove product: %v", err)
	}
	gitRun(t, repo, "add", "large.txt", "binary.dat")
	gitRun(t, repo, "rm", "product.txt")
	gitRun(t, repo, "commit", "-m", "large binary deleted")
	head := gitOutputForTest(t, repo, "rev-parse", "HEAD")

	snapshot := sampleSnapshot("snap_limits", "Preview boundaries", head)
	snapshot.Implementation.BaseCommit = base
	snapshot.Implementation.Commits = []string{head}
	writeSnapshot(t, repo, snapshot)

	handler := newRuntimeServer(repoFromRoot(repo), "localhost:0").routes()
	var files snapshotCodeFilesResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots/snap_limits/code/files", nil, &files)
	byPath := map[string]snapshotCodeFile{}
	for _, file := range files.Files {
		byPath[file.Path] = file
	}
	for path, want := range map[string]string{
		"large.txt":   "too large",
		"binary.dat":  "Binary file",
		"product.txt": "deleted",
	} {
		file, ok := byPath[path]
		if !ok {
			t.Fatalf("files = %#v, missing %s", files.Files, path)
		}
		if file.Previewable || !strings.Contains(file.Reason, want) {
			t.Fatalf("%s = %#v, want non-previewable reason containing %q", path, file, want)
		}
	}

	var largeFile snapshotCodeFileResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots/snap_limits/code/file?path=large.txt&mode=full", nil, &largeFile)
	if largeFile.Previewable || largeFile.Content != "" || !strings.Contains(largeFile.Reason, "too large") {
		t.Fatalf("large file response = %#v, want non-previewable large state", largeFile)
	}

	var deletedFile snapshotCodeFileResponse
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(repo)+"/snapshots/snap_limits/code/file?path=product.txt&mode=full", nil, &deletedFile)
	if deletedFile.Previewable || deletedFile.Content != "" || !strings.Contains(deletedFile.Reason, "deleted") {
		t.Fatalf("deleted file response = %#v, want non-previewable deleted state", deletedFile)
	}
}

func TestAddAndCommitSnapshotCLI(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	gitRun(t, repo, "add", ".eve/config.json", "AGENTS.md", "CLAUDE.md")
	gitRun(t, repo, "commit", "-m", "initialize eve")

	productPath := filepath.Join(repo, "product.txt")
	if err := os.WriteFile(productPath, []byte("product\ncli snapshot\n"), 0o644); err != nil {
		t.Fatalf("write product: %v", err)
	}
	gitRun(t, repo, "add", "product.txt")
	gitRun(t, repo, "commit", "-m", "implement cli snapshot")
	head := gitOutputForTest(t, repo, "rev-parse", "HEAD")

	mustRun(t, []string{
		"add",
		"--title", "CLI Snapshot",
		"--type", "feature",
		"--summary", "The CLI can stage and commit EVE snapshots.",
		"--validation", "go test ./...",
		"--decision", "Expose the MCP snapshot flow through CLI commands.",
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{"commit"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("commit exit code = %d stderr = %s stdout = %s", code, stderr.String(), stdout.String())
	}
	snapshots, err := repoFromRoot(repo).listSnapshots("")
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("snapshot count = %d, want 1", len(snapshots))
	}
	snapshot := snapshots[0]
	if snapshot.Title != "CLI Snapshot" || snapshot.Implementation.GitState != head {
		t.Fatalf("snapshot = %#v, want CLI title at implementation head %s", snapshot, head)
	}
	initCommit := gitOutputForTest(t, repo, "rev-parse", "HEAD~1")
	if snapshot.Implementation.BaseCommit != initCommit {
		t.Fatalf("base commit = %s, want latest committed .eve change %s", snapshot.Implementation.BaseCommit, initCommit)
	}
	if got := snapshot.Implementation.Commits; len(got) != 1 || got[0] != head {
		t.Fatalf("implementation commits = %#v, want only %s", got, head)
	}
	if len(snapshot.Validation) != 1 || snapshot.Validation[0].Command != "go test ./..." || snapshot.Validation[0].Status != "skipped" || snapshot.Validation[0].Provenance != "reported_by_agent" {
		t.Fatalf("validation = %#v, want non-passing agent-reported claim", snapshot.Validation)
	}
	if _, err := os.Stat(repoFromRoot(repo).stagedSnapshotPath()); !os.IsNotExist(err) {
		t.Fatalf("staged draft should be removed, err = %v", err)
	}
}

func TestCommitSnapshotCLIRejectsDirtyImplementationFiles(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	gitRun(t, repo, "add", ".eve/config.json", "AGENTS.md", "CLAUDE.md")
	gitRun(t, repo, "commit", "-m", "initialize eve")

	mustRun(t, []string{
		"add",
		"--title", "Dirty CLI Snapshot",
		"--summary", "This draft should not commit with dirty product files.",
	})
	if err := os.WriteFile(filepath.Join(repo, "product.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatalf("write dirty product: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"commit"}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "working tree has uncommitted changes") {
		t.Fatalf("commit code = %d stderr = %q stdout = %q, want dirty refusal", code, stderr.String(), stdout.String())
	}
}

func TestCompleteSnapshotRejectsDirtyTreeByDefault(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	if err := os.WriteFile(filepath.Join(repo, "product.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("write product: %v", err)
	}

	handler := newRuntimeServer(repoFromRoot(repo), "localhost:0").routes()
	response := mcpCall(t, handler, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"complete_snapshot","arguments":{"title":"Dirty Snapshot","type":"feature","summary":"Should be rejected."}}}`)
	for _, want := range []string{"isError", "working tree has uncommitted changes"} {
		if !strings.Contains(response, want) {
			t.Fatalf("complete_snapshot response = %s, want %q", response, want)
		}
	}
}

func TestRuntimeListsRegisteredRepositories(t *testing.T) {
	primary := initTempGitRepo(t)
	secondary := initTempGitRepo(t)
	head := gitOutputForTest(t, secondary, "rev-parse", "HEAD")
	mustRunInRepo(t, primary, []string{"init"})
	mustRunInRepo(t, secondary, []string{"init"})
	writeSnapshot(t, secondary, sampleSnapshot("snap_other", "Other Repository Snapshot", head))

	handler := newRuntimeServer(repoFromRoot(primary), "localhost:0").routes()
	var repos []repoSummary
	requestJSON(t, handler, http.MethodGet, "/api/repos", nil, &repos)
	if len(repos) != 2 {
		t.Fatalf("repos = %#v, want primary and registered secondary", repos)
	}

	var rows []snapshotSummary
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+filepath.Base(secondary)+"/snapshots", nil, &rows)
	if len(rows) != 1 || rows[0].ID != "snap_other" {
		t.Fatalf("secondary snapshots = %#v, want snap_other", rows)
	}

	var globalRows []snapshotSummary
	requestJSON(t, handler, http.MethodGet, "/api/snapshots", nil, &globalRows)
	if len(globalRows) != 1 || globalRows[0].ID != "snap_other" || globalRows[0].Repository != filepath.Base(secondary) {
		t.Fatalf("global snapshots = %#v, want secondary snap_other", globalRows)
	}

	var globalDetail snapshotDetailResponse
	requestJSON(t, handler, http.MethodGet, "/api/snapshots/snap_other", nil, &globalDetail)
	if globalDetail.Repository != filepath.Base(secondary) || globalDetail.Snapshot.Title != "Other Repository Snapshot" {
		t.Fatalf("global detail = %#v, want secondary snap_other", globalDetail)
	}

	var search snapshotSearchResponse
	requestJSON(t, handler, http.MethodGet, "/api/search?q=other", nil, &search)
	if len(search.Results) != 1 || search.Results[0].Evolution.ID != "snap_other" || search.Results[0].Evolution.Repository != filepath.Base(secondary) {
		t.Fatalf("search = %#v, want secondary snap_other", search)
	}
}

func TestRuntimeDiscoversUnregisteredSiblingRepositories(t *testing.T) {
	parent := t.TempDir()
	primary := initTempGitRepoAt(t, filepath.Join(parent, "primary"))
	secondary := initTempGitRepoAt(t, filepath.Join(parent, "secondary"))
	head := gitOutputForTest(t, secondary, "rev-parse", "HEAD")

	mustRunInRepo(t, primary, []string{"init"})
	writeSnapshotFileWithoutRegistry(t, secondary, sampleSnapshot("snap_sibling", "Sibling Snapshot", head))

	handler := newRuntimeServer(repoFromRoot(primary), "localhost:0").routes()
	var repos []repoSummary
	requestJSON(t, handler, http.MethodGet, "/api/repos", nil, &repos)
	if len(repos) != 2 {
		t.Fatalf("repos = %#v, want primary and discovered sibling", repos)
	}

	var rows []snapshotSummary
	requestJSON(t, handler, http.MethodGet, "/api/repos/secondary/snapshots", nil, &rows)
	if len(rows) != 1 || rows[0].ID != "snap_sibling" {
		t.Fatalf("secondary snapshots = %#v, want snap_sibling", rows)
	}
}

func TestCheckoutRefusesDirtyTreeUnlessForced(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	head := gitOutputForTest(t, repo, "rev-parse", "HEAD")
	mustRun(t, []string{"init"})
	writeSnapshot(t, repo, sampleSnapshot("snap_checkout", "Checkout Snapshot", head))
	if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"checkout", "snap_checkout"}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "Working tree has uncommitted changes") {
		t.Fatalf("checkout code = %d stderr = %q, want dirty refusal", code, stderr.String())
	}
}

func TestCheckoutSnapshotWithMultipleCommitsUsesGitState(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	mustRun(t, []string{"init"})
	gitRun(t, repo, "add", ".eve/config.json", "AGENTS.md", "CLAUDE.md")
	gitRun(t, repo, "commit", "-m", "initialize eve")

	productPath := filepath.Join(repo, "product.txt")
	if err := os.WriteFile(productPath, []byte("product\nfirst\n"), 0o644); err != nil {
		t.Fatalf("write first product change: %v", err)
	}
	gitRun(t, repo, "add", "product.txt")
	gitRun(t, repo, "commit", "-m", "first product change")
	firstCommit := gitOutputForTest(t, repo, "rev-parse", "HEAD")

	if err := os.WriteFile(productPath, []byte("product\nfirst\nsecond\n"), 0o644); err != nil {
		t.Fatalf("write second product change: %v", err)
	}
	gitRun(t, repo, "add", "product.txt")
	gitRun(t, repo, "commit", "-m", "second product change")
	snapshotCommit := gitOutputForTest(t, repo, "rev-parse", "HEAD")

	snapshot := sampleSnapshot("snap_multi_commit", "Multi Commit Snapshot", snapshotCommit)
	snapshot.Implementation.Commits = []string{firstCommit, snapshotCommit}
	writeSnapshot(t, repo, snapshot)
	gitRun(t, repo, "add", ".eve")
	gitRun(t, repo, "commit", "-m", "record snapshot")

	var stdout, stderr bytes.Buffer
	code := run([]string{"checkout", "snap_multi_commit"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("checkout code = %d stderr = %q stdout = %q, want success", code, stderr.String(), stdout.String())
	}
	if got := gitOutputForTest(t, repo, "rev-parse", "HEAD"); got != snapshotCommit {
		t.Fatalf("HEAD = %s, want snapshot gitState %s", got, snapshotCommit)
	}
	if !strings.Contains(stdout.String(), "Commit: "+snapshotCommit) {
		t.Fatalf("stdout = %q, want checked out gitState", stdout.String())
	}
}

func TestCompleteSnapshotDerivesGitFacts(t *testing.T) {
	repo := initTempGitRepo(t)
	facts, err := deriveGitFacts(repoFromRoot(repo))
	if err != nil {
		t.Fatalf("deriveGitFacts: %v", err)
	}
	if facts.GitState == "" || len(facts.Commits) == 0 {
		t.Fatalf("facts = %#v, want git state and commits", facts)
	}
}

func TestNormalizeRemoteURL(t *testing.T) {
	cases := map[string]string{
		"git@github.com:owner/repo.git":       "https://github.com/owner/repo",
		"ssh://git@github.com/owner/repo.git": "https://github.com/owner/repo",
		"https://github.com/owner/repo.git":   "https://github.com/owner/repo",
	}
	for input, want := range cases {
		if got := normalizeRemoteURL(input); got != want {
			t.Fatalf("normalizeRemoteURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func mustRun(t *testing.T, args []string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("%v exit code = %d stderr = %s stdout = %s", args, code, stderr.String(), stdout.String())
	}
}

func mustRunInRepo(t *testing.T, repo string, args []string) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("chdir %s: %v", repo, err)
	}
	defer func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()
	mustRun(t, args)
}

func assertCommandContains(t *testing.T, args []string, wants []string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("%v exit code = %d stderr = %s", args, code, stderr.String())
	}
	for _, want := range wants {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("%v stdout = %q, want %q", args, stdout.String(), want)
		}
	}
}

func assertCommandOmits(t *testing.T, args []string, unwanted []string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("%v exit code = %d stderr = %s", args, code, stderr.String())
	}
	for _, value := range unwanted {
		if strings.Contains(stdout.String(), value) {
			t.Fatalf("%v stdout = %q, should omit %q", args, stdout.String(), value)
		}
	}
}

func assertCommandFails(t *testing.T, args []string, want string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("%v exit code = 0 stdout = %s, want failure", args, stdout.String())
	}
	if !strings.Contains(stderr.String(), want) {
		t.Fatalf("%v stderr = %q, want %q", args, stderr.String(), want)
	}
}

func requestJSON(t *testing.T, handler http.Handler, method string, target string, body *bytes.Reader, out any) {
	t.Helper()
	if body == nil {
		body = bytes.NewReader(nil)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, body))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("%s %s status = %d body = %s", method, target, recorder.Code, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), out); err != nil {
		t.Fatalf("decode response: %v; body = %s", err, recorder.Body.String())
	}
}

func assertRequestStatus(t *testing.T, handler http.Handler, method string, target string, status int, want string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	if recorder.Code != status {
		t.Fatalf("%s %s status = %d body = %s, want %d", method, target, recorder.Code, recorder.Body.String(), status)
	}
	if !strings.Contains(recorder.Body.String(), want) {
		t.Fatalf("%s %s body = %q, want %q", method, target, recorder.Body.String(), want)
	}
}

func mcpCall(t *testing.T, handler http.Handler, body string) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	request.Header.Set("Accept", "application/json, text/event-stream")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("mcp status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	return recorder.Body.String()
}

func initTempGitRepo(t *testing.T) string {
	t.Helper()
	return initTempGitRepoAt(t, t.TempDir())
}

func initTempGitRepoAt(t *testing.T, repo string) string {
	t.Helper()
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	gitRun(t, repo, "init")
	gitRun(t, repo, "config", "user.email", "test@example.com")
	gitRun(t, repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "product.txt"), []byte("product\n"), 0o644); err != nil {
		t.Fatalf("write product: %v", err)
	}
	gitRun(t, repo, "add", "product.txt")
	gitRun(t, repo, "commit", "-m", "initial")
	return repo
}

func gitRun(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func commitProductChangeAt(t *testing.T, repo string, path string, content string, message string, committedAt time.Time) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, path), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	gitRun(t, repo, "add", path)
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Dir = repo
	gitDate := committedAt.UTC().Format(time.RFC3339)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+gitDate, "GIT_COMMITTER_DATE="+gitDate)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit %q: %v\n%s", message, err, output)
	}
}

func gitOutputForTest(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(output))
}

func readTextFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func readJSONFile(t *testing.T, path string, out any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func jsonStringSlice(t *testing.T, value any) []string {
	t.Helper()
	items, ok := value.([]any)
	if !ok {
		t.Fatalf("value = %#v, want JSON array", value)
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			t.Fatalf("item = %#v, want string", item)
		}
		result = append(result, text)
	}
	return result
}

func sampleSnapshot(id string, title string, gitState string) *eve.Snapshot {
	return &eve.Snapshot{
		ID:            id,
		SchemaVersion: eve.SnapshotSchemaVersion,
		Title:         title,
		Type:          "feature",
		Summary:       "Snapshots are canonical product units.",
		Relationships: eve.Relationships{
			Corrects:   []string{},
			Supersedes: []string{},
			Reverts:    []string{},
			DependsOn:  []string{},
			Related:    []string{},
		},
		Risks:     []eve.Risk{},
		Timeline:  []eve.TimelineEntry{},
		Decisions: []eve.Decision{},
		Validation: []eve.Validation{{
			Command: "go test ./...",
			Status:  "passed",
		}},
		Artifacts: []eve.Artifact{},
		Implementation: eve.Implementation{
			Branch:   "master",
			GitState: gitState,
			Commits:  []string{gitState},
			Dirty:    false,
		},
		CreatedAt: "2026-07-03T15:00:00Z",
	}
}

func writeSnapshot(t *testing.T, repo string, snapshot *eve.Snapshot) {
	t.Helper()
	store := repoFromRoot(repo)
	if err := store.saveSnapshot(snapshot); err != nil {
		t.Fatalf("save snapshot: %v", err)
	}
}

func writeSnapshotFileWithoutRegistry(t *testing.T, repo string, snapshot *eve.Snapshot) {
	t.Helper()
	store := repoFromRoot(repo)
	if err := os.MkdirAll(store.snapshotsDir(), 0o755); err != nil {
		t.Fatalf("mkdir snapshots: %v", err)
	}
	canonical, err := eve.CanonicalSnapshotJSON(snapshot)
	if err != nil {
		t.Fatalf("canonicalize snapshot: %v", err)
	}
	if err := os.WriteFile(store.snapshotPath(snapshot.ID), append(canonical, '\n'), 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
}

func writeLegacyEvolution(t *testing.T, repo string) {
	t.Helper()
	dir := filepath.Join(repo, ".eve", "evolutions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir evolutions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "EV-001.json"), []byte(`{"legacy":true}`), 0o644); err != nil {
		t.Fatalf("write legacy evolution: %v", err)
	}
}
