// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  ArtifactCard,
  ArtifactLogContent,
  type ArtifactCardRow,
} from "./repository-page";

function renderWithQueryClient(element: React.ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>{element}</QueryClientProvider>,
  );
}

function artifact(overrides: Partial<ArtifactCardRow> = {}): ArtifactCardRow {
  return {
    id: "snapshot-1-0",
    type: "note",
    description: "Package evidence",
    kind: "file",
    ...overrides,
  };
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("ArtifactCard", () => {
  it("opens the artifact modal when the card body is clicked", async () => {
    const onPreview = vi.fn();
    renderWithQueryClient(
      <ArtifactCard artifact={artifact()} onPreview={onPreview} />,
    );

    await userEvent.click(
      screen.getByRole("button", { name: "Open Package evidence" }),
    );

    expect(onPreview).toHaveBeenCalledWith(artifact());
  });
});

describe("ArtifactLogContent", () => {
  it("retries without a range when the server cannot satisfy the preview range", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({ ok: false, status: 416 })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        text: async () => "The log is readable.",
      });
    vi.stubGlobal("fetch", fetchMock);

    renderWithQueryClient(
      <ArtifactLogContent
        artifact={artifact({
          type: "log",
          description: "Runtime log",
          href: "/api/repos/eve/files/output/runtime.log",
          kind: "log",
        })}
      />,
    );

    expect(await screen.findByText("The log is readable.")).toBeTruthy();
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      "/api/repos/eve/files/output/runtime.log",
    );
  });
});
