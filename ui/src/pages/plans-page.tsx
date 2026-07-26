import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import {
  ArrowDown,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  FileText,
  Filter,
  GitBranch,
  Search,
  X,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { api } from "../api";
import {
  AgentAvatar,
  DashboardShell,
  Pill,
  type PillTone,
  relativeTime,
} from "../components/dashboard-chrome";
import { ErrorState } from "../components/error-state";
import { LoadingState } from "../components/loading-state";
import { PlanApprovalDialog, currentRevision } from "../components/pending-plan-banner";
import type { EvolutionSummary, PlanRequest, PlanRevision } from "../types";

export type PlanRow = {
  id: string;
  title: string;
  summary: string;
  repository: string;
  branch: string;
  agent: string;
  state: string;
  sourceState: string;
  statusLabel: string;
  statusTone: PillTone;
  updatedAt?: string;
  files: string[];
  revision?: PlanRevision;
  request?: PlanRequest;
};

const TAB_ORDER = [
  { id: "all", label: "All" },
  { id: "pending_approval", label: "Waiting approval" },
  { id: "ready", label: "Ready" },
  { id: "in_progress", label: "In progress" },
  { id: "completed", label: "Completed" },
  { id: "rejected", label: "Rejected" },
];

export type PlanSort = "newest" | "oldest" | "title" | "status";

const PLAN_PAGE_SIZE = 10;

export function PlansPage() {
  const [query, setQuery] = useState("");
  const plans = useQuery({
    queryKey: ["plan-requests", "all"],
    queryFn: () => api.planRequests(""),
    refetchInterval: 5_000,
    retry: false,
  });
  const snapshots = useQuery({ queryKey: ["snapshots"], queryFn: api.snapshots });

  return (
    <DashboardShell
      title="Plans"
      subtitle="Proposed work by AI agents, awaiting your decision."
      searchPlaceholder="Search plans..."
      searchSlot={<PlansSearchInput value={query} onChange={setQuery} />}
    >
      {plans.isLoading || snapshots.isLoading ? <LoadingState label="Loading plans" /> : null}
      {plans.error ? <ErrorState error={plans.error} /> : null}
      {snapshots.error ? <ErrorState error={snapshots.error} /> : null}
      {snapshots.data ? (
        <PlansContent plans={plans.data ?? []} snapshots={snapshots.data} query={query} />
      ) : null}
    </DashboardShell>
  );
}

function PlansSearchInput({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  return (
    <label className="flex h-12 w-full max-w-[330px] items-center gap-3 rounded-lg border border-slate-200 bg-white/70 px-4 text-sm text-slate-500 shadow-[0_1px_2px_rgba(15,23,42,0.03)] transition-colors focus-within:border-indigo-300 focus-within:bg-white sm:w-[310px] lg:w-[330px]">
      <Search className="size-5 shrink-0 text-slate-600" strokeWidth={1.8} />
      <span className="sr-only">Search plans</span>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder="Search plans..."
        className="min-w-0 flex-1 bg-transparent text-slate-950 outline-none placeholder:text-slate-500"
      />
    </label>
  );
}

function PlansContent({ plans, snapshots, query }: { plans: PlanRequest[]; snapshots: EvolutionSummary[]; query: string }) {
  const rows = useMemo(() => buildPlanRows(plans, snapshots), [plans, snapshots]);
  const [activeTab, setActiveTab] = useState("all");
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [repositoryFilter, setRepositoryFilter] = useState("all");
  const [agentFilter, setAgentFilter] = useState("all");
  const [sort, setSort] = useState<PlanSort>("newest");
  const [page, setPage] = useState(1);
  const repositories = useMemo(
    () => Array.from(new Set(rows.map((row) => row.repository).filter(Boolean))).sort((left, right) => left.localeCompare(right)),
    [rows],
  );
  const agents = useMemo(
    () => Array.from(new Set(rows.map((row) => row.agent).filter(Boolean))).sort((left, right) => left.localeCompare(right)),
    [rows],
  );
  const filtered = useMemo(
    () => filterPlanRows(rows, { activeTab, query, repositoryFilter, agentFilter, sort }),
    [rows, activeTab, query, repositoryFilter, agentFilter, sort],
  );
  const pageCount = Math.max(1, Math.ceil(filtered.length / PLAN_PAGE_SIZE));
  const currentPage = Math.min(page, pageCount);
  const visibleRows = filtered.slice((currentPage - 1) * PLAN_PAGE_SIZE, currentPage * PLAN_PAGE_SIZE);
  const [selectedId, setSelectedId] = useState(filtered[0]?.id ?? rows[0]?.id ?? "");
  const selected = filtered.find((row) => row.id === selectedId) ?? filtered[0] ?? rows[0];
  const pendingPlans = plans.filter((plan) => plan.state === "pending_approval");
  const [approvalOpen, setApprovalOpen] = useState(false);
  const hasExtraFilters = repositoryFilter !== "all" || agentFilter !== "all";

  useEffect(() => {
    setPage(1);
  }, [activeTab, query, repositoryFilter, agentFilter, sort]);

  useEffect(() => {
    if (filtered.length === 0) {
      setSelectedId("");
      return;
    }
    if (!filtered.some((row) => row.id === selectedId)) {
      setSelectedId(filtered[0].id);
    }
  }, [filtered, selectedId]);

  return (
    <div className="mt-11">
      <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
        <div className="flex gap-7 overflow-x-auto text-sm text-slate-600">
          {TAB_ORDER.map((tab) => {
            const count = tab.id === "all" ? rows.length : rows.filter((row) => row.state === tab.id).length;
            const active = activeTab === tab.id;
            return (
              <button
                key={tab.id}
                type="button"
                onClick={() => {
                  setActiveTab(tab.id);
                  setSelectedId("");
                }}
                className={`relative flex h-10 shrink-0 items-center gap-2 font-medium transition-colors after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 after:rounded-full ${
                  active ? "text-slate-950 after:bg-slate-950" : "after:bg-transparent hover:text-slate-950"
                }`}
              >
                {tab.label}
                {tab.id !== "all" ? (
                  <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-500">{count}</span>
                ) : null}
              </button>
            );
          })}
        </div>
        <div className="relative flex flex-wrap gap-3">
          <button
            type="button"
            onClick={() => setFiltersOpen((open) => !open)}
            className="inline-flex h-11 items-center gap-3 rounded-lg border border-slate-200 bg-white/60 px-4 font-medium text-slate-950"
            aria-expanded={filtersOpen}
          >
            <Filter className="size-4" />
            Filters
            {hasExtraFilters ? <span className="size-2 rounded-full bg-indigo-500" /> : null}
            <ChevronDown className="size-4" />
          </button>
          <label className="relative inline-flex h-11 items-center rounded-lg border border-slate-200 bg-white/60 font-medium text-slate-950">
            <span className="sr-only">Sort plans</span>
            <select
              value={sort}
              onChange={(event) => setSort(event.target.value as PlanSort)}
              className="h-full appearance-none bg-transparent pl-4 pr-10 outline-none"
              aria-label="Sort plans"
            >
              <option value="newest">Sort: Newest</option>
              <option value="oldest">Sort: Oldest</option>
              <option value="title">Sort: Title</option>
              <option value="status">Sort: Status</option>
            </select>
            <ChevronDown className="pointer-events-none absolute right-3 size-4" />
          </label>
          {filtersOpen ? (
            <div className="absolute right-0 top-[3.25rem] z-10 w-72 rounded-lg border border-slate-200 bg-white p-4 shadow-lg">
              <div className="space-y-4">
                <PlanFilterSelect
                  label="Repository"
                  value={repositoryFilter}
                  onChange={setRepositoryFilter}
                  options={[{ value: "all", label: "All repositories" }, ...repositories.map((repository) => ({ value: repository, label: repository }))]}
                />
                <PlanFilterSelect
                  label="Agent"
                  value={agentFilter}
                  onChange={setAgentFilter}
                  options={[{ value: "all", label: "All agents" }, ...agents.map((agent) => ({ value: agent, label: agent }))]}
                />
                <button
                  type="button"
                  onClick={() => {
                    setRepositoryFilter("all");
                    setAgentFilter("all");
                  }}
                  disabled={!hasExtraFilters}
                  className="h-9 w-full rounded-md border border-slate-200 text-sm font-medium text-slate-950 disabled:cursor-not-allowed disabled:text-slate-300"
                >
                  Reset filters
                </button>
              </div>
            </div>
          ) : null}
        </div>
      </div>

      <div className="mt-5 grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
        <section className="overflow-hidden rounded-lg border border-slate-200 bg-white/45">
          <div className="hidden min-h-[62px] grid-cols-[minmax(220px,1.4fr)_minmax(120px,0.8fr)_90px_110px] items-center gap-5 border-b border-slate-200 px-7 text-xs font-medium text-slate-600 lg:grid 2xl:grid-cols-[minmax(260px,1.45fr)_minmax(145px,0.85fr)_minmax(120px,0.6fr)_minmax(120px,0.7fr)_100px_28px]">
            <span>Plan</span>
            <span>Repository</span>
            <span>Agent</span>
            <span>Status</span>
            <span className="hidden items-center gap-2 text-slate-950 2xl:inline-flex">Updated <ArrowDown className="size-3.5" /></span>
            <span className="hidden 2xl:block" />
          </div>
          <div>
            {visibleRows.length === 0 ? (
              <div className="px-5 py-10 text-sm text-slate-500 lg:px-7">No plans match these filters.</div>
            ) : null}
            {visibleRows.map((row) => (
              <button
                key={row.id}
                type="button"
                onClick={() => setSelectedId(row.id)}
                className={`grid w-full grid-cols-1 gap-3 border-b border-slate-200 px-5 py-4 text-left transition-colors last:border-b-0 hover:bg-white lg:min-h-[78px] lg:grid-cols-[minmax(220px,1.4fr)_minmax(120px,0.8fr)_90px_110px] lg:items-center lg:gap-5 lg:px-7 2xl:grid-cols-[minmax(260px,1.45fr)_minmax(145px,0.85fr)_minmax(120px,0.6fr)_minmax(120px,0.7fr)_100px_28px] ${
                  selected?.id === row.id ? "bg-white" : ""
                }`}
              >
                <span className="flex min-w-0 gap-4">
                  <FileText className="mt-1 size-5 shrink-0 text-slate-600" strokeWidth={1.7} />
                  <span className="min-w-0">
                    <span className="block truncate text-[15px] font-semibold text-slate-950">{row.title}</span>
                    <span className="mt-1 block truncate text-sm text-slate-500">{row.summary}</span>
                  </span>
                </span>
                <span className="min-w-0 text-sm">
                  <span className="block truncate font-medium text-slate-950">{row.repository}</span>
                  <span className="mt-1 inline-flex items-center gap-1.5 text-xs text-slate-500">
                    <GitBranch className="size-3.5" /> {row.branch}
                  </span>
                </span>
                <span className="inline-flex items-center gap-3 text-sm font-medium text-slate-950">
                  <AgentAvatar agent={row.agent} /> {row.agent}
                </span>
                <span><Pill tone={row.statusTone}>{row.statusLabel}</Pill></span>
                <span className="hidden text-sm text-slate-600 2xl:block">{relativeTime(row.updatedAt)}</span>
                <ChevronRight className="hidden size-4 justify-self-end text-slate-600 2xl:block" />
              </button>
            ))}
          </div>
          <div className="flex min-h-[68px] flex-col gap-3 px-5 py-4 text-sm text-slate-600 sm:flex-row sm:items-center sm:justify-between lg:px-7">
            <span>
              Showing {visibleRows.length === 0 ? 0 : (currentPage - 1) * PLAN_PAGE_SIZE + 1}-{Math.min(currentPage * PLAN_PAGE_SIZE, filtered.length)} of {filtered.length} plans
            </span>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setPage(Math.max(1, currentPage - 1))}
                disabled={currentPage === 1}
                className="grid size-9 place-items-center rounded-md border border-slate-200 bg-white text-slate-600 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <ChevronLeft className="size-4" />
              </button>
              {visiblePlanPages(pageCount, currentPage).map((pageNumber) => (
                <button
                  key={pageNumber}
                  type="button"
                  onClick={() => setPage(pageNumber)}
                  className={pageNumber === currentPage ? "grid size-9 place-items-center rounded-md bg-slate-100 font-semibold text-slate-950" : "grid size-9 place-items-center rounded-md text-slate-600"}
                >
                  {pageNumber}
                </button>
              ))}
              <button
                type="button"
                onClick={() => setPage(Math.min(pageCount, currentPage + 1))}
                disabled={currentPage === pageCount}
                className="grid size-9 place-items-center rounded-md border border-slate-200 bg-white text-slate-600 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <ChevronRight className="size-4" />
              </button>
            </div>
          </div>
        </section>

        <PlanDetailRail
          row={selected}
          onReview={() => setApprovalOpen(true)}
          reviewDisabled={!selected?.request || selected.request.state !== "pending_approval"}
        />
      </div>

      <PlanApprovalDialog plans={pendingPlans} open={approvalOpen} onOpenChange={setApprovalOpen} />
    </div>
  );
}

function PlanFilterSelect({
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
    <label className="block text-sm font-medium text-slate-950">
      {label}
      <span className="relative mt-2 block">
        <select
          value={value}
          onChange={(event) => onChange(event.target.value)}
          className="h-10 w-full appearance-none rounded-md border border-slate-200 bg-white pl-3 pr-9 text-sm outline-none"
        >
          {options.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        <ChevronDown className="pointer-events-none absolute right-3 top-3 size-4" />
      </span>
    </label>
  );
}

function PlanDetailRail({
  row,
  onReview,
  reviewDisabled,
}: {
  row?: PlanRow;
  onReview: () => void;
  reviewDisabled: boolean;
}) {
  if (!row) {
    return (
      <aside className="rounded-lg border border-slate-200 bg-white/55 p-6 text-sm text-slate-500">
        No plans recorded yet.
      </aside>
    );
  }

  return (
    <aside className="rounded-lg border border-slate-200 bg-white/55 p-6 xl:sticky xl:top-8 xl:self-start">
      <div className="flex items-start justify-between gap-4">
        <Pill tone={row.statusTone}>{row.statusLabel}</Pill>
        <div className="flex items-center gap-5 text-xs text-slate-500">
          <span>Plan created {relativeTime(row.updatedAt)}</span>
          <X className="size-4" />
        </div>
      </div>
      <h2 className="mt-7 text-[22px] font-semibold leading-tight tracking-[-0.01em] text-slate-950">{row.title}</h2>
      <p className="mt-4 text-sm leading-6 text-slate-600">{row.summary}</p>

      <dl className="mt-8 space-y-6 text-sm">
        <div className="flex items-center justify-between gap-4">
          <dt className="font-semibold text-slate-950">Repository</dt>
          <dd className="inline-flex items-center gap-2 text-slate-700">
            {row.repository}
            <GitBranch className="size-3.5" />
            <span className="text-xs">{row.branch}</span>
          </dd>
        </div>
        <div>
          <dt className="font-semibold text-slate-950">Agent</dt>
          <dd className="mt-3 inline-flex items-center gap-3 font-medium text-slate-950">
            <AgentAvatar agent={row.agent} /> {row.agent}
          </dd>
        </div>
        <div>
          <dt className="font-semibold text-slate-950">Files ({row.files.length})</dt>
          <dd className="mt-3 space-y-1 text-sm leading-5 text-slate-600">
            {row.files.slice(0, 5).map((file) => (
              <div key={file} className="truncate">{file}</div>
            ))}
            {row.files.length > 5 ? <div>+{row.files.length - 5} more</div> : null}
          </dd>
        </div>
        <div>
          <dt className="font-semibold text-slate-950">Plan summary</dt>
          <dd className="mt-3 text-sm leading-6 text-slate-600">{row.revision?.acceptanceCriteria || row.summary}</dd>
        </div>
        <div>
          <dt className="font-semibold text-slate-950">Checks</dt>
          <dd className="mt-3 text-sm text-slate-500">
            {row.revision?.resolvedCheckIds?.length ? `${row.revision.resolvedCheckIds.length} checks declared` : "No deterministic checks yet"}
          </dd>
        </div>
      </dl>

      <div className="mt-8 space-y-3">
        <button
          type="button"
          onClick={onReview}
          disabled={reviewDisabled}
          className="h-12 w-full rounded-lg bg-slate-950 text-sm font-semibold text-white transition-colors hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-300"
        >
          Review plan
        </button>
        <button
          type="button"
          onClick={onReview}
          disabled={reviewDisabled}
          className="h-11 w-full rounded-lg border border-slate-200 bg-white text-sm font-semibold text-slate-950 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:text-slate-400"
        >
          Request changes
        </button>
      </div>
    </aside>
  );
}

function buildPlanRows(plans: PlanRequest[], snapshots: EvolutionSummary[]): PlanRow[] {
  const rows = plans.map((plan) => {
    const revision = currentRevision(plan);
    const state = normalizePlanState(plan.state);
    return {
      id: plan.planRequestId,
      title: titleFromGoal(revision?.goal) || "Plan awaiting review",
      summary: summaryFromText(revision?.acceptanceCriteria) || "Review the proposed scope, checks, and milestones.",
      repository: plan.repository,
      branch: plan.branch || "main",
      agent: "Codex",
      state,
      sourceState: plan.state,
      statusLabel: planStatusLabel(state),
      statusTone: planStatusTone(state),
      updatedAt: revision?.createdAt,
      files: revision?.allowedPathGlobs?.length ? revision.allowedPathGlobs : ["No file scope declared"],
      revision,
      request: plan,
    };
  });

  if (rows.length > 0) {
    return rows.sort((left, right) => (right.updatedAt || "").localeCompare(left.updatedAt || ""));
  }

  return snapshots.slice(0, 10).map((snapshot, index) => ({
    id: snapshot.id,
    title: snapshot.title,
    summary: snapshot.userVisibleChange || snapshot.outcome,
    repository: snapshot.repository || "eve",
    branch: "main",
    agent: snapshot.sessionProviders.includes("claude") ? "Claude" : "Codex",
    state: index % 7 === 0 ? "rejected" : "completed",
    sourceState: index % 7 === 0 ? "rejected" : "fulfilled",
    statusLabel: index % 7 === 0 ? "Rejected" : "Completed",
    statusTone: index % 7 === 0 ? "rejected" : "verified",
    updatedAt: snapshot.updatedAt || snapshot.createdAt,
    files: ["Recorded in snapshot evidence", "Implementation metadata", "Verification notes"],
  }));
}

export function filterPlanRows(
  rows: PlanRow[],
  filters: {
    activeTab: string;
    query: string;
    repositoryFilter: string;
    agentFilter: string;
    sort: PlanSort;
  },
) {
  const query = filters.query.trim().toLowerCase();
  return rows
    .filter((row) => {
      if (filters.activeTab !== "all" && row.state !== filters.activeTab) return false;
      if (filters.repositoryFilter !== "all" && row.repository !== filters.repositoryFilter) return false;
      if (filters.agentFilter !== "all" && row.agent !== filters.agentFilter) return false;
      if (!query) return true;
      return planSearchText(row).includes(query);
    })
    .sort((left, right) => comparePlanRows(left, right, filters.sort));
}

function planSearchText(row: PlanRow) {
  const approvedAlias = row.state === "ready" || row.sourceState === "locked" ? "approved" : "";
  return [
    row.id,
    row.title,
    row.summary,
    row.repository,
    row.branch,
    row.agent,
    row.state,
    row.sourceState,
    row.statusLabel,
    approvedAlias,
    ...row.files,
    row.revision?.acceptanceCriteria,
    row.revision?.goal,
  ]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
}

function comparePlanRows(left: PlanRow, right: PlanRow, sort: PlanSort) {
  if (sort === "title") return left.title.localeCompare(right.title);
  if (sort === "status") return left.statusLabel.localeCompare(right.statusLabel) || left.title.localeCompare(right.title);
  const leftTime = new Date(left.updatedAt || "").getTime() || 0;
  const rightTime = new Date(right.updatedAt || "").getTime() || 0;
  return sort === "oldest" ? leftTime - rightTime : rightTime - leftTime;
}

function visiblePlanPages(pageCount: number, currentPage: number) {
  const pages = new Set([1, pageCount, currentPage, currentPage - 1, currentPage + 1]);
  return Array.from(pages)
    .filter((page) => page >= 1 && page <= pageCount)
    .sort((left, right) => left - right);
}

function normalizePlanState(state: string) {
  if (state === "pending_approval") return "pending_approval";
  if (state === "locked") return "ready";
  if (state === "fulfilled") return "completed";
  if (state === "rejected") return "rejected";
  if (state === "stale" || state === "superseded") return "rejected";
  return "in_progress";
}

function planStatusLabel(state: string) {
  if (state === "pending_approval") return "Waiting approval";
  if (state === "ready") return "Ready";
  if (state === "completed") return "Completed";
  if (state === "rejected") return "Rejected";
  return "In progress";
}

function planStatusTone(state: string): PillTone {
  if (state === "pending_approval") return "waiting";
  if (state === "ready") return "ready";
  if (state === "completed") return "verified";
  if (state === "rejected") return "rejected";
  return "progress";
}

function titleFromGoal(value?: string) {
  if (!value) return "";
  return value.split(/\r?\n/)[0].replace(/^[-*]\s*/, "").trim();
}

function summaryFromText(value?: string) {
  if (!value) return "";
  return value.split(/\r?\n/).find((line) => line.trim())?.replace(/^[-*]\s*/, "").trim() ?? "";
}
