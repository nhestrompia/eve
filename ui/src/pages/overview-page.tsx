import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  ArrowRight,
  Boxes,
  Clock3,
  FileText,
  Gauge,
  Zap,
} from "lucide-react";
import { useState } from "react";
import { api } from "../api";
import {
  AgentAvatar,
  DashboardShell,
  Pill,
  ViewAllLink,
  relativeTime,
  shortHash,
  snapshotVisualStatus,
  verificationPercent,
} from "../components/dashboard-chrome";
import { ErrorState } from "../components/error-state";
import { PlanApprovalDialog, currentRevision } from "../components/pending-plan-banner";
import { LoadingState } from "../components/loading-state";
import type { EvolutionSummary, PlanRequest, RepositorySummary } from "../types";

export function OverviewPage() {
  const snapshots = useQuery({ queryKey: ["snapshots"], queryFn: api.snapshots });
  const repositories = useQuery({ queryKey: ["repositories"], queryFn: api.repositories });
  const plans = useQuery({
    queryKey: ["plan-requests", "all"],
    queryFn: () => api.planRequests(""),
    refetchInterval: 5_000,
    retry: false,
  });

  return (
    <DashboardShell
      title="Overview"
      subtitle="The pulse of your AI agents and product evolution."
      searchPlaceholder="Search snapshots, plans..."
    >
      {snapshots.isLoading || repositories.isLoading ? <LoadingState label="Loading overview" /> : null}
      {snapshots.error ? <ErrorState error={snapshots.error} /> : null}
      {repositories.error ? <ErrorState error={repositories.error} /> : null}
      {snapshots.data && repositories.data ? (
        <OverviewContent
          snapshots={snapshots.data}
          repositories={repositories.data}
          plans={plans.data ?? []}
        />
      ) : null}
    </DashboardShell>
  );
}

