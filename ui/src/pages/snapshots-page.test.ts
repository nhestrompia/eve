import { describe, expect, it } from "vitest";
import type { EvolutionSummary } from "../types";
import { filterSnapshots } from "./snapshots-page";

const baseSnapshot: EvolutionSummary = {
  id: "EV-001",
  repository: "eve",
  title: "Record product history",
  type: "change",
  status: "completed",
  outcome: "History can be reviewed.",
  snapshot: "abc123",
  commitCount: 1,
  decisionCount: 0,
  riskCount: 0,
  artifactCount: 0,
  failedValidationCount: 0,
  verificationState: "passed",
  verificationSummary: "verified",
  sessionProviders: ["codex"],
  createdAt: "2026-07-24T10:00:00Z",
  updatedAt: "2026-07-24T10:00:00Z",
};

describe("snapshot page filters", () => {
  it("filters snapshots by local search text and repository", () => {
    const rows = [
      baseSnapshot,
      {
        ...baseSnapshot,
        id: "EV-002",
        repository: "docs",
        title: "Document release flow",
        outcome: "Release notes are searchable.",
      },
    ];

    expect(
      filterSnapshots(rows, {
        query: "release",
        statusFilter: "all",
        agentFilter: "all",
        repositoryFilter: "docs",
        dateFilter: "all",
        sort: "newest",
      }).map((row) => row.id),
    ).toEqual(["EV-002"]);
  });

  it("filters by visual status and changes sort direction", () => {
    const rows = [
      baseSnapshot,
      {
        ...baseSnapshot,
        id: "EV-002",
        title: "Needs decision",
        failedValidationCount: 1,
        updatedAt: "2026-07-25T10:00:00Z",
      },
    ];

    expect(
      filterSnapshots(rows, {
        query: "",
        statusFilter: "waiting",
        agentFilter: "all",
        repositoryFilter: "all",
        dateFilter: "all",
        sort: "oldest",
      }).map((row) => row.id),
    ).toEqual(["EV-002"]);
  });
});
