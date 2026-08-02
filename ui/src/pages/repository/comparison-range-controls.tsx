import { ArrowRight, Search } from "lucide-react";
import { useMemo } from "react";

import type { EvolutionSummary } from "../../types";
import { ComparisonEndpointCard } from "./comparison-endpoint-card";
import { ComparisonSnapshotResult } from "./comparison-snapshot-result";
import { snapshotMatchesQuery } from "./comparison-utils";
import { RangePresetButton } from "./range-preset-button";

export function ComparisonRangeControls({
  evolutions,
  fromId,
  toId,
  fromSnapshot,
  toSnapshot,
  includedSnapshots,
  activeEndpoint,
  query,
  onQueryChange,
  onEndpointChange,
  onPickSnapshot,
  onPreset,
}: {
  evolutions: EvolutionSummary[];
  fromId: string;
  toId: string;
  fromSnapshot?: EvolutionSummary;
  toSnapshot?: EvolutionSummary;
  includedSnapshots: EvolutionSummary[];
  activeEndpoint: "from" | "to";
  query: string;
  onQueryChange: (query: string) => void;
  onEndpointChange: (endpoint: "from" | "to") => void;
  onPickSnapshot: (id: string) => void;
  onPreset: (count: number) => void;
}): React.JSX.Element {
  const newestFirst = useMemo(() => [...evolutions].reverse(), [evolutions]);
  const normalizedQuery = query.trim().toLowerCase();
  const matchingSnapshots = useMemo(() => {
    if (!normalizedQuery) return newestFirst.slice(0, 10);
    return newestFirst
      .filter((snapshot) => snapshotMatchesQuery(snapshot, normalizedQuery))
      .slice(0, 12);
  }, [newestFirst, normalizedQuery]);
  const totalMatches = normalizedQuery
    ? newestFirst.filter((snapshot) => snapshotMatchesQuery(snapshot, normalizedQuery)).length
    : newestFirst.length;

  return (
    <div className="mt-6 space-y-4">
      <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_260px]">
        <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] md:items-stretch">
          <ComparisonEndpointCard
            label="Earlier"
            snapshot={fromSnapshot}
            active={activeEndpoint === "from"}
            onClick={() => onEndpointChange("from")}
          />
          <div className="hidden items-center px-1 text-muted-foreground md:flex">
            <ArrowRight className="size-4" />
          </div>
          <ComparisonEndpointCard
            label="Later"
            snapshot={toSnapshot}
            active={activeEndpoint === "to"}
            onClick={() => onEndpointChange("to")}
          />
        </div>

        <div className="rounded-lg bg-secondary p-3 shadow-[var(--shadow-border)]">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              Quick ranges
            </span>
            <span className="text-xs text-muted-foreground">
              {includedSnapshots.length} selected
            </span>
          </div>
          <div className="mt-3 grid grid-cols-2 gap-2">
            <RangePresetButton onClick={() => onPreset(2)}>Latest pair</RangePresetButton>
            <RangePresetButton onClick={() => onPreset(3)}>Last 3</RangePresetButton>
            <RangePresetButton onClick={() => onPreset(5)}>Last 5</RangePresetButton>
            <RangePresetButton onClick={() => onPreset(evolutions.length)}>All</RangePresetButton>
          </div>
        </div>
      </div>

      <div className="rounded-lg bg-secondary p-3 shadow-[var(--shadow-border)]">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <label className="relative min-w-0 flex-1">
            <span className="sr-only">Search snapshots</span>
            <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <input
              value={query}
              onChange={(event) => onQueryChange(event.target.value)}
              placeholder={`Search snapshots for ${activeEndpoint === "from" ? "Earlier" : "Later"}`}
              className="min-h-11 w-full rounded-md bg-card pl-10 pr-3 text-sm font-medium text-foreground shadow-[var(--shadow-border)] outline-none transition-[box-shadow] focus:shadow-[0_0_0_2px_color-mix(in_oklch,var(--ring)_45%,transparent)]"
            />
          </label>
          <span className="shrink-0 text-xs text-muted-foreground">
            Showing {matchingSnapshots.length} of {totalMatches}
          </span>
        </div>

        <div className="mt-3 max-h-[220px] overflow-y-auto pr-1">
          {matchingSnapshots.length === 0 ? (
            <p className="rounded-md bg-card p-3 text-sm text-muted-foreground shadow-[var(--shadow-border)]">
              No snapshots match this search.
            </p>
          ) : (
            <div className="grid gap-2 md:grid-cols-2">
              {matchingSnapshots.map((snapshot) => (
                <ComparisonSnapshotResult
                  key={snapshot.id}
                  snapshot={snapshot}
                  selectedAs={
                    snapshot.id === fromId
                      ? "Earlier"
                      : snapshot.id === toId
                        ? "Later"
                        : undefined
                  }
                  onClick={() => onPickSnapshot(snapshot.id)}
                />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
