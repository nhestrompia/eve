// @vitest-environment jsdom

import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from "@tanstack/react-router";
import { describe, expect, it, vi } from "vitest";
import {
  parseAcceptanceCriteria,
  pullRequestReadiness,
  PullRequestsBreadcrumbLink,
} from "./pull-request-page";
import type { PullRequestSummary } from "../types";

describe("PullRequestsBreadcrumbLink", () => {
  it("navigates through the client router to the repository PR tab", async () => {
    window.scrollTo = vi.fn();
    const rootRoute = createRootRoute();
    const repositoryRoute = createRoute({
      getParentRoute: () => rootRoute,
      path: "/repositories/$repo",
      component: () => React.createElement("p", null, "Repository"),
    });
    const pullRequestRoute = createRoute({
      getParentRoute: () => rootRoute,
      path: "/repositories/$repo/pull-requests/$number",
      component: () =>
        React.createElement(PullRequestsBreadcrumbLink, {
          repository: "eve",
        }),
    });
    const router = createRouter({
      routeTree: rootRoute.addChildren([repositoryRoute, pullRequestRoute]),
      history: createMemoryHistory({
        initialEntries: ["/repositories/eve/pull-requests/32"],
      }),
    });

    await router.load();
    render(
      React.createElement(RouterProvider, {
        router: router as never,
      }),
    );

    await userEvent.click(
      await screen.findByRole("link", { name: "Pull requests" }),
    );
    await waitFor(() => {
      expect(router.state.location.pathname).toBe("/repositories/eve");
      expect(router.state.location.hash).toBe("pull-requests");
    });
  });
});

describe("parseAcceptanceCriteria", () => {
  it("normalizes markdown bullets and task-list markers", () => {
    expect(
      parseAcceptanceCriteria(
        "- [x] Show a pull request count\n- [ ] Link exact-head Snapshots\n3. Keep code secondary",
      ),
    ).toEqual([
      "Show a pull request count",
      "Link exact-head Snapshots",
      "Keep code secondary",
    ]);
  });

  it("omits empty lines", () => {
    expect(parseAcceptanceCriteria("\n- First\n\n* Second\n")).toEqual([
      "First",
      "Second",
    ]);
  });
});

describe("pullRequestReadiness", () => {
  it("names a GitHub merge conflict after Snapshot freshness is established", () => {
    const readiness = pullRequestReadiness({
      snapshotId: "snap_current",
      snapshotHeadMatch: true,
      planRevision: 1,
      planValid: true,
      planAligned: false,
      eveChecksPassed: true,
      mergeability: "conflicting",
      baseBranch: "main",
    } as PullRequestSummary);

    expect(readiness).toMatchObject({
      badge: "Merge conflict",
      title: "Resolve the merge conflict",
      variant: "destructive",
    });
  });
});