function OverviewContent({
  snapshots,
  repositories,
  plans,
}: {
  snapshots: EvolutionSummary[];
  repositories: RepositorySummary[];
  plans: PlanRequest[];
}) {
  const pendingPlans = plans.filter((plan) => plan.state === "pending_approval");
  const [approvalOpen, setApprovalOpen] = useState(false);
  const latestSnapshots = snapshots.slice(0, 5);
  const failed = snapshots.filter((snapshot) => snapshot.failedValidationCount > 0).length;
  const percent = verificationPercent(snapshots.length, failed);
  const activeAgents = buildActiveAgentRows(plans);
  const activePlans = plans.filter((plan) => ["pending_approval", "locked"].includes(plan.state));
  const agentsPaused = pendingPlans.length;
  const showAttention = pendingPlans.length > 0 || failed > 0 || agentsPaused > 0;

  return (
    <div className="mt-14 grid gap-8 xl:grid-cols-[minmax(0,1fr)_274px] xl:gap-11">
      <section className="min-w-0">
        {showAttention ? (
          <div className="flex flex-col gap-5 border-b border-slate-200 pb-11 lg:flex-row lg:items-center lg:justify-between">
            <div className="flex min-w-0 flex-col gap-4 sm:flex-row sm:items-center sm:gap-6">
              <AlertTriangle className="size-7 shrink-0 text-slate-950" strokeWidth={1.6} />
              <div className="flex min-w-0 flex-wrap items-center gap-x-6 gap-y-2 text-sm font-semibold text-slate-950">
                <span>{pendingPlans.length} {pendingPlans.length === 1 ? "plan" : "plans"} waiting for your approval</span>
                <span className="hidden text-slate-400 sm:inline">•</span>
                <span>{failed} snapshots awaiting decisions</span>
                <span className="hidden text-slate-400 sm:inline">•</span>
                <span>{agentsPaused} {agentsPaused === 1 ? "agent" : "agents"} paused</span>
              </div>
            </div>
            <button
              type="button"
              onClick={() => setApprovalOpen(true)}
              className="inline-flex h-10 w-fit items-center justify-center gap-3 rounded-lg bg-slate-950 px-5 text-sm font-semibold text-white shadow-[0_12px_28px_rgba(15,23,42,0.16)] transition-colors hover:bg-slate-800"
            >
              Review
              <ArrowRight className="size-4" strokeWidth={1.8} />
            </button>
          </div>
        ) : null}

        <section className={showAttention ? "mt-10" : "mt-0"}>
          <div className="flex items-center justify-between gap-4">
            <h2 className="text-xl font-semibold tracking-[-0.01em] text-slate-950">Recent snapshots</h2>
            <ViewAllLink to="/snapshots" />
          </div>
          <div className="mt-6 overflow-hidden border-y border-slate-200">
            {latestSnapshots.map((snapshot) => (
              <SnapshotRow key={snapshot.id} snapshot={snapshot} />
            ))}
          </div>
          <div className="mt-6">
            <ViewAllLink to="/snapshots" label="See all snapshots" />
          </div>
        </section>

        <div className="mt-16 flex flex-col gap-4 border-t border-slate-200 pt-7 text-xs text-slate-500 sm:flex-row sm:items-center sm:gap-12">
          <Legend tone="verified" label="Verified (check suite passed)" />
          <Legend tone="waiting" label="Awaiting decision (agent-reported)" />
          <Legend tone="pending" label="Investigation (no deterministic check)" />
        </div>
      </section>

      <aside className={`grid min-w-0 gap-5 md:grid-cols-2 xl:block xl:space-y-5 ${showAttention ? "xl:-mt-[66px]" : ""}`}>
        <OverviewCard>
          <div className="flex items-center justify-between gap-3">
            <h2 className="font-semibold text-slate-950">System overview</h2>
            <span className="inline-flex items-center gap-2 text-xs text-slate-500">
              <span className="size-1.5 rounded-full bg-emerald-500" />
              Live
            </span>
          </div>
          <div className="mt-7">
            <div className="text-[42px] font-semibold leading-none tracking-[-0.04em] text-slate-950">{percent}%</div>
            <div className="mt-1 text-[15px] font-semibold text-slate-950">verified</div>
            <p className="mt-3 max-w-[25ch] text-sm leading-5 text-slate-500">
              Completions that passed their declared check suites.
            </p>
          </div>
          <div className="mt-7 grid grid-cols-3 gap-3 border-t border-slate-200 pt-5">
            <MiniStat icon={Boxes} value={repositories.length} label="repos" />
            <MiniStat icon={Zap} value={activeAgents.length} label="agents" />
            <MiniStat icon={Clock3} value={activePlans.length} label="active plans" />
          </div>
        </OverviewCard>

        <OverviewCard>
          <div className="flex items-center justify-between gap-3">
            <h2 className="font-semibold text-slate-950">Active agents</h2>
            <span className="inline-flex items-center gap-2 text-xs text-slate-500">
              <span className={`size-1.5 rounded-full ${activeAgents.length > 0 ? "bg-emerald-500" : "bg-slate-300"}`} />
              {activeAgents.length} running
            </span>
          </div>
          {activeAgents.length > 0 ? (
            <div className="mt-6 space-y-6">
              {activeAgents.slice(0, 2).map((agent) => (
                <ActiveAgentRow key={`${agent.repository}-${agent.label}`} {...agent} />
              ))}
            </div>
          ) : (
            <div className="mt-6 rounded-lg border border-slate-200 bg-slate-50/70 px-4 py-4">
              <p className="text-sm font-medium text-slate-950">No active agents</p>
              <p className="mt-1 text-xs leading-5 text-slate-500">
                Locked plans appear here while implementation is underway.
              </p>
            </div>
          )}
        </OverviewCard>

        <OverviewCard className="p-0">
          <div className="flex items-center justify-between gap-3 px-5 pt-5">
            <h2 className="font-semibold text-slate-950">Plans</h2>
            <ViewAllLink to="/plans" />
          </div>
          <div className="px-5 pb-5 pt-5">
            <div className="flex gap-3">
              <FileText className="mt-0.5 size-5 shrink-0 text-slate-600" strokeWidth={1.7} />
              <div className="min-w-0">
                <p className="truncate font-medium text-slate-950">
                  {(pendingPlans[0] ? currentRevision(pendingPlans[0])?.goal : undefined) || "No plans awaiting approval"}
                </p>
                <p className="mt-1 text-xs font-medium text-orange-600">
                  {pendingPlans.length > 0 ? "Waiting approval" : "Queue clear"}
                </p>
              </div>
            </div>
          </div>
        </OverviewCard>

        <OverviewCard className="p-0">
          <div className="flex items-center justify-between gap-3 px-5 pt-5">
            <h2 className="font-semibold text-slate-950">Repositories</h2>
            <ViewAllLink to="/repositories" />
          </div>
          <div className="px-5 pb-6 pt-5">
            <div className="text-3xl font-semibold leading-none text-slate-950">{repositories.length}</div>
            <p className="mt-2 text-sm text-slate-500">Active repositories</p>
          </div>
        </OverviewCard>
      </aside>

      <PlanApprovalDialog plans={pendingPlans} open={approvalOpen} onOpenChange={setApprovalOpen} />
    </div>
  );
}

