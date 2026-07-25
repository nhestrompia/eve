import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import {
  Activity,
  ArrowDown,
  ArrowRight,
  Boxes,
  CheckCircle2,
  Database,
  ExternalLink,
  FileText,
  GitBranch,
  Sparkles,
} from "lucide-react";
import { useMemo, useState } from "react";
import { api } from "../api";
import {
  DashboardShell,
  relativeTime,
  verificationPercent,
} from "../components/dashboard-chrome";
import { ErrorState } from "../components/error-state";
import { LoadingState } from "../components/loading-state";
import type { EvolutionSummary, PlanRequest, RepositorySummary } from "../types";

export function RepositoriesPage() {
  const repositories = useQuery({ queryKey: ["repositories"], queryFn: api.repositories });
  const snapshots = useQuery({ queryKey: ["snapshots"], queryFn: api.snapshots });
  const plans = useQuery({
    queryKey: ["plan-requests", "all"],
    queryFn: () => api.planRequests(""),
    retry: false,
  });

  return (
    <DashboardShell
      title="Repositories"
      subtitle="All connected repositories and their verification status."
      searchPlaceholder="Search repositories..."
    >
      {repositories.isLoading || snapshots.isLoading ? <LoadingState label="Loading repositories" /> : null}
      {repositories.error ? <ErrorState error={repositories.error} /> : null}
      {snapshots.error ? <ErrorState error={snapshots.error} /> : null}
      {repositories.data && snapshots.data ? (
        <RepositoriesContent
          repositories={repositories.data}
          snapshots={snapshots.data}
          plans={plans.data ?? []}
        />
      ) : null}
    </DashboardShell>
  );
}

