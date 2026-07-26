import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import {
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  GitBranch,
  List,
  Monitor,
  Network,
  Search,
  SlidersHorizontal,
} from "lucide-react";
import { useMemo, useState } from "react";
import { api } from "../api";
import {
  AgentAvatar,
  Pill,
  relativeTime,
  snapshotVisualStatus,
} from "../components/dashboard-chrome";
import { ErrorState } from "../components/error-state";
import { LoadingState } from "../components/loading-state";
import type { EvolutionSummary } from "../types";

export type SnapshotStatusFilter = "all" | "verified" | "waiting" | "progress" | "pending";
export type SnapshotDateFilter = "all" | "today" | "yesterday" | "last7" | "last30";
export type SnapshotSort = "newest" | "oldest" | "title";
type SnapshotView = "timeline" | "graph";

const SNAPSHOT_PAGE_SIZE = 10;

export function SnapshotsPage() {
  const snapshots = useQuery({ queryKey: ["snapshots"], queryFn: api.snapshots });

  return (
    <main className="min-h-dvh bg-[oklch(0.986_0.003_247)] px-5 pb-10 pt-7 sm:px-8 md:px-11 md:pb-16 md:pt-8">
      <div className="w-full max-w-[1360px]">
        {snapshots.isLoading ? <LoadingState label="Loading snapshots" /> : null}
        {snapshots.error ? <ErrorState error={snapshots.error} /> : null}
        {snapshots.data ? <SnapshotsContent snapshots={snapshots.data} /> : null}
      </div>
    </main>
  );
}

