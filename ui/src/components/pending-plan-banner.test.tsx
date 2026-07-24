import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import type { PlanRequest } from "../types";
import { PendingPlanBanner } from "./pending-plan-banner";

function plan(id: string, repository: string, goal: string): PlanRequest {
  return {
    planRequestId: id,
    repository,
    repositoryRoot: `/tmp/${repository}`,
    branch: "main",
    state: "pending_approval",
    currentRevision: 1,
    revisions: [
      {
        revision: 1,
        source: "agent",
        goal,
        acceptanceCriteria: "- It works",
        allowedPathGlobs: ["src/**"],
        milestones: [],
        resolvedCheckIds: [],
        policyHash: "",
        checkDefinitionsHash: "",
        suiteDigest: "",
        baseCommit: "abc",
        branch: "main",
        createdAt: "2026-07-24T00:00:00Z",
      },
    ],
  };
}

describe("PendingPlanBanner", () => {
  it("shows multiple waiting plans without hiding their repository context", () => {
    const html = renderToStaticMarkup(
      <PendingPlanBanner
        plans={[
          plan("planreq_one", "eve", "Improve approvals"),
          plan("planreq_two", "astronomy", "Add a star map"),
        ]}
      />,
    );

    expect(html).toContain("2 plans are waiting for you");
    expect(html).toContain("eve");
    expect(html).toContain("Improve approvals");
    expect(html).toContain("astronomy");
    expect(html).toContain("Add a star map");
  });

  it("renders nothing for an empty queue", () => {
    expect(renderToStaticMarkup(<PendingPlanBanner plans={[]} />)).toBe("");
  });
});
