import { QueryClient, type QueryClientConfig } from "@tanstack/react-query";

export const RESOURCE_REFETCH_INTERVAL_MS = 30_000;
export const PLAN_REFETCH_INTERVAL_MS = 10_000;
export const PENDING_PLAN_REFETCH_INTERVAL_MS = 2_000;

const foregroundRefreshDefaults = {
  refetchIntervalInBackground: false,
  refetchOnWindowFocus: "always" as const,
  refetchOnReconnect: "always" as const,
};

const resourceQueryPrefixes = [
  ["config"],
  ["snapshots"],
  ["repositories"],
  ["repository"],
] as const;

const planQueryPrefixes = [
  ["plan-requests"],
] as const;

const pendingPlanQueryPrefixes = [
  ["pending-plan-requests"],
] as const;

export function createAppQueryClient(config?: QueryClientConfig) {
  const client = new QueryClient(config);
  for (const queryKey of resourceQueryPrefixes) {
    client.setQueryDefaults(queryKey, {
      ...foregroundRefreshDefaults,
      refetchInterval: RESOURCE_REFETCH_INTERVAL_MS,
    });
  }
  for (const queryKey of planQueryPrefixes) {
    client.setQueryDefaults(queryKey, {
      ...foregroundRefreshDefaults,
      refetchInterval: PLAN_REFETCH_INTERVAL_MS,
    });
  }
  for (const queryKey of pendingPlanQueryPrefixes) {
    client.setQueryDefaults(queryKey, {
      ...foregroundRefreshDefaults,
      refetchInterval: PENDING_PLAN_REFETCH_INTERVAL_MS,
    });
  }
  return client;
}
