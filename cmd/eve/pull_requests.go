package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nhestrompia/eve"
)

const githubPullRequestFields = "number,title,url,baseRefName,headRefName,headRefOid,state,isDraft,mergeStateStatus,mergeable,reviewDecision,statusCheckRollup,createdAt,updatedAt,mergedAt,author,additions,deletions,changedFiles"
const githubPullRequestDetailFields = githubPullRequestFields + ",body,commits"

type pullRequestCollection struct {
	Connected    bool                 `json:"connected"`
	Repository   string               `json:"repository"`
	OpenCount    int                  `json:"openCount"`
	Reason       string               `json:"reason,omitempty"`
	PullRequests []pullRequestSummary `json:"pullRequests"`
}

type pullRequestSummary struct {
	Provider          string `json:"provider"`
	Repository        string `json:"repository"`
	Number            int    `json:"number"`
	Title             string `json:"title"`
	Body              string `json:"body,omitempty"`
	URL               string `json:"url"`
	BaseBranch        string `json:"baseBranch"`
	HeadBranch        string `json:"headBranch"`
	HeadSHA           string `json:"headSha"`
	State             string `json:"state"`
	Draft             bool   `json:"draft"`
	Mergeability      string `json:"mergeability"`
	MergeState        string `json:"mergeState,omitempty"`
	ReviewDecision    string `json:"reviewDecision,omitempty"`
	Author            string `json:"author,omitempty"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
	MergedAt          string `json:"mergedAt,omitempty"`
	Additions         int    `json:"additions"`
	Deletions         int    `json:"deletions"`
	ChangedFiles      int    `json:"changedFiles"`
	CommitCount       int    `json:"commitCount"`
	ChecksTotal       int    `json:"checksTotal"`
	ChecksPassed      int    `json:"checksPassed"`
	ChecksFailed      int    `json:"checksFailed"`
	ChecksPending     int    `json:"checksPending"`
	SnapshotID        string `json:"snapshotId,omitempty"`
	SnapshotTitle     string `json:"snapshotTitle,omitempty"`
	PlanRevision      int    `json:"planRevision,omitempty"`
	PlanValid         bool   `json:"planValid"`
	PlanAligned       bool   `json:"planAligned"`
	EveChecksPassed   bool   `json:"eveChecksPassed"`
	ScopeDrift        bool   `json:"scopeDrift"`
	GitHubReady       bool   `json:"githubReady"`
	ReadyToMerge      bool   `json:"readyToMerge"`
	SnapshotHeadMatch bool   `json:"snapshotHeadMatch"`
}

type githubPullRequest struct {
	Number           int                 `json:"number"`
	Title            string              `json:"title"`
	Body             string              `json:"body"`
	URL              string              `json:"url"`
	BaseRefName      string              `json:"baseRefName"`
	HeadRefName      string              `json:"headRefName"`
	HeadRefOID       string              `json:"headRefOid"`
	State            string              `json:"state"`
	IsDraft          bool                `json:"isDraft"`
	MergeStateStatus string              `json:"mergeStateStatus"`
	Mergeable        string              `json:"mergeable"`
	ReviewDecision   string              `json:"reviewDecision"`
	StatusChecks     []githubStatusCheck `json:"statusCheckRollup"`
	CreatedAt        string              `json:"createdAt"`
	UpdatedAt        string              `json:"updatedAt"`
	MergedAt         string              `json:"mergedAt"`
	Author           struct {
		Login string `json:"login"`
	} `json:"author"`
	Additions    int               `json:"additions"`
	Deletions    int               `json:"deletions"`
	ChangedFiles int               `json:"changedFiles"`
	Commits      []json.RawMessage `json:"commits"`
}

type githubStatusCheck struct {
	TypeName   string `json:"__typename"`
	Name       string `json:"name"`
	Context    string `json:"context"`
	State      string `json:"state"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type pullRequestLoader func(context.Context, repository) ([]githubPullRequest, string, int, error)

func (server runtimeServer) handlePullRequests(w http.ResponseWriter, r *http.Request, repo repository) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	collection := server.pullRequests(r.Context(), repo)
	writeJSON(w, http.StatusOK, collection)
}

