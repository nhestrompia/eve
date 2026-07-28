import { describe, expect, it } from "vitest";
import { createAppQueryClient } from "./query-client";

describe("live query defaults", () => {
  it("does not poll repository resources", () => {
    const client = createAppQueryClient();
    for (const queryKey of [["config"], ["snapshots"], ["repositories"], ["repository", "eve"]]) {
      const options = client.defaultQueryOptions({
        queryKey,
        queryFn: async () => [],
      });

      expect(options.refetchInterval).toBeUndefined();
      expect(options.refetchIntervalInBackground).toBe(false);
      expect(options.refetchOnWindowFocus).toBe("always");
      expect(options.refetchOnReconnect).toBe("always");
    }
  });

  it("does not poll plan history", () => {
    const client = createAppQueryClient();
    const options = client.defaultQueryOptions({
      queryKey: ["plan-requests", "all"],
      queryFn: async () => [],
    });

    expect(options.refetchInterval).toBeUndefined();
    expect(options.refetchIntervalInBackground).toBe(false);
    expect(options.refetchOnWindowFocus).toBe("always");
    expect(options.refetchOnReconnect).toBe("always");
  });

  it("does not poll pending approval checks", () => {
    const client = createAppQueryClient();
    const options = client.defaultQueryOptions({
      queryKey: ["pending-plan-requests"],
      queryFn: async () => [],
    });

    expect(options.refetchInterval).toBeUndefined();
    expect(options.refetchIntervalInBackground).toBe(false);
    expect(options.refetchOnWindowFocus).toBe("always");
    expect(options.refetchOnReconnect).toBe("always");
  });
});
