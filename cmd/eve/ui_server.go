package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/nhestrompia/eve"
)

//go:embed ui_dist/* ui_dist/assets/*
var embeddedUI embed.FS

type runtimeServer struct {
	repo                 repository
	addr                 string
	searchCache          *snapshotSearchCache
	verificationRegistry *verificationRegistry
}

type apiError struct {
	Error string `json:"error"`
}

type configResponse struct {
	SnapshotSchemaVersion string           `json:"snapshotSchemaVersion"`
	CLIVersion            string           `json:"cliVersion"`
	Repository            string           `json:"repository"`
	Addr                  string           `json:"addr"`
	EveDir                string           `json:"eveDir"`
	Initialized           bool             `json:"initialized"`
	CurrentGitState       string           `json:"currentGitState,omitempty"`
	CurrentBranch         string           `json:"currentBranch,omitempty"`
	CurrentDirty          bool             `json:"currentDirty"`
	LatestSnapshot        string           `json:"latestSnapshot,omitempty"`
	LatestGitState        string           `json:"latestGitState,omitempty"`
	PendingSnapshot       *pendingSnapshot `json:"pendingSnapshot,omitempty"`
}

type snapshotSummary struct {
	ID                    string `json:"id"`
	Repository            string `json:"repository,omitempty"`
	Title                 string `json:"title"`
	Type                  string `json:"type"`
	Summary               string `json:"summary"`
	UserVisibleChange     string `json:"userVisibleChange,omitempty"`
	GitState              string `json:"gitState"`
	Branch                string `json:"branch"`
	Dirty                 bool   `json:"dirty"`
	CommitCount           int    `json:"commitCount"`
	DecisionCount         int    `json:"decisionCount"`
	RiskCount             int    `json:"riskCount"`
	ArtifactCount         int    `json:"artifactCount"`
	FailedValidationCount int    `json:"failedValidationCount"`
	ValidationState       string `json:"validationState"`
	CreatedAt             string `json:"createdAt"`
}

type snapshotDetailResponse struct {
	Repository string            `json:"repository"`
	Snapshot   *eve.Snapshot     `json:"snapshot"`
	Summary    snapshotSummary   `json:"summary"`
	Sessions   []uiSessionRecord `json:"sessions"`
	Providers  []uiProviderInfo  `json:"providers"`
	Commits    []uiGitCommit     `json:"commits"`
	RawJSON    json.RawMessage   `json:"rawJson"`
}

type snapshotSearchResponse struct {
	Query   string                 `json:"query"`
	Results []snapshotSearchResult `json:"results"`
}

type snapshotSearchResult struct {
	Evolution snapshotSummary `json:"evolution"`
	Matches   []string        `json:"matches"`
}

type indexedSnapshot struct {
	Repo       repository
	Snapshot   *eve.Snapshot
	Summary    snapshotSummary
	SearchText string
}

type snapshotSearchCache struct {
	mu        sync.Mutex
	expiresAt time.Time
	signature string
	entries   []indexedSnapshot
}

type sessionListResponse struct {
	EvolutionID string            `json:"evolutionId"`
	Sessions    []uiSessionRecord `json:"sessions"`
	Providers   []uiProviderInfo  `json:"providers"`
}

type sessionTranscriptResponse struct {
	EvolutionID string `json:"evolutionId"`
	Provider    string `json:"provider"`
	ID          string `json:"id"`
	Key         string `json:"key"`
	Title       string `json:"title"`
	Markdown    string `json:"markdown"`
	Sanitized   bool   `json:"sanitized"`
}

type uiSessionRecord struct {
	Provider      string            `json:"provider"`
	ProviderName  string            `json:"providerName"`
	ID            string            `json:"id"`
	Key           string            `json:"key"`
	URI           string            `json:"uri,omitempty"`
	Title         string            `json:"title,omitempty"`
	Transcript    string            `json:"transcript,omitempty"`
	Raw           string            `json:"raw,omitempty"`
	Sanitized     bool              `json:"sanitized"`
	Format        string            `json:"format,omitempty"`
	AttachedAt    string            `json:"attachedAt,omitempty"`
	Source        string            `json:"source,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	HasTranscript bool              `json:"hasTranscript"`
	Status        string            `json:"status"`
	CaptureHint   string            `json:"captureHint"`
	LocalSources  []uiSessionSource `json:"localSources"`
	RootsChecked  []string          `json:"rootsChecked"`
	Preview       uiSessionPreview  `json:"preview"`
}

type uiSessionSource struct {
	Path       string `json:"path"`
	Format     string `json:"format"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
	Title      string `json:"title,omitempty"`
	Match      string `json:"match,omitempty"`
}

type uiSessionPreview struct {
	EventCount     int      `json:"eventCount"`
	MessageCount   int      `json:"messageCount"`
	UserMessages   int      `json:"userMessages"`
	AgentMessages  int      `json:"agentMessages"`
	ToolCalls      int      `json:"toolCalls"`
	FirstTimestamp string   `json:"firstTimestamp,omitempty"`
	LastTimestamp  string   `json:"lastTimestamp,omitempty"`
	Headings       []string `json:"headings,omitempty"`
}

type uiProviderInfo struct {
	Provider      string   `json:"provider"`
	Name          string   `json:"name"`
	Roots         []string `json:"roots"`
	Available     bool     `json:"available"`
	ImportCommand string   `json:"importCommand"`
	Displays      []string `json:"displays"`
}

type uiGitCommit struct {
	Hash        string `json:"hash"`
	ShortHash   string `json:"shortHash"`
	Subject     string `json:"subject"`
	AuthorName  string `json:"authorName"`
	AuthoredAt  string `json:"authoredAt"`
	CommittedAt string `json:"committedAt"`
}

const snapshotCodePreviewLimit = 200 * 1024

type snapshotCodeFilesResponse struct {
	Repository string             `json:"repository"`
	SnapshotID string             `json:"snapshotId"`
	Base       string             `json:"base,omitempty"`
	Head       string             `json:"head"`
	Files      []snapshotCodeFile `json:"files"`
}

type snapshotCodeFile struct {
	Path        string `json:"path"`
	OldPath     string `json:"oldPath,omitempty"`
	Status      string `json:"status"`
	Language    string `json:"language"`
	Curated     bool   `json:"curated"`
	Evidence    string `json:"evidence,omitempty"`
	SizeBytes   int64  `json:"sizeBytes"`
	Previewable bool   `json:"previewable"`
	Reason      string `json:"reason,omitempty"`
}

type snapshotCodeFileResponse struct {
	Path            string `json:"path"`
	Language        string `json:"language"`
	Mode            string `json:"mode"`
	Content         string `json:"content"`
	HighlightedHTML string `json:"highlightedHtml,omitempty"`
	SizeBytes       int64  `json:"sizeBytes"`
	Previewable     bool   `json:"previewable"`
	Reason          string `json:"reason,omitempty"`
}

type legacyEvolution struct {
	Metadata struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		UpdatedAt string `json:"updated_at"`
		CreatedAt string `json:"created_at"`
	} `json:"metadata"`
	Sessions       []legacySession `json:"sessions"`
	Implementation struct {
		Snapshot string   `json:"snapshot"`
		Commits  []string `json:"commits"`
	} `json:"implementation"`
}

type legacySession struct {
	Provider string `json:"provider,omitempty"`
	ID       string `json:"id,omitempty"`
	URI      string `json:"uri,omitempty"`
}

func newRuntimeServer(repo repository, addr string) runtimeServer {
	return runtimeServer{repo: repo, addr: addr, searchCache: &snapshotSearchCache{}, verificationRegistry: newVerificationRegistry()}
}

func (server runtimeServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", server.handleConfig)
	mux.HandleFunc("/api/compare", server.handleCompare)
	mux.HandleFunc("/api/search", server.handleSnapshotSearch)
	mux.HandleFunc("/api/snapshots", server.handleGlobalSnapshots)
	mux.HandleFunc("/api/snapshots/", server.handleGlobalSnapshotRoutes)
	mux.HandleFunc("/api/repos", server.handleRepos)
	mux.HandleFunc("/api/repos/", server.handleRepoRoutes)
	mux.HandleFunc("/mcp", server.handleMCPHTTP)
	mux.Handle("/", spaHandler())
	return logRequests(mux)
}

