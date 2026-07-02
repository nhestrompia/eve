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
	rows := make([]evolutionSummary, 0, len(evolutions))
	for _, evolution := range evolutions {
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
			writeAPIError(w, http.StatusNotFound, fmt.Errorf("session transcript is not available"))
			return
		}
		markdown, err := os.ReadFile(filepath.FromSlash(session.Transcript))
		if err != nil {
			writeAPIError(w, http.StatusNotFound, fmt.Errorf("read session transcript: %w", err))
			return
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
		record.LocalSources = discoverSessionSources(record.Provider, record.ID)
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
		LocalSources:  discoverSessionSources(artifact.Provider, artifact.ID),
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
				"attached sanitized transcript",
				"raw artifact path and format",
				"message/event counts when a transcript is attached",
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
		return []string{}
	case "pi":
		return piSessionRoots()
	default:
		return []string{}
	}
}

func discoverSessionSources(provider string, sessionID string) []uiSessionSource {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	var sources []uiSessionSource
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
			if !strings.Contains(entry.Name(), sessionID) && !strings.Contains(path, sessionID) {
				return nil
			}
			info, statErr := entry.Info()
			if statErr != nil {
				return nil
			}
			sources = append(sources, uiSessionSource{
				Path:       filepath.ToSlash(path),
				Format:     strings.TrimPrefix(ext, "."),
				Size:       info.Size(),
				ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
			})
			return nil
		})
	}
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].ModifiedAt > sources[j].ModifiedAt
	})
	if len(sources) > 8 {
		return sources[:8]
	}
	if sources == nil {
		return []uiSessionSource{}
	}
	return sources
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
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024), 1024*1024*10)
	preview := uiSessionPreview{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		preview.EventCount++
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		role := strings.ToLower(firstString(event, "role", "type", "kind", "event"))
		if role == "user" {
			preview.UserMessages++
			preview.MessageCount++
		}
		if role == "assistant" || role == "agent" || role == "system" {
			preview.AgentMessages++
			preview.MessageCount++
		}
		if strings.Contains(role, "tool") {
			preview.ToolCalls++
		}
		if timestamp := firstString(event, "timestamp", "created_at", "createdAt", "time"); timestamp != "" {
			if preview.FirstTimestamp == "" {
				preview.FirstTimestamp = timestamp
			}
			preview.LastTimestamp = timestamp
		}
	}
	return preview
}

func previewMarkdownSession(data []byte) uiSessionPreview {
	preview := uiSessionPreview{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(line, "#"))
			if heading != "" && len(preview.Headings) < 6 {
				preview.Headings = append(preview.Headings, heading)
			}
		}
		if strings.HasPrefix(strings.ToLower(line), "### user") {
			preview.UserMessages++
			preview.MessageCount++
		}
		if strings.HasPrefix(strings.ToLower(line), "### assistant") {
			preview.AgentMessages++
			preview.MessageCount++
		}
	}
	return preview
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
