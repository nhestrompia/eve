package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/nhestrompia/eve"
)

const planRequestSchemaVersion = 1

var planRequestIDPattern = regexp.MustCompile(`^planreq_[A-Za-z0-9_-]{8,120}$`)
var planWindowsAbsolutePattern = regexp.MustCompile(`^[A-Za-z]:/`)

type planRequest struct {
	SchemaVersion       int                `json:"schemaVersion"`
	PlanRequestID       string             `json:"planRequestId"`
	PlanID              string             `json:"planId,omitempty"`
	Repository          string             `json:"repository"`
	RepositoryRoot      string             `json:"repositoryRoot"`
	Branch              string             `json:"branch"`
	State               string             `json:"state"`
	CurrentRevision     int                `json:"currentRevision"`
	LockedRevision      int                `json:"lockedRevision,omitempty"`
	Revisions           []eve.PlanRevision `json:"revisions"`
	AvailableSuites     []string           `json:"availableSuites"`
	RejectionFeedback   string             `json:"rejectionFeedback,omitempty"`
	StaleReasons        []string           `json:"staleReasons,omitempty"`
	SupersededBy        string             `json:"supersededBy,omitempty"`
	Supersedes          []string           `json:"supersedes,omitempty"`
	FulfilledSnapshotID string             `json:"fulfilledSnapshotId,omitempty"`
	CreatedAt           string             `json:"createdAt"`
	UpdatedAt           string             `json:"updatedAt"`
	LockedAt            string             `json:"lockedAt,omitempty"`
}

type planProposal struct {
	Goal               string              `json:"goal"`
	AcceptanceCriteria string              `json:"acceptanceCriteria"`
	AllowedPathGlobs   []string            `json:"allowedPathGlobs"`
	Milestones         []eve.PlanMilestone `json:"milestones,omitempty"`
	RequiredSuite      string              `json:"requiredSuite,omitempty"`
}

type declarePlanInput struct {
	CWD                string              `json:"cwd"`
	RepoID             string              `json:"repoId"`
	PlanRequestID      string              `json:"planRequestId"`
	Goal               string              `json:"goal"`
	AcceptanceCriteria string              `json:"acceptanceCriteria"`
	AllowedPathGlobs   []string            `json:"allowedPathGlobs"`
	Milestones         []eve.PlanMilestone `json:"milestones"`
	RequiredSuite      string              `json:"requiredSuite"`
}

func (input declarePlanInput) proposal() planProposal {
	return planProposal{
		Goal:               strings.TrimSpace(input.Goal),
		AcceptanceCriteria: strings.TrimSpace(input.AcceptanceCriteria),
		AllowedPathGlobs:   normalizePlanGlobs(input.AllowedPathGlobs),
		Milestones:         normalizePlanMilestones(input.Milestones),
		RequiredSuite:      strings.TrimSpace(input.RequiredSuite),
	}
}

func (input declarePlanInput) hasProposal() bool {
	return strings.TrimSpace(input.Goal) != "" ||
		strings.TrimSpace(input.AcceptanceCriteria) != "" ||
		len(input.AllowedPathGlobs) > 0 ||
		len(input.Milestones) > 0 ||
		strings.TrimSpace(input.RequiredSuite) != ""
}

func (repo repository) plansDir() string {
	return filepath.Join(repo.eveDir, "plans")
}

func (repo repository) planPath(id string) string {
	return filepath.Join(repo.plansDir(), id+".json")
}

func (repo repository) planRequestsDir() (string, error) {
	return repo.verificationPrivatePath("plan-requests")
}

func (repo repository) planRequestPath(id string) (string, error) {
	dir, err := repo.planRequestsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, id+".json"), nil
}

func (repo repository) planLockPath() (string, error) {
	return repo.verificationPrivatePath("plan-requests.lock")
}

func normalizePlanMilestones(values []eve.PlanMilestone) []eve.PlanMilestone {
	result := make([]eve.PlanMilestone, 0, len(values))
	for _, value := range values {
		value.Title = strings.TrimSpace(value.Title)
		value.Goal = strings.TrimSpace(value.Goal)
		if value.Title != "" || value.Goal != "" {
			result = append(result, value)
		}
	}
	return result
}