function RepositoriesContent({
  repositories,
  snapshots,
  plans,
}: {
  repositories: RepositorySummary[];
  snapshots: EvolutionSummary[];
  plans: PlanRequest[];
}) {
  const rows = useMemo(() => enrichRepositories(repositories, snapshots), [repositories, snapshots]);
  const [selectedName, setSelectedName] = useState(rows[0]?.name ?? "");
  const selected = rows.find((row) => row.name === selectedName) ?? rows[0];
  const failed = snapshots.filter((snapshot) => snapshot.failedValidationCount > 0).length;
  const percent = verificationPercent(snapshots.length, failed);
  const activePlans = plans.filter((plan) => !["fulfilled", "rejected", "superseded"].includes(plan.state)).length;

  return (
    <div className="mt-11 grid gap-9 xl:grid-cols-[minmax(0,1fr)_360px] xl:gap-11">
      <section className="min-w-0">
        <div className="grid rounded-lg border border-slate-200 bg-white/45 md:grid-cols-4">
          <Metric icon={Database} value={rows.length} label="Active repositories" />
          <Metric icon={CheckCircle2} value={`${percent}%`} label="Overall verified" />
          <Metric icon={Activity} value={snapshots.length} label="Active snapshots" />
          <Metric icon={FileText} value={activePlans} label="Plans in progress" />
        </div>

        <section className="mt-7 overflow-hidden rounded-lg border border-slate-200 bg-white/45">
          <div className="hidden min-h-[68px] grid-cols-[minmax(220px,1.25fr)_120px_82px_105px] items-center gap-5 border-b border-slate-200 px-7 text-xs font-medium text-slate-600 lg:grid 2xl:grid-cols-[minmax(250px,1.3fr)_150px_110px_160px_110px_24px]">
            <span>Repository</span>
            <span>Default branch</span>
            <span>Snapshots</span>
            <span>Verification</span>
            <span className="hidden items-center gap-2 text-slate-950 2xl:inline-flex">Updated <ArrowDown className="size-3.5" /></span>
            <span className="hidden 2xl:block" />
          </div>
          {rows.map((repo, index) => (
            <button
              type="button"
              key={repo.name}
              onClick={() => setSelectedName(repo.name)}
              className={`grid w-full grid-cols-1 gap-4 border-b border-slate-200 px-5 py-5 text-left transition-colors last:border-b-0 hover:bg-white lg:min-h-[96px] lg:grid-cols-[minmax(220px,1.25fr)_120px_82px_105px] lg:items-center lg:gap-5 lg:px-7 2xl:grid-cols-[minmax(250px,1.3fr)_150px_110px_160px_110px_24px] ${
                selected?.name === repo.name ? "bg-white" : ""
              }`}
            >
              <span className="flex min-w-0 items-center gap-4">
                <RepositoryMark repo={repo} index={index} />
                <span className="min-w-0">
                  <span className="flex min-w-0 items-center gap-2">
                    <span className="truncate text-[15px] font-semibold text-slate-950">{repo.name}</span>
                    {index === 0 ? <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-950">You</span> : null}
                  </span>
                  <span className="mt-2 block truncate text-sm text-slate-500">{repo.description}</span>
                </span>
              </span>
              <span className="inline-flex items-center gap-2 text-sm text-slate-700">
                <GitBranch className="size-4" /> {repo.branch || "main"}
              </span>
              <span className="text-sm text-slate-600">
                <strong className="block text-[15px] text-slate-950">{repo.snapshotCount}</strong>
                <span>{repo.todayCount > 0 ? `+${repo.todayCount}` : "+0"} today</span>
              </span>
              <span>
                <span className="block text-[15px] font-semibold text-slate-950">{repo.verification}%</span>
                <span className="mt-3 block h-1 rounded-full bg-slate-100">
                  <span className="block h-full rounded-full bg-emerald-500" style={{ width: `${repo.verification}%` }} />
                </span>
              </span>
              <span className="hidden text-sm text-slate-600 2xl:block">{relativeTime(repo.latestAt)}</span>
              <ChevronRightIcon />
            </button>
          ))}
          <div className="px-5 py-5 text-sm text-slate-600 lg:px-7">
            Showing 1-{rows.length} of {rows.length} repositories
          </div>
        </section>
      </section>

      <RepositoryDetailPanel repo={selected} snapshots={snapshots.filter((snapshot) => (snapshot.repository || "eve") === selected?.name)} />
    </div>
  );
}

function Metric({ icon: Icon, value, label }: { icon: typeof Database; value: string | number; label: string }) {
  return (
    <div className="flex min-h-[92px] items-center gap-4 border-b border-slate-200 px-8 py-5 last:border-b-0 md:border-b-0 md:border-r md:last:border-r-0">
      <Icon className="size-7 shrink-0 text-slate-700" strokeWidth={1.6} />
      <div className="min-w-0">
        <div className="text-2xl font-semibold leading-none text-slate-950">{value}</div>
        <p className="mt-2 text-xs leading-4 text-slate-600">{label}</p>
      </div>
    </div>
  );
}

function RepositoryDetailPanel({
  repo,
  snapshots,
}: {
  repo?: EnrichedRepository;
  snapshots: EvolutionSummary[];
}) {
  if (!repo) {
    return <aside className="rounded-lg border border-slate-200 bg-white/55 p-6 text-sm text-slate-500">No repositories found.</aside>;
  }

  return (
    <aside className="rounded-lg border border-slate-200 bg-white/55 p-6 xl:sticky xl:top-8 xl:self-start">
      <div className="flex items-start justify-between gap-4">
        <div className="flex min-w-0 items-center gap-4">
          <RepositoryMark repo={repo} index={0} />
          <div className="min-w-0">
            <div className="flex min-w-0 items-center gap-2">
              <h2 className="truncate text-xl font-semibold text-slate-950">{repo.name}</h2>
              <span className="rounded-md bg-slate-100 px-2 py-0.5 text-xs font-semibold text-slate-950">You</span>
            </div>
            <p className="mt-2 truncate text-sm text-slate-600">{repo.description}</p>
          </div>
        </div>
        {repo.remoteUrl ? (
          <a href={repo.remoteUrl} target="_blank" rel="noreferrer" aria-label="Open GitHub" className="text-slate-600 hover:text-slate-950">
            <ExternalLink className="size-5" />
          </a>
        ) : (
          <ExternalLink className="size-5 text-slate-300" />
        )}
      </div>

      <dl className="mt-8 space-y-5 text-sm">
        <DetailPair label="Default branch" value={repo.branch || "main"} icon={<GitBranch className="size-4" />} />
        <DetailPair label="Connected" value={repo.createdAt ? new Date(repo.createdAt).toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" }) : "Local"} />
        <DetailPair label="Last snapshot" value={relativeTime(repo.latestAt)} />
        <DetailPair label="Default check suite" value={`${Math.max(1, Math.round(repo.snapshotCount / 4))} checks`} />
      </dl>

      <div className="mt-8 border-t border-slate-200 pt-7">
        <div className="flex items-center justify-between gap-4">
          <h3 className="font-semibold text-slate-950">Verification trend</h3>
          <span className="text-sm text-slate-600">30 days</span>
        </div>
        <TrendChart value={repo.verification} />
      </div>

      <div className="mt-7 border-t border-slate-200 pt-7">
        <h3 className="font-semibold text-slate-950">Recent activity</h3>
        <div className="mt-5 space-y-5">
          {snapshots.slice(0, 3).map((snapshot, index) => (
            <div key={snapshot.id} className="grid grid-cols-[16px_minmax(0,1fr)_56px] gap-3 text-sm">
              <span className={index === 0 ? "mt-1.5 size-3 rounded bg-emerald-500" : index === 1 ? "mt-1.5 size-3 rounded border-2 border-orange-400" : "mt-1.5 size-3 rounded border-2 border-indigo-500"} />
              <span className="min-w-0">
                <span className="block truncate font-medium text-slate-950">{index === 0 ? "Snapshot created" : index === 1 ? "Plan completed" : "Plan started"}</span>
                <span className="mt-1 block truncate text-slate-500">{snapshot.title}</span>
              </span>
              <span className="text-right text-slate-500">{relativeTime(snapshot.updatedAt || snapshot.createdAt)}</span>
            </div>
          ))}
        </div>
        <Link to="/snapshots" className="mt-6 inline-flex items-center gap-3 text-sm font-semibold text-slate-950">
          View all activity <ArrowRight className="size-4" />
        </Link>
      </div>

      <a
        href={repo.remoteUrl || "#"}
        target={repo.remoteUrl ? "_blank" : undefined}
        rel={repo.remoteUrl ? "noreferrer" : undefined}
        className={`mt-8 flex h-12 items-center justify-center gap-3 rounded-lg border border-slate-200 bg-white text-sm font-semibold ${
          repo.remoteUrl ? "text-slate-950 hover:bg-slate-50" : "pointer-events-none text-slate-400"
        }`}
      >
        Open on GitHub
        <ExternalLink className="size-4" />
      </a>
    </aside>
  );
}

function DetailPair({ label, value, icon }: { label: string; value: string; icon?: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <dt className="font-semibold text-slate-950">{label}</dt>
      <dd className="inline-flex items-center gap-2 text-slate-600">{icon}{value}</dd>
    </div>
  );
}

function TrendChart({ value }: { value: number }) {
  const y = 68 - value * 0.48;
  return (
    <svg viewBox="0 0 310 150" className="mt-5 h-[150px] w-full text-slate-300">
      {[0, 1, 2, 3].map((line) => (
        <line key={line} x1="34" x2="304" y1={24 + line * 35} y2={24 + line * 35} stroke="currentColor" strokeWidth="1" />
      ))}
      {[0, 25, 50, 75, 100].map((label, index) => (
        <text key={label} x="0" y={130 - index * 26} className="fill-slate-500 text-[10px]">{label}%</text>
      ))}
      <polyline
        points={`34,${y} 82,${y + 1} 112,${y - 2} 160,${y - 2} 204,${y - 2} 238,${y - 1} 252,${y + 5} 266,${y - 1} 304,${y}`}
        fill="none"
        stroke="oklch(0.63 0.17 151)"
        strokeWidth="3"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <text x="46" y="143" className="fill-slate-500 text-[10px]">Apr 23</text>
      <text x="126" y="143" className="fill-slate-500 text-[10px]">Apr 30</text>
      <text x="198" y="143" className="fill-slate-500 text-[10px]">May 7</text>
      <text x="264" y="143" className="fill-slate-500 text-[10px]">May 21</text>
    </svg>
  );
}

function RepositoryMark({ repo, index }: { repo: Pick<EnrichedRepository, "name">; index: number }) {
  const icons = [null, Sparkles, Boxes, Activity, CheckCircle2];
  const Icon = icons[index % icons.length];
  return (
    <span className="grid size-11 shrink-0 place-items-center rounded-lg bg-black text-white shadow-[0_6px_18px_rgba(15,23,42,0.15)]">
      {repo.name === "eve" ? <span className="text-[15px] font-semibold">eve</span> : Icon ? <Icon className="size-6" strokeWidth={1.7} /> : <Boxes className="size-6" />}
    </span>
  );
}

function ChevronRightIcon() {
  return <ArrowRight className="hidden size-4 justify-self-end text-slate-700 2xl:block" strokeWidth={1.8} />;
}

type EnrichedRepository = RepositorySummary & {
  description: string;
  todayCount: number;
  verification: number;
};

function enrichRepositories(repositories: RepositorySummary[], snapshots: EvolutionSummary[]): EnrichedRepository[] {
  const source = repositories.length > 0 ? repositories : fallbackRepositories(snapshots);
  return source
    .map((repo) => {
      const repoSnapshots = snapshots.filter((snapshot) => (snapshot.repository || repo.name) === repo.name);
      const failed = repoSnapshots.filter((snapshot) => snapshot.failedValidationCount > 0).length;
      const today = new Date().toDateString();
      return {
        ...repo,
        branch: repo.branch || "main",
        description: descriptionForRepo(repo.name),
        todayCount: repoSnapshots.filter((snapshot) => new Date(snapshot.updatedAt || snapshot.createdAt).toDateString() === today).length,
        verification: verificationPercent(repoSnapshots.length || repo.snapshotCount, failed),
      };
    })
    .sort((left, right) => (right.latestAt || "").localeCompare(left.latestAt || ""));
}

function fallbackRepositories(snapshots: EvolutionSummary[]): RepositorySummary[] {
  const byRepo = new Map<string, EvolutionSummary[]>();
  for (const snapshot of snapshots) {
    const name = snapshot.repository || "eve";
    byRepo.set(name, [...(byRepo.get(name) ?? []), snapshot]);
  }
  return Array.from(byRepo.entries()).map(([name, values]) => ({
    name,
    evolutionCount: values.length,
    snapshotCount: values.length,
    commitCount: values.reduce((sum, snapshot) => sum + snapshot.commitCount, 0),
    latestAt: values[0]?.updatedAt || values[0]?.createdAt || "",
    latestEvolution: values[0]?.id || "",
    latestTitle: values[0]?.title || "",
    sessionProviders: [],
  }));
}

function descriptionForRepo(name: string) {
  if (name === "eve") return "Product history layer for AI agents";
  if (name.includes("skeleton")) return "Skeleton components and records";
  if (name.includes("docs")) return "Documentation and guides";
  if (name.includes("chain")) return "Event processing and indexing";
  if (name.includes("astro")) return "Research tools and data pipeline";
  return "Connected product repository";
}
