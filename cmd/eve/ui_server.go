package main

import (
	"bufio"
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nhestrompia/eve"
)

//go:embed ui_dist/* ui_dist/assets/*
var embeddedUI embed.FS

type uiServer struct {
	store localStore
	repo  string
	addr  string
}

type uiError struct {
	Error string `json:"error"`
}

type uiConfigResponse struct {
	Config          *configFile `json:"config,omitempty"`
	ProtocolVersion int         `json:"protocolVersion"`
	CLIVersion      string      `json:"cliVersion"`
	Repository      string      `json:"repository"`
	Addr            string      `json:"addr"`
	EveDir          string      `json:"eveDir"`
	Initialized     bool        `json:"initialized"`
}

type evolutionSummary struct {
	ID                  string   `json:"id"`
	Title               string   `json:"title"`
	Type                string   `json:"type"`
	Status              string   `json:"status"`
	Outcome             string   `json:"outcome"`
	Snapshot            string   `json:"snapshot"`
	VerificationState   string   `json:"verificationState"`
	VerificationSummary string   `json:"verificationSummary"`
	SessionProviders    []string `json:"sessionProviders"`
	CreatedAt           string   `json:"createdAt"`
	UpdatedAt           string   `json:"updatedAt"`
}

type repositorySummary struct {
	Name             string   `json:"name"`
	RemoteURL        string   `json:"remoteUrl,omitempty"`
	EvolutionCount   int      `json:"evolutionCount"`
	SnapshotCount    int      `json:"snapshotCount"`
	CommitCount      int      `json:"commitCount"`
	LatestAt         string   `json:"latestAt"`
	LatestEvolution  string   `json:"latestEvolution"`
	LatestTitle      string   `json:"latestTitle"`
	SessionProviders []string `json:"sessionProviders"`
}

type evolutionDetailResponse struct {
	Evolution *eve.Evolution    `json:"evolution"`
	Summary   evolutionSummary  `json:"summary"`
	Sessions  []uiSessionRecord `json:"sessions"`
	Providers []uiProviderInfo  `json:"providers"`
	Commits   []uiGitCommit     `json:"commits"`
	RawJSON   json.RawMessage   `json:"rawJson"`
}