function SnapshotsContent({ snapshots }: { snapshots: EvolutionSummary[] }) {
  const [view, setView] = useState<SnapshotView>("timeline");
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState<SnapshotStatusFilter>("all");
  const [agentFilter, setAgentFilter] = useState("all");
  const [repositoryFilter, setRepositoryFilter] = useState("all");
  const [dateFilter, setDateFilter] = useState<SnapshotDateFilter>("all");
  const [sort, setSort] = useState<SnapshotSort>("newest");
  const [page, setPage] = useState(1);
  const stats = snapshotStats(snapshots);
  const agents = useMemo(() => uniqueOptions(snapshots.map(agentName)), [snapshots]);
  const repositories = useMemo(
    () => uniqueOptions(snapshots.map((snapshot) => snapshot.repository || "eve")),
    [snapshots],
  );
  const filtered = useMemo(
    () => filterSnapshots(snapshots, { query, statusFilter, agentFilter, repositoryFilter, dateFilter, sort }),
    [snapshots, query, statusFilter, agentFilter, repositoryFilter, dateFilter, sort],
  );
  const pageCount = Math.max(1, Math.ceil(filtered.length / SNAPSHOT_PAGE_SIZE));
  const currentPage = Math.min(page, pageCount);
  const pageRows = filtered.slice((currentPage - 1) * SNAPSHOT_PAGE_SIZE, currentPage * SNAPSHOT_PAGE_SIZE);
  const groups = groupSnapshots(pageRows);
  const hasFilters =
    query.trim() || statusFilter !== "all" || agentFilter !== "all" || repositoryFilter !== "all" || dateFilter !== "all" || sort !== "newest";

  const updateFilter = (action: () => void) => {
    action();
    setPage(1);
  };

  const resetFilters = () => {
    setQuery("");
    setStatusFilter("all");
    setAgentFilter("all");
    setRepositoryFilter("all");
    setDateFilter("all");
    setSort("newest");
    setPage(1);
  };

  return (
    <>
      <header className="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <h1 className="text-[38px] font-semibold leading-none tracking-[-0.03em] text-slate-950">
            Snapshots
          </h1>
          <div className="mt-4 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-slate-600">
            <span>{snapshots.length} snapshots</span>
            <span aria-hidden="true">•</span>
            <span className="font-medium text-emerald-600">{stats.verifiedPercent}% verified</span>
            <span aria-hidden="true">•</span>
            <span className="font-medium text-orange-600">{stats.awaiting} awaiting decision</span>
            <span aria-hidden="true">•</span>
            <span>{stats.inProgress} in progress</span>
          </div>
        </div>
        <div className="inline-flex h-11 w-fit rounded-lg border border-slate-200 bg-white/70 p-1 shadow-[0_1px_2px_rgba(15,23,42,0.03)]">
          <button
            type="button"
            onClick={() => setView("timeline")}
            className={`inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm font-medium text-slate-950 ${view === "timeline" ? "bg-slate-100" : ""}`}
          >
            <List className="size-4 text-indigo-600" />
            Timeline
          </button>
          <button
            type="button"
            onClick={() => setView("graph")}
            className={`inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm font-medium text-slate-950 ${view === "graph" ? "bg-slate-100" : ""}`}
          >
            <Network className="size-4" />
            Graph
          </button>
        </div>
      </header>

      <div className="mt-8 flex flex-col gap-3 xl:flex-row xl:items-center">
        <label className="flex h-12 min-w-0 flex-1 items-center gap-3 rounded-lg border border-slate-200 bg-white/70 px-4 text-sm text-slate-500 shadow-[0_1px_2px_rgba(15,23,42,0.03)] transition-colors focus-within:border-indigo-300 focus-within:bg-white">
          <Search className="size-5 shrink-0 text-slate-600" strokeWidth={1.8} />
          <span className="sr-only">Search snapshots</span>
          <input
            value={query}
            onChange={(event) => updateFilter(() => setQuery(event.target.value))}
            placeholder="Search snapshots..."
            className="min-w-0 flex-1 bg-transparent text-slate-950 outline-none placeholder:text-slate-500"
          />
        </label>
        <div className="hidden h-10 w-px bg-slate-200 xl:block" />
        <div className="flex gap-3 overflow-x-auto">
          <FilterSelect
            label="Status"
            value={statusFilter}
            onChange={(value) => updateFilter(() => setStatusFilter(value as SnapshotStatusFilter))}
            options={[
              { value: "all", label: "All statuses" },
              { value: "verified", label: "Verified" },
              { value: "waiting", label: "Awaiting decision" },
              { value: "progress", label: "In progress" },
              { value: "pending", label: "Pending" },
            ]}
          />
          <FilterSelect
            label="Agent"
            value={agentFilter}
            onChange={(value) => updateFilter(() => setAgentFilter(value))}
            options={[{ value: "all", label: "All agents" }, ...agents.map((agent) => ({ value: agent, label: agent }))]}
          />
          <FilterSelect
            label="Repository"
            value={repositoryFilter}
            onChange={(value) => updateFilter(() => setRepositoryFilter(value))}
            options={[{ value: "all", label: "All repositories" }, ...repositories.map((repository) => ({ value: repository, label: repository }))]}
          />
          <FilterSelect
            label="Date"
            value={dateFilter}
            onChange={(value) => updateFilter(() => setDateFilter(value as SnapshotDateFilter))}
            options={[
              { value: "all", label: "Any date" },
              { value: "today", label: "Today" },
              { value: "yesterday", label: "Yesterday" },
              { value: "last7", label: "Last 7 days" },
              { value: "last30", label: "Last 30 days" },
            ]}
          />
          <FilterSelect
            label="Sort"
            value={sort}
            onChange={(value) => updateFilter(() => setSort(value as SnapshotSort))}
            options={[
              { value: "newest", label: "Newest" },
              { value: "oldest", label: "Oldest" },
              { value: "title", label: "Title" },
            ]}
          />
          <button
            type="button"
            onClick={resetFilters}
            disabled={!hasFilters}
            className="grid size-11 shrink-0 place-items-center rounded-lg border border-slate-200 bg-white/70 text-slate-950 disabled:cursor-not-allowed disabled:text-slate-300"
            aria-label="Reset snapshot filters"
          >
            <SlidersHorizontal className="size-4" />
          </button>
        </div>
      </div>

      {view === "timeline" ? (
        <section className="mt-8 overflow-hidden rounded-lg border border-slate-200 bg-white/35">
          {groups.length === 0 ? (
            <div className="px-5 py-10 text-sm text-slate-500">No snapshots match these filters.</div>
          ) : null}
          {groups.map((group) => (
            <div key={group.label}>
              <div className="border-b border-slate-200 bg-[oklch(0.986_0.003_247)] px-3 py-2 text-sm font-semibold text-slate-500">
                {group.label}
              </div>
              {group.rows.map((snapshot) => (
                <SnapshotTimelineRow key={snapshot.id} snapshot={snapshot} />
              ))}
            </div>
          ))}
        </section>
      ) : (
        <SnapshotGraph snapshots={pageRows} />
      )}

      <nav className="mt-5 flex items-center justify-center gap-6 text-sm text-slate-500" aria-label="Snapshot pages">
        <button
          type="button"
          onClick={() => setPage(Math.max(1, currentPage - 1))}
          disabled={currentPage === 1}
          className="inline-flex items-center gap-2 disabled:opacity-60"
        >
          <ChevronLeft className="size-4" />
          Newer
        </button>
        {visiblePages(pageCount, currentPage).map((page) => (
          <button
            key={page}
            type="button"
            onClick={() => setPage(page)}
            className={page === currentPage ? "grid size-8 place-items-center rounded-md bg-indigo-100 font-semibold text-indigo-700" : "grid size-8 place-items-center font-medium text-slate-950"}
          >
            {page}
          </button>
        ))}
        <button
          type="button"
          onClick={() => setPage(Math.min(pageCount, currentPage + 1))}
          disabled={currentPage === pageCount}
          className="inline-flex items-center gap-2 disabled:opacity-60"
        >
          Older
          <ChevronRight className="size-4" />
        </button>
      </nav>
      <p className="mt-3 text-center text-xs text-slate-500">
        Showing {pageRows.length === 0 ? 0 : (currentPage - 1) * SNAPSHOT_PAGE_SIZE + 1}-{Math.min(currentPage * SNAPSHOT_PAGE_SIZE, filtered.length)} of {filtered.length} snapshots
      </p>
    </>
  );
}

