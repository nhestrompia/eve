package main

import (
	"context"
	"net/http"
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

	got := summarizePullRequest("owner/repo", row, snapshot, true)
	if !got.ReadyToMerge || !got.SnapshotHeadMatch || got.SnapshotID != "snap_pr" {
		t.Fatalf("summarizePullRequest() = %#v, want exact verified Snapshot ready to merge", got)
	}

	got = summarizePullRequest("owner/repo", row, snapshot, false)
	if got.ReadyToMerge || got.PlanValid {
		t.Fatalf("summarizePullRequest() = %#v, want invalid Plan revision blocked", got)
	}

	row.HeadRefOID = "different"
	got = summarizePullRequest("owner/repo", row, snapshot, true)
	if got.ReadyToMerge || got.SnapshotHeadMatch {
		t.Fatalf("summarizePullRequest() = %#v, want stale review blocked", got)
	}
}

func TestSnapshotsByPullRequestHeadIncludesBranchIdentity(t *testing.T) {
	first := sampleSnapshot("snap_first_pr", "First PR", "shared")
	first.Implementation.Branch = "feature/first"
	second := sampleSnapshot("snap_second_pr", "Second PR", "shared")
	second.Implementation.Branch = "feature/second"

	byHead := snapshotsByPullRequestHead([]*eve.Snapshot{first, second})
	if got := byHead[pullRequestHeadKey("shared", "feature/first")]; got != first {
		t.Fatalf("first branch match = %#v, want snap_first_pr", got)
	}
	if got := byHead[pullRequestHeadKey("shared", "feature/second")]; got != second {
		t.Fatalf("second branch match = %#v, want snap_second_pr", got)
	}
	if got := byHead[pullRequestHeadKey("shared", "feature/other")]; got != nil {
		t.Fatalf("unrelated branch match = %#v, want nil", got)
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