type snapshotResponse struct {
	ID              string             `json:"id"`
	Title           string             `json:"title"`
	Outcome         string             `json:"outcome"`
	Behavior        eve.Behavior       `json:"behavior"`
	Verification    []eve.Verification `json:"verification"`
	Repository      string             `json:"repository"`
	Commit          string             `json:"commit"`
	CheckoutCommand string             `json:"checkoutCommand"`
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

type searchResponse struct {
	Query   string             `json:"query"`
	Results []searchResultItem `json:"results"`
}

type searchResultItem struct {
	Evolution evolutionSummary `json:"evolution"`
	Matches   []string         `json:"matches"`
}

type checkoutResponse struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
	Command    string `json:"command"`
	ExitCode   int    `json:"exitCode"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
}

func runUI(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("ui", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", "localhost:4317", "local UI listen address")
	repo := fs.String("repo", "", "repository name for snapshot resolution")
	openBrowser := fs.Bool("open", false, "open the UI in a browser")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve ui takes no positional arguments")
		return 2
	}

	server := newUIServer(newStore(), strings.TrimSpace(*repo), strings.TrimSpace(*addr))
	listener, err := net.Listen("tcp", server.addr)
	if err != nil {
		fmt.Fprintf(stderr, "start UI server: %v\n", err)
		return 1
	}
	defer listener.Close()

	actualAddr := listener.Addr().String()
	url := "http://" + actualAddr
	if strings.HasPrefix(actualAddr, "[::]") {
		url = "http://localhost" + strings.TrimPrefix(actualAddr, "[::]")
	}
	fmt.Fprintf(stdout, "EVE UI listening on %s\n", url)
	if *openBrowser {
		if err := openURL(url); err != nil {
			fmt.Fprintf(stderr, "open browser: %v\n", err)
		}
	}

	if err := http.Serve(listener, server.routes()); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintf(stderr, "serve UI: %v\n", err)
		return 1
	}
	return 0
}

func newUIServer(store localStore, repo string, addr string) uiServer {
	if strings.TrimSpace(addr) == "" {
		addr = "localhost:4317"
	}
	return uiServer{store: store, repo: strings.TrimSpace(repo), addr: addr}
}

func (server uiServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", server.handleConfig)
	mux.HandleFunc("/api/repositories", server.handleRepositories)
	mux.HandleFunc("/api/evolutions", server.handleEvolutions)
	mux.HandleFunc("/api/evolutions/", server.handleEvolutionRoutes)
	mux.HandleFunc("/api/search", server.handleSearch)
	mux.Handle("/", spaHandler())
	return logUIRequests(mux)
}

func (server uiServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	config, initialized := server.loadConfig()
	writeJSON(w, http.StatusOK, uiConfigResponse{
		Config:          config,
		ProtocolVersion: eve.ProtocolVersion,
		CLIVersion:      eve.CLIVersion,
		Repository:      server.repositoryName(),
		Addr:            server.addr,
		EveDir:          server.store.root,
		Initialized:     initialized,
	})
}

func (server uiServer) handleRepositories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	evolutions, err := server.store.loadAllCommitted()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, server.repositorySummaries(evolutions))
}

func (server uiServer) handleEvolutions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	evolutions, err := server.store.loadAllCommitted()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	repoFilter := strings.TrimSpace(r.URL.Query().Get("repo"))
	rows := make([]evolutionSummary, 0, len(evolutions))
	for _, evolution := range evolutions {
		if repoFilter != "" && !evolutionTouchesRepository(evolution, repoFilter, server.repositoryName()) {
			continue
		}
		rows = append(rows, summarizeEvolution(evolution))
	}
	sortEvolutionSummaries(rows)
	writeJSON(w, http.StatusOK, rows)
}

func (server uiServer) handleEvolutionRoutes(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/api/evolutions/")
	parts := splitPath(trimmed)
	if len(parts) == 0 || parts[0] == "" {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("evolution route not found"))
		return
	}
	id := parts[0]

	switch {
	case len(parts) == 1:
		server.handleEvolutionDetail(w, r, id)
	case len(parts) == 2 && parts[1] == "snapshot":
		server.handleSnapshot(w, r, id)
	case len(parts) == 2 && parts[1] == "sessions":
		server.handleSessions(w, r, id)
	case len(parts) == 4 && parts[1] == "sessions" && parts[2] != "":
		sessionKey := parts[2] + "/" + parts[3]
		server.handleSessionTranscript(w, r, id, sessionKey)
	case len(parts) == 3 && parts[1] == "sessions":
		server.handleSessionTranscript(w, r, id, parts[2])
	case len(parts) == 2 && parts[1] == "checkout":
		server.handleCheckout(w, r, id)
	default:
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("evolution route not found"))
	}
}

func (server uiServer) handleEvolutionDetail(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	evolution, raw, err := server.loadEvolutionWithRaw(id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, evolutionDetailResponse{
		Evolution: evolution,
		Summary:   summarizeEvolution(evolution),
		Sessions:  server.sessionRecords(evolution),
		Providers: providerInfos(),
		Commits:   gitCommits(evolution.Implementation.Commits),
		RawJSON:   raw,
	})
}

func (server uiServer) handleSnapshot(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	evolution, err := server.store.loadCommitted(id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	target, err := resolveSnapshotTarget(evolution, server.repo)
	if err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshotResponse{
		ID:              evolution.Metadata.ID,
		Title:           evolution.Metadata.Title,
		Outcome:         evolution.Outcome,
		Behavior:        evolution.Behavior,
		Verification:    evolution.Verification,
		Repository:      target.Repository,
		Commit:          target.Commit,
		CheckoutCommand: checkoutCommand(evolution.Metadata.ID, server.repo),
	})
}

func (server uiServer) handleSessions(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	evolution, err := server.store.loadCommitted(id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, sessionListResponse{
		EvolutionID: evolution.Metadata.ID,
		Sessions:    server.sessionRecords(evolution),
		Providers:   providerInfos(),
	})
}

func (server uiServer) handleSessionTranscript(w http.ResponseWriter, r *http.Request, id string, sessionKey string) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	evolution, err := server.store.loadCommitted(id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	key := strings.TrimSpace(sessionKey)
	for _, session := range server.sessionRecords(evolution) {
		if session.Key != key {
			continue
		}
		if !session.HasTranscript {
			if len(session.LocalSources) == 0 {
				writeAPIError(w, http.StatusNotFound, fmt.Errorf("session transcript is not available"))
				return
			}
			source := session.LocalSources[0]
			raw, err := os.ReadFile(filepath.FromSlash(source.Path))
			if err != nil {
				writeAPIError(w, http.StatusNotFound, fmt.Errorf("read local session candidate: %w", err))
				return
			}
			raw = sanitizeSession(raw)
			markdown := renderSessionMarkdown(session.Provider, session.ID, fallback(source.Title, session.Provider+" "+session.ID), source.Format, raw, true, source.Path)
			writeJSON(w, http.StatusOK, sessionTranscriptResponse{
				EvolutionID: evolution.Metadata.ID,
				Provider:    session.Provider,
				ID:          session.ID,
				Key:         session.Key,
				Title:       fallback(source.Title, session.Provider+" "+session.ID),
				Markdown:    string(markdown),
				Sanitized:   true,
			})
			return
		}
		var markdown []byte
		if strings.TrimSpace(session.Raw) != "" && fileExists(filepath.FromSlash(session.Raw)) {
			raw, err := os.ReadFile(filepath.FromSlash(session.Raw))
			if err != nil {
				writeAPIError(w, http.StatusNotFound, fmt.Errorf("read raw session artifact: %w", err))
				return
			}
			rawFormat := session.Metadata["raw_format"]
			if rawFormat == "" {
				rawFormat = detectRawFormat(session.Source, raw)
			}
			markdown = renderSessionMarkdown(session.Provider, session.ID, fallback(session.Title, session.Provider+" "+session.ID), rawFormat, raw, session.Sanitized, session.Source)
		} else {
			var err error
			markdown, err = os.ReadFile(filepath.FromSlash(session.Transcript))
			if err != nil {
				writeAPIError(w, http.StatusNotFound, fmt.Errorf("read session transcript: %w", err))
				return
			}
		}
		writeJSON(w, http.StatusOK, sessionTranscriptResponse{
			EvolutionID: evolution.Metadata.ID,
			Provider:    session.Provider,
			ID:          session.ID,
			Key:         session.Key,
			Title:       fallback(session.Title, session.Provider+" "+session.ID),
			Markdown:    string(markdown),
			Sanitized:   session.Sanitized,
		})
		return
	}
	writeAPIError(w, http.StatusNotFound, fmt.Errorf("session %s not found", key))
}

func (server uiServer) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if query == "" {
		writeJSON(w, http.StatusOK, searchResponse{Query: "", Results: []searchResultItem{}})
		return
	}
	evolutions, err := server.store.loadAllCommitted()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	var results []searchResultItem
	for _, evolution := range evolutions {
		matches := server.searchMatches(evolution, query)
		if len(matches) == 0 {
			continue
		}
		results = append(results, searchResultItem{
			Evolution: summarizeEvolution(evolution),
			Matches:   matches,
		})
	}
	writeJSON(w, http.StatusOK, searchResponse{Query: query, Results: results})
}

func (server uiServer) handleCheckout(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	evolution, err := server.store.loadCommitted(id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	target, err := resolveSnapshotTarget(evolution, server.repo)
	if err != nil {
		writeAPIError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if dirty, err := workingTreeDirty(); err != nil {
		writeJSON(w, http.StatusOK, checkoutResponse{
			ID:         evolution.Metadata.ID,
			Title:      evolution.Metadata.Title,
			Repository: target.Repository,
			Commit:     target.Commit,
			Command:    checkoutCommand(evolution.Metadata.ID, server.repo),
			ExitCode:   1,
			Stderr:     "check working tree: " + err.Error(),
		})
		return
	} else if dirty {
		writeJSON(w, http.StatusOK, checkoutResponse{
			ID:         evolution.Metadata.ID,
			Title:      evolution.Metadata.Title,
			Repository: target.Repository,
			Commit:     target.Commit,
			Command:    checkoutCommand(evolution.Metadata.ID, server.repo),
			ExitCode:   1,
			Stderr:     fmt.Sprintf("Working tree has uncommitted changes.\nCommit or stash them before checking out %s.\n", evolution.Metadata.ID),
		})
		return
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("git", "checkout", target.Commit)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	exitCode := 0
	if err := cmd.Run(); err != nil {
		exitCode = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		if stderr.Len() == 0 {
			fmt.Fprintf(&stderr, "git checkout %s: %v", target.Commit, err)
		}
	}
	writeJSON(w, http.StatusOK, checkoutResponse{
		ID:         evolution.Metadata.ID,
		Title:      evolution.Metadata.Title,
		Repository: target.Repository,
		Commit:     target.Commit,
		Command:    checkoutCommand(evolution.Metadata.ID, server.repo),
		ExitCode:   exitCode,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
	})
}

func (server uiServer) loadConfig() (*configFile, bool) {
	data, err := os.ReadFile(server.store.configPath())
	if err != nil {
		return nil, false
	}
	var config configFile
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, true
	}
	return &config, true
}

func (server uiServer) loadEvolutionWithRaw(id string) (*eve.Evolution, json.RawMessage, error) {
	path := server.store.evolutionPath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("load evolution %s: %w", id, err)
	}
	evolution, err := eve.Parse(data)
	if err != nil {
		return nil, nil, err
	}
	return evolution, json.RawMessage(data), nil
}

func (server uiServer) repositoryName() string {
	if server.repo != "" {
		return server.repo
	}
	return currentRepositoryName()
}

func (server uiServer) repositorySummaries(evolutions []*eve.Evolution) []repositorySummary {
	byName := map[string]*repositorySummary{}
	current := server.repositoryName()
	ensure := func(name string) *repositorySummary {
		name = strings.TrimSpace(name)
		if name == "" {
			name = current
		}
		if byName[name] == nil {
			byName[name] = &repositorySummary{Name: name}
			if name == current {
				byName[name].RemoteURL = gitRemoteURL()
			}
		}
		return byName[name]
	}
	ensure(current)
	providerSets := map[string]map[string]bool{}
	for _, evolution := range evolutions {
		repos := evolutionRepositoryNames(evolution, current)
		for _, repo := range repos {
			row := ensure(repo)
			row.EvolutionCount++
			if evolution.Implementation.Snapshot != "" {
				row.SnapshotCount++
			}
			row.CommitCount += len(evolution.Implementation.Commits)
			latest := fallback(evolution.Metadata.UpdatedAt, evolution.Metadata.CreatedAt)
			if latest > row.LatestAt {
				row.LatestAt = latest
				row.LatestEvolution = evolution.Metadata.ID
				row.LatestTitle = evolution.Metadata.Title
			}
			if providerSets[repo] == nil {
				providerSets[repo] = map[string]bool{}
			}
			for _, provider := range sessionProviders(evolution.Sessions) {
				providerSets[repo][provider] = true
			}
		}
	}
	out := make([]repositorySummary, 0, len(byName))
	for name, row := range byName {
		for provider := range providerSets[name] {
			row.SessionProviders = append(row.SessionProviders, provider)
		}
		sort.Strings(row.SessionProviders)
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LatestAt != out[j].LatestAt {
			return out[i].LatestAt > out[j].LatestAt
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func evolutionRepositoryNames(evolution *eve.Evolution, fallbackRepo string) []string {
	seen := map[string]bool{}
	var repos []string
	for repo := range evolution.Implementation.Repositories {
		repo = strings.TrimSpace(repo)
		if repo == "" || seen[repo] {
			continue
		}
		repos = append(repos, repo)
		seen[repo] = true
	}
	if len(repos) == 0 {
		repos = append(repos, fallbackRepo)
	}
	sort.Strings(repos)
	return repos
}

func evolutionTouchesRepository(evolution *eve.Evolution, repo string, fallbackRepo string) bool {
	for _, name := range evolutionRepositoryNames(evolution, fallbackRepo) {
		if name == repo {
			return true
		}
	}
	return false
}

func summarizeEvolution(evolution *eve.Evolution) evolutionSummary {
	return evolutionSummary{
		ID:                  evolution.Metadata.ID,
		Title:               evolution.Metadata.Title,
		Type:                evolution.Metadata.Type,
		Status:              evolution.Metadata.Status,
		Outcome:             evolution.Outcome,
		Snapshot:            evolution.Implementation.Snapshot,
		VerificationState:   verificationState(evolution.Verification),
		VerificationSummary: verificationSummary(evolution.Verification),
		SessionProviders:    sessionProviders(evolution.Sessions),
		CreatedAt:           evolution.Metadata.CreatedAt,
		UpdatedAt:           evolution.Metadata.UpdatedAt,
	}
}

func verificationState(values []eve.Verification) string {
	if len(values) == 0 {
		return "none"
	}
	state := "passed"
	for _, verification := range values {
		switch verification.Status {
		case "failed":
			return "failed"
		case "pending":
			if state != "failed" {
				state = "pending"
			}
		case "skipped", "generated":
			if state == "passed" {
				state = verification.Status
			}
		case "approved", "passed":
		default:
			if state == "passed" {
				state = verification.Status
			}
		}
	}
	return state
}

func verificationSummary(values []eve.Verification) string {
	if len(values) == 0 {
		return "No verification"
	}
	parts := make([]string, 0, len(values))
	for _, verification := range values {
		label := strings.TrimSpace(verification.Status)
		if verification.Reference != "" {
			label += ": " + verification.Reference
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, "; ")
}

func sessionProviders(sessions []eve.Session) []string {
	seen := map[string]bool{}
	var providers []string
	for _, session := range sessions {
		provider := fallback(session.Provider, "unknown")
		if seen[provider] {
			continue
		}
		providers = append(providers, provider)
		seen[provider] = true
	}
	sort.Strings(providers)
	return providers
}

func (server uiServer) sessionRecords(evolution *eve.Evolution) []uiSessionRecord {
	manifest := server.store.loadSessionManifest(evolution.Metadata.ID)
	artifactByKey := map[string]sessionArtifact{}
	for _, artifact := range manifest.Sessions {
		artifactByKey[sessionRecordKey(artifact.Provider, artifact.ID)] = artifact
	}

	seen := map[string]bool{}
	var records []uiSessionRecord
	for _, session := range evolution.Sessions {
		key := sessionRecordKey(session.Provider, session.ID)
		record := uiSessionRecord{
			Provider:     session.Provider,
			ProviderName: providerDisplayName(session.Provider),
			ID:           session.ID,
			Key:          key,
			URI:          session.URI,
			Status:       "reference-only",
		}
		if artifact, ok := artifactByKey[key]; ok {
			record = recordFromArtifact(session, artifact)
		} else if session.URI != "" && fileExists(filepath.FromSlash(session.URI)) {
			record.Transcript = session.URI
			record.HasTranscript = true
			record.Status = "transcript"
			record.Preview = previewSessionFile(filepath.FromSlash(session.URI))
		}
		record.CaptureHint = sessionCaptureHint(record.Provider, record.ID)
		record.RootsChecked = providerRoots(record.Provider)
		record.LocalSources = discoverSessionSources(record.Provider, record.ID, evolution)
		if !record.HasTranscript && len(record.LocalSources) > 0 {
			record.Status = "local-candidate"
			record.Preview = previewSessionFile(filepath.FromSlash(record.LocalSources[0].Path))
		}
		records = append(records, record)
		seen[key] = true
	}
	for _, artifact := range manifest.Sessions {
		key := sessionRecordKey(artifact.Provider, artifact.ID)
		if seen[key] {
			continue
		}
		records = append(records, recordFromArtifact(eve.Session{Provider: artifact.Provider, ID: artifact.ID}, artifact))
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Key < records[j].Key
	})
	return records
}

func recordFromArtifact(session eve.Session, artifact sessionArtifact) uiSessionRecord {
	return uiSessionRecord{
		Provider:      artifact.Provider,
		ProviderName:  providerDisplayName(artifact.Provider),
		ID:            artifact.ID,
		Key:           sessionRecordKey(artifact.Provider, artifact.ID),
		URI:           session.URI,
		Title:         artifact.Title,
		Transcript:    artifact.Transcript,
		Raw:           artifact.Raw,
		Sanitized:     artifact.Sanitized,
		Format:        artifact.Format,
		AttachedAt:    artifact.AttachedAt,
		Source:        artifact.Source,
		Metadata:      artifact.Metadata,
		HasTranscript: artifact.Transcript != "" && fileExists(filepath.FromSlash(artifact.Transcript)),
		Status:        "transcript",
		CaptureHint:   sessionCaptureHint(artifact.Provider, artifact.ID),
		RootsChecked:  providerRoots(artifact.Provider),
		LocalSources:  []uiSessionSource{},
		Preview:       previewSessionFile(filepath.FromSlash(artifact.Transcript)),
	}
}

func sessionCaptureHint(provider string, id string) string {
	provider = fallback(provider, "provider")
	id = fallback(id, "session-id")
	return fmt.Sprintf("eve add session %s:%s --source <transcript.jsonl|json|md>", provider, id)
}

func providerInfos() []uiProviderInfo {
	providers := []string{"codex", "claude", "opencode", "pi"}
	infos := make([]uiProviderInfo, 0, len(providers))
	for _, provider := range providers {
		roots := providerRoots(provider)
		available := false
		for _, root := range roots {
			if info, err := os.Stat(root); err == nil && info.IsDir() {
				available = true
				break
			}
		}
		infos = append(infos, uiProviderInfo{
			Provider:      provider,
			Name:          providerDisplayName(provider),
			Roots:         roots,
			Available:     available,
			ImportCommand: sessionCaptureHint(provider, "<session-id>"),
			Displays: []string{
				"session provider and id",
				"user and agent messages",
				"message, event, and tool-call analytics",
				"raw artifact path and format",
				"local matching transcript candidates when found",
			},
		})
	}
	return infos
}

func providerDisplayName(provider string) string {
	switch normalizeProvider(provider) {
	case "codex":
		return "Codex"
	case "claude":
		return "Claude Code"
	case "opencode":
		return "OpenCode"
	case "pi":
		return "Pi"
	default:
		return fallback(provider, "Unknown")
	}
}

func providerRoots(provider string) []string {
	switch normalizeProvider(provider) {
	case "codex":
		return codexSessionRoots()
	case "claude":
		return claudeSessionRoots()
	case "opencode":
		return opencodeSessionRoots()
	case "pi":
		return piSessionRoots()
	default:
		return []string{}
	}
}

func opencodeSessionRoots() []string {
	home, _ := os.UserHomeDir()
	var roots []string
	if opencodeHome := os.Getenv("OPENCODE_HOME"); opencodeHome != "" {
		roots = append(roots, opencodeHome)
	}
	if home != "" {
		roots = append(roots,
			filepath.Join(home, ".local", "share", "opencode", "storage"),
			filepath.Join(home, "Library", "Application Support", "opencode"),
			filepath.Join(home, ".opencode"),
		)
	}
	return roots
}

func discoverSessionSources(provider string, sessionID string, evolution *eve.Evolution) []uiSessionSource {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	type scoredSource struct {
		source uiSessionSource
		score  int
	}
	keywords := sessionMatchKeywords(sessionID, evolution)
	var files []string
	for _, root := range providerRoots(provider) {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext != ".jsonl" && ext != ".json" && ext != ".md" && ext != ".txt" {
				return nil
			}
			files = append(files, path)
			return nil
		})
	}
	sort.Slice(files, func(i, j int) bool {
		left, leftErr := os.Stat(files[i])
		right, rightErr := os.Stat(files[j])
		if leftErr != nil || rightErr != nil {
			return files[i] > files[j]
		}
		return left.ModTime().After(right.ModTime())
	})
	files = filterSessionFilesByTime(files, evolution)
	if len(files) > 500 {
		files = files[:500]
	}
	var matches []scoredSource
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() || info.Size() > 20*1024*1024 {
			continue
		}
		score, reason, title := scoreSessionCandidate(file, sessionID, keywords, evolution)
		if score == 0 {
			continue
		}
		matches = append(matches, scoredSource{
			score: score,
			source: uiSessionSource{
				Path:       filepath.ToSlash(file),
				Format:     strings.TrimPrefix(strings.ToLower(filepath.Ext(file)), "."),
				Size:       info.Size(),
				ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
				Title:      title,
				Match:      reason,
			},
		})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].source.ModifiedAt > matches[j].source.ModifiedAt
	})
	var sources []uiSessionSource
	for _, match := range matches {
		sources = append(sources, match.source)
		if len(sources) == 8 {
			break
		}
	}
	if len(sources) == 0 {
		return []uiSessionSource{}
	}
	return sources
}

func sessionMatchKeywords(sessionID string, evolution *eve.Evolution) []string {
	seen := map[string]bool{}
	var keywords []string
	add := func(value string) {
		value = strings.ToLower(strings.TrimSpace(value))
		value = strings.Trim(value, "-_ .:/")
		if len(value) < 4 || seen[value] {
			return
		}
		keywords = append(keywords, value)
		seen[value] = true
	}
	add(sessionID)
	for _, part := range strings.FieldsFunc(sessionID, func(r rune) bool {
		return r == '-' || r == '_' || r == ':' || r == '/'
	}) {
		add(part)
	}
	if evolution == nil {
		return keywords
	}
	add(evolution.Metadata.ID)
	add(evolution.Metadata.Title)
	add(evolution.Intent)
	add(evolution.Outcome)
	for _, claim := range evolution.Behavior.Added {
		add(claim.Description)
	}
	for _, claim := range evolution.Behavior.Changed {
		add(claim.Description)
	}
	for _, claim := range evolution.Behavior.Fixed {
		add(claim.Description)
	}
	for _, claim := range evolution.Behavior.Removed {
		add(claim.Description)
	}
	return keywords
}

func filterSessionFilesByTime(files []string, evolution *eve.Evolution) []string {
	if evolution == nil {
		if len(files) > 80 {
			return files[:80]
		}
		return files
	}
	center, ok := parseEvolutionTime(evolution)
	if !ok {
		if len(files) > 160 {
			return files[:160]
		}
		return files
	}
	start := center.AddDate(0, 0, -14)
	end := center.AddDate(0, 0, 2)
	filtered := make([]string, 0, len(files))
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		if info.ModTime().Before(start) || info.ModTime().After(end) {
			continue
		}
		filtered = append(filtered, file)
	}
	if len(filtered) == 0 {
		if len(files) > 120 {
			return files[:120]
		}
		return files
	}
	return filtered
}

func parseEvolutionTime(evolution *eve.Evolution) (time.Time, bool) {
	for _, value := range []string{evolution.Metadata.UpdatedAt, evolution.Metadata.CreatedAt} {
		if value == "" {
			continue
		}
		parsed, err := time.Parse(time.RFC3339, value)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func scoreSessionCandidate(file string, sessionID string, keywords []string, evolution *eve.Evolution) (int, string, string) {
	lowerPath := strings.ToLower(file)
	lowerName := strings.ToLower(filepath.Base(file))
	lowerID := strings.ToLower(sessionID)
	score := 0
	var reasons []string
	if strings.Contains(lowerName, lowerID) || strings.Contains(lowerPath, lowerID) {
		score += 100
		reasons = append(reasons, "id")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return score, strings.Join(reasons, ", "), ""
	}
	text := strings.ToLower(string(data))
	if strings.Contains(text, lowerID) {
		score += 60
		reasons = append(reasons, "session reference")
	}
	if evolution != nil {
		if cwd, err := os.Getwd(); err == nil && strings.Contains(text, strings.ToLower(filepath.ToSlash(cwd))) {
			score += 20
			reasons = append(reasons, "repo path")
		}
		if evolution.Metadata.ID != "" && strings.Contains(text, strings.ToLower(evolution.Metadata.ID)) {
			score += 30
			reasons = append(reasons, "EV ID")
		}
	}
	keywordHits := 0
	for _, keyword := range keywords {
		if keyword == lowerID {
			continue
		}
		if strings.Contains(lowerPath, keyword) || strings.Contains(text, keyword) {
			keywordHits++
		}
	}
	if keywordHits > 0 {
		score += min(keywordHits, 5) * 8
		reasons = append(reasons, "content")
	}
	if score < 24 {
		return 0, "", ""
	}
	return score, strings.Join(uniqueStrings(reasons), ", "), sessionCandidateTitle(data, file)
}

func sessionCandidateTitle(data []byte, file string) string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event map[string]any
		if json.Unmarshal([]byte(line), &event) == nil {
			if title := firstMeaningfulTitle(firstString(event, "thread_name", "title", "summary", "name")); title != "" {
				return title
			}
			if payload, ok := event["payload"].(map[string]any); ok {
				if title := firstMeaningfulTitle(firstString(payload, "thread_name", "title", "summary", "name")); title != "" {
					return title
				}
			}
		}
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
	}
	return strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
}

func firstMeaningfulTitle(value string) string {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "", "auto", "untitled", "exec_command", "write_stdin", "response_item", "event_msg", "turn_context":
		return ""
	default:
		return value
	}
}

func previewSessionFile(path string) uiSessionPreview {
	if strings.TrimSpace(path) == "" {
		return uiSessionPreview{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return uiSessionPreview{}
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jsonl":
		return previewJSONLSession(data)
	case ".md", ".markdown":
		return previewMarkdownSession(data)
	default:
		return previewTextSession(data)
	}
}

func previewJSONLSession(data []byte) uiSessionPreview {
	return previewConversation(extractSessionConversation("jsonl", data))
}

func previewMarkdownSession(data []byte) uiSessionPreview {
	preview := previewConversation(extractSessionConversation("md", data))
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if heading != "" && len(preview.Headings) < 6 {
				preview.Headings = append(preview.Headings, heading)
			}
		}
	}
	return preview
}

func previewConversation(conversation sessionConversation) uiSessionPreview {
	return uiSessionPreview{
		EventCount:     conversation.EventCount,
		MessageCount:   conversation.MessageCount,
		UserMessages:   conversation.UserMessages,
		AgentMessages:  conversation.AgentMessages,
		ToolCalls:      conversation.ToolCalls,
		FirstTimestamp: conversation.FirstTimestamp,
		LastTimestamp:  conversation.LastTimestamp,
	}
}

func previewTextSession(data []byte) uiSessionPreview {
	text := strings.TrimSpace(string(data))
	if text == "" {
		return uiSessionPreview{}
	}
	return uiSessionPreview{EventCount: len(strings.Split(text, "\n"))}
}

func gitCommits(commits []string) []uiGitCommit {
	out := make([]uiGitCommit, 0, len(commits))
	for _, commit := range commits {
		if info, err := gitCommit(commit); err == nil {
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

func gitCommit(commit string) (uiGitCommit, error) {
	format := "%H%x00%h%x00%s%x00%an%x00%aI%x00%cI"
	output, err := exec.Command("git", "show", "-s", "--format="+format, commit).Output()
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

func gitRemoteURL() string {
	output, err := exec.Command("git", "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func shortHash(commit string) string {
	if len(commit) <= 12 {
		return commit
	}
	return commit[:12]
}

func (server uiServer) searchMatches(evolution *eve.Evolution, query string) []string {
	var matches []string
	for _, field := range searchableFields(evolution) {
		if strings.Contains(strings.ToLower(field), query) {
			matches = append(matches, field)
		}
	}
	for _, session := range server.sessionRecords(evolution) {
		if session.Title != "" && strings.Contains(strings.ToLower(session.Title), query) {
			matches = append(matches, "session: "+session.Title)
		}
		if session.HasTranscript {
			data, err := os.ReadFile(filepath.FromSlash(session.Transcript))
			if err == nil && strings.Contains(strings.ToLower(string(data)), query) {
				matches = append(matches, "session transcript: "+fallback(session.Title, session.Key))
			}
		}
	}
	return uniqueStrings(matches)
}

func sortEvolutionSummaries(rows []evolutionSummary) {
	sort.Slice(rows, func(i, j int) bool {
		left := rows[i]
		right := rows[j]
		if left.CreatedAt != "" && right.CreatedAt != "" && left.CreatedAt != right.CreatedAt {
			return left.CreatedAt > right.CreatedAt
		}
		return left.ID > right.ID
	})
}

func splitPath(value string) []string {
	var parts []string
	for _, part := range strings.Split(value, "/") {
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	return parts
}

func sessionRecordKey(provider string, id string) string {
	return strings.TrimSpace(provider) + ":" + strings.TrimSpace(id)
}

func checkoutCommand(id string, repo string) string {
	if strings.TrimSpace(repo) == "" {
		return "eve checkout " + id
	}
	return "eve checkout " + id + " --repo " + strings.TrimSpace(repo)
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(value)
}

func writeAPIError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, uiError{Error: err.Error()})
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeAPIError(w, http.StatusMethodNotAllowed, fmt.Errorf("method not allowed"))
}

func spaHandler() http.Handler {
	dist, err := fs.Sub(embeddedUI, "ui_dist")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "EVE UI assets are not embedded", http.StatusInternalServerError)
		})
	}
	fileServer := http.FileServer(http.FS(dist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "." || name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(dist, name); err != nil {
			r.URL.Path = "/"
			name = "index.html"
		}
		if name == "index.html" {
			w.Header().Set("Cache-Control", "no-store")
		}
		fileServer.ServeHTTP(w, r)
	})
}

func logUIRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func openURL(url string) error {
	switch {
	case commandExists("open"):
		return exec.Command("open", url).Start()
	case commandExists("xdg-open"):
		return exec.Command("xdg-open", url).Start()
	default:
		return fmt.Errorf("no supported browser opener found")
	}
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
