import { describe, expect, it } from "vitest";
import type { PlanRow } from "./plans-page";
import { filterPlanRows } from "./plans-page";

const basePlan: PlanRow = {
  id: "plan-1",
  title: "Add approval flow",
  summary: "Review and approve pending work.",
  repository: "eve",
  branch: "main",
  agent: "Codex",
  state: "pending_approval",
  sourceState: "pending_approval",
  statusLabel: "Waiting approval",
  statusTone: "waiting",
  updatedAt: "2026-07-24T10:00:00Z",
  files: ["ui/src/pages/plans-page.tsx"],
};

describe("plans page filters", () => {
  it("uses the page search to filter plan rows", () => {
    const rows: PlanRow[] = [
      basePlan,
      {
        ...basePlan,
        id: "plan-2",
        title: "Fix snapshots graph",
        summary: "Make graph nodes clickable.",
        state: "completed",
        sourceState: "fulfilled",
        statusLabel: "Completed",
        statusTone: "verified",
      },
    ];

    expect(
      filterPlanRows(rows, {
        activeTab: "all",
        query: "clickable",
        repositoryFilter: "all",
        agentFilter: "all",
        sort: "newest",
      }).map((row) => row.id),
    ).toEqual(["plan-2"]);
  });

  it("matches approved as an alias for locked plans and applies tab filters", () => {
    const lockedPlan: PlanRow = {
      ...basePlan,
      id: "plan-2",
      title: "Prepare release",
      state: "ready",
      sourceState: "locked",
      statusLabel: "Ready",
      statusTone: "ready",
    };

    expect(
      filterPlanRows([basePlan, lockedPlan], {
        activeTab: "ready",
        query: "approved",
        repositoryFilter: "all",
        agentFilter: "all",
        sort: "newest",
      }).map((row) => row.id),
    ).toEqual(["plan-2"]);
  });
});
