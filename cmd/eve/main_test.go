package main

import (
	"bytes"
	"encoding/json"
	"io"
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
	if !strings.Contains(stdout.String(), "protocol v1") {
		t.Fatalf("stdout = %q, want protocol version", stdout.String())
	}
}

func TestRunValidateValidFile(t *testing.T) {
	path := writeTempEvolution(t, validCLIJSON())
	var stdout, stderr bytes.Buffer

	code := run([]string{"validate", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "valid") {
		t.Fatalf("stdout = %q, want valid", stdout.String())
	}
}

func TestRunValidateInvalidFile(t *testing.T) {
	path := writeTempEvolution(t, `{"eve":{"version":2}}`)
	var stdout, stderr bytes.Buffer

	code := run([]string{"validate", path}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "invalid evolution") {
		t.Fatalf("stderr = %q, want validation error", stderr.String())
	}
}

func TestRunCanonicalize(t *testing.T) {
	path := writeTempEvolution(t, validCLIJSON())
	var stdout, stderr bytes.Buffer

	code := run([]string{"canonicalize", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if strings.Contains(strings.TrimSpace(stdout.String()), "\n") {
		t.Fatalf("stdout contains embedded newline: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"eve":{"version":1}`) {
		t.Fatalf("stdout = %q, want canonical JSON", stdout.String())
	}
}

func TestInitCreatesStructure(t *testing.T) {
	eveDir := filepath.Join(t.TempDir(), ".eve")
	t.Setenv("EVE_DIR", eveDir)

	var stdout, stderr bytes.Buffer
	code := run([]string{"init"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	for _, path := range []string{
		filepath.Join(eveDir, "config.json"),
		filepath.Join(eveDir, "staged"),
		filepath.Join(eveDir, "evolutions"),
		filepath.Join(eveDir, "sessions"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(eveDir, "next_id")); !os.IsNotExist(err) {
		t.Fatalf("next_id should not exist, err = %v", err)
	}
}

func TestAddStatusCommitWorkflow(t *testing.T) {
	eveDir := filepath.Join(t.TempDir(), ".eve")
	t.Setenv("EVE_DIR", eveDir)
	source := writeTranscriptSource(t)
	head := currentHead(t)

	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)

	stdout.Reset()
	stderr.Reset()
	code := run([]string{
		"add",
		"--title", "Enterprise SSO",
		"--type", "feature",
		"--behavior-added", "Organizations can log in via Okta",
		"--outcome", "Organizations can authenticate with Okta.",
		"--verification", "passed: go test ./...",
		"--session", "codex:session_912",
		"--session-source", source,
		"--implementation", "HEAD",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("add exit code = %d, want 0; stderr = %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"status"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("status exit code = %d, want 0; stderr = %s\nstdout = %s", code, stderr.String(), stdout.String())
	}
	for _, want := range []string{
		"Ready: yes",
		"Next ID: EV-001",
		"Commit message: EV-001 Enterprise SSO",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("status stdout = %q, want %q", stdout.String(), want)
		}
	}

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"commit"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("commit exit code = %d, want 0; stderr = %s\nstdout = %s", code, stderr.String(), stdout.String())
	}
	for _, want := range []string{
		"Created EV-001 Enterprise SSO",
		"git add .eve/",
		`git commit -m "EV-001 Enterprise SSO"`,
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("commit stdout = %q, want %q", stdout.String(), want)
		}
	}

	evolutionPath := filepath.Join(eveDir, "evolutions", "EV-001.json")
	if _, err := os.Stat(evolutionPath); err != nil {
		t.Fatalf("expected committed evolution: %v", err)
	}
	if _, err := os.Stat(filepath.Join(eveDir, "staged", "evolution.json")); !os.IsNotExist(err) {
		t.Fatalf("staged evolution should be cleared, err = %v", err)
	}
	for _, path := range []string{
		filepath.Join(eveDir, "sessions", "EV-001", "codex-session-912.md"),
		filepath.Join(eveDir, "sessions", "EV-001", "codex-session-912.jsonl"),
		filepath.Join(eveDir, "sessions", "EV-001", "manifest.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected session artifact %s: %v", path, err)
		}
	}

	raw, err := os.ReadFile(filepath.Join(eveDir, "sessions", "EV-001", "codex-session-912.jsonl"))
	if err != nil {
		t.Fatalf("read raw artifact: %v", err)
	}
	if strings.Contains(string(raw), "sk-1234567890abcdef") {
		t.Fatalf("raw artifact was not sanitized: %s", string(raw))
	}
	if !strings.Contains(readFile(t, evolutionPath), head) {
		t.Fatalf("committed evolution does not contain resolved HEAD %s", head)
	}
}

func TestManualAddSubcommands(t *testing.T) {
	eveDir := filepath.Join(t.TempDir(), ".eve")
	t.Setenv("EVE_DIR", eveDir)
	source := writeTranscriptSource(t)
	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)

	commands := [][]string{
		{"add", "title", "Enterprise SSO", "--type", "feature"},
		{"add", "behavior", "--added", "Organizations can log in via Okta"},
		{"add", "verification", "--status", "passed", "--reference", "go test ./..."},
		{"add", "session", "codex:session_912", "--source", source},
		{"add", "outcome", "Organizations can authenticate with Okta."},
		{"add", "implementation", "--snapshot", "HEAD", "--commit", "HEAD", "--repository", "eve", "--status", "merged"},
	}
	for _, command := range commands {
		stdout.Reset()
		stderr.Reset()
		code := run(command, &stdout, &stderr)
		if code != 0 {
			t.Fatalf("%v exit code = %d, stderr = %s", command, code, stderr.String())
		}
	}

	stdout.Reset()
	stderr.Reset()
	code := run([]string{"status"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("status exit code = %d, want 0; stderr = %s\nstdout = %s", code, stderr.String(), stdout.String())
	}
}

func TestStatusFailsWhenRequiredFieldsMissing(t *testing.T) {
	t.Setenv("EVE_DIR", filepath.Join(t.TempDir(), ".eve"))
	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)
	mustRun(t, []string{"add", "title", "Incomplete", "--type", "feature"}, &stdout, &stderr)

	stdout.Reset()
	stderr.Reset()
	code := run([]string{"status"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("status exit code = %d, want 1", code)
	}
	for _, want := range []string{"Ready: no", "outcome", "behavior", "verification", "session", "implementation snapshot"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("status stdout = %q, want %q", stdout.String(), want)
		}
	}
}

func TestStatusFailsForInvalidType(t *testing.T) {
	t.Setenv("EVE_DIR", filepath.Join(t.TempDir(), ".eve"))
	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)
	mustRun(t, []string{"add", "title", "Enterprise SSO", "--type", "migration"}, &stdout, &stderr)

	stdout.Reset()
	stderr.Reset()
	code := run([]string{"status"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("status exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "type must be one of feature, fix, refactor, docs, test, chore") {
		t.Fatalf("status stdout = %q, want allowed type validation", stdout.String())
	}
}

func TestNextIDScansExistingEvolutions(t *testing.T) {
	eveDir := filepath.Join(t.TempDir(), ".eve")
	t.Setenv("EVE_DIR", eveDir)
	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)
	writeCommittedEvolution(t, eveDir, "EV-001", "First")
	writeCommittedEvolution(t, eveDir, "EV-003", "Third")

	mustStageCompleteEvolution(t, eveDir)
	stdout.Reset()
	stderr.Reset()
	code := run([]string{"commit"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("commit exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Created EV-004") {
		t.Fatalf("commit stdout = %q, want EV-004", stdout.String())
	}
}

func TestSessionProvidersFromSource(t *testing.T) {
	for _, provider := range []string{"codex", "claude", "opencode", "pi"} {
		t.Run(provider, func(t *testing.T) {
			t.Setenv("EVE_DIR", filepath.Join(t.TempDir(), ".eve"))
			source := filepath.Join(t.TempDir(), provider+".md")
			if err := os.WriteFile(source, []byte("# transcript\n\nhello\n"), 0o600); err != nil {
				t.Fatalf("write source: %v", err)
			}
			var stdout, stderr bytes.Buffer
			mustRun(t, []string{"init"}, &stdout, &stderr)
			code := run([]string{"add", "session", provider + ":abc123", "--source", source}, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("add session exit code = %d; stderr = %s", code, stderr.String())
			}
		})
	}
}

func TestShowListTimelineGraphSearchAndSnapshot(t *testing.T) {
	eveDir := filepath.Join(t.TempDir(), ".eve")
	t.Setenv("EVE_DIR", eveDir)
	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)
	writeCommittedEvolution(t, eveDir, "EV-001", "Base")
	child := sampleEvolution("EV-002", "Enterprise SSO")
	child.Relationships.Extends = []string{"EV-001"}
	child.Behavior.Added = behaviorClaims("Organizations can log in via Okta")
	saveEvolutionJSON(t, filepath.Join(eveDir, "evolutions", "EV-002.json"), child)

	assertCommandContains(t, []string{"list"}, []string{"EV-001", "EV-002", "Enterprise SSO"})
	assertCommandContains(t, []string{"show", "EV-002"}, []string{"Enterprise SSO", "+ Organizations can log in via Okta", "Commits:"})
	assertCommandContains(t, []string{"timeline", "EV-002"}, []string{"EV-002 timeline"})
	assertCommandContains(t, []string{"graph"}, []string{"EV-001 Base", "`-- EV-002 Enterprise SSO"})
	assertCommandContains(t, []string{"search", "okta"}, []string{"EV-002"})
	assertCommandContains(t, []string{"snapshot", "EV-002"}, []string{"Snapshot", "Repository: eve", "Commit:"})
}

func TestSnapshotFailsWithoutImplementationCommit(t *testing.T) {
	eveDir := filepath.Join(t.TempDir(), ".eve")
	t.Setenv("EVE_DIR", eveDir)
	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)
	ev := sampleEvolution("EV-001", "No Commit")
	ev.Implementation.Snapshot = ""
	saveEvolutionJSON(t, filepath.Join(eveDir, "evolutions", "EV-001.json"), ev)

	code := run([]string{"snapshot", "EV-001"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("snapshot exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no resolvable code snapshot") {
		t.Fatalf("stderr = %q, want missing commit failure", stderr.String())
	}
}

func TestSnapshotAllowsExternalRepositoryMetadata(t *testing.T) {
	eveDir := filepath.Join(t.TempDir(), ".eve")
	t.Setenv("EVE_DIR", eveDir)
	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)
	ev := sampleEvolution("EV-001", "Multi Repo")
	ev.Implementation.Repositories["api"] = eve.Repository{Status: "merged"}
	saveEvolutionJSON(t, filepath.Join(eveDir, "evolutions", "EV-001.json"), ev)

	code := run([]string{"snapshot", "EV-001"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("snapshot exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Snapshot") {
		t.Fatalf("stdout = %q, want snapshot", stdout.String())
	}
}

func TestCheckoutFailsOnDirtyWorkingTree(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)

	eveDir := filepath.Join(t.TempDir(), ".eve")
	t.Setenv("EVE_DIR", eveDir)
	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)
	writeCommittedEvolution(t, eveDir, "EV-001", "Enterprise SSO")
	if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	code := run([]string{"checkout", "EV-001"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("checkout exit code = %d, want dirty-tree failure", code)
	}
	if !strings.Contains(stderr.String(), "Working tree has uncommitted changes.") {
		t.Fatalf("stderr = %q, want dirty-tree message", stderr.String())
	}
}

func TestCheckoutRestoresCleanRepository(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	first := gitOutput(t, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(repo, "product.txt"), []byte("second\n"), 0o644); err != nil {
		t.Fatalf("write product file: %v", err)
	}
	gitRun(t, "add", "product.txt")
	gitRun(t, "commit", "-m", "second")

	eveDir := filepath.Join(t.TempDir(), ".eve")
	t.Setenv("EVE_DIR", eveDir)
	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)
	ev := sampleEvolution("EV-001", "Enterprise SSO")
	ev.Implementation.Snapshot = first
	saveEvolutionJSON(t, filepath.Join(eveDir, "evolutions", "EV-001.json"), ev)

	stdout.Reset()
	stderr.Reset()
	code := run([]string{"checkout", "EV-001"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("checkout exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Product snapshot restored") {
		t.Fatalf("stdout = %q, want restored message", stdout.String())
	}
	if got := strings.TrimSpace(gitOutput(t, "rev-parse", "HEAD")); got != first {
		t.Fatalf("HEAD = %s, want %s", got, first)
	}
}

func TestUIAPIServesStaticAndEvolutionData(t *testing.T) {
	eveDir := filepath.Join(t.TempDir(), ".eve")
	t.Setenv("EVE_DIR", eveDir)
	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)
	ev := sampleEvolution("EV-001", "Git-like product staging")
	ev.Outcome = "Product history is browsable as snapshots."
	saveEvolutionJSON(t, filepath.Join(eveDir, "evolutions", "EV-001.json"), ev)

	handler := newUIServer(newStore(), "", "localhost:0").routes()

	static := httptest.NewRecorder()
	handler.ServeHTTP(static, httptest.NewRequest(http.MethodGet, "/", nil))
	if static.Code != http.StatusOK {
		t.Fatalf("static status = %d, want 200", static.Code)
	}
	if !strings.Contains(static.Body.String(), "EVE UI") {
		t.Fatalf("static body = %q, want EVE UI", static.Body.String())
	}

	var rows []evolutionSummary
	requestJSON(t, handler, http.MethodGet, "/api/evolutions", nil, &rows)
	if len(rows) != 1 || rows[0].ID != "EV-001" || rows[0].Snapshot == "" {
		t.Fatalf("timeline rows = %#v, want EV-001 with snapshot", rows)
	}

	var detail evolutionDetailResponse
	requestJSON(t, handler, http.MethodGet, "/api/evolutions/EV-001", nil, &detail)
	if detail.Evolution.Metadata.Title != "Git-like product staging" || len(detail.RawJSON) == 0 {
		t.Fatalf("detail = %#v, want evolution and raw JSON", detail)
	}

	var snapshot snapshotResponse
	requestJSON(t, handler, http.MethodGet, "/api/evolutions/EV-001/snapshot", nil, &snapshot)
	if snapshot.CheckoutCommand != "eve checkout EV-001" || snapshot.Commit == "" {
		t.Fatalf("snapshot = %#v, want checkout command and commit", snapshot)
	}

	var search searchResponse
	requestJSON(t, handler, http.MethodGet, "/api/search?q=snapshots", nil, &search)
	if len(search.Results) != 1 || search.Results[0].Evolution.ID != "EV-001" {
		t.Fatalf("search = %#v, want EV-001", search)
	}
}

func TestUIAPISessionTranscript(t *testing.T) {
	eveDir := filepath.Join(t.TempDir(), ".eve")
	t.Setenv("EVE_DIR", eveDir)
	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)
	ev := sampleEvolution("EV-001", "Session Reader")
	saveEvolutionJSON(t, filepath.Join(eveDir, "evolutions", "EV-001.json"), ev)
	store := newStore()
	_, err := store.writeSessionArtifacts(
		store.sessionDir("EV-001"),
		store.sessionManifestPath("EV-001"),
		"EV-001",
		"codex",
		"session_912",
		"Implementation Session",
		"md",
		[]byte("# Implementation Session\n\nVerified transcript search.\n"),
		true,
		"fixture.md",
	)
	if err != nil {
		t.Fatalf("write session artifacts: %v", err)
	}

	handler := newUIServer(newStore(), "", "localhost:0").routes()
	var sessions sessionListResponse
	requestJSON(t, handler, http.MethodGet, "/api/evolutions/EV-001/sessions", nil, &sessions)
	if len(sessions.Sessions) != 1 || !sessions.Sessions[0].HasTranscript {
		t.Fatalf("sessions = %#v, want transcript", sessions)
	}

	var transcript sessionTranscriptResponse
	requestJSON(t, handler, http.MethodGet, "/api/evolutions/EV-001/sessions/codex%3Asession_912", nil, &transcript)
	if !strings.Contains(transcript.Markdown, "Verified transcript search.") {
		t.Fatalf("transcript = %#v, want markdown", transcript)
	}

	var search searchResponse
	requestJSON(t, handler, http.MethodGet, "/api/search?q=transcript", nil, &search)
	if len(search.Results) != 1 {
		t.Fatalf("search = %#v, want transcript match", search)
	}
}

func TestUIAPICheckoutReportsDirtyWorkingTree(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	if err := os.WriteFile(filepath.Join(repo, "dirty.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	eveDir := filepath.Join(t.TempDir(), ".eve")
	t.Setenv("EVE_DIR", eveDir)
	var stdout, stderr bytes.Buffer
	mustRun(t, []string{"init"}, &stdout, &stderr)
	writeCommittedEvolution(t, eveDir, "EV-001", "Dirty Checkout")

	handler := newUIServer(newStore(), "", "localhost:0").routes()
	var checkout checkoutResponse
	requestJSON(t, handler, http.MethodPost, "/api/evolutions/EV-001/checkout", nil, &checkout)
	if checkout.ExitCode != 1 || !strings.Contains(checkout.Stderr, "Working tree has uncommitted changes.") {
		t.Fatalf("checkout = %#v, want dirty-tree failure", checkout)
	}
}

func mustRun(t *testing.T, args []string, stdout *bytes.Buffer, stderr *bytes.Buffer) {
	t.Helper()
	stdout.Reset()
	stderr.Reset()
	code := run(args, stdout, stderr)
	if code != 0 {
		t.Fatalf("%v exit code = %d, stderr = %s", args, code, stderr.String())
	}
}

func requestJSON(t *testing.T, handler http.Handler, method string, target string, body io.Reader, out any) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, body))
	if recorder.Code < 200 || recorder.Code >= 300 {
		t.Fatalf("%s %s status = %d; body = %s", method, target, recorder.Code, recorder.Body.String())
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), out); err != nil {
		t.Fatalf("decode %s %s: %v; body = %s", method, target, err, recorder.Body.String())
	}
}

func assertCommandContains(t *testing.T, args []string, wants []string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("%v exit code = %d; stderr = %s", args, code, stderr.String())
	}
	for _, want := range wants {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("%v stdout = %q, want %q", args, stdout.String(), want)
		}
	}
}

