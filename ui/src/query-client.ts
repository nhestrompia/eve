import { QueryClient, type QueryClientConfig } from "@tanstack/react-query";

const foregroundRefreshDefaults = {
  refetchIntervalInBackground: false,
  refetchOnWindowFocus: "always" as const,
  refetchOnReconnect: "always" as const,
};

const liveQueryPrefixes = [
  ["config"],
  ["snapshots"],
  ["repositories"],
  ["repository"],
  ["plan-requests"],
  ["pending-plan-requests"],
  ["agents"],
] as const;

export function createAppQueryClient(config?: QueryClientConfig) {
  const client = new QueryClient(config);
  for (const queryKey of liveQueryPrefixes) {
    client.setQueryDefaults(queryKey, {
      ...foregroundRefreshDefaults,
    });
  }
  return client;
}
