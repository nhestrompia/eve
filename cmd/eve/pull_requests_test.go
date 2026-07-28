package main

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/nhestrompia/eve"
)

func TestGitHubRepositorySlug(t *testing.T) {
	for input, want := range map[string]string{
		"https://github.com/owner/repo":     "owner/repo",
		"https://github.com/owner/repo.git": "owner/repo",
		"git@github.com:owner/repo.git":     "owner/repo",
		"ssh://git@github.com/owner/repo":   "owner/repo",
	} {
		got, ok := githubRepositorySlug(input)
		if !ok || got != want {
			t.Fatalf("githubRepositorySlug(%q) = %q, %v; want %q, true", input, got, ok, want)
		}
	}
	for _, input := range []string{"", "https://gitlab.com/owner/repo", "owner/repo/extra"} {
		if got, ok := githubRepositorySlug(input); ok {
			t.Fatalf("githubRepositorySlug(%q) = %q, true; want false", input, got)
		}
	}
}

func TestSummarizePullRequestRequiresExactVerifiedSnapshot(t *testing.T) {
	snapshot := sampleSnapshot("snap_pr", "Verified product change", "abc123")
	snapshot.Implementation.Branch = "agent/eve-release"
	snapshot.Plan = &eve.PlanReference{ID: "plan_1", Revision: 4}
	snapshot.PlanConformance = &eve.PlanConformance{
		Status:               "matched",
		RequiredChecksStatus: "passed",
		PolicyMatched:        true,
	}
	row := sampleGitHubPullRequest()

	got := summarizePullRequest("owner/repo", row, snapshot, true, true)
	if !got.ReadyToMerge || !got.SnapshotHeadMatch || got.SnapshotID != "snap_pr" {
		t.Fatalf("summarizePullRequest() = %#v, want exact verified Snapshot ready to merge", got)
	}

	got = summarizePullRequest("owner/repo", row, snapshot, false, true)
	if got.ReadyToMerge || got.PlanValid {
		t.Fatalf("summarizePullRequest() = %#v, want invalid Plan revision blocked", got)
	}

	row.HeadRefOID = "different"
	got = summarizePullRequest("owner/repo", row, snapshot, true, false)
	if got.ReadyToMerge || got.SnapshotHeadMatch {
		t.Fatalf("summarizePullRequest() = %#v, want stale review blocked", got)
	}
}

func TestMatchingPullRequestSnapshotRequiresUnambiguousBranchIdentity(t *testing.T) {
	first := sampleSnapshot("snap_first_pr", "First PR", "shared")
	first.Implementation.Branch = "feature/first"
	second := sampleSnapshot("snap_second_pr", "Second PR", "shared")
	second.Implementation.Branch = "feature/second"

	contexts := []pullRequestSnapshotContext{
		{Snapshot: first},
		{Snapshot: second},
	}
	firstRow := sampleGitHubPullRequest()
	firstRow.HeadRefOID = "shared"
	firstRow.HeadRefName = "feature/first"
	if got := matchingPullRequestSnapshot(contexts, firstRow); got == nil || got.Snapshot != first {
		t.Fatalf("first branch match = %#v, want snap_first_pr", got)
	}
	secondRow := sampleGitHubPullRequest()
	secondRow.HeadRefOID = "shared"
	secondRow.HeadRefName = "feature/second"
	if got := matchingPullRequestSnapshot(contexts, secondRow); got == nil || got.Snapshot != second {
		t.Fatalf("second branch match = %#v, want snap_second_pr", got)
	}
	otherRow := sampleGitHubPullRequest()
	otherRow.HeadRefOID = "shared"
	otherRow.HeadRefName = "feature/other"
	if got := matchingPullRequestSnapshot(contexts, otherRow); got != nil {
		t.Fatalf("unrelated branch match = %#v, want nil", got)
	}

	duplicate := sampleSnapshot("snap_duplicate_pr", "Duplicate PR", "shared")
	duplicate.Implementation.Branch = "feature/first"
	ambiguous := append(contexts, pullRequestSnapshotContext{Snapshot: duplicate})
	if got := matchingPullRequestSnapshot(ambiguous, firstRow); got != nil {
		t.Fatalf("ambiguous branch match = %#v, want nil", got)
	}
}

