import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import {
  ArrowRight,
  GitCompareArrows,
  RotateCcw,
  Search,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { api } from "../../api";
import { LoadingState } from "../../components/loading-state";
import { Button } from "../../components/ui/button";
import { compactDate, humanDate, statusLabel } from "../../format";
import {
  compareEvolutionOrder,
  defaultComparisonPair,
  orderedComparisonPair,
} from "../../lib/comparison";
import type {
  ComparisonResponse,
  ComparisonTimelineItem,
  EvolutionSummary,
} from "../../types";

export function RepositoryComparePanel({
  evolutions,
}: {
  evolutions: EvolutionSummary[];
}): React.JSX.Element {
  const [fromId, setFromId] = useState("");
  const [toId, setToId] = useState("");
  const [activeEndpoint, setActiveEndpoint] = useState<"from" | "to">("from");
  const [snapshotQuery, setSnapshotQuery] = useState("");
  const chronological = useMemo(
    () => [...evolutions].sort(compareEvolutionOrder),
    [evolutions],
  );
  useEffect(() => {
    if (chronological.length < 2) {
      setFromId("");
      setToId("");
      return;
    }
    const ids = new Set(chronological.map((evolution) => evolution.id));
    if (ids.has(fromId) && ids.has(toId)) return;
    const pair = defaultComparisonPair(chronological);
    setFromId(pair?.from ?? "");
    setToId(pair?.to ?? "");
  }, [chronological, fromId, toId]);

  const orderedPair = orderedComparisonPair(chronological, fromId, toId);
  const comparisonFromId = orderedPair?.from ?? "";
  const comparisonToId = orderedPair?.to ?? "";

  const comparison = useQuery({
    queryKey: ["compare", comparisonFromId, comparisonToId],
    queryFn: () => api.compare(comparisonFromId, comparisonToId),
    enabled: Boolean(comparisonFromId && comparisonToId),
  });
  const fromSnapshot = chronological.find(
    (evolution) => evolution.id === comparisonFromId,
  );
  const toSnapshot = chronological.find(
    (evolution) => evolution.id === comparisonToId,
  );
  const fromIndex = chronological.findIndex(
    (evolution) => evolution.id === comparisonFromId,
  );
  const toIndex = chronological.findIndex(
    (evolution) => evolution.id === comparisonToId,
  );
  const includedSnapshots =
    fromIndex >= 0 && toIndex >= 0
      ? chronological.slice(fromIndex, toIndex + 1)
      : [];

  const setComparisonPair = (firstId: string, secondId: string) => {
    const next = orderedComparisonPair(chronological, firstId, secondId);
    if (!next) return;
    setFromId(next.from);
    setToId(next.to);
  };

  const pickSnapshot = (id: string) => {
    if (activeEndpoint === "from") {
      setComparisonPair(id, comparisonToId || toId);
      setActiveEndpoint("to");
      return;
    }
    setComparisonPair(comparisonFromId || fromId, id);
    setActiveEndpoint("from");
  };

  const resetToLatestRange = () => {
    const pair = defaultComparisonPair(chronological);
    if (pair) {
      setFromId(pair.from);
      setToId(pair.to);
    }
  };

  if (chronological.length < 2) {
    return (
      <section className="rounded-lg bg-card p-5 shadow-[var(--shadow-border)]">
        <h2 className="flex items-center gap-2 text-base font-semibold">
          <GitCompareArrows className="size-4 text-blue-600" />
          Compare snapshots
        </h2>
        <p className="mt-3 text-sm text-muted-foreground">
          Record at least two snapshots in this repository to compare product states.
        </p>
      </section>
    );
  }

  return (
    <section className="space-y-4">
      <section className="overflow-hidden rounded-lg bg-card text-card-foreground shadow-[var(--shadow-border)]">
        <div className="p-5 sm:p-6">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div className="max-w-[64ch]">
              <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                Product range
              </p>
              <h2 className="mt-2 flex items-center gap-2 text-xl font-semibold text-balance">
                <GitCompareArrows className="size-5 text-blue-600" />
                Compare snapshots
              </h2>
              <p className="mt-3 text-sm leading-6 text-muted-foreground text-pretty">
                Choose an endpoint, then search snapshots by title, type, date,
                or id. EVE keeps the final range in chronological order.
              </p>
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="min-h-10 shrink-0 gap-2 transition-[scale,background-color,box-shadow] active:scale-[0.96]"
              onClick={resetToLatestRange}
            >
              <RotateCcw className="size-3.5" />
              Latest range
            </Button>
          </div>

          <ComparisonRangeControls
            evolutions={chronological}
            fromId={comparisonFromId}
            toId={comparisonToId}
            fromSnapshot={fromSnapshot}
            toSnapshot={toSnapshot}
            includedSnapshots={includedSnapshots}
            activeEndpoint={activeEndpoint}
            query={snapshotQuery}
            onQueryChange={setSnapshotQuery}
            onEndpointChange={setActiveEndpoint}
            onPickSnapshot={pickSnapshot}
            onPreset={(count) => {
              const pair = comparisonPairByCount(chronological, count);
              if (pair) {
                setFromId(pair.from);
                setToId(pair.to);
              }
            }}
          />
        </div>
      </section>

      <div className="min-w-0">
        {!orderedPair ? (
          <div className="rounded-lg bg-card p-5 text-sm text-muted-foreground shadow-[var(--shadow-border)]">
            Choose two different snapshots to compare.
          </div>
        ) : null}
        {comparison.isLoading ? <LoadingState label="Comparing Snapshots" /> : null}
        {comparison.error ? <ComparisonInlineError error={comparison.error} /> : null}
        {comparison.data ? <ComparisonBoard comparison={comparison.data} /> : null}
      </div>
    </section>
  );
}

function ComparisonEndpointCard({
  label,
  snapshot,
  active,
  onClick,
}: {
  label: string;
  snapshot?: EvolutionSummary;
  active: boolean;
  onClick: () => void;
}): React.JSX.Element {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`min-h-[96px] rounded-lg bg-card p-4 text-left shadow-[var(--shadow-border)] transition-[scale,background-color,box-shadow] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.98] ${
        active ? "bg-accent text-accent-foreground" : "hover:bg-secondary"
      }`}
    >
      <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {label}
      </p>
      <h3 className="mt-2 truncate text-base font-semibold">
        {snapshot?.title ?? "Select snapshot"}
      </h3>
      <p className="mt-1 text-sm text-muted-foreground">
        {snapshot
          ? `${statusLabel(snapshot.type)} · ${humanDate(snapshot.createdAt)}`
          : "Choose a snapshot"}
      </p>
    </button>
  );
}