func (server runtimeServer) handleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	fromID := strings.TrimSpace(r.URL.Query().Get("from"))
	toID := strings.TrimSpace(r.URL.Query().Get("to"))
	if fromID == "" || toID == "" {
		writeAPIError(w, http.StatusBadRequest, fmt.Errorf("from and to query parameters are required"))
		return
	}
	fromRepo, ok := server.repoForSnapshot(fromID)
	if !ok {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("snapshot %s not found", fromID))
		return
	}
	toRepo, ok := server.repoForSnapshot(toID)
	if !ok {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("snapshot %s not found", toID))
		return
	}
	if fromRepo.Root != toRepo.Root {
		writeAPIError(w, http.StatusBadRequest, fmt.Errorf("snapshots must belong to the same repository"))
		return
	}
	comparison, err := compareSnapshots(fromRepo, fromID, toID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, comparison)
}

func (server runtimeServer) handleGlobalSnapshots(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	entries, err := server.indexedSnapshots()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	rows := make([]snapshotSummary, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, entry.Summary)
	}
	writeJSON(w, http.StatusOK, rows)
}

func (server runtimeServer) handleGlobalSnapshotRoutes(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/snapshots/"), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 1 || parts[0] == "" || r.Method != http.MethodGet {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("snapshot route not found"))
		return
	}
	repo, ok := server.repoForSnapshot(parts[0])
	if !ok {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("snapshot %s not found", parts[0]))
		return
	}
	server.handleSnapshotDetail(w, r, repo, parts[0])
}

func (server runtimeServer) handleSnapshotSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	normalized := strings.ToLower(query)
	limit := 50
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
			limit = min(parsed, 200)
		}
	}
	entries, err := server.indexedSnapshots()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	results := make([]snapshotSearchResult, 0, min(len(entries), limit))
	for _, entry := range entries {
		if normalized != "" && !strings.Contains(entry.SearchText, normalized) {
			continue
		}
		results = append(results, snapshotSearchResult{
			Evolution: entry.Summary,
			Matches:   snapshotSearchMatches(entry.Snapshot, entry.Repo, normalized),
		})
		if len(results) >= limit {
			break
		}
	}
	writeJSON(w, http.StatusOK, snapshotSearchResponse{Query: query, Results: results})
}

func (server runtimeServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	_, err := os.Stat(server.repo.configPath())
	summary, summaryErr := server.repo.summary()
	facts, factsErr := deriveGitFacts(server.repo)
	if summaryErr != nil {
		summary = repoSummary{}
	}
	if factsErr != nil {
		facts = gitFacts{}
	}
	writeJSON(w, http.StatusOK, configResponse{
		SnapshotSchemaVersion: eve.SnapshotSchemaVersion,
		CLIVersion:            eve.CLIVersion,
		Repository:            server.repo.ID,
		Addr:                  server.addr,
		EveDir:                server.repo.eveDir,
		Initialized:           err == nil,
		CurrentGitState:       facts.GitState,
		CurrentBranch:         facts.Branch,
		CurrentDirty:          facts.Dirty,
		LatestSnapshot:        summary.LatestSnapshot,
		LatestGitState:        summary.LatestGitState,
		PendingSnapshot:       summary.PendingSnapshot,
	})
}

func (server runtimeServer) handleRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	repos := server.repositories()
	summaries := make([]repoSummary, 0, len(repos))
	for _, repo := range repos {
		summary, err := repo.summary()
		if err != nil {
			if repo.Root == server.repo.Root {
				writeAPIError(w, http.StatusInternalServerError, err)
				return
			}
			continue
		}
		summaries = append(summaries, summary)
	}
	writeJSON(w, http.StatusOK, summaries)
}

func (server runtimeServer) handleRepoRoutes(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/repos/"), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("repo route not found"))
		return
	}
	repoID := parts[0]
	repo, ok := server.repoByID(repoID)
	if !ok {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("repository %s not found", repoID))
		return
	}
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		detail, err := repo.detail()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, detail)
	case len(parts) == 2 && parts[1] == "pending" && r.Method == http.MethodGet:
		server.handlePendingSnapshot(w, r, repo)
	case len(parts) == 2 && parts[1] == "open-editor" && r.Method == http.MethodPost:
		writeJSON(w, http.StatusOK, openRepositoryInEditor(repo))
	case len(parts) >= 3 && parts[1] == "artifacts" && r.Method == http.MethodGet:
		server.handleArtifactFile(w, r, repo, strings.Join(parts[2:], "/"))
	case len(parts) == 2 && parts[1] == "snapshots" && r.Method == http.MethodGet:
		server.handleSnapshots(w, r, repo)
	case len(parts) == 3 && parts[1] == "snapshots" && r.Method == http.MethodGet:
		server.handleSnapshotDetail(w, r, repo, parts[2])
	case len(parts) == 5 && parts[1] == "snapshots" && parts[3] == "code" && parts[4] == "files" && r.Method == http.MethodGet:
		server.handleSnapshotCodeFiles(w, r, repo, parts[2])
	case len(parts) == 5 && parts[1] == "snapshots" && parts[3] == "code" && parts[4] == "file" && r.Method == http.MethodGet:
		server.handleSnapshotCodeFile(w, r, repo, parts[2])
	case len(parts) == 4 && parts[1] == "snapshots" && parts[3] == "checkout" && r.Method == http.MethodPost:
		server.handleCheckout(w, r, repo, parts[2])
	case len(parts) == 4 && parts[1] == "snapshots" && parts[3] == "sessions" && r.Method == http.MethodGet:
		server.handleSessions(w, r, repo, parts[2])
	case len(parts) == 5 && parts[1] == "snapshots" && parts[3] == "sessions" && r.Method == http.MethodGet:
		server.handleSessionTranscript(w, r, repo, parts[2], parts[4])
	default:
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("repo route not found"))
	}
}

func (server runtimeServer) handlePendingSnapshot(w http.ResponseWriter, r *http.Request, repo repository) {
	pending, err := repo.detectPending(pendingOptions{Initialize: true, Now: time.Now().UTC()})
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, pendingSnapshotResponse{
		Pending:         pending != nil,
		PendingSnapshot: pending,
	})
}

func (server runtimeServer) repositories() []repository {
	return knownRepositories(server.repo)
}

func (server runtimeServer) repoByID(repoID string) (repository, bool) {
	for _, repo := range server.repositories() {
		if repo.ID == repoID {
			return repo, true
		}
	}
	return repository{}, false
}

func (server runtimeServer) handleArtifactFile(w http.ResponseWriter, r *http.Request, repo repository, artifactPath string) {
	if artifactPath == "" {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("artifact not found"))
		return
	}
	artifactRoot, err := filepath.Abs(repo.artifactsDir())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	cleanPath := strings.TrimPrefix(path.Clean("/"+artifactPath), "/")
	target, err := filepath.Abs(filepath.Join(artifactRoot, filepath.FromSlash(cleanPath)))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	if target != artifactRoot && !strings.HasPrefix(target, artifactRoot+string(os.PathSeparator)) {
		writeAPIError(w, http.StatusBadRequest, fmt.Errorf("artifact path escapes repository artifacts directory"))
		return
	}
	if info, err := os.Stat(target); err != nil || info.IsDir() {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("artifact not found"))
		return
	}
	http.ServeFile(w, r, target)
}

