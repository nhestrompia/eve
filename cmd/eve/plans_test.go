package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nhestrompia/eve"
)

func setupPlanTestRepo(t *testing.T) repository {
	t.Helper()
	root := initTempGitRepo(t)
	t.Chdir(root)
	var stdout, stderr bytes.Buffer
	if code := run([]string{"init", "--no-agent-instructions"}, &stdout, &stderr); code != 0 {
		t.Fatalf("eve init = %d: %s", code, stderr.String())
	}
	gitRun(t, root, "add", ".eve/config.json")
	gitRun(t, root, "commit", "-m", "initialize eve")
	repo := repoFromRoot(root)
	if _, err := repo.createSkip("test fixture baseline", skipAgent{Provider: "test", ID: "fixture"}); err != nil {
		t.Fatalf("create baseline skip: %v", err)
	}
	gitRun(t, root, "add", ".eve/skips")
	gitRun(t, root, "commit", "-m", "record fixture baseline")
	branch := gitOutputForTest(t, root, "branch", "--show-current")
	head := gitOutputForTest(t, root, "rev-parse", "HEAD")
	if err := repo.resolvePendingBranch(branch, head); err != nil {
		t.Fatalf("resolve fixture baseline: %v", err)
	}
	return repo
}

func testPlanInput(id string) declarePlanInput {
	return declarePlanInput{
		PlanRequestID:      id,
		Goal:               "Add resumable plan approval",
		AcceptanceCriteria: "- The request survives cancellation\n- Scope drift is recorded",
		AllowedPathGlobs:   []string{"product.txt", "cmd/**"},
		Milestones:         []eve.PlanMilestone{{Title: "Protocol"}, {Title: "Surface"}},
	}
}

func TestPlanGlobValidationAndMatching(t *testing.T) {
	for _, invalid := range []string{"", "/absolute/**", "C:/absolute/**", "../escape", "cmd/../escape", "!secret/**", "cmd/[ab].go", `cmd\*.go`, "cmd//*.go", "cmd/***.go"} {
		if err := validatePlanGlob(invalid); err == nil {
			t.Errorf("validatePlanGlob(%q) succeeded", invalid)
		}
	}
	cases := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"cmd/*.go", "cmd/main.go", true},
		{"cmd/*.go", "cmd/eve/main.go", false},
		{"cmd/**", "cmd/eve/main.go", true},
		{"**/*.go", "main.go", true},
		{"**/*.go", "cmd/main.go", true},
		{"README.*", "README.md", true},
		{"README.*", "readme.md", false},
	}
	for _, tc := range cases {
		if got := planGlobMatches(tc.pattern, tc.path); got != tc.want {
			t.Errorf("planGlobMatches(%q, %q) = %t, want %t", tc.pattern, tc.path, got, tc.want)
		}
	}
}

func TestPlanRequestResumeSupersedeEditAndLockedConflict(t *testing.T) {
	repo := setupPlanTestRepo(t)
	ctx := context.Background()
	first, err := repo.createOrResumePlanRequest(ctx, testPlanInput("planreq_first0001"))
	if err != nil {
		t.Fatalf("declare first: %v", err)
	}
	if first.State != "pending_approval" {
		t.Fatalf("first state = %s", first.State)
	}
	privatePath, _ := repo.planRequestPath(first.PlanRequestID)
	if !strings.HasPrefix(privatePath, filepath.Join(repo.Root, ".git", "eve")) {
		t.Fatalf("private request path = %s", privatePath)
	}
	if status := gitOutputForTest(t, repo.Root, "status", "--porcelain"); status != "" {
		t.Fatalf("request dirtied implementation tree: %q", status)
	}
	resumed, err := repo.createOrResumePlanRequest(ctx, testPlanInput("planreq_first0001"))
	if err != nil || resumed.CreatedAt != first.CreatedAt {
		t.Fatalf("idempotent resume = %#v, %v", resumed, err)
	}
	conflict := testPlanInput("planreq_first0001")
	conflict.Goal = "Different goal"
	if _, err := repo.createOrResumePlanRequest(ctx, conflict); err == nil {
		t.Fatal("conflicting ID reuse succeeded")
	}

	second, err := repo.createOrResumePlanRequest(ctx, testPlanInput("planreq_second002"))
	if err != nil {
		t.Fatalf("declare second: %v", err)
	}
	first, _ = repo.loadPlanRequest(first.PlanRequestID)
	if first.State != "superseded" || first.SupersededBy != second.PlanRequestID ||
		!containsString(second.Supersedes, first.PlanRequestID) {
		t.Fatalf("superseded request = %#v", first)
	}
	edited := second.Revisions[0]
	proposal := planProposal{
		Goal:               edited.Goal + " with human edits",
		AcceptanceCriteria: edited.AcceptanceCriteria,
		AllowedPathGlobs:   edited.AllowedPathGlobs,
		Milestones:         edited.Milestones,
		RequiredSuite:      edited.ConfiguredSuite,
	}
	locked, err := repo.approvePlanRequest(ctx, second.PlanRequestID, 1, &proposal)
	if err != nil {
		t.Fatalf("approve edited: %v", err)
	}
	if locked.State != "locked" || locked.LockedRevision != 2 || len(locked.Revisions) != 2 ||
		locked.Revisions[0].Source != "agent" || locked.Revisions[1].Source != "human" {
		t.Fatalf("locked request = %#v", locked)
	}
	if _, err := repo.createOrResumePlanRequest(ctx, testPlanInput("planreq_third0003")); err == nil || !strings.Contains(err.Error(), "must be fulfilled") {
		t.Fatalf("locked conflict error = %v", err)
	}
}

