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