func writeTempEvolution(t *testing.T, input string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "evolution.json")
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatalf("write temp evolution: %v", err)
	}
	return path
}

func writeTranscriptSource(t *testing.T) string {
	t.Helper()
	source := filepath.Join(t.TempDir(), "codex-session_912.jsonl")
	transcript := `{"role":"user","content":"implement Okta login","api_key":"sk-1234567890abcdef"}
{"role":"assistant","content":"implemented Okta callback flow"}
`
	if err := os.WriteFile(source, []byte(transcript), 0o600); err != nil {
		t.Fatalf("write source transcript: %v", err)
	}
	return source
}

func currentHead(t *testing.T) string {
	t.Helper()
	commit, err := resolveCommit("HEAD")
	if err != nil {
		t.Fatalf("resolve HEAD: %v", err)
	}
	return commit
}

func mustStageCompleteEvolution(t *testing.T, eveDir string) {
	t.Helper()
	source := writeTranscriptSource(t)
	var stdout, stderr bytes.Buffer
	commands := [][]string{
		{"add", "--title", "Enterprise SSO", "--type", "feature", "--behavior-added", "Organizations can log in via Okta", "--outcome", "Organizations can authenticate with Okta.", "--verification", "passed: go test ./...", "--session", "codex:session_912", "--session-source", source, "--implementation", "HEAD"},
	}
	for _, command := range commands {
		mustRun(t, command, &stdout, &stderr)
	}
}