func TestPullRequestAPIListsAndLoadsLinkedSnapshot(t *testing.T) {
	root := initTempGitRepo(t)
	t.Chdir(root)
	mustRun(t, []string{"init"})
	gitRun(t, root, "switch", "-c", "agent/eve-release")
	gitRun(t, root, "add", ".eve/config.json", "AGENTS.md", "CLAUDE.md")
	gitRun(t, root, "commit", "-m", "initialize eve")
	head := gitOutputForTest(t, root, "rev-parse", "HEAD")
	locked := approvePlanForTest(
		t,
		repoFromRoot(root),
		"planreq_pullrequestapi",
		"Review a linked pull request",
		[]string{"product.txt"},
	)
	snapshot := sampleSnapshot("snap_pr_api", "Repository PR review", head)
	snapshot.Implementation.Branch = "agent/eve-release"
	snapshot.Plan = &eve.PlanReference{ID: locked.PlanID, Revision: locked.LockedRevision}
	snapshot.PlanConformance = &eve.PlanConformance{
		Status:               "matched",
		RequiredChecksStatus: "passed",
		PolicyMatched:        true,
	}
	writeSnapshot(t, root, snapshot)
	if _, err := repoFromRoot(root).savePlanRecord(locked, snapshot.ID); err != nil {
		t.Fatalf("save plan record: %v", err)
	}
	if err := repoFromRoot(root).fulfillPlanRequest(context.Background(), locked, snapshot.ID); err != nil {
		t.Fatalf("fulfill plan request: %v", err)
	}

	row := sampleGitHubPullRequest()
	row.HeadRefOID = head
	server := newRuntimeServer(repoFromRoot(root), "localhost:0")
	server.pullRequestLoader = func(context.Context, repository) ([]githubPullRequest, string, int, error) {
		return []githubPullRequest{row}, "owner/repo", 1, nil
	}
	handler := server.routes()

	var collection pullRequestCollection
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+repoFromRoot(root).ID+"/pull-requests", nil, &collection)
	if !collection.Connected || collection.OpenCount != 1 || len(collection.PullRequests) != 1 {
		t.Fatalf("collection = %#v, want one connected pull request", collection)
	}
	if collection.PullRequests[0].SnapshotID != "snap_pr_api" || !collection.PullRequests[0].ReadyToMerge {
		t.Fatalf("pull request = %#v, want linked ready Snapshot", collection.PullRequests[0])
	}

	var detail pullRequestSummary
	requestJSON(t, handler, http.MethodGet, "/api/repos/"+repoFromRoot(root).ID+"/pull-requests/142", nil, &detail)
	if detail.Number != 142 || detail.SnapshotID != "snap_pr_api" {
		t.Fatalf("detail = %#v, want PR #142 linked to snap_pr_api", detail)
	}
	assertRequestStatus(t, handler, http.MethodGet, "/api/repos/"+repoFromRoot(root).ID+"/pull-requests/999", http.StatusNotFound, "not found")
}