func (server runtimeServer) handlePullRequest(w http.ResponseWriter, r *http.Request, repo repository, rawNumber string) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	number, err := strconv.Atoi(rawNumber)
	if err != nil || number <= 0 {
		writeAPIError(w, http.StatusBadRequest, fmt.Errorf("invalid pull request number"))
		return
	}
	collection := server.pullRequests(r.Context(), repo)
	if !collection.Connected {
		writeAPIError(w, http.StatusServiceUnavailable, errors.New(collection.Reason))
		return
	}
	for _, pullRequest := range collection.PullRequests {
		if pullRequest.Number == number {
			writeJSON(w, http.StatusOK, pullRequest)
			return
		}
	}
	if server.pullRequestLoader == nil {
		pullRequest, err := server.pullRequestByNumber(r.Context(), repo, number)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, pullRequest)
		return
	}
	writeAPIError(w, http.StatusNotFound, fmt.Errorf("pull request #%d not found", number))
}

func (server runtimeServer) pullRequests(ctx context.Context, repo repository) pullRequestCollection {
	loader := server.pullRequestLoader
	if loader == nil {
		loader = loadGitHubPullRequests
	}
	timeoutContext, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	rows, githubRepository, openCount, err := loader(timeoutContext, repo)
	if err != nil {
		return pullRequestCollection{
			Connected:    false,
			Repository:   repo.ID,
			Reason:       err.Error(),
			PullRequests: []pullRequestSummary{},
		}
	}

	snapshots, snapshotErr := repo.listSnapshots("")
	if snapshotErr != nil {
		return pullRequestCollection{
			Connected:    false,
			Repository:   repo.ID,
			Reason:       fmt.Sprintf("read Snapshots: %v", snapshotErr),
			PullRequests: []pullRequestSummary{},
		}
	}
	byHead := snapshotsByPullRequestHead(snapshots)
	pullRequestsByHead := make(map[string]int, len(rows))
	for _, row := range rows {
		pullRequestsByHead[pullRequestHeadKey(row.HeadRefOID, row.HeadRefName)]++
	}
	pullRequests := make([]pullRequestSummary, 0, len(rows))
	for _, row := range rows {
		key := pullRequestHeadKey(row.HeadRefOID, row.HeadRefName)
		var snapshot *eve.Snapshot
		if pullRequestsByHead[key] == 1 {
			snapshot = byHead[key]
		}
		planValid := snapshotPlanIsValid(repo, snapshot)
		pullRequests = append(
			pullRequests,
			summarizePullRequest(githubRepository, row, snapshot, planValid),
		)
	}
	sort.SliceStable(pullRequests, func(i, j int) bool {
		return pullRequests[i].UpdatedAt > pullRequests[j].UpdatedAt
	})
	return pullRequestCollection{
		Connected:    true,
		Repository:   repo.ID,
		OpenCount:    openCount,
		PullRequests: pullRequests,
	}
}

func loadGitHubPullRequests(ctx context.Context, repo repository) ([]githubPullRequest, string, int, error) {
	summary, err := repo.summary()
	if err != nil {
		return nil, "", 0, fmt.Errorf("read repository: %w", err)
	}
	githubRepository, ok := githubRepositorySlug(summary.RemoteURL)
	if !ok {
		return nil, "", 0, errors.New("Add a GitHub remote to review pull requests in EVE.")
	}
	command := exec.CommandContext(
		ctx,
		"gh",
		"pr",
		"list",
		"--repo",
		githubRepository,
		"--state",
		"open",
		"--limit",
		"50",
		"--json",
		githubPullRequestFields,
	)
	command.Dir = repo.Root
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			if errors.Is(err, exec.ErrNotFound) {
				message = "GitHub CLI is not installed."
			} else {
				message = err.Error()
			}
		}
		return nil, githubRepository, 0, fmt.Errorf("GitHub pull requests are unavailable: %s", message)
	}
	var rows []githubPullRequest
	if err := json.Unmarshal(output, &rows); err != nil {
		return nil, githubRepository, 0, fmt.Errorf("decode GitHub pull requests: %w", err)
	}
	openCount := len(rows)
	if len(rows) == 50 {
		count, err := loadGitHubOpenPullRequestCount(ctx, repo, githubRepository)
		if err != nil {
			return nil, githubRepository, 0, err
		}
		openCount = count
	}
	return rows, githubRepository, openCount, nil
}

