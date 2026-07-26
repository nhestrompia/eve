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
import { api } from "../api";
import {
  AgentAvatar,
  Pill,
  openDashboardSearch,
  relativeTime,
  snapshotVisualStatus,
} from "../components/dashboard-chrome";
import { ErrorState } from "../components/error-state";
import { LoadingState } from "../components/loading-state";
import type { EvolutionSummary } from "../types";

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
  const stats = snapshotStats(snapshots);
  const groups = groupSnapshots(snapshots.slice(0, 9));

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
          <button type="button" className="inline-flex h-9 items-center gap-2 rounded-md bg-slate-100 px-3 text-sm font-medium text-slate-950">
            <List className="size-4 text-indigo-600" />
            Timeline
          </button>
          <button type="button" className="inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm font-medium text-slate-950">
            <Network className="size-4" />
            Graph
          </button>
        </div>
      </header>

      <div className="mt-8 flex flex-col gap-3 xl:flex-row xl:items-center">
        <button
          type="button"
          onClick={() => openDashboardSearch()}
          className="flex h-12 min-w-0 flex-1 items-center gap-3 rounded-lg border border-slate-200 bg-white/70 px-4 text-left text-sm text-slate-500 shadow-[0_1px_2px_rgba(15,23,42,0.03)] transition-colors hover:bg-white"
        >
          <Search className="size-5 shrink-0 text-slate-600" strokeWidth={1.8} />
          <span className="min-w-0 flex-1 truncate">Search snapshots...</span>
          <kbd className="rounded-md bg-slate-100 px-2 py-1 font-mono text-[11px] font-medium text-slate-500">⌘K</kbd>
        </button>
        <div className="hidden h-10 w-px bg-slate-200 xl:block" />
        <div className="flex gap-3 overflow-x-auto">
          {["Status", "Agent", "Repository", "Date"].map((label) => (
            <button
              key={label}
              type="button"
              className="inline-flex h-11 shrink-0 items-center gap-3 rounded-lg border border-slate-200 bg-white/70 px-4 text-sm font-medium text-slate-950"
            >
              {label}
              <ChevronDown className="size-4" />
            </button>
          ))}
          <button type="button" className="grid size-11 shrink-0 place-items-center rounded-lg border border-slate-200 bg-white/70 text-slate-950">
            <SlidersHorizontal className="size-4" />
          </button>
        </div>
      </div>

      <section className="mt-8 overflow-hidden rounded-lg border border-slate-200 bg-white/35">
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

      <nav className="mt-5 flex items-center justify-center gap-6 text-sm text-slate-500" aria-label="Snapshot pages">
        <button type="button" className="inline-flex items-center gap-2 opacity-60">
          <ChevronLeft className="size-4" />
          Newer
        </button>
        {[1, 2, 3, 4].map((page) => (
          <button
            key={page}
            type="button"
            className={page === 1 ? "grid size-8 place-items-center rounded-md bg-indigo-100 font-semibold text-indigo-700" : "grid size-8 place-items-center font-medium text-slate-950"}
          >
            {page}
          </button>
        ))}
        <span className="text-slate-950">...</span>
        <button type="button" className="grid size-8 place-items-center font-medium text-slate-950">16</button>
        <button type="button" className="inline-flex items-center gap-2">
          Older
          <ChevronRight className="size-4" />
        </button>
      </nav>
    </>
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