func TestPullRequestAPILinksSnapshotFromPRWorktreeAfterEVERecordCommit(t *testing.T) {
	root := initTempGitRepo(t)
	t.Chdir(root)
	mustRun(t, []string{"init"})
	gitRun(t, root, "add", ".eve/config.json", "AGENTS.md", "CLAUDE.md")
	gitRun(t, root, "commit", "-m", "initialize eve")

	featureRoot := filepath.Join(t.TempDir(), "feature-worktree")
	gitRun(t, root, "worktree", "add", "-b", "agent/eve-context", featureRoot)
	featureRepo := repoFromRoot(featureRoot)
	featureBase := gitOutputForTest(t, featureRoot, "rev-parse", "HEAD")
	baseBranch := gitOutputForTest(t, root, "branch", "--show-current")
	locked := approvePlanForTest(
		t,
		featureRepo,
		"planreq_prworktree",
		"Review a Snapshot recorded on a PR branch",
		[]string{"product.txt"},
	)
	if err := os.WriteFile(filepath.Join(featureRoot, "product.txt"), []byte("product\nreviewed change\n"), 0o644); err != nil {
		t.Fatalf("write product change: %v", err)
	}
	gitRun(t, featureRoot, "add", "product.txt")
	gitRun(t, featureRoot, "commit", "-m", "implement reviewed change")
	implementationHead := gitOutputForTest(t, featureRoot, "rev-parse", "HEAD")

	snapshot := sampleSnapshot("snap_pr_worktree", "PR worktree Snapshot", implementationHead)
	snapshot.Implementation.Branch = "agent/eve-context"
	snapshot.Implementation.BaseCommit = featureBase
	snapshot.Plan = &eve.PlanReference{ID: locked.PlanID, Revision: locked.LockedRevision}
	snapshot.PlanConformance = &eve.PlanConformance{
		Status:               "matched",
		RequiredChecksStatus: "passed",
		PolicyMatched:        true,
	}
	writeSnapshot(t, featureRoot, snapshot)
	if _, err := featureRepo.savePlanRecord(locked, snapshot.ID); err != nil {
		t.Fatalf("save plan record: %v", err)
	}
	if err := featureRepo.fulfillPlanRequest(context.Background(), locked, snapshot.ID); err != nil {
		t.Fatalf("fulfill plan request: %v", err)
	}
	gitRun(t, featureRoot, "add", ".eve")
	gitRun(t, featureRoot, "commit", "-m", "record eve product history")

	if err := os.WriteFile(filepath.Join(root, "base-only.txt"), []byte("new base work\n"), 0o644); err != nil {
		t.Fatalf("write base change: %v", err)
	}
	gitRun(t, root, "add", "base-only.txt")
	gitRun(t, root, "commit", "-m", "advance base branch")
	gitRun(t, featureRoot, "merge", baseBranch, "--no-edit")
	pullRequestHead := gitOutputForTest(t, featureRoot, "rev-parse", "HEAD")

	row := sampleGitHubPullRequest()
	row.BaseRefName = baseBranch
	row.HeadRefName = "agent/eve-context"
	row.HeadRefOID = pullRequestHead
	server := newRuntimeServer(repoFromRoot(root), "localhost:0")
	server.pullRequestLoader = func(context.Context, repository) ([]githubPullRequest, string, int, error) {
		return []githubPullRequest{row}, "owner/repo", 1, nil
	}

	var collection pullRequestCollection
	requestJSON(t, server.routes(), http.MethodGet, "/api/repos/"+repoFromRoot(root).ID+"/pull-requests", nil, &collection)
	if len(collection.PullRequests) != 1 {
		t.Fatalf("collection = %#v, want one pull request", collection)
	}
	got := collection.PullRequests[0]
	if got.SnapshotID != snapshot.ID || !got.SnapshotHeadMatch || !got.ReadyToMerge {
		t.Fatalf("pull request = %#v, want current Snapshot from PR worktree", got)
	}

	if err := os.WriteFile(filepath.Join(root, "product.txt"), []byte("product\nbase change\n"), 0o644); err != nil {
		t.Fatalf("write conflicting base change: %v", err)
	}
	gitRun(t, root, "add", "product.txt")
	gitRun(t, root, "commit", "-m", "change product on base branch")
	merge := exec.Command("git", "merge", baseBranch, "--no-edit")
	merge.Dir = featureRoot
	if output, err := merge.CombinedOutput(); err == nil {
		t.Fatalf("conflicting merge unexpectedly succeeded:\n%s", output)
	}
	if err := os.WriteFile(
		filepath.Join(featureRoot, "product.txt"),
		[]byte("product\nreviewed change\nunrecorded merge resolution\n"),
		0o644,
	); err != nil {
		t.Fatalf("write unrecorded merge resolution: %v", err)
	}
	gitRun(t, featureRoot, "add", "product.txt")
	gitRun(t, featureRoot, "commit", "-m", "resolve base merge with product change")
	row.HeadRefOID = gitOutputForTest(t, featureRoot, "rev-parse", "HEAD")

	var stale pullRequestCollection
	requestJSON(t, server.routes(), http.MethodGet, "/api/repos/"+repoFromRoot(root).ID+"/pull-requests", nil, &stale)
	if stale.PullRequests[0].SnapshotID != snapshot.ID || stale.PullRequests[0].SnapshotHeadMatch {
		t.Fatalf("stale pull request = %#v, want linked Snapshot marked stale", stale.PullRequests[0])
	}
}