func (server runtimeServer) handleSnapshots(w http.ResponseWriter, r *http.Request, repo repository) {
	snapshots, err := repo.listSnapshots(r.URL.Query().Get("type"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	rows := make([]snapshotSummary, 0, len(snapshots))
	for _, snapshot := range snapshots {
		rows = append(rows, summarizeSnapshotForRepo(repo, snapshot))
	}
	writeJSON(w, http.StatusOK, rows)
}

func (server runtimeServer) handleSnapshotDetail(w http.ResponseWriter, r *http.Request, repo repository, id string) {
	snapshot, raw, err := server.loadSnapshotWithRaw(repo, id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	commits := server.gitCommits(repo, snapshotImplementationCommits(repo, snapshot))
	writeJSON(w, http.StatusOK, snapshotDetailResponse{
		Repository: repo.ID,
		Snapshot:   snapshot,
		Summary:    summarizeSnapshotForRepo(repo, snapshot),
		Sessions:   server.snapshotSessionRecords(repo, snapshot),
		Providers:  providerInfos(),
		Commits:    commits,
		RawJSON:    raw,
	})
}

func (server runtimeServer) handleSessions(w http.ResponseWriter, r *http.Request, repo repository, id string) {
	snapshot, err := repo.loadSnapshot(id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, sessionListResponse{
		EvolutionID: id,
		Sessions:    server.snapshotSessionRecords(repo, snapshot),
		Providers:   providerInfos(),
	})
}

func (server runtimeServer) handleSessionTranscript(w http.ResponseWriter, r *http.Request, repo repository, id string, sessionKey string) {
	snapshot, err := repo.loadSnapshot(id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	for _, session := range server.snapshotSessionRecords(repo, snapshot) {
		if session.Key != sessionKey {
			continue
		}
		if strings.TrimSpace(session.Transcript) == "" {
			writeAPIError(w, http.StatusNotFound, fmt.Errorf("session transcript is not available"))
			return
		}
		data, err := os.ReadFile(filepath.FromSlash(session.Transcript))
		if err != nil {
			writeAPIError(w, http.StatusNotFound, fmt.Errorf("read session transcript: %w", err))
			return
		}
		writeJSON(w, http.StatusOK, sessionTranscriptResponse{
			EvolutionID: id,
			Provider:    session.Provider,
			ID:          session.ID,
			Key:         session.Key,
			Title:       fallback(session.Title, session.ProviderName+" "+session.ID),
			Markdown:    string(data),
			Sanitized:   session.Sanitized,
		})
		return
	}
	writeAPIError(w, http.StatusNotFound, fmt.Errorf("session %s not found", sessionKey))
}

func (server runtimeServer) handleCheckout(w http.ResponseWriter, r *http.Request, repo repository, id string) {
	snapshot, err := repo.loadSnapshot(id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	force := r.URL.Query().Get("force") == "true"
	writeJSON(w, http.StatusOK, checkoutSnapshot(repo, snapshot, force))
}

func (server runtimeServer) handleSnapshotCodeFiles(w http.ResponseWriter, r *http.Request, repo repository, id string) {
	snapshot, err := repo.loadSnapshot(id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	base, head := snapshotCodeRange(repo, snapshot)
	files, err := snapshotChangedCodeFiles(repo, snapshot, base, head)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshotCodeFilesResponse{
		Repository: repo.ID,
		SnapshotID: snapshot.ID,
		Base:       base,
		Head:       head,
		Files:      files,
	})
}

func (server runtimeServer) handleSnapshotCodeFile(w http.ResponseWriter, r *http.Request, repo repository, id string) {
	snapshot, err := repo.loadSnapshot(id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	filePath, err := cleanSnapshotCodePath(r.URL.Query().Get("path"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	if mode == "" {
		mode = "diff"
	}
	if mode != "diff" && mode != "full" {
		writeAPIError(w, http.StatusBadRequest, fmt.Errorf("mode must be diff or full"))
		return
	}

	base, head := snapshotCodeRange(repo, snapshot)
	files, err := snapshotChangedCodeFiles(repo, snapshot, base, head)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	var selected snapshotCodeFile
	found := false
	for _, file := range files {
		if file.Path == filePath {
			selected = file
			found = true
			break
		}
	}
	if !found {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("file %s is not changed in snapshot %s", filePath, snapshot.ID))
		return
	}

	response := snapshotCodeFileResponse{
		Path:        selected.Path,
		Language:    languageFromPath(selected.Path),
		Mode:        mode,
		SizeBytes:   selected.SizeBytes,
		Previewable: true,
	}
	if selected.Reason != "" {
		response.Previewable = false
		response.Reason = selected.Reason
		writeJSON(w, http.StatusOK, response)
		return
	}

	switch mode {
	case "diff":
		content, err := snapshotFileDiff(repo, base, head, selected.Path)
		if err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		if int64(len(content)) > snapshotCodePreviewLimit {
			response.Previewable = false
			response.Reason = "File diff is too large to preview."
			writeJSON(w, http.StatusOK, response)
			return
		}
		response.Content = content
	case "full":
		if selected.Status == "D" {
			response.Previewable = false
			response.Reason = "File was deleted in this Snapshot."
			writeJSON(w, http.StatusOK, response)
			return
		}
		if selected.SizeBytes > snapshotCodePreviewLimit {
			response.Previewable = false
			response.Reason = "File too large to preview. Open in editor or GitHub."
			writeJSON(w, http.StatusOK, response)
			return
		}
		content, err := snapshotFileAtGitState(repo, head, selected.Path)
		if err != nil {
			writeAPIError(w, http.StatusNotFound, err)
			return
		}
		response.Content = content
	}
	writeJSON(w, http.StatusOK, response)
}

func (server runtimeServer) loadSnapshotWithRaw(repo repository, id string) (*eve.Snapshot, json.RawMessage, error) {
	data, err := os.ReadFile(repo.snapshotPath(id))
	if err != nil {
		return nil, nil, err
	}
	snapshot, err := eve.ParseSnapshot(data)
	if err != nil {
		return nil, nil, err
	}
	verifySnapshotRunEvidence(repo, snapshot)
	return snapshot, json.RawMessage(data), nil
}

func summarizeSnapshot(snapshot *eve.Snapshot) snapshotSummary {
	state := validationState(snapshot.Validation)
	if snapshot.Verification != nil && snapshot.Verification.Status != "" {
		state = snapshot.Verification.Status
	}
	return snapshotSummary{
		ID:                    snapshot.ID,
		Title:                 snapshot.Title,
		Type:                  snapshot.Type,
		Summary:               snapshot.Summary,
		UserVisibleChange:     snapshot.UserVisibleChange,
		GitState:              snapshot.Implementation.GitState,
		Branch:                snapshot.Implementation.Branch,
		Dirty:                 snapshot.Implementation.Dirty,
		CommitCount:           len(snapshot.Implementation.Commits),
		DecisionCount:         len(snapshot.Decisions),
		RiskCount:             len(snapshot.Risks),
		ArtifactCount:         len(snapshot.Artifacts),
		FailedValidationCount: failedValidationCount(snapshot.Validation),
		ValidationState:       state,
		CreatedAt:             snapshot.CreatedAt,
	}
}

func summarizeSnapshotForRepo(repo repository, snapshot *eve.Snapshot) snapshotSummary {
	summary := summarizeSnapshot(snapshot)
	summary.Repository = repo.ID
	summary.CommitCount = len(snapshotImplementationCommits(repo, snapshot))
	return summary
}

func (server runtimeServer) indexedSnapshots() ([]indexedSnapshot, error) {
	repos := server.repositories()
	signature := repositorySignature(repos)
	now := time.Now()
	cache := server.searchCache
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.signature == signature && now.Before(cache.expiresAt) {
		return append([]indexedSnapshot(nil), cache.entries...), nil
	}

	entries := make([]indexedSnapshot, 0)
	for _, repo := range repos {
		snapshots, err := repo.listSnapshots("")
		if err != nil {
			if repo.Root == server.repo.Root {
				return nil, err
			}
			continue
		}
		for _, snapshot := range snapshots {
			entries = append(entries, indexedSnapshot{
				Repo:       repo,
				Snapshot:   snapshot,
				Summary:    summarizeSnapshotForRepo(repo, snapshot),
				SearchText: snapshotSearchText(repo, snapshot),
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Summary.CreatedAt == entries[j].Summary.CreatedAt {
			return entries[i].Summary.ID < entries[j].Summary.ID
		}
		return entries[i].Summary.CreatedAt > entries[j].Summary.CreatedAt
	})
	cache.signature = signature
	cache.expiresAt = now.Add(5 * time.Second)
	cache.entries = append([]indexedSnapshot(nil), entries...)
	return entries, nil
}

func (server runtimeServer) repoForSnapshot(id string) (repository, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return repository{}, false
	}
	if entries, err := server.indexedSnapshots(); err == nil {
		for _, entry := range entries {
			if entry.Summary.ID == id {
				return entry.Repo, true
			}
		}
	}
	for _, repo := range server.repositories() {
		if _, err := os.Stat(repo.snapshotPath(id)); err == nil {
			return repo, true
		}
	}
	return repository{}, false
}

func repositorySignature(repos []repository) string {
	parts := make([]string, 0, len(repos))
	for _, repo := range repos {
		parts = append(parts, repo.ID+"\x00"+repo.Root)
	}
	return strings.Join(parts, "\x00")
}

func snapshotSearchText(repo repository, snapshot *eve.Snapshot) string {
	values := []string{
		repo.ID,
		repo.Root,
		snapshot.ID,
		snapshot.Title,
		snapshot.Type,
		snapshot.Summary,
		snapshot.UserVisibleChange,
		snapshot.Implementation.GitState,
		snapshot.Implementation.Branch,
		strings.Join(snapshot.Implementation.Commits, " "),
	}
	for _, validation := range snapshot.Validation {
		values = append(values, validation.Command, validation.Status, validation.Output)
	}
	for _, decision := range snapshot.Decisions {
		values = append(values, decision.Title, decision.Rationale)
	}
	for _, risk := range snapshot.Risks {
		values = append(values, risk.Title, risk.Severity, risk.Mitigation)
	}
	return strings.ToLower(strings.Join(values, "\n"))
}

func snapshotSearchMatches(snapshot *eve.Snapshot, repo repository, query string) []string {
	candidates := []string{snapshot.Title, snapshot.Summary, snapshot.UserVisibleChange, snapshot.ID, repo.ID}
	for _, validation := range snapshot.Validation {
		candidates = append(candidates, validation.Command)
	}
	for _, commit := range snapshot.Implementation.Commits {
		candidates = append(candidates, commit)
	}
	if snapshot.Implementation.GitState != "" {
		candidates = append(candidates, snapshot.Implementation.GitState)
	}
	matches := make([]string, 0, 3)
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		if query == "" || strings.Contains(strings.ToLower(candidate), query) {
			matches = append(matches, candidate)
			seen[candidate] = true
		}
		if len(matches) >= 3 {
			break
		}
	}
	if len(matches) == 0 {
		matches = append(matches, snapshot.Title)
	}
	return matches
}

func (server runtimeServer) snapshotSessionRecords(repo repository, snapshot *eve.Snapshot) []uiSessionRecord {
	seen := map[string]bool{}
	var records []uiSessionRecord
	add := func(record uiSessionRecord) {
		if strings.TrimSpace(record.Provider) == "" {
			record.Provider = "codex"
		}
		if strings.TrimSpace(record.ID) == "" {
			record.ID = "current"
		}
		record.ProviderName = providerDisplayName(record.Provider)
		record.Key = sessionRecordKey(record.Provider, record.ID)
		record.CaptureHint = sessionCaptureHint(record.Provider, record.ID)
		record.RootsChecked = providerRoots(record.Provider)
		if record.Status == "" {
			record.Status = "reference-only"
		}
		if record.LocalSources == nil {
			record.LocalSources = []uiSessionSource{}
		}
		if record.Key == "" || seen[record.Key] {
			return
		}
		seen[record.Key] = true
		records = append(records, record)
	}

	for _, artifact := range snapshot.Artifacts {
		if artifact.Type != "conversation" {
			continue
		}
		provider, id := conversationArtifactSession(artifact)
		transcript := artifact.Path
		hasTranscript := transcript != "" && fileExists(filepath.FromSlash(transcript))
		add(uiSessionRecord{
			Provider:      provider,
			ID:            id,
			Title:         artifact.Description,
			Transcript:    transcript,
			URI:           fallback(artifact.URI, artifact.URL),
			Source:        fallback(artifact.Path, fallback(artifact.URL, artifact.URI)),
			Sanitized:     true,
			Format:        strings.TrimPrefix(strings.ToLower(filepath.Ext(transcript)), "."),
			HasTranscript: hasTranscript,
			Status:        mapBool(hasTranscript, "transcript", "reference-only"),
			Preview:       previewSessionFile(filepath.FromSlash(transcript)),
		})
	}

	for _, session := range server.legacySessionsForSnapshot(repo, snapshot) {
		add(uiSessionRecord{
			Provider: session.Provider,
			ID:       session.ID,
			URI:      session.URI,
			Status:   "reference-only",
		})
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Key < records[j].Key
	})
	return records
}

func (server runtimeServer) legacySessionsForSnapshot(repo repository, snapshot *eve.Snapshot) []legacySession {
	entries, err := os.ReadDir(filepath.Join(repo.eveDir, "evolutions"))
	if err != nil {
		return nil
	}
	commits := map[string]bool{}
	if snapshot.Implementation.GitState != "" {
		commits[snapshot.Implementation.GitState] = true
	}
	for _, commit := range snapshot.Implementation.Commits {
		if strings.TrimSpace(commit) != "" {
			commits[commit] = true
		}
	}
	var sessions []legacySession
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repo.eveDir, "evolutions", entry.Name()))
		if err != nil {
			continue
		}
		var evolution legacyEvolution
		if err := json.Unmarshal(data, &evolution); err != nil {
			continue
		}
		if !legacyEvolutionMatchesSnapshot(evolution, commits) {
			continue
		}
		sessions = append(sessions, evolution.Sessions...)
	}
	return sessions
}

func legacyEvolutionMatchesSnapshot(evolution legacyEvolution, commits map[string]bool) bool {
	if commits[evolution.Implementation.Snapshot] {
		return true
	}
	for _, commit := range evolution.Implementation.Commits {
		if commits[commit] {
			return true
		}
	}
	return false
}

func conversationArtifactSession(artifact eve.Artifact) (string, string) {
	text := strings.ToLower(strings.Join([]string{artifact.Description, artifact.Path, artifact.URI, artifact.URL}, " "))
	provider := "codex"
	switch {
	case strings.Contains(text, "claude"):
		provider = "claude"
	case strings.Contains(text, "opencode"):
		provider = "opencode"
	}
	id := strings.TrimSuffix(filepath.Base(firstNonEmpty(artifact.Path, artifact.URI, artifact.URL)), filepath.Ext(firstNonEmpty(artifact.Path, artifact.URI, artifact.URL)))
	if id == "" {
		id = "conversation"
	}
	return provider, id
}

func (server runtimeServer) gitCommits(repo repository, commits []string) []uiGitCommit {
	seen := map[string]bool{}
	ordered := make([]string, 0, len(commits))
	add := func(commit string) {
		commit = strings.TrimSpace(commit)
		if commit == "" || seen[commit] {
			return
		}
		seen[commit] = true
		ordered = append(ordered, commit)
	}
	for _, commit := range commits {
		add(commit)
	}

	out := make([]uiGitCommit, 0, len(ordered))
	for _, commit := range ordered {
		if info, err := server.gitCommit(repo, commit); err == nil {
			out = append(out, info)
			continue
		}
		out = append(out, uiGitCommit{
			Hash:      commit,
			ShortHash: shortHash(commit),
			Subject:   "Commit metadata unavailable",
		})
	}
	return out
}

func snapshotCodeRange(repo repository, snapshot *eve.Snapshot) (string, string) {
	if snapshot == nil {
		return "", ""
	}
	head := strings.TrimSpace(snapshot.Implementation.GitState)
	base := strings.TrimSpace(snapshot.Implementation.BaseCommit)
	if base == "" {
		base = latestCommittedEVEChangeBefore(repo, head)
	}
	if base == "" && head != "" {
		base = firstParentCommit(repo, head)
	}
	if base == "" {
		base = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	}
	return base, head
}

func firstParentCommit(repo repository, ref string) string {
	commit, err := gitOutput(repo.Root, "rev-parse", ref+"^")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(commit)
}

func snapshotChangedCodeFiles(repo repository, snapshot *eve.Snapshot, base string, head string) ([]snapshotCodeFile, error) {
	if strings.TrimSpace(head) == "" {
		return nil, fmt.Errorf("snapshot git state is not recorded")
	}
	output, err := gitOutput(repo.Root, "diff", "--name-status", "--find-renames", base, head, "--")
	if err != nil {
		return nil, fmt.Errorf("git diff name-status: %w", err)
	}
	haystack, evidenceLabel := snapshotCodeEvidence(snapshot)
	files := make([]snapshotCodeFile, 0)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		file, ok := parseSnapshotCodeNameStatus(line)
		if !ok {
			continue
		}
		size := snapshotCodeFileSize(repo, head, file.Path)
		binary := snapshotCodeFileIsBinary(repo, base, head, file.Path)
		file.Language = languageFromPath(file.Path)
		file.SizeBytes = size
		file.Previewable = true
		if file.Status == "D" {
			file.Previewable = false
			file.Reason = "File was deleted in this Snapshot."
		} else if binary {
			file.Previewable = false
			file.Reason = "Binary file preview is not available."
		} else if size > snapshotCodePreviewLimit {
			file.Previewable = false
			file.Reason = "File too large to preview. Open in editor or GitHub."
		}
		if snapshotCodePathReferenced(haystack, file.Path) || (file.OldPath != "" && snapshotCodePathReferenced(haystack, file.OldPath)) {
			file.Curated = true
			file.Evidence = "Evidence for: " + evidenceLabel
		}
		files = append(files, file)
	}
	sort.SliceStable(files, func(i, j int) bool {
		if files[i].Curated != files[j].Curated {
			return files[i].Curated
		}
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func parseSnapshotCodeNameStatus(line string) (snapshotCodeFile, bool) {
	fields := strings.Split(line, "\t")
	if len(fields) < 2 {
		return snapshotCodeFile{}, false
	}
	status := fields[0]
	code := status[:1]
	file := snapshotCodeFile{Status: code}
	if code == "R" || code == "C" {
		if len(fields) < 3 {
			return snapshotCodeFile{}, false
		}
		oldPath, err := cleanSnapshotCodePath(fields[1])
		if err != nil {
			return snapshotCodeFile{}, false
		}
		newPath, err := cleanSnapshotCodePath(fields[2])
		if err != nil {
			return snapshotCodeFile{}, false
		}
		file.OldPath = oldPath
		file.Path = newPath
		return file, true
	}
	cleaned, err := cleanSnapshotCodePath(fields[1])
	if err != nil {
		return snapshotCodeFile{}, false
	}
	file.Path = cleaned
	return file, true
}

func cleanSnapshotCodePath(value string) (string, error) {
	trimmed := strings.TrimSpace(filepath.ToSlash(value))
	if trimmed == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.HasPrefix(trimmed, "/") || strings.Contains(trimmed, "\x00") {
		return "", fmt.Errorf("invalid path")
	}
	for _, segment := range strings.Split(trimmed, "/") {
		if segment == ".." {
			return "", fmt.Errorf("invalid path")
		}
	}
	cleaned := strings.TrimPrefix(path.Clean("/"+trimmed), "/")
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("invalid path")
	}
	return cleaned, nil
}

func snapshotCodeEvidence(snapshot *eve.Snapshot) (string, string) {
	if snapshot == nil {
		return "", "Snapshot evidence"
	}
	values := []string{snapshot.Title, snapshot.Summary, snapshot.UserVisibleChange}
	for _, validation := range snapshot.Validation {
		values = append(values, validation.Command, validation.Output)
	}
	for _, decision := range snapshot.Decisions {
		values = append(values, decision.Title, decision.Rationale)
	}
	for _, risk := range snapshot.Risks {
		values = append(values, risk.Title, risk.Mitigation)
	}
	for _, entry := range snapshot.Timeline {
		values = append(values, entry.Title, entry.Summary)
	}
	for _, artifact := range snapshot.Artifacts {
		values = append(values, artifact.Path, artifact.URI, artifact.URL, artifact.Description)
	}
	label := firstNonEmpty(snapshot.UserVisibleChange, snapshot.Summary, snapshot.Title)
	if label == "" {
		label = "Snapshot evidence"
	}
	return strings.Join(values, "\n"), label
}

func snapshotCodePathReferenced(haystack string, filePath string) bool {
	if strings.TrimSpace(haystack) == "" {
		return false
	}
	return strings.Contains(haystack, filePath) || strings.Contains(haystack, filepath.Base(filePath))
}

func snapshotCodeFileSize(repo repository, head string, filePath string) int64 {
	output, err := gitOutput(repo.Root, "cat-file", "-s", head+":"+filePath)
	if err != nil {
		return 0
	}
	size, err := strconv.ParseInt(strings.TrimSpace(output), 10, 64)
	if err != nil {
		return 0
	}
	return size
}

func snapshotCodeFileIsBinary(repo repository, base string, head string, filePath string) bool {
	output, err := gitOutput(repo.Root, "diff", "--numstat", base, head, "--", filePath)
	if err == nil {
		for _, line := range strings.Split(output, "\n") {
			fields := strings.Fields(line)
			if len(fields) >= 2 && fields[0] == "-" && fields[1] == "-" {
				return true
			}
		}
	}
	data, err := gitBinaryOutput(repo.Root, "show", head+":"+filePath)
	if err != nil {
		return false
	}
	return bytesLookBinary(data)
}

func snapshotFileDiff(repo repository, base string, head string, filePath string) (string, error) {
	output, err := gitOutput(repo.Root, "diff", "--unified=3", "--no-ext-diff", base, head, "--", filePath)
	if err != nil {
		return "", fmt.Errorf("git diff file: %w", err)
	}
	return diffHunksOnly(output), nil
}

func snapshotFileAtGitState(repo repository, head string, filePath string) (string, error) {
	data, err := gitBinaryOutput(repo.Root, "show", head+":"+filePath)
	if err != nil {
		return "", fmt.Errorf("read file at snapshot git state: %w", err)
	}
	if bytesLookBinary(data) {
		return "", fmt.Errorf("binary file preview is not available")
	}
	return string(data), nil
}

func gitBinaryOutput(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	return cmd.Output()
}

func bytesLookBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	sample := data
	if len(sample) > 8000 {
		sample = sample[:8000]
	}
	if strings.Contains(string(sample), "\x00") {
		return true
	}
	return !utf8.Valid(sample)
}

func diffHunksOnly(diff string) string {
	lines := strings.Split(diff, "\n")
	hunks := make([]string, 0, len(lines))
	inHunk := false
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			inHunk = true
		}
		if inHunk {
			hunks = append(hunks, line)
		}
	}
	return strings.TrimRight(strings.Join(hunks, "\n"), "\n")
}

func languageFromPath(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	base := strings.ToLower(filepath.Base(filePath))
	switch base {
	case "dockerfile":
		return "dockerfile"
	case "makefile":
		return "makefile"
	case "go.mod", "go.sum":
		return "go"
	case "package.json", "tsconfig.json":
		return "json"
	}
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "tsx"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".json":
		return "json"
	case ".css":
		return "css"
	case ".html":
		return "html"
	case ".md", ".mdx":
		return "markdown"
	case ".yml", ".yaml":
		return "yaml"
	case ".sh", ".bash", ".zsh":
		return "bash"
	case ".py":
		return "python"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".sql":
		return "sql"
	case ".xml":
		return "xml"
	default:
		return "text"
	}
}

