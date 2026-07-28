import { describe, expect, it } from "vitest";
import {
  createAppQueryClient,
  PLAN_REFETCH_INTERVAL_MS,
  PENDING_PLAN_REFETCH_INTERVAL_MS,
  RESOURCE_REFETCH_INTERVAL_MS,
} from "./query-client";

describe("live query defaults", () => {
  it("keeps repository resources fresh without aggressive background polling", () => {
    const client = createAppQueryClient();
    for (const queryKey of [["config"], ["snapshots"], ["repositories"], ["repository", "eve"]]) {
      const options = client.defaultQueryOptions({
        queryKey,
        queryFn: async () => [],
      });

      expect(options.refetchInterval).toBe(RESOURCE_REFETCH_INTERVAL_MS);
      expect(options.refetchIntervalInBackground).toBe(false);
      expect(options.refetchOnWindowFocus).toBe("always");
      expect(options.refetchOnReconnect).toBe("always");
    }
  });

  it("keeps plan history current without using the fastest approval poll", () => {
    const client = createAppQueryClient();
    const options = client.defaultQueryOptions({
      queryKey: ["plan-requests", "all"],
      queryFn: async () => [],
    });

    expect(options.refetchInterval).toBe(PLAN_REFETCH_INTERVAL_MS);
    expect(options.refetchIntervalInBackground).toBe(false);
    expect(options.refetchOnWindowFocus).toBe("always");
    expect(options.refetchOnReconnect).toBe("always");
  });

  it("keeps pending approval checks fast for interactive plan review", () => {
    const client = createAppQueryClient();
    const options = client.defaultQueryOptions({
      queryKey: ["pending-plan-requests"],
      queryFn: async () => [],
    });

    expect(options.refetchInterval).toBe(PENDING_PLAN_REFETCH_INTERVAL_MS);
    expect(options.refetchIntervalInBackground).toBe(false);
    expect(options.refetchOnWindowFocus).toBe("always");
    expect(options.refetchOnReconnect).toBe("always");
  });
});
