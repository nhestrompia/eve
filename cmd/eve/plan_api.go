package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func (server runtimeServer) handlePlanRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	requests, err := server.cachedPlanRequests(r.Context(), status)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, requests)
}

func (server runtimeServer) handlePlanRequestRoutes(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/plan-requests/"), "/")
	if trimmed == "events" {
		server.handlePlanRequestEvents(w, r)
		return
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("plan request route not found"))
		return
	}
	repo, request, err := server.findPlanRequest(parts[0])
	if err != nil {
		writeAPIError(w, http.StatusNotFound, err)
		return
	}
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		refreshed, refreshErr := repo.refreshPlanRequestState(r.Context(), request.PlanRequestID)
		if refreshErr != nil {
			writeAPIError(w, http.StatusInternalServerError, refreshErr)
			return
		}
		writeJSON(w, http.StatusOK, refreshed)
	case len(parts) == 2 && parts[1] == "approve" && r.Method == http.MethodPost:
		var input struct {
			ExpectedRevision int           `json:"expectedRevision"`
			Proposal         *planProposal `json:"proposal,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		approved, approveErr := repo.approvePlanRequest(r.Context(), request.PlanRequestID, input.ExpectedRevision, input.Proposal)
		if approveErr != nil {
			writePlanMutationError(w, approveErr)
			return
		}
		server.publishPlanRequestChange(repo)
		writeJSON(w, http.StatusOK, approved)
	case len(parts) == 2 && parts[1] == "reject" && r.Method == http.MethodPost:
		var input struct {
			ExpectedRevision int    `json:"expectedRevision"`
			Feedback         string `json:"feedback"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		rejected, rejectErr := repo.rejectPlanRequest(r.Context(), request.PlanRequestID, input.ExpectedRevision, input.Feedback)
		if rejectErr != nil {
			writePlanMutationError(w, rejectErr)
			return
		}
		server.publishPlanRequestChange(repo)
		writeJSON(w, http.StatusOK, rejected)
	case len(parts) == 2 && parts[1] == "dismiss" && r.Method == http.MethodPost:
		var input struct {
			ExpectedRevision int `json:"expectedRevision"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeAPIError(w, http.StatusBadRequest, err)
			return
		}
		dismissed, dismissErr := repo.dismissStalePlanRequest(r.Context(), request.PlanRequestID, input.ExpectedRevision)
		if dismissErr != nil {
			writePlanMutationError(w, dismissErr)
			return
		}
		server.publishPlanRequestChange(repo)
		writeJSON(w, http.StatusOK, dismissed)
	default:
		writeAPIError(w, http.StatusNotFound, fmt.Errorf("plan request route not found"))
	}
}

func (server runtimeServer) publishPlanRequestChange(repo repository) {
	if server.events == nil {
		return
	}
	server.events.publish(runtimeEvent{Kind: runtimeEventPlans, Repository: repo.ID})
}

func writePlanMutationError(w http.ResponseWriter, err error) {
	message := err.Error()
	switch {
	case strings.Contains(message, "revision conflict"),
		strings.Contains(message, "is stale"),
		strings.Contains(message, "plan request is "):
		writeAPIError(w, http.StatusConflict, err)
	default:
		writeAPIError(w, http.StatusUnprocessableEntity, err)
	}
}

func (server runtimeServer) planRequests(ctx context.Context, status string) ([]*planRequest, error) {
	return planRequestsFromRepositories(ctx, server.planRequestRepositories(), status)
}

func (server runtimeServer) cachedPlanRequests(ctx context.Context, status string) ([]*planRequest, error) {
	return cachedRuntimeValue(
		ctx,
		server.derivedCache,
		"plan-requests:"+status,
		server.events.generationFor(runtimeEventPlans),
		func() ([]*planRequest, error) {
			return server.planRequests(ctx, status)
		},
	)
}

func planRequestsFromRepositories(ctx context.Context, repositories []repository, status string) ([]*planRequest, error) {
	result := make([]*planRequest, 0)
	seen := make(map[string]bool)
	for _, repo := range repositories {
		requests, err := repo.listPlanRequests()
		if err != nil {
			continue
		}
		for _, request := range requests {
			if request.State == "pending_approval" {
				if refreshed, refreshErr := repo.refreshPlanRequestState(ctx, request.PlanRequestID); refreshErr == nil {
					request = refreshed
				}
			}
			if planRequestMatchesStatus(request, status) {
				if seen[request.PlanRequestID] {
					continue
				}
				seen[request.PlanRequestID] = true
				result = append(result, request)
			}
		}
	}
	return result, nil
}

func planRequestMatchesStatus(request *planRequest, status string) bool {
	switch status {
	case "":
		return true
	case "review":
		return request.State == "pending_approval" || request.State == "stale"
	default:
		return request.State == status
	}
}

func (server runtimeServer) findPlanRequest(id string) (repository, *planRequest, error) {
	for _, repo := range server.planRequestRepositories() {
		request, err := repo.loadPlanRequest(id)
		if err == nil {
			return repo, request, nil
		}
	}
	return repository{}, nil, fmt.Errorf("plan request %s not found", id)
}

func (server runtimeServer) planRequestRepositories() []repository {
	repositories := make([]repository, 0)
	seen := map[string]bool{}
	add := func(repo repository) {
		root := cleanRegistryRoot(repo.Root)
		if root == "" || seen[root] {
			return
		}
		seen[root] = true
		repositories = append(repositories, repo)
	}
	for _, repo := range server.repositories() {
		add(repo)
		for _, worktree := range repo.gitWorktreeRepositories() {
			add(worktree)
		}
	}
	return repositories
}

func (server runtimeServer) handlePlanRequestEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeAPIError(w, http.StatusInternalServerError, fmt.Errorf("streaming is unavailable"))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	if server.events == nil {
		writeAPIError(w, http.StatusServiceUnavailable, fmt.Errorf("runtime events are unavailable"))
		return
	}
	stream, unsubscribe := server.events.subscribe()
	defer unsubscribe()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	lastSignature := ""
	send := func() bool {
		requests, err := server.cachedPlanRequests(r.Context(), "")
		if err != nil {
			return false
		}
		data, _ := json.Marshal(requests)
		signature := string(data)
		if signature == lastSignature {
			return true
		}
		lastSignature = signature
		_, _ = fmt.Fprintf(w, "event: plan-requests\ndata: %s\n\n", data)
		flusher.Flush()
		return true
	}
	if !send() {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-stream:
			switch event.Kind {
			case runtimeEventAll, runtimeEventConfig, runtimeEventPlans, runtimeEventRepositories:
				if !send() {
					return
				}
			}
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}