func snapshotImplementationCommits(repo repository, snapshot *eve.Snapshot) []string {
	if snapshot == nil {
		return nil
	}
	head := strings.TrimSpace(snapshot.Implementation.GitState)
	if head == "" {
		return normalizedCommitList(snapshot.Implementation.Commits)
	}
	baseCommit := strings.TrimSpace(snapshot.Implementation.BaseCommit)
	if baseCommit == "" {
		baseCommit = latestCommittedEVEChangeBefore(repo, head)
	}
	commits, err := implementationCommits(repo, baseCommit, head)
	if err == nil {
		return commits
	}
	return normalizedCommitList(snapshot.Implementation.Commits)
}

func latestCommittedEVEChangeBefore(repo repository, ref string) string {
	commit, err := gitOutput(repo.Root, "log", "-n", "1", "--format=%H", ref, "--", ".eve")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(commit)
}

func normalizedCommitList(commits []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(commits))
	for _, commit := range commits {
		commit = strings.TrimSpace(commit)
		if commit == "" || seen[commit] {
			continue
		}
		seen[commit] = true
		out = append(out, commit)
	}
	return out
}

func (server runtimeServer) gitCommit(repo repository, commit string) (uiGitCommit, error) {
	format := "%H%x00%h%x00%s%x00%an%x00%aI%x00%cI"
	cmd := exec.Command("git", "show", "-s", "--format="+format, commit)
	cmd.Dir = repo.Root
	output, err := cmd.Output()
	if err != nil {
		return uiGitCommit{}, err
	}
	parts := strings.Split(strings.TrimSpace(string(output)), "\x00")
	if len(parts) < 6 {
		return uiGitCommit{}, fmt.Errorf("unexpected git show output")
	}
	return uiGitCommit{
		Hash:        parts[0],
		ShortHash:   parts[1],
		Subject:     parts[2],
		AuthorName:  parts[3],
		AuthoredAt:  parts[4],
		CommittedAt: parts[5],
	}, nil
}

