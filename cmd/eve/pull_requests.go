package main

import (
	"context"
	"crypto/sha256"
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

type pullRequestSnapshotContext struct {
	Snapshot   *eve.Snapshot
	Repository repository
}

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

	snapshotContexts, snapshotErr := loadPullRequestSnapshotContexts(repo)
	if snapshotErr != nil {
		return pullRequestCollection{
			Connected:    false,
			Repository:   repo.ID,
			Reason:       fmt.Sprintf("read Snapshots from Git worktrees: %v", snapshotErr),
			PullRequests: []pullRequestSummary{},
		}
	}
	pullRequestsByHead := make(map[string]int, len(rows))
	for _, row := range rows {
		pullRequestsByHead[pullRequestHeadKey(row.HeadRefOID, row.HeadRefName)]++
	}
	pullRequests := make([]pullRequestSummary, 0, len(rows))
	for _, row := range rows {
		key := pullRequestHeadKey(row.HeadRefOID, row.HeadRefName)
		pullRequests = append(
			pullRequests,
			summarizePullRequestWithSnapshotContext(
				githubRepository,
				row,
				snapshotContexts,
				pullRequestsByHead[key] == 1,
			),
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
		return nil, "", 0, errors.New("Add a GitHub remote to review pull requests in eve.")
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

func loadPullRequestSnapshotContexts(repo repository) ([]pullRequestSnapshotContext, error) {
	worktrees := repo.gitWorktreeRepositories()
	currentIncluded := false
	for _, worktree := range worktrees {
		if worktree.Root == repo.Root {
			currentIncluded = true
			break
		}
	}
	if !currentIncluded {
		worktrees = append(worktrees, repo)
	}

	bySnapshotID := make(map[string]pullRequestSnapshotContext)
	for _, worktree := range worktrees {
		snapshots, err := worktree.listSnapshots("")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", worktree.Root, err)
		}
		for _, snapshot := range snapshots {
			if snapshot == nil ||
				strings.TrimSpace(snapshot.ID) == "" ||
				strings.TrimSpace(snapshot.Implementation.GitState) == "" ||
				strings.TrimSpace(snapshot.Implementation.Branch) == "" {
				continue
			}
			if _, exists := bySnapshotID[snapshot.ID]; !exists {
				bySnapshotID[snapshot.ID] = pullRequestSnapshotContext{
					Snapshot:   snapshot,
					Repository: worktree,
				}
			}
		}
	}

	contexts := make([]pullRequestSnapshotContext, 0, len(bySnapshotID))
	for _, context := range bySnapshotID {
		contexts = append(contexts, context)
	}
	sort.Slice(contexts, func(i, j int) bool {
		if contexts[i].Snapshot.CreatedAt == contexts[j].Snapshot.CreatedAt {
			return contexts[i].Snapshot.ID < contexts[j].Snapshot.ID
		}
		return contexts[i].Snapshot.CreatedAt > contexts[j].Snapshot.CreatedAt
	})
	return contexts, nil
}

func matchingPullRequestSnapshot(
	contexts []pullRequestSnapshotContext,
	row githubPullRequest,
) *pullRequestSnapshotContext {
	context, current := pullRequestSnapshotForRow(contexts, row)
	if !current {
		return nil
	}
	return context
}

func pullRequestSnapshotForRow(
	contexts []pullRequestSnapshotContext,
	row githubPullRequest,
) (*pullRequestSnapshotContext, bool) {
	var fallback *pullRequestSnapshotContext
	var matched *pullRequestSnapshotContext
	for index := range contexts {
		context := &contexts[index]
		snapshot := context.Snapshot
		if strings.TrimSpace(snapshot.Implementation.Branch) != strings.TrimSpace(row.HeadRefName) {
			continue
		}
		if fallback == nil &&
			snapshotRecordedOnPullRequestHead(context.Repository, snapshot, row.HeadRefOID) {
			fallback = context
		}
		if !snapshotMatchesPullRequestHead(
			context.Repository,
			snapshot,
			row.HeadRefOID,
			row.BaseRefName,
		) {
			continue
		}
		if matched != nil && matched.Snapshot.ID != snapshot.ID {
			return fallback, false
		}
		matched = context
	}
	if matched != nil {
		return matched, true
	}
	return fallback, false
}

func snapshotRecordedOnPullRequestHead(
	repo repository,
	snapshot *eve.Snapshot,
	pullRequestHead string,
) bool {
	implementationHead := strings.TrimSpace(snapshot.Implementation.GitState)
	pullRequestHead = strings.TrimSpace(pullRequestHead)
	if implementationHead == "" || pullRequestHead == "" ||
		!repo.isAncestor(implementationHead, pullRequestHead) {
		return false
	}
	snapshotPath := ".eve/snapshots/" + snapshot.ID + ".json"
	_, err := gitOutput(repo.Root, "cat-file", "-e", pullRequestHead+":"+snapshotPath)
	return err == nil
}

func snapshotMatchesPullRequestHead(
	repo repository,
	snapshot *eve.Snapshot,
	pullRequestHead string,
	baseBranch string,
) bool {
	implementationHead := strings.TrimSpace(snapshot.Implementation.GitState)
	pullRequestHead = strings.TrimSpace(pullRequestHead)
	if implementationHead == "" || pullRequestHead == "" {
		return false
	}
	if implementationHead == pullRequestHead {
		return true
	}
	if !repo.isAncestor(implementationHead, pullRequestHead) {
		return false
	}
	if !snapshotRecordedOnPullRequestHead(repo, snapshot, pullRequestHead) {
		return false
	}
	return snapshotFollowedOnlyByHistoryAndBaseMerges(
		repo,
		snapshot,
		pullRequestHead,
		baseBranch,
	)
}

func snapshotFollowedOnlyByHistoryAndBaseMerges(
	repo repository,
	snapshot *eve.Snapshot,
	pullRequestHead string,
	baseBranch string,
) bool {
	implementationHead := strings.TrimSpace(snapshot.Implementation.GitState)
	output, err := gitOutput(
		repo.Root,
		"rev-list",
		"--first-parent",
		"--reverse",
		implementationHead+".."+pullRequestHead,
	)
	if err != nil {
		return false
	}

	previous := implementationHead
	baseHead := ""
	for _, commit := range strings.Fields(output) {
		parentOutput, err := gitOutput(repo.Root, "show", "-s", "--format=%P", commit)
		if err != nil {
			return false
		}
		parents := strings.Fields(parentOutput)
		if len(parents) == 0 || parents[0] != previous {
			return false
		}
		if len(parents) == 1 {
			if !commitChangesOnlyEVEHistory(repo, previous, commit) {
				return false
			}
		} else {
			if baseHead == "" {
				baseHead = resolvePullRequestBaseHead(repo, baseBranch)
			}
			if baseHead == "" {
				return false
			}
			for _, mergedParent := range parents[1:] {
				if !repo.isAncestor(mergedParent, baseHead) {
					return false
				}
			}
		}
		previous = commit
	}
	return previous == pullRequestHead &&
		snapshotProductPatchMatchesPullRequest(repo, snapshot, pullRequestHead, baseBranch)
}

func commitChangesOnlyEVEHistory(repo repository, from string, to string) bool {
	changedPaths, err := gitOutput(
		repo.Root,
		"diff",
		"--name-only",
		"--no-renames",
		from+".."+to,
		"--",
	)
	if err != nil {
		return false
	}
	for _, changedPath := range strings.Split(changedPaths, "\n") {
		changedPath = strings.TrimSpace(changedPath)
		if changedPath == "" {
			continue
		}
		if changedPath != ".eve" && !strings.HasPrefix(changedPath, ".eve/") {
			return false
		}
	}
	return true
}

func resolvePullRequestBaseHead(repo repository, branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return ""
	}
	for _, ref := range []string{
		"refs/remotes/origin/" + branch,
		"refs/heads/" + branch,
	} {
		value, err := gitOutput(repo.Root, "rev-parse", "--verify", ref+"^{commit}")
		if err == nil && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func snapshotProductPatchMatchesPullRequest(
	repo repository,
	snapshot *eve.Snapshot,
	pullRequestHead string,
	baseBranch string,
) bool {
	snapshotBase := strings.TrimSpace(snapshot.Implementation.BaseCommit)
	implementationHead := strings.TrimSpace(snapshot.Implementation.GitState)
	baseHead := resolvePullRequestBaseHead(repo, baseBranch)
	if snapshotBase == "" || implementationHead == "" || baseHead == "" {
		return false
	}
	pullRequestBase, err := gitOutput(repo.Root, "merge-base", baseHead, pullRequestHead)
	if err != nil || strings.TrimSpace(pullRequestBase) == "" {
		return false
	}
	snapshotPatchID, err := normalizedProductDiffID(repo, snapshotBase, implementationHead)
	if err != nil {
		return false
	}
	pullRequestPatchID, err := normalizedProductDiffID(
		repo,
		strings.TrimSpace(pullRequestBase),
		pullRequestHead,
	)
	return err == nil && snapshotPatchID == pullRequestPatchID
}

func normalizedProductDiffID(repo repository, from string, to string) (string, error) {
	diff := exec.Command(
		"git",
		"diff",
		"--unified=0",
		"--no-ext-diff",
		"--no-textconv",
		"--no-renames",
		strings.TrimSpace(from)+".."+strings.TrimSpace(to),
		"--",
		".",
		":(exclude).eve",
		":(exclude).eve/**",
		":(exclude)cmd/eve/ui_dist",
		":(exclude)cmd/eve/ui_dist/**",
	)
	diff.Dir = repo.Root
	patch, err := diff.Output()
	if err != nil {
		return "", err
	}
	normalized := normalizeProductDiff(string(patch))
	if strings.TrimSpace(normalized) == "" {
		return "", nil
	}
	sum := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", sum), nil
}

func normalizeProductDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	normalized := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "index ") {
			continue
		}
		if strings.HasPrefix(line, "@@ ") {
			if marker := strings.Index(line, " @@"); marker >= 0 {
				line = "@@" + line[marker+3:]
			}
		}
		normalized = append(normalized, line)
	}
	return strings.Join(normalized, "\n")
}