func githubRepositorySlug(remoteURL string) (string, bool) {
	value := strings.TrimSpace(strings.TrimSuffix(remoteURL, ".git"))
	value = strings.TrimPrefix(value, "https://github.com/")
	value = strings.TrimPrefix(value, "http://github.com/")
	value = strings.TrimPrefix(value, "ssh://git@github.com/")
	value = strings.TrimPrefix(value, "git@github.com:")
	value = strings.Trim(value, "/")
	if strings.Count(value, "/") != 1 || strings.ContainsAny(value, "?#") {
		return "", false
	}
	return value, true
}

func snapshotsByPullRequestHead(snapshots []*eve.Snapshot) map[string]*eve.Snapshot {
	result := make(map[string]*eve.Snapshot, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot == nil ||
			strings.TrimSpace(snapshot.Implementation.GitState) == "" ||
			strings.TrimSpace(snapshot.Implementation.Branch) == "" {
			continue
		}
		key := pullRequestHeadKey(
			snapshot.Implementation.GitState,
			snapshot.Implementation.Branch,
		)
		current := result[key]
		if current == nil || snapshot.CreatedAt > current.CreatedAt {
			result[key] = snapshot
		}
	}
	return result
}

func pullRequestHeadKey(sha string, branch string) string {
	return strings.TrimSpace(sha) + "\x00" + strings.TrimSpace(branch)
}

func summarizePullRequest(
	repository string,
	row githubPullRequest,
	snapshot *eve.Snapshot,
	planValid bool,
) pullRequestSummary {
	passed, failed, pending := summarizeGitHubChecks(row.StatusChecks)
	state := strings.ToLower(row.State)
	mergeability := strings.ToLower(row.Mergeable)
	mergeState := strings.ToLower(row.MergeStateStatus)
	reviewDecision := strings.ToLower(row.ReviewDecision)
	githubReady := state == "open" &&
		!row.IsDraft &&
		mergeability == "mergeable" &&
		mergeState == "clean" &&
		failed == 0 &&
		pending == 0 &&
		reviewDecision != "changes_requested"

	result := pullRequestSummary{
		Provider:       "github",
		Repository:     repository,
		Number:         row.Number,
		Title:          row.Title,
		Body:           row.Body,
		URL:            row.URL,
		BaseBranch:     row.BaseRefName,
		HeadBranch:     row.HeadRefName,
		HeadSHA:        row.HeadRefOID,
		State:          state,
		Draft:          row.IsDraft,
		Mergeability:   mergeability,
		MergeState:     mergeState,
		ReviewDecision: reviewDecision,
		Author:         row.Author.Login,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		MergedAt:       row.MergedAt,
		Additions:      row.Additions,
		Deletions:      row.Deletions,
		ChangedFiles:   row.ChangedFiles,
		CommitCount:    len(row.Commits),
		ChecksTotal:    len(row.StatusChecks),
		ChecksPassed:   passed,
		ChecksFailed:   failed,
		ChecksPending:  pending,
		GitHubReady:    githubReady,
	}
	if snapshot == nil {
		return result
	}

	result.CommitCount = len(snapshot.Implementation.Commits)
	result.SnapshotID = snapshot.ID
	result.SnapshotTitle = snapshot.Title
	result.SnapshotHeadMatch = snapshot.Implementation.GitState == row.HeadRefOID
	result.PlanValid = planValid
	if snapshot.Plan != nil {
		result.PlanRevision = snapshot.Plan.Revision
	}
	if snapshot.PlanConformance != nil {
		result.PlanAligned = snapshot.PlanConformance.Status == "matched"
		result.EveChecksPassed = snapshot.PlanConformance.RequiredChecksStatus == "passed" ||
			snapshot.PlanConformance.RequiredChecksStatus == "not_configured"
		result.ScopeDrift = snapshot.PlanConformance.ScopeDrift
	}
	eveReady := result.SnapshotHeadMatch &&
		snapshot.Plan != nil &&
		result.PlanValid &&
		result.PlanAligned &&
		result.EveChecksPassed &&
		!result.ScopeDrift
	result.ReadyToMerge = githubReady && eveReady
	return result
}

