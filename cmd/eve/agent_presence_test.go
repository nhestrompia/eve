package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestAgentLeaseReportsLiveThenOfflinePlan(t *testing.T) {
	repo := setupPlanTestRepo(t)
	request, err := repo.createOrResumePlanRequest(t.Context(), testPlanInput("planreq_agentlive1"))
	if err != nil {
		t.Fatal(err)
	}
	locked, err := repo.approvePlanRequest(t.Context(), request.PlanRequestID, 1, nil)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	owner := newAgentLeaseOwner(repo, "Codex", "agent_fixture", time.Minute, time.Hour)
	t.Cleanup(owner.close)
	if err := owner.update(locked, agentStateRunning, now); err != nil {
		t.Fatal(err)
	}

	server := newRuntimeServer(repo, "")
	activities := server.agentActivities(now.Add(30 * time.Second))
	if len(activities) != 1 || activities[0].Status != agentStateRunning ||
		activities[0].Provider != "Codex" || activities[0].PlanRequestID != request.PlanRequestID {
		t.Fatalf("live activities = %#v", activities)
	}

	if err := owner.remove(); err != nil {
		t.Fatal(err)
	}
	activities = server.agentActivities(now.Add(31 * time.Second))
	if len(activities) != 1 || activities[0].Status != agentStateOffline {
		t.Fatalf("offline activities = %#v", activities)
	}
}

func TestExpiredAgentLeaseIsNotLive(t *testing.T) {
	repo := setupPlanTestRepo(t)
	owner := newAgentLeaseOwner(repo, "Codex", "agent_expired", time.Second, time.Hour)
	t.Cleanup(owner.close)

	now := time.Now().UTC()
	request := &planRequest{
		PlanRequestID: "planreq_expired1",
		Repository:    repo.ID,
		State:         "locked",
	}
	if err := owner.update(request, agentStateRunning, now); err != nil {
		t.Fatal(err)
	}
	leases := repo.agentLeases(now.Add(2 * time.Second))
	if len(leases) != 0 {
		t.Fatalf("expired leases = %#v, want none", leases)
	}
}

func TestAgentLeaseExpiryEmitsEventWithoutPolling(t *testing.T) {
	repo := setupPlanTestRepo(t)
	owner := newAgentLeaseOwner(repo, "Codex", "agent_expiry_event", 50*time.Millisecond, time.Hour)
	t.Cleanup(owner.close)
	request := &planRequest{
		PlanRequestID: "planreq_expiry01",
		Repository:    repo.ID,
		State:         "locked",
	}
	if err := owner.update(request, agentStateRunning, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	events := newRuntimeEvents(time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, unsubscribe := events.subscribe()
	defer unsubscribe()
	go events.watchAgentExpirations(ctx, []repository{repo})

	select {
	case event := <-stream:
		if event.Kind != runtimeEventAgents {
			t.Fatalf("expiry event = %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("lease expiry did not emit an agent event")
	}
}

func TestMCPPlanLookupActivatesProcessLease(t *testing.T) {
	repo := setupPlanTestRepo(t)
	request, err := repo.createOrResumePlanRequest(t.Context(), testPlanInput("planreq_mcplease1"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.approvePlanRequest(t.Context(), request.PlanRequestID, 1, nil); err != nil {
		t.Fatal(err)
	}
	server := newRuntimeServer(repo, "")
	server.agentLease = newAgentLeaseOwner(repo, "Codex", "agent_mcp", time.Minute, time.Hour)
	t.Cleanup(server.agentLease.close)
	arguments, _ := json.Marshal(map[string]string{"planRequestId": request.PlanRequestID})
	params, _ := json.Marshal(map[string]any{"name": "get_plan_request", "arguments": json.RawMessage(arguments)})
	result, rpcErr := server.callMCPTool(t.Context(), params)
	if rpcErr != nil || !toolCallSucceeded(result) {
		t.Fatalf("get_plan_request = %#v, %#v", result, rpcErr)
	}
	leases := repo.agentLeases(time.Now().UTC())
	if len(leases) != 1 || leases[0].Status != agentStateRunning ||
		leases[0].PlanRequestID != request.PlanRequestID {
		t.Fatalf("MCP leases = %#v", leases)
	}
}