func TestSnapshotHistoryCommitRemainsCurrentAcrossEarlierFeatureWork(t *testing.T) {
	root := initTempGitRepo(t)
	t.Chdir(root)
	mustRun(t, []string{"init"})
	gitRun(t, root, "add", ".eve/config.json", "AGENTS.md", "CLAUDE.md")
	gitRun(t, root, "commit", "-m", "initialize eve")
	baseBranch := gitOutputForTest(t, root, "branch", "--show-current")

	featureRoot := filepath.Join(t.TempDir(), "feature-worktree")
	gitRun(t, root, "worktree", "add", "-b", "agent/multi-snapshot-pr", featureRoot)
	featureRepo := repoFromRoot(featureRoot)
	if err := os.WriteFile(filepath.Join(featureRoot, "earlier.txt"), []byte("earlier feature work\n"), 0o644); err != nil {
		t.Fatalf("write earlier feature work: %v", err)
	}
	gitRun(t, featureRoot, "add", "earlier.txt")
	gitRun(t, featureRoot, "commit", "-m", "implement earlier Snapshot")
	snapshotBase := gitOutputForTest(t, featureRoot, "rev-parse", "HEAD")

	if err := os.WriteFile(filepath.Join(featureRoot, "latest.txt"), []byte("latest feature work\n"), 0o644); err != nil {
		t.Fatalf("write latest feature work: %v", err)
	}
	gitRun(t, featureRoot, "add", "latest.txt")
	gitRun(t, featureRoot, "commit", "-m", "implement latest Snapshot")
	implementationHead := gitOutputForTest(t, featureRoot, "rev-parse", "HEAD")

	snapshot := sampleSnapshot("snap_multi_snapshot_pr", "Latest PR Snapshot", implementationHead)
	snapshot.Implementation.Branch = "agent/multi-snapshot-pr"
	snapshot.Implementation.BaseCommit = snapshotBase
	writeSnapshot(t, featureRoot, snapshot)
	gitRun(t, featureRoot, "add", ".eve")
	gitRun(t, featureRoot, "commit", "-m", "record latest eve product history")
	pullRequestHead := gitOutputForTest(t, featureRoot, "rev-parse", "HEAD")

	if !snapshotMatchesPullRequestHead(featureRepo, snapshot, pullRequestHead, baseBranch) {
		t.Fatal("Snapshot followed only by its eve history commit was marked stale")
	}
}

func sampleGitHubPullRequest() githubPullRequest {
	row := githubPullRequest{
		Number:           142,
		Title:            "Prepare Eve release",
		URL:              "https://github.com/owner/repo/pull/142",
		BaseRefName:      "main",
		HeadRefName:      "agent/eve-release",
		HeadRefOID:       "abc123",
		State:            "OPEN",
		MergeStateStatus: "CLEAN",
		Mergeable:        "MERGEABLE",
		ReviewDecision:   "APPROVED",
		UpdatedAt:        "2026-07-28T10:00:00Z",
		CreatedAt:        "2026-07-28T09:00:00Z",
		ChangedFiles:     4,
		Additions:        120,
		Deletions:        20,
		Commits:          nil,
		StatusChecks: []githubStatusCheck{
			{Conclusion: "SUCCESS"},
			{State: "SUCCESS"},
		},
	}
	row.Author.Login = "octocat"
	return row
}