func providerInfos() []uiProviderInfo {
	providers := []string{"codex", "claude", "opencode"}
	infos := make([]uiProviderInfo, 0, len(providers))
	for _, provider := range providers {
		infos = append(infos, uiProviderInfo{
			Provider:      provider,
			Name:          providerDisplayName(provider),
			Roots:         providerRoots(provider),
			Available:     false,
			ImportCommand: sessionCaptureHint(provider, "<session-id>"),
			Displays:      []string{"session provider and id", "conversation artifacts"},
		})
	}
	return infos
}

func sessionCaptureHint(provider string, id string) string {
	return fmt.Sprintf("Attach a %s conversation artifact for %s.", fallback(provider, "provider"), fallback(id, "session-id"))
}

func providerDisplayName(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "codex":
		return "Codex"
	case "claude":
		return "Claude"
	case "opencode":
		return "OpenCode"
	default:
		return fallback(provider, "Other")
	}
}

func providerRoots(provider string) []string {
	return []string{}
}

func sessionRecordKey(provider string, id string) string {
	key := strings.ToLower(strings.TrimSpace(provider)) + "-" + strings.TrimSpace(id)
	key = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '-'
	}, key)
	return strings.Trim(key, "-")
}

func previewSessionFile(path string) uiSessionPreview {
	if strings.TrimSpace(path) == "" {
		return uiSessionPreview{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return uiSessionPreview{}
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return uiSessionPreview{}
	}
	lines := strings.Split(text, "\n")
	return uiSessionPreview{
		EventCount:   len(lines),
		MessageCount: len(lines),
	}
}

func shortHash(commit string) string {
	if len(commit) <= 12 {
		return commit
	}
	return commit[:12]
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func fallback(value string, fallbackValue string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallbackValue
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func mapBool(condition bool, truthy string, falsy string) string {
	if condition {
		return truthy
	}
	return falsy
}

func validationState(values []eve.Validation) string {
	if len(values) == 0 {
		return "skipped"
	}
	hasSkipped := false
	for _, value := range values {
		switch value.Status {
		case "failed":
			return "failed"
		case "skipped":
			hasSkipped = true
		}
	}
	if hasSkipped {
		return "skipped"
	}
	return "passed"
}

func failedValidationCount(values []eve.Validation) int {
	count := 0
	for _, value := range values {
		if strings.EqualFold(value.Status, "failed") {
			count++
		}
	}
	return count
}

func spaHandler() http.Handler {
	sub, err := fs.Sub(embeddedUI, "ui_dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if clean == "." || clean == "" {
			clean = "index.html"
		}
		if _, err := fs.Stat(sub, clean); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (server runtimeServer) handleMCPHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err)
		return
	}
	response := server.handleMCPMessage(r.Context(), body)
	if len(response) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(response)
}

func (server runtimeServer) handleMCPMessage(ctx context.Context, data []byte) []byte {
	var req mcpRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return marshalMCP(mcpResponse{JSONRPC: "2.0", Error: &mcpError{Code: -32700, Message: err.Error()}})
	}
	if len(req.ID) == 0 {
		return nil
	}
	result, rpcErr := server.dispatchMCP(ctx, req)
	if rpcErr != nil {
		return marshalMCP(mcpResponse{JSONRPC: "2.0", ID: req.ID, Error: rpcErr})
	}
	return marshalMCP(mcpResponse{JSONRPC: "2.0", ID: req.ID, Result: result})
}