func TestPlanRequestIdempotencyPreservesConfiguredAndResolvedSuite(t *testing.T) {
	repo := setupPlanTestRepo(t)
	configurePlanPolicy(t, repo, `{"schemaVersion":3,"snapshotSchema":"0.3.0","verification":{"checks":{"unit":{"argv":["go","version"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":1000}},"suites":{"change":["unit"]},"profileRules":[{"default":"change"}]}}`+"\n")
	input := testPlanInput("planreq_defaultsuite")
	request, err := repo.createOrResumePlanRequest(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if request.Revisions[0].ConfiguredSuite != "" || request.Revisions[0].ResolvedSuite != "change" ||
		!containsString(request.AvailableSuites, "change") {
		t.Fatalf("suite references = %#v, options = %#v", request.Revisions[0], request.AvailableSuites)
	}
	if _, err := repo.createOrResumePlanRequest(context.Background(), input); err != nil {
		t.Fatalf("identical default-suite retry conflicted: %v", err)
	}
}

func TestPlanWaitersWakeForApprovalRejectionAndSupersession(t *testing.T) {
	t.Run("approval", func(t *testing.T) {
		repo := setupPlanTestRepo(t)
		request, err := repo.createOrResumePlanRequest(context.Background(), testPlanInput("planreq_waitlock1"))
		if err != nil {
			t.Fatal(err)
		}
		result := make(chan *planRequest, 1)
		go func() {
			waited, _ := repo.waitForPlanRequest(context.Background(), request.PlanRequestID)
			result <- waited
		}()
		if _, err := repo.approvePlanRequest(context.Background(), request.PlanRequestID, 1, nil); err != nil {
			t.Fatal(err)
		}
		if waited := <-result; waited == nil || waited.State != "locked" {
			t.Fatalf("approval waiter = %#v", waited)
		}
	})

	t.Run("rejection", func(t *testing.T) {
		repo := setupPlanTestRepo(t)
		request, err := repo.createOrResumePlanRequest(context.Background(), testPlanInput("planreq_waitreject"))
		if err != nil {
			t.Fatal(err)
		}
		result := make(chan *planRequest, 1)
		go func() {
			waited, _ := repo.waitForPlanRequest(context.Background(), request.PlanRequestID)
			result <- waited
		}()
		if _, err := repo.rejectPlanRequest(context.Background(), request.PlanRequestID, 1, "Not yet"); err != nil {
			t.Fatal(err)
		}
		if waited := <-result; waited == nil || waited.State != "rejected" || waited.RejectionFeedback != "Not yet" {
			t.Fatalf("rejection waiter = %#v", waited)
		}
	})

	t.Run("supersession is non-error", func(t *testing.T) {
		repo := setupPlanTestRepo(t)
		first, err := repo.createOrResumePlanRequest(context.Background(), testPlanInput("planreq_waitfirst1"))
		if err != nil {
			t.Fatal(err)
		}
		result := make(chan *planRequest, 1)
		go func() {
			waited, _ := repo.waitForPlanRequest(context.Background(), first.PlanRequestID)
			result <- waited
		}()
		second, err := repo.createOrResumePlanRequest(context.Background(), testPlanInput("planreq_waitsecond"))
		if err != nil {
			t.Fatal(err)
		}
		if waited := <-result; waited == nil || waited.State != "superseded" || waited.SupersededBy != second.PlanRequestID {
			t.Fatalf("supersession waiter = %#v", waited)
		}
	})
}

func TestPlanRejectRequiresFeedbackAndPlanBecomesStale(t *testing.T) {
	t.Run("reject", func(t *testing.T) {
		repo := setupPlanTestRepo(t)
		request, err := repo.createOrResumePlanRequest(context.Background(), testPlanInput("planreq_reject001"))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repo.rejectPlanRequest(context.Background(), request.PlanRequestID, 1, "  "); err == nil {
			t.Fatal("empty rejection feedback succeeded")
		}
		rejected, err := repo.rejectPlanRequest(context.Background(), request.PlanRequestID, 1, "Scope is too broad")
		if err != nil || rejected.State != "rejected" || rejected.RejectionFeedback == "" {
			t.Fatalf("reject = %#v, %v", rejected, err)
		}
	})

	t.Run("tracked tree changes", func(t *testing.T) {
		repo := setupPlanTestRepo(t)
		request, err := repo.createOrResumePlanRequest(context.Background(), testPlanInput("planreq_stale0001"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo.Root, "product.txt"), []byte("changed\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		stale, err := repo.refreshPlanRequestState(context.Background(), request.PlanRequestID)
		if err != nil {
			t.Fatal(err)
		}
		if stale.State != "stale" || !containsString(stale.StaleReasons, "working tree changed") {
			t.Fatalf("stale request = %#v", stale)
		}
		if _, err := repo.approvePlanRequest(context.Background(), request.PlanRequestID, 1, nil); err == nil {
			t.Fatal("stale approval succeeded")
		}
	})

	t.Run("reject detects stale repository context", func(t *testing.T) {
		repo := setupPlanTestRepo(t)
		request, err := repo.createOrResumePlanRequest(context.Background(), testPlanInput("planreq_rejectstale"))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo.Root, "product.txt"), []byte("changed before rejection\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.rejectPlanRequest(context.Background(), request.PlanRequestID, 1, "Try again"); err == nil ||
			!strings.Contains(err.Error(), "stale") {
			t.Fatalf("stale rejection error = %v", err)
		}
		stale, err := repo.loadPlanRequest(request.PlanRequestID)
		if err != nil || stale.State != "stale" || stale.RejectionFeedback != "" {
			t.Fatalf("stale rejected request = %#v, %v", stale, err)
		}
	})

	t.Run("head changes", func(t *testing.T) {
		repo := setupPlanTestRepo(t)
		request, err := repo.createOrResumePlanRequest(context.Background(), testPlanInput("planreq_stalehead"))
		if err != nil {
			t.Fatal(err)
		}
		commitProductChangeAt(t, repo.Root, "product.txt", "new head\n", "change head", time.Now())
		stale, err := repo.refreshPlanRequestState(context.Background(), request.PlanRequestID)
		if err != nil {
			t.Fatal(err)
		}
		if stale.State != "stale" || !containsString(stale.StaleReasons, "repository HEAD changed") {
			t.Fatalf("stale reasons = %#v", stale.StaleReasons)
		}
		if containsString(stale.StaleReasons, "working tree changed") {
			t.Fatalf("HEAD-only change was also reported as a tree change: %#v", stale.StaleReasons)
		}
	})
}

func TestEveryPlanStalenessContextIsDetected(t *testing.T) {
	cases := []struct {
		name   string
		reason string
		change func(t *testing.T, repo repository)
	}{
		{
			name: "branch", reason: "repository branch changed",
			change: func(t *testing.T, repo repository) { gitRun(t, repo.Root, "checkout", "-b", "other") },
		},
		{
			name: "untracked tree", reason: "working tree changed",
			change: func(t *testing.T, repo repository) {
				if err := os.WriteFile(filepath.Join(repo.Root, "new.txt"), []byte("new\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "policy and suite definitions", reason: "verification policy hash changed",
			change: func(t *testing.T, repo repository) {
				config := `{"schemaVersion":3,"snapshotSchema":"0.3.0","verification":{"checks":{"unit":{"argv":["go","version"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":1000}},"suites":{"change":["unit"]},"profileRules":[{"default":"change"}]}}` + "\n"
				if err := os.WriteFile(repo.configPath(), []byte(config), 0o644); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := setupPlanTestRepo(t)
			request, err := repo.createOrResumePlanRequest(context.Background(), testPlanInput("planreq_context01"))
			if err != nil {
				t.Fatal(err)
			}
			tc.change(t, repo)
			stale, err := repo.refreshPlanRequestState(context.Background(), request.PlanRequestID)
			if err != nil {
				t.Fatal(err)
			}
			if stale.State != "stale" || !containsString(stale.StaleReasons, tc.reason) {
				t.Fatalf("reasons = %#v, want %q", stale.StaleReasons, tc.reason)
			}
			if tc.name == "policy and suite definitions" && !containsString(stale.StaleReasons, "resolved check suite changed") {
				t.Fatalf("suite change reason missing: %#v", stale.StaleReasons)
			}
		})
	}
}

func TestPlanApprovalAPIStatusCodesAndHumanRevision(t *testing.T) {
	repo := setupPlanTestRepo(t)
	request, err := repo.createOrResumePlanRequest(context.Background(), testPlanInput("planreq_api000001"))
	if err != nil {
		t.Fatal(err)
	}
	handler := newRuntimeServer(repo, "").routes()
	listRecorder := httptest.NewRecorder()
	handler.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/api/plan-requests?status=pending_approval", nil))
	if listRecorder.Code != http.StatusOK || !strings.Contains(listRecorder.Body.String(), request.PlanRequestID) {
		t.Fatalf("pending queue = %d: %s", listRecorder.Code, listRecorder.Body.String())
	}

	post := func(path string, body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		handler.ServeHTTP(recorder, req)
		return recorder
	}
	if got := post("/api/plan-requests/"+request.PlanRequestID+"/approve", `{"expectedRevision":9}`); got.Code != http.StatusConflict {
		t.Fatalf("revision conflict = %d: %s", got.Code, got.Body.String())
	}
	if got := post("/api/plan-requests/"+request.PlanRequestID+"/reject", `{"expectedRevision":1,"feedback":""}`); got.Code != http.StatusUnprocessableEntity {
		t.Fatalf("empty reject = %d: %s", got.Code, got.Body.String())
	}
	body := `{"expectedRevision":1,"proposal":{"goal":"Human goal","acceptanceCriteria":"- accepted","allowedPathGlobs":["product.txt"],"milestones":[]}}`
	got := post("/api/plan-requests/"+request.PlanRequestID+"/approve", body)
	if got.Code != http.StatusOK {
		t.Fatalf("approve = %d: %s", got.Code, got.Body.String())
	}
	var approved planRequest
	if err := json.Unmarshal(got.Body.Bytes(), &approved); err != nil {
		t.Fatal(err)
	}
	if approved.LockedRevision != 2 || approved.Revisions[1].Source != "human" {
		t.Fatalf("approved = %#v", approved)
	}
}

func TestPlanApprovalAPIEmptyQueueIsAnArray(t *testing.T) {
	repo := setupPlanTestRepo(t)
	handler := newRuntimeServer(repo, "").routes()
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/plan-requests?status=pending_approval", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("empty pending queue = %d: %s", recorder.Code, recorder.Body.String())
	}
	if got := strings.TrimSpace(recorder.Body.String()); got != "[]" {
		t.Fatalf("empty pending queue = %s, want []", got)
	}
}

func TestPlanRejectionAPIReturnsConflictForStaleRepositoryContext(t *testing.T) {
	repo := setupPlanTestRepo(t)
	request, err := repo.createOrResumePlanRequest(context.Background(), testPlanInput("planreq_apistale01"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo.Root, "product.txt"), []byte("changed before API rejection\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	handler := newRuntimeServer(repo, "").routes()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(
		http.MethodPost,
		"/api/plan-requests/"+request.PlanRequestID+"/reject",
		strings.NewReader(`{"expectedRevision":1,"feedback":"Try again"}`),
	))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("stale rejection = %d: %s", recorder.Code, recorder.Body.String())
	}
	stale, err := repo.loadPlanRequest(request.PlanRequestID)
	if err != nil || stale.State != "stale" || !containsString(stale.StaleReasons, "working tree changed") {
		t.Fatalf("stale request = %#v, %v", stale, err)
	}
}

func TestPlanRequestSSEStartsWithFullQueueAndDaemonRejectsRemoteBinding(t *testing.T) {
	repo := setupPlanTestRepo(t)
	request, err := repo.createOrResumePlanRequest(context.Background(), testPlanInput("planreq_sse000001"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	recorder := newLockedSSERecorder()
	done := make(chan struct{})
	go func() {
		newRuntimeServer(repo, "").routes().ServeHTTP(
			recorder,
			httptest.NewRequest(http.MethodGet, "/api/plan-requests/events", nil).WithContext(ctx),
		)
		close(done)
	}()
	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(recorder.String(), "\n\n") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	<-done
	reader := bufio.NewReader(strings.NewReader(recorder.String()))
	event, _ := reader.ReadString('\n')
	data, _ := reader.ReadString('\n')
	if event != "event: plan-requests\n" || !strings.Contains(data, request.PlanRequestID) {
		t.Fatalf("initial SSE event = %q %q", event, data)
	}

	var stdout, stderr bytes.Buffer
	if code := runDaemon([]string{"--addr", "0.0.0.0:4317", "--cwd", repo.Root}, &stdout, &stderr); code != 2 ||
		!strings.Contains(stderr.String(), "only binds to localhost") {
		t.Fatalf("remote bind = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if isLocalhostAddr(":4317") || !isLocalhostAddr("[::1]:4317") {
		t.Fatal("localhost address validation accepted a wildcard bind or rejected IPv6 loopback")
	}
}

type lockedSSERecorder struct {
	mu     sync.Mutex
	header http.Header
	body   bytes.Buffer
	status int
}

func newLockedSSERecorder() *lockedSSERecorder {
	return &lockedSSERecorder{header: make(http.Header)}
}

func (recorder *lockedSSERecorder) Header() http.Header { return recorder.header }

func (recorder *lockedSSERecorder) WriteHeader(status int) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.status = status
}

func (recorder *lockedSSERecorder) Write(data []byte) (int, error) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return recorder.body.Write(data)
}

func (recorder *lockedSSERecorder) Flush() {}

func (recorder *lockedSSERecorder) String() string {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return recorder.body.String()
}

func TestMCPStdioPersistsBeforeWaitAndProcessesCancellation(t *testing.T) {
	repo := setupPlanTestRepo(t)
	reader, writer := io.Pipe()
	var stdout, stderr bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- runMCPStdio([]string{"--cwd", repo.Root}, reader, &stdout, &stderr)
	}()
	declare := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"declare_plan","arguments":{"planRequestId":"planreq_stdio001","goal":"Wait safely","acceptanceCriteria":"- persisted","allowedPathGlobs":["product.txt"]}}}`
	list := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	cancel := `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":1}}`
	if _, err := io.WriteString(writer, declare+"\n"+list+"\n"+cancel+"\n"); err != nil {
		t.Fatal(err)
	}
	_ = writer.Close()
	if code := <-done; code != 0 {
		t.Fatalf("stdio code = %d stderr = %s", code, stderr.String())
	}
	request, err := repo.loadPlanRequest("planreq_stdio001")
	if err != nil || request.State != "pending_approval" {
		t.Fatalf("persisted request = %#v, %v", request, err)
	}
	output := stdout.String()
	if !strings.Contains(output, `"id":2`) || !strings.Contains(output, `"declare_plan"`) {
		t.Fatalf("concurrent tools/list response missing: %s", output)
	}
}

func TestPlanLockSerializesAcrossProcesses(t *testing.T) {
	repo := setupPlanTestRepo(t)
	firstReady := filepath.Join(t.TempDir(), "first-ready")
	secondReady := filepath.Join(t.TempDir(), "second-ready")
	command := func(ready string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestPlanLockProcessHelper$")
		cmd.Env = append(os.Environ(),
			"EVE_PLAN_LOCK_HELPER=1",
			"EVE_PLAN_LOCK_REPO="+repo.Root,
			"EVE_PLAN_LOCK_READY="+ready,
		)
		return cmd
	}
	first := command(firstReady)
	if err := first.Start(); err != nil {
		t.Fatal(err)
	}
	waitForFile(t, firstReady, 2*time.Second)
	started := time.Now()
	second := command(secondReady)
	if err := second.Start(); err != nil {
		t.Fatal(err)
	}
	waitForFile(t, secondReady, 2*time.Second)
	if elapsed := time.Since(started); elapsed < 250*time.Millisecond {
		t.Fatalf("second process acquired lock after %s; expected serialization", elapsed)
	}
	if err := first.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := second.Wait(); err != nil {
		t.Fatal(err)
	}
}

func TestPlanLockProcessHelper(t *testing.T) {
	if os.Getenv("EVE_PLAN_LOCK_HELPER") != "1" {
		return
	}
	repo := repoFromRoot(os.Getenv("EVE_PLAN_LOCK_REPO"))
	err := repo.withPlanLock(context.Background(), func() error {
		if err := os.WriteFile(os.Getenv("EVE_PLAN_LOCK_READY"), []byte("ready"), 0o600); err != nil {
			return err
		}
		time.Sleep(400 * time.Millisecond)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func waitForFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func TestPlanRenameConformanceEvaluatesOldAndNewPaths(t *testing.T) {
	repo := setupPlanTestRepo(t)
	request, err := repo.createOrResumePlanRequest(context.Background(), declarePlanInput{
		PlanRequestID:      "planreq_rename001",
		Goal:               "Move the product file",
		AcceptanceCriteria: "- rename is visible",
		AllowedPathGlobs:   []string{"docs/**"},
	})
	if err != nil {
		t.Fatal(err)
	}
	locked, err := repo.approvePlanRequest(context.Background(), request.PlanRequestID, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo.Root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo.Root, "mv", "product.txt", "docs/product.txt")
	gitRun(t, repo.Root, "commit", "-m", "move product")
	facts, err := deriveGitFacts(repo)
	if err != nil {
		t.Fatal(err)
	}
	verification, err := snapshotVerification(repo, facts)
	if err != nil {
		t.Fatal(err)
	}
	conformance, err := repo.evaluatePlanConformance(locked, facts, verification)
	if err != nil {
		t.Fatal(err)
	}
	if !conformance.ScopeDrift || !containsString(conformance.ChangedPaths, "product.txt") ||
		!containsString(conformance.ChangedPaths, "docs/product.txt") || !containsString(conformance.OutOfScopePaths, "product.txt") {
		t.Fatalf("rename conformance = %#v", conformance)
	}
}

func TestPlanConformanceRecordsFailedChecksAndPostLockPolicyDrift(t *testing.T) {
	t.Run("missing run evidence is incomplete", func(t *testing.T) {
		repo := setupPlanTestRepo(t)
		configurePlanPolicy(t, repo, `{"schemaVersion":3,"snapshotSchema":"0.3.0","verification":{"checks":{"unit":{"argv":["go","version"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":1000}},"suites":{"change":["unit"]},"profileRules":[{"default":"change"}]}}`+"\n")
		request, err := repo.createOrResumePlanRequest(context.Background(), declarePlanInput{
			PlanRequestID: "planreq_incomplete", Goal: "Record absent evidence",
			AcceptanceCriteria: "- absence is incomplete", AllowedPathGlobs: []string{"product.txt"}, RequiredSuite: "change",
		})
		if err != nil {
			t.Fatal(err)
		}
		locked, err := repo.approvePlanRequest(context.Background(), request.PlanRequestID, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		commitProductChangeAt(t, repo.Root, "product.txt", "missing evidence\n", "implement without verification", time.Now())
		facts := completionFactsForTest(t, repo)
		verification, err := snapshotVerification(repo, facts)
		if err != nil {
			t.Fatal(err)
		}
		conformance, err := repo.evaluatePlanConformance(locked, facts, verification)
		if err != nil {
			t.Fatal(err)
		}
		if conformance.Status != "incomplete" || conformance.RequiredChecksStatus != "incomplete" {
			t.Fatalf("missing-evidence conformance = %#v", conformance)
		}
	})

	t.Run("different selected suite fails definition match", func(t *testing.T) {
		repo := setupPlanTestRepo(t)
		configurePlanPolicy(t, repo, `{"schemaVersion":3,"snapshotSchema":"0.3.0","verification":{"checks":{"unit":{"argv":["go","version"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":1000}},"suites":{"change":["unit"],"extended":["unit"]},"profileRules":[{"default":"change"}]}}`+"\n")
		request, err := repo.createOrResumePlanRequest(context.Background(), declarePlanInput{
			PlanRequestID: "planreq_wrongsuite", Goal: "Require the declared suite",
			AcceptanceCriteria: "- suite identity matches", AllowedPathGlobs: []string{"product.txt"}, RequiredSuite: "change",
		})
		if err != nil {
			t.Fatal(err)
		}
		locked, err := repo.approvePlanRequest(context.Background(), request.PlanRequestID, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		commitProductChangeAt(t, repo.Root, "product.txt", "wrong suite\n", "implement wrong suite", time.Now())
		server := newRuntimeServer(repo, "")
		run, err := server.startVerificationRun(context.Background(), repo, "", "extended", "")
		if err != nil {
			t.Fatal(err)
		}
		waitForVerificationRun(t, server, repo, run.RunID)
		facts := completionFactsForTest(t, repo)
		verification, err := snapshotVerification(repo, facts)
		if err != nil {
			t.Fatal(err)
		}
		conformance, err := repo.evaluatePlanConformance(locked, facts, verification)
		if err != nil {
			t.Fatal(err)
		}
		if conformance.Status != "failed" || conformance.CheckDefinitionsMatch {
			t.Fatalf("wrong-suite conformance = %#v", conformance)
		}
	})

	t.Run("required check failure", func(t *testing.T) {
		repo := setupPlanTestRepo(t)
		configurePlanPolicy(t, repo, `{"schemaVersion":3,"snapshotSchema":"0.3.0","verification":{"checks":{"unit":{"argv":["sh","-c","exit 9"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":1000}},"suites":{"change":["unit"]},"profileRules":[{"default":"change"}]}}`+"\n")
		request, err := repo.createOrResumePlanRequest(context.Background(), declarePlanInput{
			PlanRequestID: "planreq_failcheck1", Goal: "Run a failing check",
			AcceptanceCriteria: "- failure is recorded", AllowedPathGlobs: []string{"product.txt"}, RequiredSuite: "change",
		})
		if err != nil {
			t.Fatal(err)
		}
		locked, err := repo.approvePlanRequest(context.Background(), request.PlanRequestID, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		commitProductChangeAt(t, repo.Root, "product.txt", "failed check\n", "implement failing plan", time.Now())
		server := newRuntimeServer(repo, "")
		run, err := server.startVerificationRun(context.Background(), repo, "", "change", "")
		if err != nil {
			t.Fatal(err)
		}
		terminal := waitForVerificationRun(t, server, repo, run.RunID)
		facts := completionFactsForTest(t, repo)
		verification, err := snapshotVerification(repo, facts)
		if err != nil {
			t.Fatal(err)
		}
		conformance, err := repo.evaluatePlanConformance(locked, facts, verification)
		if err != nil {
			t.Fatal(err)
		}
		if conformance.Status != "failed" || conformance.RequiredChecksStatus != "failed" {
			t.Fatalf("failed-check run=%#v verification=%#v conformance=%#v", terminal, verification, conformance)
		}
	})

	t.Run("policy drift after lock", func(t *testing.T) {
		repo := setupPlanTestRepo(t)
		initial := `{"schemaVersion":3,"snapshotSchema":"0.3.0","verification":{"checks":{"unit":{"argv":["go","version"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":1000}},"suites":{"change":["unit"]},"profileRules":[{"default":"change"}]}}` + "\n"
		configurePlanPolicy(t, repo, initial)
		request, err := repo.createOrResumePlanRequest(context.Background(), declarePlanInput{
			PlanRequestID: "planreq_policydrift", Goal: "Record policy drift",
			AcceptanceCriteria: "- drift fails conformance",
			AllowedPathGlobs:   []string{"product.txt", ".eve/config.json"}, RequiredSuite: "change",
		})
		if err != nil {
			t.Fatal(err)
		}
		locked, err := repo.approvePlanRequest(context.Background(), request.PlanRequestID, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		changed := strings.Replace(initial, `"outputLimitBytes":1000`, `"outputLimitBytes":2000`, 1)
		if err := os.WriteFile(repo.configPath(), []byte(changed), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo.Root, "product.txt"), []byte("policy drift\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitRun(t, repo.Root, "add", ".eve/config.json", "product.txt")
		gitRun(t, repo.Root, "commit", "-m", "implement with changed policy")
		server := newRuntimeServer(repo, "")
		run, err := server.startVerificationRun(context.Background(), repo, "", "change", "")
		if err != nil {
			t.Fatal(err)
		}
		waitForVerificationRun(t, server, repo, run.RunID)
		facts := completionFactsForTest(t, repo)
		verification, err := snapshotVerification(repo, facts)
		if err != nil {
			t.Fatal(err)
		}
		conformance, err := repo.evaluatePlanConformance(locked, facts, verification)
		if err != nil {
			t.Fatal(err)
		}
		if conformance.Status != "failed" || conformance.PolicyMatched || conformance.CheckDefinitionsMatch {
			t.Fatalf("policy-drift conformance = %#v", conformance)
		}
	})
}

func TestSnapshotCompletionFulfillsLockedPlanAndRecordsConformance(t *testing.T) {
	repo := setupPlanTestRepo(t)
	config := `{"schemaVersion":3,"snapshotSchema":"0.3.0","verification":{"checks":{"unit":{"argv":["go","version"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":1000}},"suites":{"change":["unit"]},"profileRules":[{"default":"change"}]}}` + "\n"
	if err := os.WriteFile(repo.configPath(), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo.Root, "add", ".eve/config.json")
	gitRun(t, repo.Root, "commit", "-m", "configure plan verification")
	branch := gitOutputForTest(t, repo.Root, "branch", "--show-current")
	configHead := gitOutputForTest(t, repo.Root, "rev-parse", "HEAD")
	if err := repo.resolvePendingBranch(branch, configHead); err != nil {
		t.Fatal(err)
	}
	request, err := repo.createOrResumePlanRequest(context.Background(), declarePlanInput{
		PlanRequestID:      "planreq_complete01",
		Goal:               "Change the product",
		AcceptanceCriteria: "- product is changed",
		AllowedPathGlobs:   []string{"product.txt"},
		RequiredSuite:      "change",
	})
	if err != nil {
		t.Fatal(err)
	}
	locked, err := repo.approvePlanRequest(context.Background(), request.PlanRequestID, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	commitProductChangeAt(t, repo.Root, "product.txt", "implemented\n", "implement plan", time.Now())
	server := newRuntimeServer(repo, "")
	run, err := server.startVerificationRun(context.Background(), repo, "", "change", "")
	if err != nil {
		t.Fatal(err)
	}
	run = waitForVerificationRun(t, server, repo, run.RunID)
	if run.Status != "completed" {
		t.Fatalf("verification run = %#v", run)
	}

	if _, err := completeSnapshot(repo, completeSnapshotInput{
		Title: "Wrong revision", Type: "feature", Summary: "Must fail.",
		PlanID: locked.PlanID, PlanRevision: 2,
	}, nil); err == nil {
		t.Fatal("invalid Plan revision was accepted")
	}
	snapshot, err := completeSnapshot(repo, snapshotInputForTest("Fulfill Plan", locked.PlanID, locked.LockedRevision), nil)
	if err != nil {
		t.Fatalf("complete snapshot: %v", err)
	}
	if snapshot.PlanConformance == nil || snapshot.PlanConformance.Status != "matched" || snapshot.PlanConformance.NoPlanOnFile {
		t.Fatalf("conformance = %#v", snapshot.PlanConformance)
	}
	fulfilled, err := repo.loadPlanRequest(request.PlanRequestID)
	if err != nil || fulfilled.State != "fulfilled" || fulfilled.FulfilledSnapshotID != snapshot.ID {
		t.Fatalf("fulfilled request = %#v, %v", fulfilled, err)
	}
	record, err := repo.loadPlanRecord(locked.PlanID)
	if err != nil || record.Status != "fulfilled" || record.FulfilledBy != snapshot.ID || record.LockedRevision != 1 {
		t.Fatalf("plan record = %#v, %v", record, err)
	}
}

func TestSnapshotCompletionWithoutPlanRecordsLoudNoPlanResult(t *testing.T) {
	repo := setupPlanTestRepo(t)
	commitProductChangeAt(t, repo.Root, "product.txt", "unplanned\n", "unplanned implementation", time.Now())
	snapshot, err := completeSnapshot(repo, snapshotInputForTest("Unplanned work", "", 0), nil)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Plan != nil || snapshot.PlanConformance == nil ||
		snapshot.PlanConformance.Status != "no_plan" || !snapshot.PlanConformance.NoPlanOnFile {
		t.Fatalf("no-plan snapshot = %#v", snapshot)
	}
}

func TestCodexStyleMCPPlanSmokeFlow(t *testing.T) {
	repo := setupPlanTestRepo(t)
	config := `{"schemaVersion":3,"snapshotSchema":"0.3.0","verification":{"checks":{"unit":{"argv":["go","version"],"timeoutSeconds":10,"successExitCodes":[0],"outputLimitBytes":1000}},"suites":{"change":["unit"]},"profileRules":[{"default":"change"}]}}` + "\n"
	if err := os.WriteFile(repo.configPath(), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo.Root, "add", ".eve/config.json")
	gitRun(t, repo.Root, "commit", "-m", "configure smoke verification")
	branch := gitOutputForTest(t, repo.Root, "branch", "--show-current")
	head := gitOutputForTest(t, repo.Root, "rev-parse", "HEAD")
	if err := repo.resolvePendingBranch(branch, head); err != nil {
		t.Fatal(err)
	}

	server := newRuntimeServer(repo, "")
	declareResponse := make(chan []byte, 1)
	declareMessage := `{"jsonrpc":"2.0","id":"codex-declare","method":"tools/call","params":{"name":"declare_plan","arguments":{"planRequestId":"planreq_codexsmoke","goal":"Implement the smoke change","acceptanceCriteria":"- Product changes\n- Verification passes","allowedPathGlobs":["product.txt"],"requiredSuite":"change"}}}`
	go func() {
		declareResponse <- server.handleMCPMessage(context.Background(), []byte(declareMessage))
	}()
	var pending *planRequest
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		pending, _ = repo.loadPlanRequest("planreq_codexsmoke")
		if pending != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if pending == nil || pending.State != "pending_approval" {
		t.Fatalf("declare did not persist before waiting: %#v", pending)
	}
	recorder := httptest.NewRecorder()
	server.routes().ServeHTTP(recorder, httptest.NewRequest(
		http.MethodPost,
		"/api/plan-requests/planreq_codexsmoke/approve",
		strings.NewReader(`{"expectedRevision":1}`),
	))
	if recorder.Code != http.StatusOK {
		t.Fatalf("approve = %d: %s", recorder.Code, recorder.Body.String())
	}
	response := <-declareResponse
	if !strings.Contains(string(response), `"state":"locked"`) {
		t.Fatalf("declare response = %s", response)
	}
	locked, _ := repo.loadPlanRequest("planreq_codexsmoke")

	commitProductChangeAt(t, repo.Root, "product.txt", "codex smoke\n", "implement codex smoke", time.Now())
	run, err := server.startVerificationRun(context.Background(), repo, "", "change", "codex")
	if err != nil {
		t.Fatal(err)
	}
	if terminal := waitForVerificationRun(t, server, repo, run.RunID); terminal.Status != "completed" {
		t.Fatalf("suite = %#v", terminal)
	}
	completeMessage := `{"jsonrpc":"2.0","id":"codex-complete","method":"tools/call","params":{"name":"complete_snapshot","arguments":{"title":"Codex smoke","type":"feature","summary":"The MCP Plan flow completed.","planId":"` + locked.PlanID + `","planRevision":1}}}`
	completed := string(server.handleMCPMessage(context.Background(), []byte(completeMessage)))
	if !strings.Contains(completed, `matched`) || !strings.Contains(completed, locked.PlanID) {
		t.Fatalf("complete response = %s", completed)
	}
	fulfilled, _ := repo.loadPlanRequest("planreq_codexsmoke")
	if fulfilled == nil || fulfilled.State != "fulfilled" {
		t.Fatalf("fulfilled = %#v", fulfilled)
	}
}

func snapshotInputForTest(title string, planID string, revision int) completeSnapshotInput {
	return completeSnapshotInput{
		Title:   title,
		Type:    "feature",
		Summary: "The product behavior was implemented and the durable result was recorded.",
		Relationships: eve.Relationships{
			Corrects: []string{}, Supersedes: []string{}, Reverts: []string{}, DependsOn: []string{}, Related: []string{},
		},
		Risks: []eve.Risk{}, Timeline: []eve.TimelineEntry{}, Decisions: []eve.Decision{},
		Validation: []eve.Validation{}, Artifacts: []eve.Artifact{},
		PlanID: planID, PlanRevision: revision,
	}
}

func configurePlanPolicy(t *testing.T, repo repository, config string) {
	t.Helper()
	if err := os.WriteFile(repo.configPath(), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo.Root, "add", ".eve/config.json")
	gitRun(t, repo.Root, "commit", "-m", "configure plan policy")
	branch := gitOutputForTest(t, repo.Root, "branch", "--show-current")
	head := gitOutputForTest(t, repo.Root, "rev-parse", "HEAD")
	if err := repo.resolvePendingBranch(branch, head); err != nil {
		t.Fatal(err)
	}
}

func completionFactsForTest(t *testing.T, repo repository) gitFacts {
	t.Helper()
	evidence, err := verificationNewEvidencePaths(repo)
	if err != nil {
		t.Fatal(err)
	}
	facts, err := deriveGitFactsIgnoring(repo, evidence)
	if err != nil {
		t.Fatal(err)
	}
	return facts
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
