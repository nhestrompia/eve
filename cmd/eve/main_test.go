package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nhestrompia/eve"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "snapshot schema 0.1.0") {
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
	response = mcpCall(t, handler, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"complete_snapshot","arguments":{"title":"Completed by MCP","type":"feature","summary":"MCP writes snapshots.","validation":[{"command":"go test ./...","status":"passed"}]}}}`)
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
	repo := t.TempDir()
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
