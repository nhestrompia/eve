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
	"path"
	"strings"

	"github.com/nhestrompia/eve"
)

//go:embed ui_dist/* ui_dist/assets/*
var embeddedUI embed.FS

type runtimeServer struct {
	repo repository
	addr string
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
}

type snapshotSummary struct {
	ID                string `json:"id"`
	Title             string `json:"title"`
	Type              string `json:"type"`
	Summary           string `json:"summary"`
	UserVisibleChange string `json:"userVisibleChange,omitempty"`
	GitState          string `json:"gitState"`
	Branch            string `json:"branch"`
	Dirty             bool   `json:"dirty"`
	ValidationState   string `json:"validationState"`
	CreatedAt         string `json:"createdAt"`
}

type snapshotDetailResponse struct {
	Snapshot *eve.Snapshot   `json:"snapshot"`
	Summary  snapshotSummary `json:"summary"`
	RawJSON  json.RawMessage `json:"rawJson"`
}

func newRuntimeServer(repo repository, addr string) runtimeServer {
	return runtimeServer{repo: repo, addr: addr}
}

func (server runtimeServer) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", server.handleConfig)
	mux.HandleFunc("/api/repos", server.handleRepos)
	mux.HandleFunc("/api/repos/", server.handleRepoRoutes)
	mux.HandleFunc("/mcp", server.handleMCPHTTP)
	mux.Handle("/", spaHandler())
	return logRequests(mux)
}

func (server runtimeServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	_, err := os.Stat(server.repo.configPath())
	writeJSON(w, http.StatusOK, configResponse{
		SnapshotSchemaVersion: eve.SnapshotSchemaVersion,
		CLIVersion:            eve.CLIVersion,
		Repository:            server.repo.ID,
		Addr:                  server.addr,
		EveDir:                server.repo.eveDir,
		Initialized:           err == nil,
	})
}

func (server runtimeServer) handleRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	summary, err := server.repo.summary()
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, []repoSummary{summary})
}

func (server runtimeServer) handleRepoRoutes(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/repos/"), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("repo route not found"))
		return
	}
	repoID := parts[0]
	if repoID != server.repo.ID {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("repository %s not found", repoID))
		return
	}
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		summary, err := server.repo.summary()
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, summary)
	case len(parts) == 2 && parts[1] == "snapshots" && r.Method == http.MethodGet:
		server.handleSnapshots(w, r)
	case len(parts) == 3 && parts[1] == "snapshots" && r.Method == http.MethodGet:
		server.handleSnapshotDetail(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "snapshots" && parts[3] == "checkout" && r.Method == http.MethodPost:
		server.handleCheckout(w, r, parts[2])
	default:
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("repo route not found"))
	}
}