function FilterSelect({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: Array<{ value: string; label: string }>;
  onChange: (value: string) => void;
}) {
  return (
    <label className="relative inline-flex h-11 shrink-0 items-center rounded-lg border border-slate-200 bg-white/70 text-sm font-medium text-slate-950">
      <span className="sr-only">{label}</span>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="h-full min-w-32 appearance-none bg-transparent pl-4 pr-10 outline-none"
        aria-label={label}
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {value === "all" && option.value === "all" ? label : option.label}
          </option>
        ))}
      </select>
      <ChevronDown className="pointer-events-none absolute right-3 size-4" />
    </label>
  );
}

function SnapshotGraph({ snapshots }: { snapshots: EvolutionSummary[] }) {
  if (snapshots.length === 0) {
    return (
      <section className="mt-8 rounded-lg border border-slate-200 bg-white/45 px-5 py-10 text-sm text-slate-500">
        No snapshots match these filters.
      </section>
    );
  }

  return (
    <section className="mt-8 rounded-lg border border-slate-200 bg-white/45 p-5" aria-label="Snapshot graph">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {snapshots.map((snapshot, index) => {
          const status = snapshotVisualStatus(snapshot);
          return (
            <Link
              key={snapshot.id}
              to="/snapshots/$id"
              params={{ id: snapshot.id }}
              className="relative min-h-36 rounded-lg border border-slate-200 bg-white px-4 py-4 text-left transition-colors hover:border-indigo-200 hover:bg-indigo-50/30"
            >
              {index > 0 ? <span className="absolute -left-4 top-1/2 hidden h-px w-4 bg-slate-200 md:block" aria-hidden="true" /> : null}
              <span className="flex items-start gap-3">
                <span className={`mt-1 ${status.markerClass}`} />
                <span className="min-w-0">
                  <span className="block truncate text-sm font-semibold text-slate-950">{snapshot.title}</span>
                  <span className="mt-2 line-clamp-3 text-sm leading-5 text-slate-600">
                    {snapshot.userVisibleChange || snapshot.outcome}
                  </span>
                </span>
              </span>
              <span className="mt-4 flex items-center justify-between gap-3 text-xs text-slate-500">
                <span>{snapshot.repository || "eve"}</span>
                <span>{relativeTime(snapshot.updatedAt || snapshot.createdAt)}</span>
              </span>
            </Link>
          );
        })}
      </div>
    </section>
  );
}

function SnapshotTimelineRow({ snapshot }: { snapshot: EvolutionSummary }) {
  const status = snapshotVisualStatus(snapshot);
  const agent = agentName(snapshot);
  return (
    <Link
      to="/snapshots/$id"
      params={{ id: snapshot.id }}
      className="grid min-h-[86px] grid-cols-[32px_minmax(0,1fr)] gap-4 border-b border-slate-200 px-4 py-4 last:border-b-0 transition-colors hover:bg-white sm:grid-cols-[36px_minmax(260px,1.5fr)_160px_150px_180px_90px_24px] sm:items-center sm:px-5 lg:px-6"
    >
      <span className={status.markerClass} />
      <span className="min-w-0">
        <span className="block truncate text-[15px] font-semibold text-slate-950">{snapshot.title}</span>
        <span className="mt-1 block max-w-[44ch] text-sm leading-5 text-slate-600 sm:line-clamp-2">
          {snapshot.userVisibleChange || snapshot.outcome}
        </span>
      </span>
      <span className="hidden min-w-0 items-center gap-3 text-sm font-medium text-slate-950 sm:inline-flex">
        <Monitor className="size-4 text-slate-700" />
        <span className="truncate">{snapshot.repository || "eve"}</span>
      </span>
      <span className="hidden min-w-0 items-center gap-3 text-sm font-medium text-slate-950 sm:inline-flex">
        <AgentAvatar agent={agent} />
        <span className="truncate lowercase">{agent}</span>
      </span>
      <span className="hidden sm:block">
        <Pill tone={status.tone}>
          <span className={`mr-2 size-1.5 rounded-full ${status.dotClass}`} />
          {status.label}
        </Pill>
      </span>
      <span className="hidden text-sm text-slate-500 sm:block">{relativeTime(snapshot.updatedAt || snapshot.createdAt)}</span>
      <ChevronRight className="hidden size-4 justify-self-end text-slate-950 sm:block" />
    </Link>
  );
}

