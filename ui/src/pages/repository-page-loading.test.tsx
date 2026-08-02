// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, waitFor } from "@testing-library/react";
import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { RepositoryPage } from "./repository-page";

const apiMock = vi.hoisted(() => {
  const pending = () => new Promise<never>(() => undefined);
  return {
    repositories: vi.fn(pending),
    snapshots: vi.fn(pending),
    repository: vi.fn(pending),
    pullRequests: vi.fn(pending),
    snapshotDetail: vi.fn(pending),
  };
});

vi.mock("../api", () => ({ api: apiMock }));

vi.mock("@tanstack/react-router", () => ({
  Link: () => null,
  useParams: () => ({ repo: "eve" }),
}));

describe("repository page loading", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("only requests critical repository data on the initial render", async () => {
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      <QueryClientProvider client={client}>
        <RepositoryPage />
      </QueryClientProvider>,
    );

    await waitFor(() => {
      expect(apiMock.snapshots).toHaveBeenCalledWith("eve");
      expect(apiMock.repository).toHaveBeenCalledWith("eve");
      expect(apiMock.pullRequests).toHaveBeenCalledWith("eve");
    });

    expect(apiMock.snapshots).toHaveBeenCalledTimes(1);
    expect(apiMock.repositories).not.toHaveBeenCalled();
    expect(apiMock.snapshotDetail).not.toHaveBeenCalled();
  });
});
