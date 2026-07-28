import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  connectRuntimeEvents,
  type RuntimeEventSource,
  type RuntimeVisibilitySource,
} from "./runtime-events";

class FakeEventSource implements RuntimeEventSource {
  static instance: FakeEventSource;
  static instances: FakeEventSource[] = [];
  listeners = new Map<string, Set<(event: MessageEvent) => void>>();
  close = vi.fn();

  constructor(readonly url: string) {
    FakeEventSource.instance = this;
    FakeEventSource.instances.push(this);
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

class FakeVisibilitySource implements RuntimeVisibilitySource {
  listeners = new Set<() => void>();

  constructor(public hidden: boolean) {}

  addEventListener(_name: "visibilitychange", listener: () => void) {
    this.listeners.add(listener);
  }

  removeEventListener(_name: "visibilitychange", listener: () => void) {
    this.listeners.delete(listener);
  }

  setHidden(hidden: boolean) {
    this.hidden = hidden;
    for (const listener of this.listeners) listener();
  }
}

describe("runtime events", () => {
  beforeEach(() => {
    FakeEventSource.instances = [];
  });

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

  it("coalesces overlapping event bursts into one invalidation per query family", async () => {
    vi.useFakeTimers();
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries").mockResolvedValue();
    const disconnect = connectRuntimeEvents(client, FakeEventSource);

    FakeEventSource.instance.emit("repositories");
    FakeEventSource.instance.emit("snapshots");
    expect(invalidate).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(50);

    const keys = invalidate.mock.calls.map(([filters]) => JSON.stringify(filters?.queryKey));
    expect(keys).toHaveLength(new Set(keys).size);
    expect(keys).toContain(JSON.stringify(["repositories"]));
    expect(keys).toContain(JSON.stringify(["snapshots"]));

    disconnect();
    vi.useRealTimers();
  });

  it("disconnects hidden tabs and performs one catch-up refresh when visible", async () => {
    vi.useFakeTimers();
    const client = new QueryClient();
    const invalidate = vi.spyOn(client, "invalidateQueries").mockResolvedValue();
    const visibility = new FakeVisibilitySource(true);
    const disconnect = connectRuntimeEvents(client, FakeEventSource, {
      visibilitySource: visibility,
    });

    expect(FakeEventSource.instances).toHaveLength(0);
    visibility.setHidden(false);
    expect(FakeEventSource.instances).toHaveLength(1);
    await vi.advanceTimersByTimeAsync(50);
    expect(invalidate).toHaveBeenCalledOnce();
    expect(invalidate).toHaveBeenCalledWith();

    const firstSource = FakeEventSource.instance;
    invalidate.mockClear();
    visibility.setHidden(true);
    expect(firstSource.close).toHaveBeenCalledOnce();
    firstSource.emit("repositories");
    await vi.advanceTimersByTimeAsync(50);
    expect(invalidate).not.toHaveBeenCalled();

    visibility.setHidden(false);
    expect(FakeEventSource.instances).toHaveLength(2);
    await vi.advanceTimersByTimeAsync(50);
    expect(invalidate).toHaveBeenCalledOnce();
    expect(invalidate).toHaveBeenCalledWith();

    disconnect();
    vi.useRealTimers();
  });
});
