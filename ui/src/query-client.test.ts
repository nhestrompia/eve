import { describe, expect, it } from "vitest";
import { createAppQueryClient, LIVE_REFETCH_INTERVAL_MS } from "./query-client";

describe("live query defaults", () => {
  it("keeps snapshots, plans, and repository data fresh without a page refresh", () => {
    const client = createAppQueryClient();
    for (const queryKey of [["snapshots"], ["repositories"], ["repository", "eve"], ["plan-requests", "all"]]) {
      const options = client.defaultQueryOptions({
        queryKey,
        queryFn: async () => [],
      });

      expect(options.refetchInterval).toBe(LIVE_REFETCH_INTERVAL_MS);
      expect(options.refetchIntervalInBackground).toBe(true);
      expect(options.refetchOnWindowFocus).toBe("always");
      expect(options.refetchOnReconnect).toBe("always");
    }
  });
});