func summarizePullRequestWithSnapshotContext(
	githubRepository string,
	row githubPullRequest,
	contexts []pullRequestSnapshotContext,
	allowMatch bool,
) pullRequestSummary {
	var snapshot *eve.Snapshot
	var snapshotRepo repository
	snapshotHeadMatches := false
	if allowMatch {
		if context, current := pullRequestSnapshotForRow(contexts, row); context != nil {
			snapshot = context.Snapshot
			snapshotRepo = context.Repository
			snapshotHeadMatches = current
		}
	}
	return summarizePullRequest(
		githubRepository,
		row,
		snapshot,
		snapshotPlanIsValid(snapshotRepo, snapshot),
		snapshotHeadMatches,
	)
}

func pullRequestHeadKey(sha string, branch string) string {
	return strings.TrimSpace(sha) + "\x00" + strings.TrimSpace(branch)
}

func summarizePullRequest(
	repository string,
	row githubPullRequest,
	snapshot *eve.Snapshot,
	planValid bool,
	snapshotHeadMatches bool,
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
	result.SnapshotHeadMatch = snapshotHeadMatches
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
	snapshotContexts, err := loadPullRequestSnapshotContexts(repo)
	if err != nil {
		return pullRequestSummary{}, fmt.Errorf("read Snapshots from Git worktrees: %w", err)
	}
	return summarizePullRequestWithSnapshotContext(
		githubRepository,
		row,
		snapshotContexts,
		true,
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
