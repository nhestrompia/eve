import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import { connectRuntimeEvents, type RuntimeEventSource } from "./runtime-events";

class FakeEventSource implements RuntimeEventSource {
  static instance: FakeEventSource;
  listeners = new Map<string, Set<(event: MessageEvent) => void>>();
  close = vi.fn();

  constructor(readonly url: string) {
    FakeEventSource.instance = this;
  }

  addEventListener(name: string, listener: (event: MessageEvent) => void) {
    const listeners = this.listeners.get(name) ?? new Set();
    listeners.add(listener);
    this.listeners.set(name, listeners);
  }

  removeEventListener(name: string, listener: (event: MessageEvent) => void) {
    this.listeners.get(name)?.delete(listener);
  }

  emit(name: string, data = "{}") {
    for (const listener of this.listeners.get(name) ?? []) {
      listener(new MessageEvent(name, { data }));
    }
  }
}

describe("runtime events", () => {
  it("invalidates only the query families affected by an event", async () => {
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries").mockResolvedValue();
    const disconnect = connectRuntimeEvents(client, FakeEventSource);
    const source = FakeEventSource.instance;

    expect(source.url).toBe("/api/events");
    source.emit("snapshots", '{"repository":"eve"}');
    await vi.waitFor(() => expect(invalidate).toHaveBeenCalled());

    expect(invalidate.mock.calls.map(([filters]) => filters?.queryKey)).toEqual([
      ["snapshots"],
      ["repositories"],
      ["repository"],
      ["config"],
    ]);

    disconnect();
    expect(source.close).toHaveBeenCalledOnce();
  });

  it("maps plan and agent events without invalidating snapshots", async () => {
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries").mockResolvedValue();
    const disconnect = connectRuntimeEvents(client, FakeEventSource);

    FakeEventSource.instance.emit("plans");
    await vi.waitFor(() => expect(invalidate).toHaveBeenCalled());

    expect(invalidate.mock.calls.map(([filters]) => filters?.queryKey)).toEqual([
      ["plan-requests"],
      ["pending-plan-requests"],
      ["agents"],
    ]);
    disconnect();
  });

  it("recovers missed events after EventSource reconnects", async () => {
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries").mockResolvedValue();
    const disconnect = connectRuntimeEvents(client, FakeEventSource);

    FakeEventSource.instance.emit("ready");
    expect(invalidate).not.toHaveBeenCalled();
    FakeEventSource.instance.emit("ready");
    await vi.waitFor(() => expect(invalidate).toHaveBeenCalledOnce());
    expect(invalidate).toHaveBeenCalledWith();
    disconnect();
  });
});
