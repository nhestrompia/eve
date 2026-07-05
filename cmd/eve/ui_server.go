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

	"github.com/nhestrompia/eve"
)

//go:embed ui_dist/* ui_dist/assets/*
var embeddedUI embed.FS

type runtimeServer struct {
	repo        repository
	addr        string
	searchCache *snapshotSearchCache
}

type apiError struct {
	Error string `json:"error"`
}

type configResponse struct {
	SnapshotSchemaVersion string `json:"snapshotSchemaVersion"`
	CLIVersion            string `json:"cliVersion"`
	Repository            string `json:"repository"`
	Addr                  string `json:"addr"`
	EveDir                string `json:"eveDir"`
	Initialized           bool   `json:"initialized"`
	CurrentGitState       string `json:"currentGitState,omitempty"`
	CurrentBranch         string `json:"currentBranch,omitempty"`
	CurrentDirty          bool   `json:"currentDirty"`
	LatestSnapshot        string `json:"latestSnapshot,omitempty"`
	LatestGitState        string `json:"latestGitState,omitempty"`
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
	return runtimeServer{repo: repo, addr: addr, searchCache: &snapshotSearchCache{}}
}

func (server runtimeServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", server.handleConfig)
	mux.HandleFunc("/api/search", server.handleSnapshotSearch)
	mux.HandleFunc("/api/snapshots", server.handleGlobalSnapshots)
	mux.HandleFunc("/api/snapshots/", server.handleGlobalSnapshotRoutes)
	mux.HandleFunc("/api/repos", server.handleRepos)
	mux.HandleFunc("/api/repos/", server.handleRepoRoutes)
	mux.HandleFunc("/mcp", server.handleMCPHTTP)
	mux.Handle("/", spaHandler())
	return logRequests(mux)
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
	case len(parts) == 2 && parts[1] == "open-editor" && r.Method == http.MethodPost:
		writeJSON(w, http.StatusOK, openRepositoryInEditor(repo))
	case len(parts) >= 3 && parts[1] == "artifacts" && r.Method == http.MethodGet:
		server.handleArtifactFile(w, r, repo, strings.Join(parts[2:], "/"))
	case len(parts) == 2 && parts[1] == "snapshots" && r.Method == http.MethodGet:
		server.handleSnapshots(w, r, repo)
	case len(parts) == 3 && parts[1] == "snapshots" && r.Method == http.MethodGet:
		server.handleSnapshotDetail(w, r, repo, parts[2])
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
	commits := server.gitCommits(repo, snapshot.Implementation.GitState, snapshot.Implementation.Commits)
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

func (server runtimeServer) loadSnapshotWithRaw(repo repository, id string) (*eve.Snapshot, json.RawMessage, error) {
	data, err := os.ReadFile(repo.snapshotPath(id))
	if err != nil {
		return nil, nil, err
	}
	snapshot, err := eve.ParseSnapshot(data)
	if err != nil {
		return nil, nil, err
	}
	return snapshot, json.RawMessage(data), nil
}

func summarizeSnapshot(snapshot *eve.Snapshot) snapshotSummary {
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
		ValidationState:       validationState(snapshot.Validation),
		CreatedAt:             snapshot.CreatedAt,
	}
}

func summarizeSnapshotForRepo(repo repository, snapshot *eve.Snapshot) snapshotSummary {
	summary := summarizeSnapshot(snapshot)
	summary.Repository = repo.ID
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

func (server runtimeServer) gitCommits(repo repository, snapshot string, commits []string) []uiGitCommit {
	seen := map[string]bool{}
	ordered := make([]string, 0, len(commits)+1)
	add := func(commit string) {
		commit = strings.TrimSpace(commit)
		if commit == "" || seen[commit] {
			return
		}
		seen[commit] = true
		ordered = append(ordered, commit)
	}
	add(snapshot)
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
	case "complete_snapshot":
		return server.completeSnapshotTool(ctx, call.Arguments)
	case "skip_snapshot":
		var input struct {
			Reason string `json:"reason"`
		}
		_ = json.Unmarshal(call.Arguments, &input)
		return toolResult(map[string]string{"status": "skipped", "reason": input.Reason}), nil
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
	return eve.Validation{Command: value, Status: inferValidationStatus(value)}
}

func normalizeValidation(validation eve.Validation) eve.Validation {
	if strings.TrimSpace(validation.Status) == "" {
		validation.Status = inferValidationStatus(validation.Command + " " + validation.Output)
	}
	return validation
}

func inferValidationStatus(value string) string {
	lowered := strings.ToLower(value)
	switch {
	case strings.Contains(lowered, "skip"):
		return "skipped"
	case strings.Contains(lowered, "fail"):
		return "failed"
	default:
		return "passed"
	}
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