function snapshotStats(snapshots: EvolutionSummary[]) {
  const statuses = snapshots.map(snapshotVisualStatus);
  const verified = statuses.filter((status) => status.tone === "verified").length;
  const awaiting = statuses.filter((status) => status.tone === "waiting").length;
  const inProgress = statuses.filter((status) => status.tone === "progress").length;
  return {
    verifiedPercent: snapshots.length === 0 ? 100 : Math.round((verified / snapshots.length) * 100),
    awaiting,
    inProgress,
  };
}

function groupSnapshots(snapshots: EvolutionSummary[]) {
  const groups = new Map<string, EvolutionSummary[]>();
  for (const snapshot of snapshots) {
    const label = dateGroupLabel(snapshot.updatedAt || snapshot.createdAt);
    groups.set(label, [...(groups.get(label) ?? []), snapshot]);
  }
  return Array.from(groups.entries()).map(([label, rows]) => ({ label, rows }));
}

export function filterSnapshots(
  snapshots: EvolutionSummary[],
  filters: {
    query: string;
    statusFilter: SnapshotStatusFilter;
    agentFilter: string;
    repositoryFilter: string;
    dateFilter: SnapshotDateFilter;
    sort: SnapshotSort;
  },
) {
  const query = filters.query.trim().toLowerCase();
  const now = new Date();
  return snapshots
    .filter((snapshot) => {
      const status = snapshotVisualStatus(snapshot);
      if (filters.statusFilter !== "all" && status.tone !== filters.statusFilter) return false;
      if (filters.agentFilter !== "all" && agentName(snapshot) !== filters.agentFilter) return false;
      if (filters.repositoryFilter !== "all" && (snapshot.repository || "eve") !== filters.repositoryFilter) return false;
      if (!matchesDateFilter(snapshot.updatedAt || snapshot.createdAt, filters.dateFilter, now)) return false;
      if (!query) return true;
      return snapshotSearchText(snapshot, status.label).includes(query);
    })
    .sort((left, right) => compareSnapshots(left, right, filters.sort));
}

function snapshotSearchText(snapshot: EvolutionSummary, statusLabel: string) {
  return [
    snapshot.id,
    snapshot.title,
    snapshot.type,
    snapshot.status,
    snapshot.outcome,
    snapshot.userVisibleChange,
    snapshot.repository,
    statusLabel,
    snapshot.verificationState,
    snapshot.verificationSummary,
    ...snapshot.sessionProviders,
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
}

function compareSnapshots(left: EvolutionSummary, right: EvolutionSummary, sort: SnapshotSort) {
  if (sort === "title") return left.title.localeCompare(right.title);
  const leftTime = new Date(left.updatedAt || left.createdAt).getTime() || 0;
  const rightTime = new Date(right.updatedAt || right.createdAt).getTime() || 0;
  return sort === "oldest" ? leftTime - rightTime : rightTime - leftTime;
}

function matchesDateFilter(value: string | undefined, filter: SnapshotDateFilter, now: Date) {
  if (filter === "all") return true;
  if (!value) return false;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return false;
  if (filter === "today") return date.toDateString() === now.toDateString();
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  if (filter === "yesterday") return date.toDateString() === yesterday.toDateString();
  const days = filter === "last7" ? 7 : 30;
  return now.getTime() - date.getTime() <= days * 24 * 60 * 60 * 1000;
}

function uniqueOptions(values: string[]) {
  return Array.from(new Set(values.filter(Boolean))).sort((left, right) => left.localeCompare(right));
}

function visiblePages(pageCount: number, currentPage: number) {
  const pages = new Set([1, pageCount, currentPage, currentPage - 1, currentPage + 1]);
  return Array.from(pages)
    .filter((page) => page >= 1 && page <= pageCount)
    .sort((left, right) => left - right);
}

function dateGroupLabel(value?: string) {
  if (!value) return "Undated";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Undated";
  const today = new Date();
  if (date.toDateString() === today.toDateString()) return "Today";
  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) return "Yesterday";
  return date.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
}

function agentName(snapshot: EvolutionSummary) {
  const provider = snapshot.sessionProviders.find(Boolean) || "Codex";
  return provider.charAt(0).toUpperCase() + provider.slice(1);
}