func (server runtimeServer) dispatchMCP(ctx context.Context, req mcpRequest) (any, *mcpError) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": "2025-06-18",
			"serverInfo": map[string]string{
				"name":    "eve",
				"version": eve.CLIVersion,
			},
			"capabilities": map[string]any{
				"tools":     map[string]bool{"listChanged": false},
				"resources": map[string]bool{"listChanged": false},
			},
		}, nil
	case "tools/list":
		return map[string]any{"tools": mcpTools()}, nil
	case "tools/call":
		return server.callMCPTool(ctx, req.Params)
	case "resources/list":
		resources, err := server.mcpResources()
		if err != nil {
			return nil, &mcpError{Code: -32000, Message: err.Error()}
		}
		return map[string]any{"resources": resources}, nil
	case "resources/read":
		return server.readMCPResource(req.Params)
	default:
		return nil, &mcpError{Code: -32601, Message: "method not found: " + req.Method}
	}
}

func mcpTools() []map[string]any {
	return []map[string]any{
		{"name": "list_repos", "description": "List repositories known to the local EVE runtime.", "inputSchema": objectSchema(nil, nil)},
		{"name": "list_snapshots", "description": "List snapshots for a repository.", "inputSchema": objectSchema(map[string]any{"cwd": stringSchema(), "repoId": stringSchema(), "type": enumSchema("feature", "bugfix", "experiment", "refactor", "release")}, nil)},
		{"name": "get_snapshot", "description": "Get one snapshot by id.", "inputSchema": objectSchema(map[string]any{"cwd": stringSchema(), "repoId": stringSchema(), "snapshotId": stringSchema()}, []string{"snapshotId"})},
		{"name": "pending_snapshot", "description": "Report unresolved committed work that needs a Snapshot or Skip.", "inputSchema": objectSchema(map[string]any{"cwd": stringSchema(), "repoId": stringSchema()}, nil)},
		{"name": "start_suite", "description": "Start an explicit repository-defined verification suite against the current committed HEAD.", "inputSchema": objectSchema(map[string]any{"cwd": stringSchema(), "repoId": stringSchema(), "commit": stringSchema(), "suite": stringSchema(), "actorClaim": stringSchema()}, nil)},
		{"name": "get_suite_run", "description": "Read progress and results for a verification suite run.", "inputSchema": objectSchema(map[string]any{"cwd": stringSchema(), "repoId": stringSchema(), "runId": stringSchema()}, []string{"runId"})},
		{"name": "cancel_suite", "description": "Cancel an in-progress verification suite run.", "inputSchema": objectSchema(map[string]any{"cwd": stringSchema(), "repoId": stringSchema(), "runId": stringSchema()}, []string{"runId"})},
		{"name": "complete_snapshot", "description": "Create a completed product Snapshot after implementation changes are committed. This writes .eve files only; after it succeeds, stage and Git commit the generated .eve record separately.", "inputSchema": completeSnapshotSchema()},
		{"name": "skip_snapshot", "description": "Record that the current task does not deserve product history.", "inputSchema": objectSchema(map[string]any{"cwd": stringSchema(), "repoId": stringSchema(), "reason": stringSchema()}, []string{"reason"})},
		{"name": "checkout_snapshot", "description": "Checkout the Git state for a Snapshot.", "inputSchema": objectSchema(map[string]any{"cwd": stringSchema(), "repoId": stringSchema(), "snapshotId": stringSchema(), "force": map[string]string{"type": "boolean"}}, []string{"snapshotId"})},
	}
}