func (server runtimeServer) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots, err := server.repo.listSnapshots(r.URL.Query().Get("type"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	rows := make([]snapshotSummary, 0, len(snapshots))
	for _, snapshot := range snapshots {
		rows = append(rows, summarizeSnapshot(snapshot))
	}
	writeJSON(w, http.StatusOK, rows)
}

func (server runtimeServer) handleSnapshotDetail(w http.ResponseWriter, r *http.Request, id string) {
	snapshot, raw, err := server.loadSnapshotWithRaw(id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshotDetailResponse{
		Snapshot: snapshot,
		Summary:  summarizeSnapshot(snapshot),
		RawJSON:  raw,
	})
}

func (server runtimeServer) handleCheckout(w http.ResponseWriter, r *http.Request, id string) {
	snapshot, err := server.repo.loadSnapshot(id)
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	force := r.URL.Query().Get("force") == "true"
	writeJSON(w, http.StatusOK, checkoutSnapshot(server.repo, snapshot, force))
}

func (server runtimeServer) loadSnapshotWithRaw(id string) (*eve.Snapshot, json.RawMessage, error) {
	data, err := os.ReadFile(server.repo.snapshotPath(id))
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
		ID:                snapshot.ID,
		Title:             snapshot.Title,
		Type:              snapshot.Type,
		Summary:           snapshot.Summary,
		UserVisibleChange: snapshot.UserVisibleChange,
		GitState:          snapshot.Implementation.GitState,
		Branch:            snapshot.Implementation.Branch,
		Dirty:             snapshot.Implementation.Dirty,
		ValidationState:   validationState(snapshot.Validation),
		CreatedAt:         snapshot.CreatedAt,
	}
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
		{"name": "complete_snapshot", "description": "Create a completed product Snapshot and derive Git implementation facts.", "inputSchema": completeSnapshotSchema()},
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
		"relationships":     map[string]string{"type": "object"},
		"risks":             map[string]string{"type": "array"},
		"timeline":          map[string]string{"type": "array"},
		"decisions":         map[string]string{"type": "array"},
		"validation":        map[string]string{"type": "array"},
		"artifacts":         map[string]string{"type": "array"},
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
		summary, err := server.repo.summary()
		if err != nil {
			return toolError(err.Error()), nil
		}
		return toolResult([]repoSummary{summary}), nil
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
	var input struct {
		CWD               string              `json:"cwd"`
		RepoID            string              `json:"repoId"`
		Title             string              `json:"title"`
		Type              string              `json:"type"`
		Summary           string              `json:"summary"`
		UserVisibleChange string              `json:"userVisibleChange"`
		Relationships     eve.Relationships   `json:"relationships"`
		Risks             []eve.Risk          `json:"risks"`
		Timeline          []eve.TimelineEntry `json:"timeline"`
		Decisions         []eve.Decision      `json:"decisions"`
		Validation        []eve.Validation    `json:"validation"`
		Artifacts         []eve.Artifact      `json:"artifacts"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, &mcpError{Code: -32602, Message: err.Error()}
	}
	repo, err := server.resolveToolRepo(input.CWD, input.RepoID)
	if err != nil {
		return toolError(err.Error()), nil
	}
	facts, err := deriveGitFacts(repo)
	if err != nil {
		return toolError(err.Error()), nil
	}
	snapshot := &eve.Snapshot{
		ID:                newSnapshotID(),
		SchemaVersion:     eve.SnapshotSchemaVersion,
		Title:             strings.TrimSpace(input.Title),
		Type:              strings.TrimSpace(input.Type),
		Summary:           strings.TrimSpace(input.Summary),
		UserVisibleChange: strings.TrimSpace(input.UserVisibleChange),
		Relationships:     input.Relationships,
		Risks:             input.Risks,
		Timeline:          input.Timeline,
		Decisions:         input.Decisions,
		Validation:        input.Validation,
		Artifacts:         input.Artifacts,
		Implementation: eve.Implementation{
			Branch:     facts.Branch,
			GitState:   facts.GitState,
			BaseCommit: facts.BaseCommit,
			Commits:    facts.Commits,
			Dirty:      facts.Dirty,
		},
		CreatedAt: nowUTC(),
	}
	if err := repo.saveSnapshot(snapshot); err != nil {
		return toolError(err.Error()), nil
	}
	return toolResult(snapshot), nil
}

func (server runtimeServer) resolveToolRepo(cwd string, repoID string) (repository, error) {
	if strings.TrimSpace(repoID) == "" && strings.TrimSpace(cwd) == "" {
		return server.repo, nil
	}
	if strings.TrimSpace(repoID) == server.repo.ID {
		return server.repo, nil
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
	snapshots, err := server.repo.listSnapshots("")
	if err != nil {
		return nil, err
	}
	resources := []map[string]string{
		{"uri": "eve://repos", "name": "repos", "title": "EVE repositories", "mimeType": "application/json"},
		{"uri": "eve://repos/" + server.repo.ID, "name": server.repo.ID, "title": server.repo.ID, "mimeType": "application/json"},
		{"uri": "eve://repos/" + server.repo.ID + "/snapshots", "name": "snapshots", "title": "Snapshots", "mimeType": "application/json"},
	}
	for _, snapshot := range snapshots {
		resources = append(resources, map[string]string{
			"uri":      "eve://repos/" + server.repo.ID + "/snapshots/" + snapshot.ID,
			"name":     snapshot.ID,
			"title":    snapshot.Title,
			"mimeType": "application/json",
		})
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
		summary, err := server.repo.summary()
		if err != nil {
			return nil, &mcpError{Code: -32000, Message: err.Error()}
		}
		value = []repoSummary{summary}
	case "eve://repos/" + server.repo.ID:
		summary, err := server.repo.summary()
		if err != nil {
			return nil, &mcpError{Code: -32000, Message: err.Error()}
		}
		value = summary
	case "eve://repos/" + server.repo.ID + "/snapshots":
		snapshots, err := server.repo.listSnapshots("")
		if err != nil {
			return nil, &mcpError{Code: -32000, Message: err.Error()}
		}
		rows := make([]snapshotSummary, 0, len(snapshots))
		for _, snapshot := range snapshots {
			rows = append(rows, summarizeSnapshot(snapshot))
		}
		value = rows
	default:
		prefix := "eve://repos/" + server.repo.ID + "/snapshots/"
		if !strings.HasPrefix(input.URI, prefix) {
			return nil, &mcpError{Code: -32602, Message: "unknown resource: " + input.URI}
		}
		snapshot, err := server.repo.loadSnapshot(strings.TrimPrefix(input.URI, prefix))
		if err != nil {
			return nil, &mcpError{Code: -32000, Message: err.Error()}
		}
		value = snapshot
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
