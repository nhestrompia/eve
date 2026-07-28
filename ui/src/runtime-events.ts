import type { QueryClient, QueryKey } from "@tanstack/react-query";

export type RuntimeEventSource = {
  addEventListener(name: string, listener: (event: MessageEvent) => void): void;
  removeEventListener(name: string, listener: (event: MessageEvent) => void): void;
  close(): void;
};

type RuntimeEventSourceConstructor = new (url: string) => RuntimeEventSource;

export type RuntimeVisibilitySource = {
  readonly hidden: boolean;
  addEventListener(name: "visibilitychange", listener: () => void): void;
  removeEventListener(name: "visibilitychange", listener: () => void): void;
};

type RuntimeEventEnvironment = {
  visibilitySource?: RuntimeVisibilitySource;
  coalesceMs?: number;
};

const queryKeysByEvent: Record<string, QueryKey[]> = {
  snapshots: [["snapshots"], ["repositories"], ["repository"], ["config"]],
  repositories: [["repositories"], ["repository"], ["config"], ["plan-requests"], ["pending-plan-requests"]],
  config: [["config"], ["repositories"], ["repository"], ["plan-requests"], ["pending-plan-requests"]],
  plans: [["plan-requests"], ["pending-plan-requests"], ["agents"]],
  agents: [["agents"]],
  verification: [["repositories"], ["repository"], ["config"]],
};

export function connectRuntimeEvents(
  queryClient: QueryClient,
  EventSourceConstructor: RuntimeEventSourceConstructor = EventSource,
  environment: RuntimeEventEnvironment = {},
) {
  const visibilitySource =
    environment.visibilitySource ??
    (typeof document === "undefined" ? undefined : document);
  const coalesceMs = environment.coalesceMs ?? 50;
  const listeners = new Map<string, (event: MessageEvent) => void>();
  const pendingKeys = new Map<string, QueryKey>();
  let pendingAll = false;
  let flushTimer: ReturnType<typeof setTimeout> | undefined;
  let source: RuntimeEventSource | undefined;
  let connectedOnce = false;

  const flush = () => {
    flushTimer = undefined;
    if (pendingAll) {
      pendingAll = false;
      pendingKeys.clear();
      void queryClient.invalidateQueries();
      return;
    }
    const queryKeys = [...pendingKeys.values()];
    pendingKeys.clear();
    for (const queryKey of queryKeys) {
      void queryClient.invalidateQueries({ queryKey });
    }
  };
  const scheduleFlush = () => {
    if (flushTimer === undefined) {
      flushTimer = setTimeout(flush, coalesceMs);
    }
  };
  const queueKeys = (queryKeys: QueryKey[]) => {
    if (pendingAll) return;
    for (const queryKey of queryKeys) {
      pendingKeys.set(JSON.stringify(queryKey), queryKey);
    }
    scheduleFlush();
  };
  const queueAll = () => {
    pendingAll = true;
    pendingKeys.clear();
    scheduleFlush();
  };
  const readyListener = () => {
    if (connectedOnce) {
      queueAll();
    }
    connectedOnce = true;
  };
  listeners.set("ready", readyListener);
  for (const [name, queryKeys] of Object.entries(queryKeysByEvent)) {
    const listener = () => {
      queueKeys(queryKeys);
    };
    listeners.set(name, listener);
  }
  const allListener = queueAll;
  listeners.set("all", allListener);

  const disconnectSource = () => {
    if (!source) return;
    for (const [name, listener] of listeners) {
      source.removeEventListener(name, listener);
    }
    source.close();
    source = undefined;
  };
  const connectSource = () => {
    if (source || visibilitySource?.hidden) return;
    source = new EventSourceConstructor("/api/events");
    for (const [name, listener] of listeners) {
      source.addEventListener(name, listener);
    }
  };
  const handleVisibilityChange = () => {
    if (visibilitySource?.hidden) {
      disconnectSource();
      if (flushTimer !== undefined) {
        clearTimeout(flushTimer);
        flushTimer = undefined;
      }
      pendingAll = false;
      pendingKeys.clear();
      return;
    }
    queueAll();
    connectSource();
  };

  visibilitySource?.addEventListener("visibilitychange", handleVisibilityChange);
  connectSource();

  return () => {
    visibilitySource?.removeEventListener("visibilitychange", handleVisibilityChange);
    disconnectSource();
    if (flushTimer !== undefined) {
      clearTimeout(flushTimer);
    }
  };
}