function SnapshotRow({ snapshot }: { snapshot: EvolutionSummary }) {
  const status = snapshotVisualStatus(snapshot);
  const agent = agentName(snapshot);
  return (
    <a
      href={`/snapshots/${encodeURIComponent(snapshot.id)}`}
      className="grid min-h-[86px] grid-cols-[24px_minmax(70px,96px)_minmax(0,1fr)] items-center gap-5 border-t border-slate-200 px-0 py-4 first:border-t-0 sm:grid-cols-[24px_92px_minmax(0,1fr)_80px_120px] sm:gap-6"
    >
      <span className={status.markerClass} />
      <span className="font-mono text-sm font-semibold text-slate-900">{shortHash(snapshot.snapshot || snapshot.id)}</span>
      <span className="min-w-0">
        <span className="block truncate text-[15px] font-medium text-slate-950">{snapshot.title}</span>
        <span className="mt-1 block truncate text-xs text-slate-500">
          {snapshot.repository || "eve"} <span className="px-1.5">•</span> {agent}
        </span>
      </span>
      <span className="hidden text-xs text-slate-500 sm:block">{relativeTime(snapshot.updatedAt || snapshot.createdAt)}</span>
      <span className="hidden justify-self-end sm:block"><Pill tone={status.tone}>{status.label}</Pill></span>
    </a>
  );
}

function Legend({ tone, label }: { tone: "verified" | "waiting" | "pending"; label: string }) {
  const classes = {
    verified: "size-3.5 rounded bg-emerald-500",
    waiting: "size-3.5 rounded border-2 border-orange-400",
    pending: "size-3.5 rounded border border-dashed border-slate-500",
  };
  return (
    <span className="inline-flex items-center gap-3">
      <span className={classes[tone]} />
      {label}
    </span>
  );
}

function OverviewCard({ children, className = "" }: { children: React.ReactNode; className?: string }) {
  return (
    <section className={`rounded-lg border border-slate-200 bg-white/55 p-5 shadow-[0_1px_2px_rgba(15,23,42,0.025)] ${className}`}>
      {children}
    </section>
  );
}

function MiniStat({ icon: Icon, value, label }: { icon: typeof Gauge; value: number; label: string }) {
  return (
    <div className="min-w-0 text-center">
      <div className="flex items-center justify-center gap-2 text-slate-950">
        <Icon className="size-4" strokeWidth={1.8} />
        <span className="text-lg font-semibold leading-none">{value}</span>
      </div>
      <p className="mt-2 truncate text-xs text-slate-500">{label}</p>
    </div>
  );
}

function ActiveAgentRow({ agent, label, repository }: { agent: string; label: string; repository: string }) {
  return (
    <div className="grid grid-cols-[36px_minmax(0,1fr)] gap-3">
      <AgentAvatar agent={agent} />
      <div className="min-w-0">
        <div className="flex items-center justify-between gap-3">
          <div className="min-w-0">
            <p className="truncate font-semibold text-slate-950">{agent}</p>
            <p className="mt-0.5 truncate text-[11px] text-slate-500">{label}</p>
          </div>
          <Pill tone="progress">In progress</Pill>
        </div>
        <p className="mt-2 truncate text-[11px] text-slate-500">{repository}</p>
      </div>
    </div>
  );
}

function agentName(snapshot: EvolutionSummary) {
  const provider = snapshot.sessionProviders.find(Boolean) || "Codex";
  return provider.charAt(0).toUpperCase() + provider.slice(1);
}

export function buildActiveAgentRows(plans: PlanRequest[]) {
  return plans
    .filter((plan) => plan.state === "locked")
    .map((plan) => ({
      agent: "Agent",
      label: currentRevision(plan)?.goal || "Implementing an approved plan",
      repository: plan.repository,
    }));
}