func normalizePlanGlobs(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = filepath.ToSlash(strings.TrimSpace(value))
		for strings.Contains(value, "//") {
			value = strings.ReplaceAll(value, "//", "/")
		}
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func validatePlanProposal(proposal planProposal) error {
	var problems []string
	if proposal.Goal == "" {
		problems = append(problems, "goal is required")
	}
	if proposal.AcceptanceCriteria == "" {
		problems = append(problems, "acceptanceCriteria is required")
	}
	if len(proposal.AllowedPathGlobs) == 0 {
		problems = append(problems, "allowedPathGlobs requires at least one pattern")
	}
	for i, pattern := range proposal.AllowedPathGlobs {
		if err := validatePlanGlob(pattern); err != nil {
			problems = append(problems, fmt.Sprintf("allowedPathGlobs[%d]: %v", i, err))
		}
	}
	for i, milestone := range proposal.Milestones {
		if strings.TrimSpace(milestone.Title) == "" {
			problems = append(problems, fmt.Sprintf("milestones[%d].title is required", i))
		}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func validatePlanGlob(pattern string) error {
	if pattern == "" {
		return errors.New("pattern cannot be empty")
	}
	if strings.HasPrefix(pattern, "/") || filepath.IsAbs(pattern) || planWindowsAbsolutePattern.MatchString(pattern) {
		return errors.New("pattern must be repository-relative")
	}
	if strings.HasPrefix(pattern, "!") {
		return errors.New("negated patterns are not supported")
	}
	if strings.Contains(pattern, `\`) {
		return errors.New("use forward slashes")
	}
	if strings.ContainsAny(pattern, "?[]{}") {
		return errors.New("only * and ** wildcards are supported")
	}
	for _, segment := range strings.Split(pattern, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return errors.New("pattern contains an invalid path segment")
		}
		if strings.Contains(segment, "**") && segment != "**" {
			return errors.New("** must occupy an entire path segment")
		}
	}
	return nil
}

func planGlobMatches(pattern string, changedPath string) bool {
	if validatePlanGlob(pattern) != nil {
		return false
	}
	var expression strings.Builder
	expression.WriteString("^")
	for index := 0; index < len(pattern); {
		if pattern[index] == '*' {
			if index+1 < len(pattern) && pattern[index+1] == '*' {
				if index+2 < len(pattern) && pattern[index+2] == '/' {
					expression.WriteString("(?:.*/)?")
					index += 3
					continue
				}
				expression.WriteString(".*")
				index += 2
				continue
			}
			expression.WriteString("[^/]*")
			index++
			continue
		}
		expression.WriteString(regexp.QuoteMeta(string(pattern[index])))
		index++
	}
	expression.WriteString("$")
	matched, err := regexp.MatchString(expression.String(), filepath.ToSlash(changedPath))
	return err == nil && matched
}

func proposalMatchesRevision(proposal planProposal, revision eve.PlanRevision) bool {
	left, _ := json.Marshal(proposal)
	right, _ := json.Marshal(planProposal{
		Goal:               revision.Goal,
		AcceptanceCriteria: revision.AcceptanceCriteria,
		AllowedPathGlobs:   revision.AllowedPathGlobs,
		Milestones:         revision.Milestones,
		RequiredSuite:      revision.ConfiguredSuite,
	})
	return string(left) == string(right)
}

func (repo repository) createOrResumePlanRequest(ctx context.Context, input declarePlanInput) (*planRequest, error) {
	if !planRequestIDPattern.MatchString(strings.TrimSpace(input.PlanRequestID)) {
		return nil, errors.New("planRequestId must start with planreq_ and contain 8-120 letters, numbers, underscores, or hyphens")
	}
	var request *planRequest
	err := repo.withPlanLock(ctx, func() error {
		existing, err := repo.loadPlanRequest(input.PlanRequestID)
		if err == nil {
			if input.hasProposal() && !proposalMatchesRevision(input.proposal(), existing.Revisions[0]) {
				return errors.New("planRequestId already exists with different proposal content")
			}
			request = existing
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if !input.hasProposal() {
			return errors.New("goal, acceptanceCriteria, and allowedPathGlobs are required for a new plan request")
		}
		proposal := input.proposal()
		if err := validatePlanProposal(proposal); err != nil {
			return err
		}
		revision, err := repo.buildPlanRevision(proposal, 1, "agent")
		if err != nil {
			return err
		}
		if pending, err := repo.detectPending(pendingOptions{Initialize: true, Now: time.Now().UTC()}); err != nil {
			return err
		} else if pending != nil {
			return errors.New("unresolved committed work exists; complete or skip it before declaring a new plan")
		}
		requests, err := repo.listPlanRequests()
		if err != nil {
			return err
		}
		availableSuites, err := repo.configuredPlanSuites()
		if err != nil {
			return err
		}
		supersedes := make([]string, 0)
		for _, candidate := range requests {
			if candidate.Branch != revision.Branch || candidate.PlanRequestID == input.PlanRequestID {
				continue
			}
			if candidate.State == "locked" {
				return fmt.Errorf("locked plan %s revision %d must be fulfilled before another plan can be declared on %s", candidate.PlanID, candidate.LockedRevision, candidate.Branch)
			}
			if candidate.State == "pending_approval" {
				supersedes = append(supersedes, candidate.PlanRequestID)
				candidate.State = "superseded"
				candidate.SupersededBy = input.PlanRequestID
				candidate.UpdatedAt = nowUTC()
				if err := repo.savePlanRequest(candidate); err != nil {
					return err
				}
			}
		}
		request = &planRequest{
			SchemaVersion:   planRequestSchemaVersion,
			PlanRequestID:   input.PlanRequestID,
			Repository:      repo.ID,
			RepositoryRoot:  repo.Root,
			Branch:          revision.Branch,
			State:           "pending_approval",
			CurrentRevision: 1,
			Revisions:       []eve.PlanRevision{revision},
			AvailableSuites: availableSuites,
			StaleReasons:    []string{},
			Supersedes:      supersedes,
			CreatedAt:       nowUTC(),
			UpdatedAt:       nowUTC(),
		}
		return repo.savePlanRequest(request)
	})
	if err != nil {
		return nil, err
	}
	return request, nil
}

func (repo repository) buildPlanRevision(proposal planProposal, revision int, source string) (eve.PlanRevision, error) {
	facts, err := deriveGitFacts(repo)
	if err != nil {
		return eve.PlanRevision{}, err
	}
	if facts.Branch == "" {
		return eve.PlanRevision{}, errors.New("plans require an attached Git branch")
	}
	if facts.Dirty {
		return eve.PlanRevision{}, errors.New("working tree must be clean before declaring or approving a plan")
	}
	resolvedSuite, checks, checkIDs, policyHash, definitionsHash, err := repo.resolvePlanChecks(proposal.RequiredSuite, facts.Branch, facts.GitState)
	if err != nil {
		return eve.PlanRevision{}, err
	}
	fingerprint, err := repo.planTreeFingerprint()
	if err != nil {
		return eve.PlanRevision{}, err
	}
	return eve.PlanRevision{
		Revision:             revision,
		Source:               source,
		Goal:                 proposal.Goal,
		AcceptanceCriteria:   proposal.AcceptanceCriteria,
		AllowedPathGlobs:     append([]string{}, proposal.AllowedPathGlobs...),
		Milestones:           append([]eve.PlanMilestone{}, proposal.Milestones...),
		ConfiguredSuite:      proposal.RequiredSuite,
		ResolvedSuite:        resolvedSuite,
		ResolvedChecks:       checks,
		ResolvedCheckIDs:     checkIDs,
		PolicyHash:           policyHash,
		CheckDefinitionsHash: definitionsHash,
		SuiteDigest:          definitionsHash,
		BaseCommit:           facts.GitState,
		Branch:               facts.Branch,
		TreeFingerprint:      fingerprint,
		CreatedAt:            nowUTC(),
	}, nil
}

func (repo repository) resolvePlanChecks(requestedSuite string, branch string, commit string) (string, map[string]eve.PlanResolvedCheck, []string, string, string, error) {
	config, configHash, err := repo.verificationPolicy()
	if err != nil {
		return "", nil, nil, "", "", err
	}
	if len(config.Verification.Suites) == 0 {
		hash, err := planCheckDefinitionsHash("", map[string]eve.PlanResolvedCheck{}, []string{})
		return "", map[string]eve.PlanResolvedCheck{}, []string{}, configHash, hash, err
	}
	ref := resolveVerificationProfile(repo, config, branch, gitTagsAt(repo, commit))
	suite := strings.TrimSpace(requestedSuite)
	if suite == "" {
		suite = ref.ResolvedProfile
	}
	ids, ok := config.Verification.Suites[suite]
	if !ok {
		return "", nil, nil, "", "", fmt.Errorf("verification suite %q is not configured", suite)
	}
	required := config.Verification.Suites[ref.ResolvedProfile]
	if !containsAll(ids, required) {
		return "", nil, nil, "", "", fmt.Errorf("suite %q does not include all checks required by profile %q", suite, ref.ResolvedProfile)
	}
	checks := make(map[string]eve.PlanResolvedCheck, len(ids))
	for _, id := range ids {
		check := config.Verification.Checks[id]
		checks[id] = eve.PlanResolvedCheck{
			Argv:               append([]string{}, check.Argv...),
			WorkingDirectory:   check.WorkingDirectory,
			TimeoutSeconds:     check.TimeoutSeconds,
			SuccessExitCodes:   append([]int{}, check.SuccessExitCodes...),
			OutputLimitBytes:   check.OutputLimitBytes,
			InheritEnvironment: append([]string{}, check.InheritEnvironment...),
			Environment:        cloneStringMap(check.Environment),
		}
	}
	hash, err := planCheckDefinitionsHash(suite, checks, ids)
	return suite, checks, append([]string{}, ids...), configHash, hash, err
}

func (repo repository) configuredPlanSuites() ([]string, error) {
	config, _, err := repo.verificationPolicy()
	if err != nil {
		return nil, err
	}
	suites := make([]string, 0, len(config.Verification.Suites))
	for suite := range config.Verification.Suites {
		suites = append(suites, suite)
	}
	sort.Strings(suites)
	return suites, nil
}

func cloneStringMap(value map[string]string) map[string]string {
	result := make(map[string]string, len(value))
	for key, item := range value {
		result[key] = item
	}
	return result
}

func planCheckDefinitionsHash(suite string, checks map[string]eve.PlanResolvedCheck, ids []string) (string, error) {
	payload := struct {
		Suite  string                           `json:"suite"`
		IDs    []string                         `json:"ids"`
		Checks map[string]eve.PlanResolvedCheck `json:"checks"`
	}{Suite: suite, IDs: ids, Checks: checks}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (repo repository) planTreeFingerprint() (string, error) {
	status, err := gitOutput(repo.Root, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(status))
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (repo repository) refreshPlanRequestState(ctx context.Context, id string) (*planRequest, error) {
	var request *planRequest
	err := repo.withPlanLock(ctx, func() error {
		loaded, err := repo.loadPlanRequest(id)
		if err != nil {
			return err
		}
		request = loaded
		if request.State != "pending_approval" {
			return nil
		}
		revision := request.Revisions[request.CurrentRevision-1]
		reasons := repo.planStaleReasons(revision)
		if len(reasons) == 0 {
			return nil
		}
		request.State = "stale"
		request.StaleReasons = reasons
		request.UpdatedAt = nowUTC()
		return repo.savePlanRequest(request)
	})
	return request, err
}

func (repo repository) planStaleReasons(revision eve.PlanRevision) []string {
	var reasons []string
	facts, err := deriveGitFacts(repo)
	if err != nil {
		return []string{"repository state could not be resolved"}
	}
	if facts.GitState != revision.BaseCommit {
		reasons = append(reasons, "repository HEAD changed")
	}
	if facts.Branch != revision.Branch {
		reasons = append(reasons, "repository branch changed")
	}
	fingerprint, err := repo.planTreeFingerprint()
	if err != nil || fingerprint != revision.TreeFingerprint {
		reasons = append(reasons, "working tree changed")
	}
	suite, _, _, policyHash, definitionsHash, err := repo.resolvePlanChecks(revision.ConfiguredSuite, revision.Branch, revision.BaseCommit)
	if err != nil {
		reasons = append(reasons, "verification policy could not be resolved")
	} else {
		if policyHash != revision.PolicyHash {
			reasons = append(reasons, "verification policy hash changed")
		}
		if suite != revision.ResolvedSuite || definitionsHash != revision.CheckDefinitionsHash || definitionsHash != revision.SuiteDigest {
			reasons = append(reasons, "resolved check suite changed")
		}
	}
	sort.Strings(reasons)
	return reasons
}

func (repo repository) waitForPlanRequest(ctx context.Context, id string) (*planRequest, error) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		request, err := repo.refreshPlanRequestState(ctx, id)
		if err != nil {
			return nil, err
		}
		if request.State != "pending_approval" {
			return request, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (repo repository) approvePlanRequest(ctx context.Context, id string, expectedRevision int, edited *planProposal) (*planRequest, error) {
	var request *planRequest
	err := repo.withPlanLock(ctx, func() error {
		loaded, err := repo.loadPlanRequest(id)
		if err != nil {
			return err
		}
		request = loaded
		if request.State != "pending_approval" {
			return fmt.Errorf("plan request is %s", request.State)
		}
		if expectedRevision != request.CurrentRevision {
			return fmt.Errorf("revision conflict: current revision is %d", request.CurrentRevision)
		}
		current := request.Revisions[request.CurrentRevision-1]
		if reasons := repo.planStaleReasons(current); len(reasons) > 0 {
			request.State = "stale"
			request.StaleReasons = reasons
			request.UpdatedAt = nowUTC()
			if saveErr := repo.savePlanRequest(request); saveErr != nil {
				return saveErr
			}
			return fmt.Errorf("plan request is stale: %s", strings.Join(reasons, "; "))
		}
		if edited != nil {
			edited.Goal = strings.TrimSpace(edited.Goal)
			edited.AcceptanceCriteria = strings.TrimSpace(edited.AcceptanceCriteria)
			edited.AllowedPathGlobs = normalizePlanGlobs(edited.AllowedPathGlobs)
			edited.Milestones = normalizePlanMilestones(edited.Milestones)
			edited.RequiredSuite = strings.TrimSpace(edited.RequiredSuite)
			if err := validatePlanProposal(*edited); err != nil {
				return err
			}
			if !proposalMatchesRevision(*edited, current) {
				next, err := repo.buildPlanRevision(*edited, request.CurrentRevision+1, "human")
				if err != nil {
					return err
				}
				request.Revisions = append(request.Revisions, next)
				request.CurrentRevision = next.Revision
			}
		}
		request.PlanID = newRecordID("plan")
		request.State = "locked"
		request.LockedRevision = request.CurrentRevision
		request.LockedAt = nowUTC()
		request.UpdatedAt = request.LockedAt
		return repo.savePlanRequest(request)
	})
	return request, err
}

func (repo repository) rejectPlanRequest(ctx context.Context, id string, expectedRevision int, feedback string) (*planRequest, error) {
	feedback = strings.TrimSpace(feedback)
	if feedback == "" {
		return nil, errors.New("rejection feedback is required")
	}
	var request *planRequest
	err := repo.withPlanLock(ctx, func() error {
		loaded, err := repo.loadPlanRequest(id)
		if err != nil {
			return err
		}
		request = loaded
		if request.State != "pending_approval" {
			return fmt.Errorf("plan request is %s", request.State)
		}
		if expectedRevision != request.CurrentRevision {
			return fmt.Errorf("revision conflict: current revision is %d", request.CurrentRevision)
		}
		current := request.Revisions[request.CurrentRevision-1]
		if reasons := repo.planStaleReasons(current); len(reasons) > 0 {
			request.State = "stale"
			request.StaleReasons = reasons
			request.UpdatedAt = nowUTC()
			if saveErr := repo.savePlanRequest(request); saveErr != nil {
				return saveErr
			}
			return fmt.Errorf("plan request is stale: %s", strings.Join(reasons, "; "))
		}
		request.State = "rejected"
		request.RejectionFeedback = feedback
		request.UpdatedAt = nowUTC()
		return repo.savePlanRequest(request)
	})
	return request, err
}

func (repo repository) loadPlanRequest(id string) (*planRequest, error) {
	path, err := repo.planRequestPath(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var request planRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("parse plan request %s: %w", id, err)
	}
	return &request, nil
}

func (repo repository) listPlanRequests() ([]*planRequest, error) {
	dir, err := repo.planRequestsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []*planRequest{}, nil
	}
	if err != nil {
		return nil, err
	}
	requests := make([]*planRequest, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		request, err := repo.loadPlanRequest(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	sort.Slice(requests, func(i, j int) bool {
		if requests[i].UpdatedAt == requests[j].UpdatedAt {
			return requests[i].PlanRequestID < requests[j].PlanRequestID
		}
		return requests[i].UpdatedAt > requests[j].UpdatedAt
	})
	return requests, nil
}

func (repo repository) savePlanRequest(request *planRequest) error {
	path, err := repo.planRequestPath(request.PlanRequestID)
	if err != nil {
		return err
	}
	request.SchemaVersion = planRequestSchemaVersion
	if request.StaleReasons == nil {
		request.StaleReasons = []string{}
	}
	if request.AvailableSuites == nil {
		request.AvailableSuites = []string{}
	}
	return writeJSONAtomic(path, request, 0o600)
}

func writeJSONAtomic(path string, value any, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func (repo repository) withPlanLock(ctx context.Context, operation func() error) error {
	path, err := repo.planLockPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	for {
		file, openErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if openErr == nil {
			_, _ = file.WriteString(fmt.Sprintf("pid=%d\ncreated=%s\n", os.Getpid(), nowUTC()))
			_ = file.Close()
			defer os.Remove(path)
			return operation()
		}
		if !errors.Is(openErr, os.ErrExist) {
			return openErr
		}
		if info, statErr := os.Stat(path); statErr == nil && time.Since(info.ModTime()) > 30*time.Second {
			_ = os.Remove(path)
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func (repo repository) findLockedPlan(planID string, revision int) (*planRequest, error) {
	requests, err := repo.listPlanRequests()
	if err != nil {
		return nil, err
	}
	for _, request := range requests {
		if request.PlanID == planID {
			if request.State != "locked" {
				return nil, fmt.Errorf("plan %s is %s", planID, request.State)
			}
			if request.LockedRevision != revision {
				return nil, fmt.Errorf("plan %s locked revision is %d, not %d", planID, request.LockedRevision, revision)
			}
			return request, nil
		}
	}
	return nil, fmt.Errorf("plan %s not found", planID)
}

func (repo repository) savePlanRecord(request *planRequest, snapshotID string) (*eve.PlanRecord, error) {
	record := &eve.PlanRecord{
		ID:             request.PlanID,
		SchemaVersion:  eve.PlanSchemaVersion,
		PlanRequestID:  request.PlanRequestID,
		Repository:     request.Repository,
		Status:         "fulfilled",
		Revisions:      append([]eve.PlanRevision{}, request.Revisions...),
		LockedRevision: request.LockedRevision,
		LockedAt:       request.LockedAt,
		ApprovedBy:     "local_ui",
		FulfilledBy:    snapshotID,
	}
	if err := eve.ValidatePlanRecord(record); err != nil {
		return nil, err
	}
	if err := writeJSONAtomic(repo.planPath(record.ID), record, 0o644); err != nil {
		return nil, err
	}
	return record, nil
}

func (repo repository) loadPlanRecord(id string) (*eve.PlanRecord, error) {
	data, err := os.ReadFile(repo.planPath(id))
	if err != nil {
		return nil, fmt.Errorf("read plan %s: %w", id, err)
	}
	var record eve.PlanRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("parse plan %s: %w", id, err)
	}
	return &record, nil
}

func (repo repository) evaluatePlanConformance(request *planRequest, facts gitFacts, verification *eve.Verification) (*eve.PlanConformance, error) {
	revision := request.Revisions[request.LockedRevision-1]
	if facts.Branch != revision.Branch {
		return nil, fmt.Errorf("implementation branch %q does not match locked plan branch %q", facts.Branch, revision.Branch)
	}
	ancestor, err := gitOutput(repo.Root, "merge-base", "--is-ancestor", revision.BaseCommit, facts.GitState)
	if err != nil {
		_ = ancestor
		return nil, errors.New("locked plan base commit is not an ancestor of implementation HEAD")
	}
	changedPaths, err := repo.planChangedPaths(revision.BaseCommit, facts.GitState)
	if err != nil {
		return nil, err
	}
	outOfScope := make([]string, 0)
	for _, changedPath := range changedPaths {
		matched := false
		for _, pattern := range revision.AllowedPathGlobs {
			if planGlobMatches(pattern, changedPath) {
				matched = true
				break
			}
		}
		if !matched {
			outOfScope = append(outOfScope, changedPath)
		}
	}
	policyEvidencePresent := verification != nil && verification.ConfigBlobHash != ""
	policyMatched := policyEvidencePresent && verification.ConfigBlobHash == revision.PolicyHash
	definitionsMatched := false
	runEvidencePresent := verification != nil && verification.SelectedRunID != ""
	checkStatus := "incomplete"
	if len(revision.ResolvedCheckIDs) == 0 {
		definitionsMatched = true
		checkStatus = "not_configured"
	} else if runEvidencePresent {
		run, runErr := (&runtimeServer{verificationRegistry: newVerificationRegistry()}).verificationRun(repo, verification.SelectedRunID)
		if runErr == nil {
			if run.Suite == revision.ResolvedSuite && equalStrings(run.ResolvedSuite, revision.ResolvedCheckIDs) {
				runChecks := make(map[string]eve.PlanResolvedCheck, len(revision.ResolvedCheckIDs))
				for _, id := range revision.ResolvedCheckIDs {
					if check, ok := run.ResolvedChecks[id]; ok {
						runChecks[id] = eve.PlanResolvedCheck{
							Argv: append([]string{}, check.Argv...), WorkingDirectory: check.WorkingDirectory,
							TimeoutSeconds: check.TimeoutSeconds, SuccessExitCodes: append([]int{}, check.SuccessExitCodes...),
							OutputLimitBytes: check.OutputLimitBytes, InheritEnvironment: append([]string{}, check.InheritEnvironment...),
							Environment: cloneStringMap(check.Environment),
						}
					}
				}
				hash, _ := planCheckDefinitionsHash(run.Suite, runChecks, run.ResolvedSuite)
				definitionsMatched = hash == revision.CheckDefinitionsHash
			}
		}
		resultByID := map[string]string{}
		for _, result := range verification.CheckResults {
			resultByID[result.CheckID] = result.Status
		}
		checkStatus = "passed"
		for _, id := range revision.ResolvedCheckIDs {
			status, ok := resultByID[id]
			if !ok {
				checkStatus = "incomplete"
				break
			}
			if status != "passed" {
				checkStatus = "failed"
				break
			}
		}
	}
	conformance := &eve.PlanConformance{
		Status:                "matched",
		NoPlanOnFile:          false,
		RequiredChecksStatus:  checkStatus,
		PolicyMatched:         policyMatched,
		CheckDefinitionsMatch: definitionsMatched,
		ScopeDrift:            len(outOfScope) > 0,
		ChangedPaths:          changedPaths,
		OutOfScopePaths:       outOfScope,
	}
	if checkStatus == "incomplete" || checkStatus == "not_configured" {
		conformance.Status = "incomplete"
	}
	if checkStatus == "failed" || (policyEvidencePresent && !policyMatched) || (runEvidencePresent && !definitionsMatched) || conformance.ScopeDrift {
		conformance.Status = "failed"
	}
	return conformance, nil
}

func (repo repository) planChangedPaths(base string, head string) ([]string, error) {
	output, err := gitOutput(repo.Root, "diff", "--name-status", "-z", "--find-renames", base+".."+head)
	if err != nil {
		return nil, err
	}
	tokens := strings.Split(output, "\x00")
	paths := make([]string, 0)
	seen := map[string]bool{}
	for index := 0; index < len(tokens); {
		status := strings.TrimSpace(tokens[index])
		index++
		if status == "" {
			continue
		}
		count := 1
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			count = 2
		}
		for item := 0; item < count && index < len(tokens); item++ {
			value := filepath.ToSlash(strings.TrimSpace(tokens[index]))
			index++
			if value != "" && !seen[value] {
				seen[value] = true
				paths = append(paths, value)
			}
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func (repo repository) fulfillPlanRequest(ctx context.Context, request *planRequest, snapshotID string) error {
	return repo.withPlanLock(ctx, func() error {
		current, err := repo.loadPlanRequest(request.PlanRequestID)
		if err != nil {
			return err
		}
		if current.State != "locked" {
			return fmt.Errorf("plan request is %s", current.State)
		}
		current.State = "fulfilled"
		current.FulfilledSnapshotID = snapshotID
		current.UpdatedAt = nowUTC()
		return repo.savePlanRequest(current)
	})
}

func noPlanConformance() *eve.PlanConformance {
	return &eve.PlanConformance{
		Status:                "no_plan",
		NoPlanOnFile:          true,
		RequiredChecksStatus:  "incomplete",
		PolicyMatched:         false,
		CheckDefinitionsMatch: false,
		ScopeDrift:            false,
		ChangedPaths:          []string{},
		OutOfScopePaths:       []string{},
	}
}
