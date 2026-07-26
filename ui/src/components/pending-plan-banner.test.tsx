import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type React from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import type { PlanRequest } from "../types";
import {
  PendingPlanBanner,
  currentRevision,
  planToProposal,
  proposalValidationMessage,
} from "./pending-plan-banner";

function plan(id: string, repository: string, goal: string, state = "pending_approval"): PlanRequest {
  return {
    planRequestId: id,
    repository,
    repositoryRoot: `/tmp/${repository}`,
    branch: "main",
    state,
    currentRevision: 1,
    staleReasons: state === "stale" ? ["repository HEAD changed"] : undefined,
    availableSuites: ["fast"],
    revisions: [
      {
        revision: 1,
        source: "agent",
        goal,
        acceptanceCriteria: "- It works",
        allowedPathGlobs: ["src/**"],
        milestones: [{ title: "Build", goal: "Ship the change" }],
        configuredSuite: "fast",
        resolvedCheckIds: ["unit"],
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

function renderWithQueryClient(element: React.ReactElement) {
  const client = new QueryClient();
  return renderToStaticMarkup(
    <QueryClientProvider client={client}>{element}</QueryClientProvider>,
  );
}

describe("PendingPlanBanner", () => {
  it("shows multiple waiting plans without hiding their repository context", () => {
    const html = renderWithQueryClient(
      <PendingPlanBanner
        plans={[
          plan("planreq_one", "eve", "Improve approvals"),
          plan("planreq_two", "astronomy", "Add a star map"),
        ]}
      />,
    );

    expect(html).toContain("2 plans are waiting for you");
    expect(html).toContain("Review plans");
    expect(html).toContain("eve");
    expect(html).toContain("Improve approvals");
    expect(html).toContain("astronomy");
    expect(html).toContain("Add a star map");
  });

  it("renders nothing for an empty queue", () => {
    expect(renderWithQueryClient(<PendingPlanBanner plans={[]} />)).toBe("");
  });

  it("keeps stale plans in the review queue so they can be removed", () => {
    const html = renderWithQueryClient(
      <PendingPlanBanner plans={[plan("planreq_stale", "eve", "Old plan", "stale")]} />,
    );

    expect(html).toContain("1 plan is waiting for you");
    expect(html).toContain("Old plan");
  });

  it("builds edited approval proposals from the current revision", () => {
    const request = plan("planreq_one", "eve", "Improve approvals");

    expect(currentRevision(request)?.goal).toBe("Improve approvals");
    expect(planToProposal(request)).toMatchObject({
      goal: "Improve approvals",
      acceptanceCriteria: "- It works",
      allowedPathGlobs: ["src/**"],
      requiredSuite: "fast",
    });
  });

  it("requires goal, criteria, and scope for edited approvals", () => {
    expect(
      proposalValidationMessage({
        goal: "",
        acceptanceCriteria: "- It works",
        allowedPathGlobs: ["src/**"],
        milestones: [],
      }),
    ).toBe("Goal is required.");
    expect(
      proposalValidationMessage({
        goal: "Ship",
        acceptanceCriteria: " ",
        allowedPathGlobs: ["src/**"],
        milestones: [],
      }),
    ).toBe("Acceptance criteria are required.");
    expect(
      proposalValidationMessage({
        goal: "Ship",
        acceptanceCriteria: "- It works",
        allowedPathGlobs: [" "],
        milestones: [],
      }),
    ).toBe("At least one allowed path glob is required.");
  });
});
