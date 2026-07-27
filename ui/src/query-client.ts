import { QueryClient, type QueryClientConfig } from "@tanstack/react-query";

export const LIVE_REFETCH_INTERVAL_MS = 5_000;

const liveQueryDefaults = {
  refetchInterval: LIVE_REFETCH_INTERVAL_MS,
  refetchIntervalInBackground: true,
  refetchOnWindowFocus: "always" as const,
  refetchOnReconnect: "always" as const,
};

const liveQueryPrefixes = [
  ["snapshots"],
  ["repositories"],
  ["repository"],
  ["plan-requests"],
  ["pending-plan-requests"],
] as const;

export function createAppQueryClient(config?: QueryClientConfig) {
  const client = new QueryClient(config);
  for (const queryKey of liveQueryPrefixes) {
    client.setQueryDefaults(queryKey, liveQueryDefaults);
  }
  return client;
}