func writeCommittedEvolution(t *testing.T, eveDir string, id string, title string) {
	t.Helper()
	saveEvolutionJSON(t, filepath.Join(eveDir, "evolutions", id+".json"), sampleEvolution(id, title))
}

func saveEvolutionJSON(t *testing.T, path string, evolution *eve.Evolution) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	canonical, err := eve.CanonicalJSON(evolution)
	if err != nil {
		t.Fatalf("canonical JSON: %v", err)
	}
	if err := os.WriteFile(path, append(canonical, '\n'), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func sampleEvolution(id string, title string) *eve.Evolution {
	head, _ := resolveCommit("HEAD")
	if head == "" {
		head = "HEAD"
	}
	return &eve.Evolution{
		EVE: eve.EVEHeader{Version: eve.ProtocolVersion},
		Metadata: eve.Metadata{
			ID:        id,
			Title:     title,
			Type:      "feature",
			Status:    "completed",
			CreatedBy: "test",
			CreatedAt: "2026-07-02T00:00:00Z",
			UpdatedAt: "2026-07-02T00:00:00Z",
		},
		Intent:  title,
		Outcome: "Organizations can authenticate with Okta.",
		Behavior: eve.Behavior{
			Added: []eve.BehaviorClaim{{Description: "Organizations can log in via Okta"}},
		},
		Decisions:     []json.RawMessage{},
		Risks:         []json.RawMessage{},
		Verification:  []eve.Verification{{Type: "tests", Status: "passed", Reference: "go test ./..."}},
		Sessions:      []eve.Session{{Provider: "codex", ID: "session_912"}},
		Timeline:      []eve.TimelineEntry{{Timestamp: "2026-07-02T00:00:00Z", Event: "committed", Description: "Committed evolution."}},
		Relationships: eve.Relationships{},
		Implementation: eve.Implementation{
			Repositories: map[string]eve.Repository{"eve": {Status: "merged"}},
			Snapshot:     head,
			Commits:      []string{head},
		},
		Extensions: map[string]json.RawMessage{},
	}
}

func behaviorClaims(description string) []eve.BehaviorClaim {
	return []eve.BehaviorClaim{{Description: description}}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func initTempGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitIn(t, repo, "init")
	runGitIn(t, repo, "config", "user.email", "test@example.com")
	runGitIn(t, repo, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(repo, "product.txt"), []byte("first\n"), 0o644); err != nil {
		t.Fatalf("write product file: %v", err)
	}
	runGitIn(t, repo, "add", "product.txt")
	runGitIn(t, repo, "commit", "-m", "first")
	return repo
}

func gitRun(t *testing.T, args ...string) {
	t.Helper()
	runGitIn(t, "", args...)
}

func runGitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func gitOutput(t *testing.T, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(string(output))
}

func validCLIJSON() string {
	return `{
  "eve": {"version": 1},
  "metadata": {"status": "active", "type": "custom"},
  "intent": "Document work.",
  "outcome": "Work is documented.",
  "behavior": {},
  "decisions": [],
  "risks": [],
  "verification": [{"status": "passed"}],
  "sessions": [],
  "timeline": [],
  "relationships": {},
  "implementation": {"repositories": {"web": {"status": "merged"}}},
  "extensions": {"acme": {"rollout": "25%"}}
}`
}
