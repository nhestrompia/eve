import { describe, expect, it } from "vitest";
import type { PlanRequest } from "../types";
import { buildActiveAgentRows } from "./overview-page";

function plan(state: string, goal: string): PlanRequest {
  return {
    planRequestId: `planreq_${state}`,
    repository: "eve",
    repositoryRoot: "/tmp/eve",
    branch: "main",
    state,
    currentRevision: 1,
    revisions: [
      {
        revision: 1,
        source: "agent",
        goal,
        acceptanceCriteria: "- It works",
        allowedPathGlobs: ["ui/**"],
        milestones: [],
        resolvedCheckIds: [],
        policyHash: "",
        checkDefinitionsHash: "",
        suiteDigest: "",
        baseCommit: "abc123",
        branch: "main",
        createdAt: "2026-07-28T10:00:00Z",
      },
    ],
  };
}

describe("overview active agents", () => {
  it("does not present completed work as a running agent", () => {
    expect(buildActiveAgentRows([plan("fulfilled", "Prepare eve 0.5.0 release")])).toEqual([]);
  });

  it("uses only locked plans as the durable signal for active work", () => {
    expect(buildActiveAgentRows([plan("locked", "Fix live dashboard updates")])).toEqual([
      {
        agent: "Agent",
        label: "Fix live dashboard updates",
        repository: "eve",
      },
    ]);
  });
});