func snapshotPlanIsValid(repo repository, snapshot *eve.Snapshot) bool {
	if snapshot == nil || snapshot.Plan == nil {
		return false
	}
	record, err := repo.loadPlanRecord(snapshot.Plan.ID)
	if err != nil ||
		record.Status != "fulfilled" ||
		record.LockedRevision != snapshot.Plan.Revision ||
		record.FulfilledBy != snapshot.ID {
		return false
	}
	branchMatches := false
	for _, revision := range record.Revisions {
		if revision.Revision == snapshot.Plan.Revision {
			branchMatches = revision.Branch == snapshot.Implementation.Branch
			break
		}
	}
	if !branchMatches {
		return false
	}
	request, err := repo.loadPlanRequest(record.PlanRequestID)
	return err == nil &&
		request.State == "fulfilled" &&
		request.PlanID == snapshot.Plan.ID &&
		request.LockedRevision == snapshot.Plan.Revision &&
		request.FulfilledSnapshotID == snapshot.ID
}

func loadGitHubOpenPullRequestCount(
	ctx context.Context,
	repo repository,
	githubRepository string,
) (int, error) {
	parts := strings.Split(githubRepository, "/")
	command := exec.CommandContext(
		ctx,
		"gh",
		"api",
		"graphql",
		"-f",
		"query=query($owner:String!,$name:String!){repository(owner:$owner,name:$name){pullRequests(states:OPEN){totalCount}}}",
		"-F",
		"owner="+parts[0],
		"-F",
		"name="+parts[1],
	)
	command.Dir = repo.Root
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return 0, fmt.Errorf("GitHub pull request count is unavailable: %s", message)
	}
	var response struct {
		Data struct {
			Repository struct {
				PullRequests struct {
					TotalCount int `json:"totalCount"`
				} `json:"pullRequests"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return 0, fmt.Errorf("decode GitHub pull request count: %w", err)
	}
	return response.Data.Repository.PullRequests.TotalCount, nil
}

func (server runtimeServer) pullRequestByNumber(
	ctx context.Context,
	repo repository,
	number int,
) (pullRequestSummary, error) {
	summary, err := repo.summary()
	if err != nil {
		return pullRequestSummary{}, fmt.Errorf("read repository: %w", err)
	}
	githubRepository, ok := githubRepositorySlug(summary.RemoteURL)
	if !ok {
		return pullRequestSummary{}, errors.New("GitHub remote is not configured")
	}
	timeoutContext, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	command := exec.CommandContext(
		timeoutContext,
		"gh",
		"pr",
		"view",
		strconv.Itoa(number),
		"--repo",
		githubRepository,
		"--json",
		githubPullRequestDetailFields,
	)
	command.Dir = repo.Root
	output, err := command.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return pullRequestSummary{}, fmt.Errorf("pull request #%d is unavailable: %s", number, message)
	}
	var row githubPullRequest
	if err := json.Unmarshal(output, &row); err != nil {
		return pullRequestSummary{}, fmt.Errorf("decode pull request #%d: %w", number, err)
	}
	snapshots, err := repo.listSnapshots("")
	if err != nil {
		return pullRequestSummary{}, fmt.Errorf("read Snapshots: %w", err)
	}
	byHead := snapshotsByPullRequestHead(snapshots)
	snapshot := byHead[pullRequestHeadKey(row.HeadRefOID, row.HeadRefName)]
	return summarizePullRequest(
		githubRepository,
		row,
		snapshot,
		snapshotPlanIsValid(repo, snapshot),
	), nil
}

func summarizeGitHubChecks(checks []githubStatusCheck) (passed int, failed int, pending int) {
	for _, check := range checks {
		value := strings.ToUpper(strings.TrimSpace(check.Conclusion))
		if value == "" {
			value = strings.ToUpper(strings.TrimSpace(check.State))
		}
		if value == "" {
			value = strings.ToUpper(strings.TrimSpace(check.Status))
		}
		switch value {
		case "SUCCESS", "NEUTRAL", "SKIPPED":
			passed++
		case "FAILURE", "ERROR", "CANCELLED", "TIMED_OUT", "ACTION_REQUIRED", "STARTUP_FAILURE":
			failed++
		default:
			pending++
		}
	}
	return passed, failed, pending
}