func completeSnapshotSchema() map[string]any {
	return objectSchema(map[string]any{
		"cwd":               stringSchema(),
		"repoId":            stringSchema(),
		"title":             stringSchema(),
		"type":              enumSchema("feature", "bugfix", "experiment", "refactor", "release"),
		"summary":           stringSchema(),
		"userVisibleChange": stringSchema(),
		"relationships": objectSchema(map[string]any{
			"corrects":   stringArraySchema(),
			"supersedes": stringArraySchema(),
			"reverts":    stringArraySchema(),
			"dependsOn":  stringArraySchema(),
			"related":    stringArraySchema(),
		}, nil),
		"risks": arraySchema(objectSchema(map[string]any{
			"title":      stringSchema(),
			"severity":   enumSchema("low", "medium", "high"),
			"mitigation": stringSchema(),
		}, []string{"title", "severity"})),
		"timeline": arraySchema(objectSchema(map[string]any{
			"phase":      enumSchema("planning", "implementation", "validation", "review", "release"),
			"title":      stringSchema(),
			"summary":    stringSchema(),
			"occurredAt": stringSchema(),
		}, []string{"phase", "title"})),
		"decisions": arraySchema(objectSchema(map[string]any{
			"title":     stringSchema(),
			"rationale": stringSchema(),
		}, []string{"title"})),
		"validation": arraySchema(objectSchema(map[string]any{
			"command": stringSchema(),
			"status":  enumSchema("passed", "failed", "skipped"),
			"output":  stringSchema(),
		}, []string{"command", "status"})),
		"artifacts": arraySchema(objectSchema(map[string]any{
			"type":        enumSchema("screenshot", "video", "preview", "url", "note", "log", "conversation"),
			"uri":         stringSchema(),
			"path":        stringSchema(),
			"url":         stringSchema(),
			"mimeType":    stringSchema(),
			"description": stringSchema(),
		}, []string{"type"})),
		"allowDirty": boolSchema(),
	}, []string{"title", "type", "summary"})
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	if properties == nil {
		properties = map[string]any{}
	}
	schema := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema() map[string]string { return map[string]string{"type": "string"} }

func boolSchema() map[string]string { return map[string]string{"type": "boolean"} }

func arraySchema(items any) map[string]any {
	return map[string]any{"type": "array", "items": items}
}

func stringArraySchema() map[string]any {
	return arraySchema(stringSchema())
}

func enumSchema(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}

func (server runtimeServer) callMCPTool(ctx context.Context, params json.RawMessage) (any, *mcpError) {
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, &mcpError{Code: -32602, Message: err.Error()}
	}
	switch call.Name {
	case "list_repos":
		var rows []repoSummary
		for _, repo := range server.repositories() {
			summary, err := repo.summary()
			if err != nil {
				if repo.Root == server.repo.Root {
					return toolError(err.Error()), nil
				}
				continue
			}
			rows = append(rows, summary)
		}
		return toolResult(rows), nil
	case "list_snapshots":
		var input struct {
			CWD    string `json:"cwd"`
			RepoID string `json:"repoId"`
			Type   string `json:"type"`
		}
		_ = json.Unmarshal(call.Arguments, &input)
		repo, err := server.resolveToolRepo(input.CWD, input.RepoID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		snapshots, err := repo.listSnapshots(input.Type)
		if err != nil {
			return toolError(err.Error()), nil
		}
		rows := make([]snapshotSummary, 0, len(snapshots))
		for _, snapshot := range snapshots {
			rows = append(rows, summarizeSnapshot(snapshot))
		}
		return toolResult(rows), nil
	case "get_snapshot":
		var input struct {
			CWD        string `json:"cwd"`
			RepoID     string `json:"repoId"`
			SnapshotID string `json:"snapshotId"`
		}
		if err := json.Unmarshal(call.Arguments, &input); err != nil {
			return nil, &mcpError{Code: -32602, Message: err.Error()}
		}
		repo, err := server.resolveToolRepo(input.CWD, input.RepoID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		snapshot, err := repo.loadSnapshot(input.SnapshotID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(snapshot), nil
	case "pending_snapshot":
		var input struct {
			CWD    string `json:"cwd"`
			RepoID string `json:"repoId"`
		}
		if err := json.Unmarshal(call.Arguments, &input); err != nil {
			return nil, &mcpError{Code: -32602, Message: err.Error()}
		}
		repo, err := server.resolveToolRepo(input.CWD, input.RepoID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		pending, err := repo.detectPending(pendingOptions{Initialize: true, Now: time.Now().UTC()})
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(pendingSnapshotResponse{Pending: pending != nil, PendingSnapshot: pending}), nil
	case "start_suite":
		var input struct{ CWD, RepoID, Commit, Suite, ActorClaim string }
		if err := json.Unmarshal(call.Arguments, &input); err != nil {
			return nil, &mcpError{Code: -32602, Message: err.Error()}
		}
		repo, err := server.resolveToolRepo(input.CWD, input.RepoID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		run, err := server.startVerificationRun(ctx, repo, input.Commit, input.Suite, input.ActorClaim)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(run), nil
	case "get_suite_run":
		var input struct{ CWD, RepoID, RunID string }
		if err := json.Unmarshal(call.Arguments, &input); err != nil {
			return nil, &mcpError{Code: -32602, Message: err.Error()}
		}
		repo, err := server.resolveToolRepo(input.CWD, input.RepoID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		run, err := server.verificationRun(repo, input.RunID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(run), nil
	case "cancel_suite":
		var input struct{ CWD, RepoID, RunID string }
		if err := json.Unmarshal(call.Arguments, &input); err != nil {
			return nil, &mcpError{Code: -32602, Message: err.Error()}
		}
		repo, err := server.resolveToolRepo(input.CWD, input.RepoID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		run, err := server.verificationRun(repo, input.RunID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		if run.Status == "running" || run.Status == "queued" {
			if err := repo.requestVerificationCancellation(input.RunID); err != nil {
				return toolError(err.Error()), nil
			}
			server.verificationRegistry.mu.RLock()
			cancel := server.verificationRegistry.cancels[input.RunID]
			server.verificationRegistry.mu.RUnlock()
			if cancel != nil {
				cancel()
			}
			deadline := time.Now().Add(3500 * time.Millisecond)
			for time.Now().Before(deadline) {
				current, readErr := server.verificationRun(repo, input.RunID)
				if readErr == nil {
					run = current
					if run.Status != "running" && run.Status != "queued" {
						break
					}
				}
				time.Sleep(10 * time.Millisecond)
			}
			if run.Status == "running" || run.Status == "queued" {
				return toolError("verification cancellation was not acknowledged by the owning process"), nil
			}
		}
		return toolResult(run), nil
	case "complete_snapshot":
		return server.completeSnapshotTool(ctx, call.Arguments)
	case "skip_snapshot":
		var input struct {
			CWD      string `json:"cwd"`
			RepoID   string `json:"repoId"`
			Reason   string `json:"reason"`
			Provider string `json:"provider"`
			AgentID  string `json:"agentId"`
		}
		if err := json.Unmarshal(call.Arguments, &input); err != nil {
			return nil, &mcpError{Code: -32602, Message: err.Error()}
		}
		if strings.TrimSpace(input.Reason) == "" {
			return toolError("skip reason is required"), nil
		}
		repo, err := server.resolveToolRepo(input.CWD, input.RepoID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		record, err := repo.createSkip(input.Reason, skipAgent{Provider: input.Provider, ID: input.AgentID})
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(record), nil
	case "checkout_snapshot":
		var input struct {
			CWD        string `json:"cwd"`
			RepoID     string `json:"repoId"`
			SnapshotID string `json:"snapshotId"`
			Force      bool   `json:"force"`
		}
		if err := json.Unmarshal(call.Arguments, &input); err != nil {
			return nil, &mcpError{Code: -32602, Message: err.Error()}
		}
		repo, err := server.resolveToolRepo(input.CWD, input.RepoID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		snapshot, err := repo.loadSnapshot(input.SnapshotID)
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult(checkoutSnapshot(repo, snapshot, input.Force)), nil
	default:
		return nil, &mcpError{Code: -32602, Message: "unknown tool: " + call.Name}
	}
}

func (server runtimeServer) completeSnapshotTool(ctx context.Context, args json.RawMessage) (any, *mcpError) {
	var raw completeSnapshotInputRaw
	if err := json.Unmarshal(args, &raw); err != nil {
		return nil, &mcpError{Code: -32602, Message: err.Error()}
	}
	input, err := normalizeCompleteSnapshotInput(raw)
	if err != nil {
		return toolError(err.Error()), nil
	}
	repo, err := server.resolveToolRepo(input.CWD, input.RepoID)
	if err != nil {
		return toolError(err.Error()), nil
	}
	snapshot, err := completeSnapshot(repo, input, nil)
	if err != nil {
		return toolError(err.Error()), nil
	}
	return toolResult(snapshot), nil
}

type completeSnapshotInputRaw struct {
	CWD               string          `json:"cwd"`
	RepoID            string          `json:"repoId"`
	Title             string          `json:"title"`
	Type              string          `json:"type"`
	Summary           string          `json:"summary"`
	UserVisibleChange string          `json:"userVisibleChange"`
	Relationships     json.RawMessage `json:"relationships"`
	Risks             json.RawMessage `json:"risks"`
	Timeline          json.RawMessage `json:"timeline"`
	Decisions         json.RawMessage `json:"decisions"`
	Validation        json.RawMessage `json:"validation"`
	Artifacts         json.RawMessage `json:"artifacts"`
	AllowDirty        bool            `json:"allowDirty"`
}

type completeSnapshotInput struct {
	CWD               string
	RepoID            string
	Title             string
	Type              string
	Summary           string
	UserVisibleChange string
	Relationships     eve.Relationships
	Risks             []eve.Risk
	Timeline          []eve.TimelineEntry
	Decisions         []eve.Decision
	Validation        []eve.Validation
	Artifacts         []eve.Artifact
	AllowDirty        bool
}

func normalizeCompleteSnapshotInput(raw completeSnapshotInputRaw) (completeSnapshotInput, error) {
	relationships, err := decodeOptionalObject[eve.Relationships](raw.Relationships)
	if err != nil {
		return completeSnapshotInput{}, fmt.Errorf("relationships: %w", err)
	}
	risks, err := decodeFlexibleArray(raw.Risks, riskFromString, normalizeRisk)
	if err != nil {
		return completeSnapshotInput{}, fmt.Errorf("risks: %w", err)
	}
	timeline, err := decodeFlexibleArray(raw.Timeline, timelineFromString, normalizeTimelineEntry)
	if err != nil {
		return completeSnapshotInput{}, fmt.Errorf("timeline: %w", err)
	}
	decisions, err := decodeFlexibleArray(raw.Decisions, decisionFromString, normalizeDecision)
	if err != nil {
		return completeSnapshotInput{}, fmt.Errorf("decisions: %w", err)
	}
	validation, err := decodeFlexibleArray(raw.Validation, validationFromString, normalizeValidation)
	if err != nil {
		return completeSnapshotInput{}, fmt.Errorf("validation: %w", err)
	}
	for i, claim := range validation {
		if claim.Provenance != "reported_by_agent" {
			return completeSnapshotInput{}, fmt.Errorf("validation[%d].provenance must be reported_by_agent", i)
		}
	}
	artifacts, err := decodeFlexibleArray(raw.Artifacts, artifactFromString, normalizeArtifact)
	if err != nil {
		return completeSnapshotInput{}, fmt.Errorf("artifacts: %w", err)
	}

	return completeSnapshotInput{
		CWD:               raw.CWD,
		RepoID:            raw.RepoID,
		Title:             raw.Title,
		Type:              raw.Type,
		Summary:           raw.Summary,
		UserVisibleChange: raw.UserVisibleChange,
		Relationships:     relationships,
		Risks:             risks,
		Timeline:          timeline,
		Decisions:         decisions,
		Validation:        validation,
		Artifacts:         artifacts,
		AllowDirty:        raw.AllowDirty,
	}, nil
}

func decodeOptionalObject[T any](raw json.RawMessage) (T, error) {
	var zero T
	if len(raw) == 0 || string(raw) == "null" {
		return zero, nil
	}
	if err := json.Unmarshal(raw, &zero); err != nil {
		return zero, err
	}
	return zero, nil
}

func decodeFlexibleArray[T any](raw json.RawMessage, fromString func(string) T, normalize func(T) T) ([]T, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var values []T
	if err := json.Unmarshal(raw, &values); err == nil {
		for i, value := range values {
			values[i] = normalize(value)
		}
		return values, nil
	}
	var stringValues []string
	if err := json.Unmarshal(raw, &stringValues); err != nil {
		return nil, err
	}
	values = make([]T, 0, len(stringValues))
	for _, value := range stringValues {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			values = append(values, fromString(trimmed))
		}
	}
	return values, nil
}

func riskFromString(value string) eve.Risk {
	return eve.Risk{Title: value, Severity: "medium"}
}

func normalizeRisk(risk eve.Risk) eve.Risk {
	if strings.TrimSpace(risk.Severity) == "" {
		risk.Severity = "medium"
	}
	return risk
}

func timelineFromString(value string) eve.TimelineEntry {
	return eve.TimelineEntry{Phase: "implementation", Title: value}
}

func normalizeTimelineEntry(entry eve.TimelineEntry) eve.TimelineEntry {
	if strings.TrimSpace(entry.Phase) == "" {
		entry.Phase = "implementation"
	}
	return entry
}

func decisionFromString(value string) eve.Decision {
	return eve.Decision{Title: value}
}

func normalizeDecision(decision eve.Decision) eve.Decision {
	return decision
}

func validationFromString(value string) eve.Validation {
	return eve.Validation{Command: value, Status: "skipped", Provenance: "reported_by_agent"}
}

func normalizeValidation(validation eve.Validation) eve.Validation {
	if strings.TrimSpace(validation.Provenance) == "" {
		validation.Provenance = "reported_by_agent"
	}
	if strings.TrimSpace(validation.Status) == "" {
		validation.Status = "skipped"
	}
	return validation
}

func artifactFromString(value string) eve.Artifact {
	artifact := eve.Artifact{Type: "note", URI: value}
	switch {
	case strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://"):
		artifact.Type = "url"
		artifact.URL = value
		artifact.URI = ""
	case strings.HasPrefix(value, ".") || strings.HasPrefix(value, "/"):
		artifact.Path = value
		artifact.URI = ""
	}
	return artifact
}

func normalizeArtifact(artifact eve.Artifact) eve.Artifact {
	if strings.TrimSpace(artifact.Type) == "" {
		artifact.Type = "note"
	}
	return artifact
}

func (server runtimeServer) resolveToolRepo(cwd string, repoID string) (repository, error) {
	if strings.TrimSpace(repoID) == "" && strings.TrimSpace(cwd) == "" {
		return server.repo, nil
	}
	if strings.TrimSpace(repoID) == server.repo.ID {
		return server.repo, nil
	}
	if strings.TrimSpace(repoID) != "" {
		if repo, ok := server.repoByID(strings.TrimSpace(repoID)); ok {
			return repo, nil
		}
	}
	return resolveRepo(repoRequest{CWD: cwd, RepoID: repoID})
}

func toolResult(value any) map[string]any {
	data, _ := json.Marshal(value)
	return map[string]any{
		"content":           []map[string]string{{"type": "text", "text": string(data)}},
		"structuredContent": value,
		"isError":           false,
	}
}

func toolError(message string) map[string]any {
	return map[string]any{
		"content": []map[string]string{{"type": "text", "text": message}},
		"isError": true,
	}
}

func (server runtimeServer) mcpResources() ([]map[string]string, error) {
	resources := []map[string]string{
		{"uri": "eve://repos", "name": "repos", "title": "EVE repositories", "mimeType": "application/json"},
	}
	for _, repo := range server.repositories() {
		snapshots, err := repo.listSnapshots("")
		if err != nil {
			if repo.Root == server.repo.Root {
				return nil, err
			}
			continue
		}
		resources = append(resources,
			map[string]string{"uri": "eve://repos/" + repo.ID, "name": repo.ID, "title": repo.ID, "mimeType": "application/json"},
			map[string]string{"uri": "eve://repos/" + repo.ID + "/snapshots", "name": repo.ID + "-snapshots", "title": repo.ID + " Snapshots", "mimeType": "application/json"},
		)
		for _, snapshot := range snapshots {
			resources = append(resources, map[string]string{
				"uri":      "eve://repos/" + repo.ID + "/snapshots/" + snapshot.ID,
				"name":     snapshot.ID,
				"title":    snapshot.Title,
				"mimeType": "application/json",
			})
		}
	}
	return resources, nil
}

func (server runtimeServer) readMCPResource(params json.RawMessage) (any, *mcpError) {
	var input struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(params, &input); err != nil {
		return nil, &mcpError{Code: -32602, Message: err.Error()}
	}
	var value any
	switch input.URI {
	case "eve://repos":
		var rows []repoSummary
		for _, repo := range server.repositories() {
			summary, err := repo.summary()
			if err != nil {
				if repo.Root == server.repo.Root {
					return nil, &mcpError{Code: -32000, Message: err.Error()}
				}
				continue
			}
			rows = append(rows, summary)
		}
		value = rows
	default:
		const repoPrefix = "eve://repos/"
		if !strings.HasPrefix(input.URI, repoPrefix) {
			return nil, &mcpError{Code: -32602, Message: "unknown resource: " + input.URI}
		}
		rest := strings.TrimPrefix(input.URI, repoPrefix)
		parts := strings.Split(rest, "/")
		if len(parts) == 0 || parts[0] == "" {
			return nil, &mcpError{Code: -32602, Message: "unknown resource: " + input.URI}
		}
		repo, ok := server.repoByID(parts[0])
		if !ok {
			return nil, &mcpError{Code: -32602, Message: "unknown repository: " + parts[0]}
		}
		switch {
		case len(parts) == 1:
			summary, err := repo.summary()
			if err != nil {
				return nil, &mcpError{Code: -32000, Message: err.Error()}
			}
			value = summary
		case len(parts) == 2 && parts[1] == "snapshots":
			snapshots, err := repo.listSnapshots("")
			if err != nil {
				return nil, &mcpError{Code: -32000, Message: err.Error()}
			}
			rows := make([]snapshotSummary, 0, len(snapshots))
			for _, snapshot := range snapshots {
				rows = append(rows, summarizeSnapshot(snapshot))
			}
			value = rows
		case len(parts) == 3 && parts[1] == "snapshots":
			snapshot, err := repo.loadSnapshot(parts[2])
			if err != nil {
				return nil, &mcpError{Code: -32000, Message: err.Error()}
			}
			value = snapshot
		default:
			return nil, &mcpError{Code: -32602, Message: "unknown resource: " + input.URI}
		}
	}
	data, _ := json.Marshal(value)
	return map[string]any{
		"contents": []map[string]string{{
			"uri":      input.URI,
			"mimeType": "application/json",
			"text":     string(data),
		}},
	}, nil
}

func marshalMCP(response mcpResponse) []byte {
	data, _ := json.Marshal(response)
	return data
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, apiError{Error: err.Error()})
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeAPIError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