function ComparisonRangeControls({
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

function ComparisonSnapshotResult({
  snapshot,
  selectedAs,
  onClick,
}: {
  snapshot: EvolutionSummary;
  selectedAs?: "Earlier" | "Later";
  onClick: () => void;
}): React.JSX.Element {
  return (
    <button
      type="button"
      onClick={onClick}
      className="grid min-h-14 w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-md bg-card px-3 py-2.5 text-left shadow-[var(--shadow-border)] transition-[scale,background-color,box-shadow] hover:bg-background active:scale-[0.98]"
    >
      <span className="min-w-0">
        <span className="block truncate text-sm font-semibold">{snapshot.title}</span>
        <span className="mt-1 block truncate text-xs text-muted-foreground">
          {snapshot.type} · {compactDate(snapshot.createdAt)} · {snapshot.id}
        </span>
      </span>
      {selectedAs ? (
        <span className="rounded-md bg-accent px-2 py-1 text-xs font-semibold text-accent-foreground">
          {selectedAs}
        </span>
      ) : (
        <span className="text-xs font-medium text-muted-foreground">Pick</span>
      )}
    </button>
  );
}

function RangePresetButton({
  children,
  onClick,
}: {
  children: React.ReactNode;
  onClick: () => void;
}): React.JSX.Element {
  return (
    <button
      type="button"
      onClick={onClick}
      className="rounded-md bg-card px-3 py-2 text-xs font-semibold text-foreground shadow-[var(--shadow-border)] transition-[scale,background-color,box-shadow] hover:bg-accent hover:text-accent-foreground active:scale-[0.97]"
    >
      {children}
    </button>
  );
}

function snapshotMatchesQuery(snapshot: EvolutionSummary, query: string): boolean {
  const haystack = [
    snapshot.title,
    snapshot.id,
    snapshot.type,
    snapshot.status,
    snapshot.repository,
    snapshot.createdAt,
    compactDate(snapshot.createdAt),
  ]
    .join(" ")
    .toLowerCase();
  return haystack.includes(query);
}

function ComparisonIncludedStrip({
  snapshots,
}: {
  snapshots: EvolutionSummary[];
}): React.JSX.Element {
  return (
    <div className="xl:col-span-2">
      <div className="flex gap-2 overflow-x-auto pb-1">
        {snapshots.map((snapshot, index) => (
          <Link
            key={snapshot.id}
            to="/snapshots/$id"
            params={{ id: snapshot.id }}
            className="min-w-[180px] rounded-lg bg-card px-3 py-2 shadow-[var(--shadow-border)] transition-[background-color,box-shadow] hover:bg-secondary"
          >
            <span className="text-[11px] font-semibold uppercase tracking-wide text-primary">
              {index + 1}
            </span>
            <span className="mt-1 block truncate text-xs font-semibold text-foreground">
              {snapshot.title}
            </span>
            <span className="mt-1 block text-[11px] text-muted-foreground">
              {snapshot.type} · {compactDate(snapshot.createdAt)}
            </span>
          </Link>
        ))}
      </div>
    </div>
  );
}

function comparisonPairByCount(
  evolutions: EvolutionSummary[],
  count: number,
): { from: string; to: string } | undefined {
  if (evolutions.length < 2) return undefined;
  const size = Math.max(2, Math.min(count, evolutions.length));
  const to = evolutions[evolutions.length - 1];
  const from = evolutions[evolutions.length - size];
  return { from: from.id, to: to.id };
}

type ComparisonPanelItem = {
  key: string;
  snapshotId: string;
  title: string;
  meta?: string;
};

function ComparisonBoard({
  comparison,
}: {
  comparison: ComparisonResponse;
}): React.JSX.Element {
  const totalChanges =
    comparison.added.length +
    comparison.changed.length +
    comparison.fixed.length;
  const selectedSnapshotCount = comparison.range.length + 1;

  return (
    <section className="space-y-4">
      <div className="space-y-4">
        <div className="rounded-lg bg-card p-5 shadow-[var(--shadow-border)]">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="max-w-[68ch]">
              <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                Comparison summary
              </p>
              <h3 className="mt-1 text-xl font-semibold text-balance">
                {comparison.from.title} to {comparison.to.title}
              </h3>
              <p className="mt-2 text-sm leading-6 text-muted-foreground text-pretty">
                Product changes, decisions, risks, validation, and timeline
                entries are grouped so the delta is visible without reading the
                full record stream.
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              <ComparisonStat label="Selected" value={selectedSnapshotCount} />
              <ComparisonStat label="Changes" value={totalChanges} />
              <ComparisonStat label="Decisions" value={comparison.decisions.length} />
              <ComparisonStat label="Risks" value={comparison.risks.length} />
            </div>
          </div>
        </div>

        <div className="grid gap-4 lg:grid-cols-3">
          <ComparisonListPanel
            title="Added"
            count={comparison.added.length}
            empty="No added product changes."
            items={comparison.added.map((item) => ({
              key: `${item.snapshotId}-${item.text}`,
              snapshotId: item.snapshotId,
              title: item.text,
              meta: `${item.snapshotTitle} · ${humanDate(item.createdAt)}`,
            }))}
          />
          <ComparisonListPanel
            title="Changed"
            count={comparison.changed.length}
            empty="No changed product behavior."
            items={comparison.changed.map((item) => ({
              key: `${item.snapshotId}-${item.text}`,
              snapshotId: item.snapshotId,
              title: item.text,
              meta: `${item.snapshotTitle} · ${humanDate(item.createdAt)}`,
            }))}
          />
          <ComparisonListPanel
            title="Fixed"
            count={comparison.fixed.length}
            empty="No fixes in this span."
            items={comparison.fixed.map((item) => ({
              key: `${item.snapshotId}-${item.text}`,
              snapshotId: item.snapshotId,
              title: item.text,
              meta: `${item.snapshotTitle} · ${humanDate(item.createdAt)}`,
            }))}
          />
        </div>

        <ComparisonTimelineCompact items={comparison.timeline} />
      </div>

      <aside className="grid gap-4 lg:grid-cols-3">
        <ComparisonListPanel
          title="Decisions"
          count={comparison.decisions.length}
          empty="No decisions recorded."
          compact
          items={comparison.decisions.map((item) => ({
            key: `${item.snapshotId}-${item.title}`,
            snapshotId: item.snapshotId,
            title: item.title,
            meta: item.rationale || item.snapshotTitle,
          }))}
        />
        <ComparisonListPanel
          title="Risks"
          count={comparison.risks.length}
          empty="No risks recorded."
          compact
          items={comparison.risks.map((item) => ({
            key: `${item.snapshotId}-${item.title}`,
            snapshotId: item.snapshotId,
            title: item.title,
            meta: `${statusLabel(item.severity)}${item.mitigation ? ` · ${item.mitigation}` : ""}`,
          }))}
        />
        <ComparisonListPanel
          title="Validation"
          count={comparison.validation.length}
          empty="No validation recorded."
          compact
          items={comparison.validation.map((item) => ({
            key: `${item.snapshotId}-${item.command}`,
            snapshotId: item.snapshotId,
            title: item.command,
            meta: `${item.snapshotTitle} · ${statusLabel(item.status)}`,
          }))}
        />
      </aside>
    </section>
  );
}

function ComparisonStat({
  label,
  value,
}: {
  label: string;
  value: number;
}): React.JSX.Element {
  return (
    <div className="min-w-20 rounded-lg bg-secondary px-3 py-2 text-right shadow-[var(--shadow-border)]">
      <div className="text-lg font-semibold tabular-nums">{value}</div>
      <div className="text-[11px] font-medium text-muted-foreground">{label}</div>
    </div>
  );
}

function ComparisonListPanel({
  title,
  count,
  empty,
  items,
  compact = false,
}: {
  title: string;
  count: number;
  empty: string;
  items: ComparisonPanelItem[];
  compact?: boolean;
}): React.JSX.Element {
  return (
    <section className="overflow-hidden rounded-lg bg-card shadow-[var(--shadow-border)]">
      <div className="flex min-h-12 items-center justify-between gap-3 border-b px-4">
        <h3 className="text-sm font-semibold">{title}</h3>
        <span className="rounded-md bg-secondary px-2 py-1 text-xs font-medium text-muted-foreground tabular-nums">
          {count}
        </span>
      </div>
      <div className={`${compact ? "max-h-[224px]" : "max-h-[300px]"} overflow-y-auto`}>
        {items.length === 0 ? (
          <p className="p-4 text-sm text-muted-foreground">{empty}</p>
        ) : (
          <div className="divide-y">
            {items.map((item) => (
              <ComparisonCompactItem key={item.key} item={item} />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}

function ComparisonCompactItem({
  item,
}: {
  item: ComparisonPanelItem;
}): React.JSX.Element {
  return (
    <Link
      to="/snapshots/$id"
      params={{ id: item.snapshotId }}
      className="block px-4 py-3 transition-[background-color] hover:bg-secondary"
    >
      <p className="line-clamp-3 text-sm font-medium leading-5 text-foreground">
        {item.title}
      </p>
      {item.meta ? (
        <p className="mt-1 line-clamp-2 text-xs leading-5 text-muted-foreground">
          {item.meta}
        </p>
      ) : null}
    </Link>
  );
}

function ComparisonTimelineCompact({
  items,
}: {
  items: ComparisonTimelineItem[];
}): React.JSX.Element {
  return (
    <section className="rounded-lg bg-card p-4 shadow-[var(--shadow-border)]">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 className="text-sm font-semibold">Timeline</h3>
        <span className="rounded-md bg-secondary px-2 py-1 text-xs font-medium text-muted-foreground tabular-nums">
          {items.length}
        </span>
      </div>
      {items.length === 0 ? (
        <p className="mt-3 text-sm text-muted-foreground">
          No timeline entries recorded.
        </p>
      ) : (
        <div className="mt-4 overflow-x-auto overscroll-x-contain pb-2">
          <div className="relative min-w-max">
            {items.length > 1 ? (
              <span className="absolute left-[15px] right-[15px] top-[7px] h-px bg-blue-600" />
            ) : null}
            <div className="flex gap-3">
              {items.map((item) => (
                <Link
                  key={`${item.snapshotId}-${item.phase}-${item.title}-${item.occurredAt}`}
                  to="/snapshots/$id"
                  params={{ id: item.snapshotId }}
                  className="group relative w-[260px] shrink-0 pt-7"
                >
                  <span className="absolute left-0 top-0 flex size-3.5 rounded-full border-2 border-blue-600 bg-card ring-4 ring-secondary transition-[transform,box-shadow] group-hover:scale-110 group-hover:ring-accent" />
                  <span className="block min-h-[140px] rounded-lg bg-secondary px-4 py-3 shadow-[var(--shadow-border)] transition-[background-color,box-shadow] group-hover:bg-accent">
                    <span className="text-[11px] font-semibold uppercase tracking-wide text-primary">
                      {statusLabel(item.phase)}
                    </span>
                    <span className="mt-2 block line-clamp-2 text-sm font-semibold leading-5 text-foreground">
                      {item.title}
                    </span>
                    <span className="mt-2 block line-clamp-2 text-xs leading-5 text-muted-foreground">
                      {item.snapshotTitle} · {compactDate(item.occurredAt)}
                    </span>
                  </span>
                </Link>
              ))}
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

function ComparisonInlineError({ error }: { error: unknown }): React.JSX.Element {
  const message = error instanceof Error ? error.message : String(error);
  return (
    <div className="rounded-lg bg-card p-5 shadow-[0_0_0_1px_rgba(239,68,68,0.18),0_1px_2px_-1px_rgba(15,23,42,0.08)]">
      <p className="text-xs font-semibold uppercase tracking-wide text-red-600">
        Unable to compare
      </p>
      <p className="mt-2 text-sm font-medium text-foreground text-pretty">
        {message}
      </p>
    </div>
  );
}
