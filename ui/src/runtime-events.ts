import type { QueryClient, QueryKey } from "@tanstack/react-query";

export type RuntimeEventSource = {
  addEventListener(name: string, listener: (event: MessageEvent) => void): void;
  removeEventListener(name: string, listener: (event: MessageEvent) => void): void;
  close(): void;
};

type RuntimeEventSourceConstructor = new (url: string) => RuntimeEventSource;

const queryKeysByEvent: Record<string, QueryKey[]> = {
  snapshots: [["snapshots"], ["repositories"], ["repository"], ["config"]],
  repositories: [["repositories"], ["repository"], ["config"], ["plan-requests"], ["pending-plan-requests"]],
  config: [["config"], ["repositories"], ["repository"], ["plan-requests"], ["pending-plan-requests"]],
  plans: [["plan-requests"], ["pending-plan-requests"], ["agents"]],
  agents: [["agents"], ["repositories"], ["repository"], ["config"]],
  verification: [["repositories"], ["repository"], ["config"]],
};

export function connectRuntimeEvents(
  queryClient: QueryClient,
  EventSourceConstructor: RuntimeEventSourceConstructor = EventSource,
) {
  const source = new EventSourceConstructor("/api/events");
  const listeners = new Map<string, (event: MessageEvent) => void>();
  let connected = false;
  const readyListener = () => {
    if (connected) {
      void queryClient.invalidateQueries();
      return;
    }
    connected = true;
  };
  listeners.set("ready", readyListener);
  source.addEventListener("ready", readyListener);
  for (const [name, queryKeys] of Object.entries(queryKeysByEvent)) {
    const listener = () => {
      for (const queryKey of queryKeys) {
        void queryClient.invalidateQueries({ queryKey });
      }
    };
    listeners.set(name, listener);
    source.addEventListener(name, listener);
  }
  const allListener = () => {
    void queryClient.invalidateQueries();
  };
  listeners.set("all", allListener);
  source.addEventListener("all", allListener);

  return () => {
    for (const [name, listener] of listeners) {
      source.removeEventListener(name, listener);
    }
    source.close();
  };
}
