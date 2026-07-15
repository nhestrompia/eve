import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import {
  AlertTriangle,
  ArrowRight,
  BookOpen,
  CircleHelp,
  Code2,
  Globe,
  Package,
  ShieldAlert,
  Zap,
  type LucideIcon,
} from "lucide-react";
import type { ReactNode } from "react";
import { useEffect, useRef, useState } from "react";
import { api } from "../api";
import { compactDate } from "../format";
import type {
  DetailResponse,
  EvolutionSummary,
  RepositorySummary,
} from "../types";
import { StatusBadge } from "./status-badge";

export function RepositoryActivityView({
  repositories,
  evolutions,
  selectedRepo,
}: {
  repositories: RepositorySummary[];
  evolutions: EvolutionSummary[];
  selectedRepo?: string;
}) {
  const title = selectedRepo ?? "Product snapshots";
  const details = useQuery({
    queryKey: [
      "activity-overview-details",
      evolutions
        .map((evolution) => `${evolution.repository ?? ""}:${evolution.id}`)
        .join(","),
      evolutions
        .map((evolution) => `${evolution.repository ?? ""}:${evolution.id}`)
        .join(","),
    ],
    queryFn: () =>
      Promise.all(
        evolutions.map((evolution) =>
          api.snapshotDetail(evolution.id, evolution.repository),
        ),
      ),
    enabled: evolutions.length > 0,
    staleTime: 30_000,
  });
  const rows = [...evolutions].sort(
    (left, right) =>
      timestamp(right.updatedAt || right.createdAt) -
      timestamp(left.updatedAt || left.createdAt),
  );
  const repoRows =
    repositories.length > 0
      ? repositories
      : buildFallbackRepositories(evolutions, selectedRepo);
  const latestRepoRows = sortRepositoriesByLatestUse(repoRows);
  const detailRows = details.data ?? [];
  const stats = buildPlatformStats(evolutions, detailRows, repoRows);

  return (
    <main className="min-h-dvh min-w-0 bg-background px-4 py-5 sm:px-6 sm:py-7 lg:px-8 lg:py-8">
      <div className="grid grid-cols-1 gap-6 2xl:grid-cols-[minmax(0,1fr)_350px] 2xl:gap-7">
        <section className="min-w-0 space-y-7">
          <header className="space-y-5">
            <div>
              <p className="font-mono text-xs font-semibold uppercase tracking-wide text-slate-600">
                Overview
              </p>
              <h1 className="mt-4 text-2xl font-semibold leading-tight tracking-[-0.01em] text-balance sm:text-[34px]">
                {title}
              </h1>
              <p className="mt-3 max-w-[54ch] text-sm leading-6 text-muted-foreground text-pretty">
                {selectedRepo
                  ? "Review the product states EVE has recorded for this repository, including snapshots, verification, and implementation history."
                  : "Review recorded product states and jump back into the repositories you touched most recently."}
              </p>
            </div>
            <section
              className="min-w-0 space-y-2.5"
              aria-label="Recently used repositories"
            >
              <div className="flex items-center justify-between gap-3">
                <h2 className="text-sm font-semibold text-slate-700">
                  Recent repositories
                </h2>
                <span className="text-xs text-muted-foreground">
                  {latestRepoRows.length}{" "}
                  {latestRepoRows.length === 1 ? "repo" : "repos"}
                </span>
              </div>
              <div className="scrollbar-none -mx-4 flex min-w-0 snap-x gap-3 overflow-x-auto px-4 pb-1 sm:-mx-6 sm:px-6 lg:-mx-8 lg:px-8">
                {latestRepoRows.map((repo, index) => (
                  <RepositoryOverviewCard
                    key={repo.name}
                    repo={repo}
                    tone={REPO_TONES[index % REPO_TONES.length]}
                    selected={selectedRepo === repo.name}
                    activityValues={repositoryVelocityBuckets(
                      repo,
                      evolutions,
                      repoRows,
                    )}
                  />
                ))}
              </div>
            </section>
          </header>

          <section className="space-y-3">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between sm:gap-4">
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-semibold">Snapshot activity</h2>
                <CircleHelp className="size-4 text-muted-foreground" />
              </div>
              {/* <button className="inline-flex h-9 w-fit items-center gap-2 rounded-md bg-white px-3 text-xs font-semibold shadow-[0_0_0_1px_rgba(15,23,42,0.12)] transition-colors hover:bg-slate-50">
                Last 12 months <ChevronDown className="size-4" />
              </button> */}
            </div>
            <ContributionGraph evolutions={evolutions} />
          </section>

          <RecentActivityPanel
            evolutions={rows}
            repositories={repoRows}
            details={detailRows}
            selectedRepo={selectedRepo}
          />
        </section>

        <aside className="grid min-w-0 grid-cols-1 gap-5 lg:grid-cols-2 2xl:block 2xl:space-y-5">
          <PlatformOverview stats={stats} />
          <VelocityPanel repositories={repoRows} evolutions={evolutions} />
          <AgentContributionPanel
            evolutions={evolutions}
            details={detailRows}
          />
          <NeedsAttentionPanel stats={stats} />
        </aside>
      </div>
    </main>
  );
}

function RepositoryOverviewCard({
  repo,
  tone,
  selected,
  activityValues,
}: {
  repo: RepositorySummary;
  tone: RepoTone;
  selected: boolean;
  activityValues: number[];
}) {
  const Icon = tone.icon;
  return (
    <Link
      to="/repositories/$repo"
      params={{ repo: repo.name }}
      className={`min-w-[224px] w-[224px] snap-start rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.1)] transition-[background-color,box-shadow,scale] duration-150 hover:bg-slate-50 active:scale-[0.96] sm:min-w-[252px] sm:w-[252px] ${selected ? "ring-2 ring-blue-500/20" : ""}`}
    >
      <div className="flex items-center justify-between gap-3">
        <span className="flex min-w-0 flex-1 items-center gap-2">
          <span className={`size-2.5 rounded-full ${tone.bg}`} />
          <span className="min-w-0 truncate text-sm font-semibold">
            {repo.name}
          </span>
        </span>
        <Icon className={`size-4 shrink-0 ${tone.text}`} />
      </div>
      <div className="mt-4">
        <RepoSparkline tone={tone} values={activityValues} />
      </div>
      <p className="mt-3 min-w-0 text-xs leading-4 text-muted-foreground">
        <span className="block overflow-hidden text-ellipsis whitespace-nowrap">
          {repo.evolutionCount} EVs · {repo.snapshotCount} snaps
        </span>
        <span className="block overflow-hidden text-ellipsis whitespace-nowrap">
          {repo.commitCount} {repo.commitCount === 1 ? "commit" : "commits"}
        </span>
      </p>
    </Link>
  );
}

type RepoTone = {
  bg: string;
  text: string;
  soft: string;
  line: string;
  icon: LucideIcon;
};

const REPO_TONES: RepoTone[] = [
  {
    bg: "bg-blue-600",
    text: "text-blue-600",
    soft: "bg-blue-50",
    line: "#2563eb",
    icon: BookOpen,
  },
  {
    bg: "bg-emerald-500",
    text: "text-emerald-600",
    soft: "bg-emerald-50",
    line: "#10b981",
    icon: BookOpen,
  },
  {
    bg: "bg-violet-600",
    text: "text-violet-600",
    soft: "bg-violet-50",
    line: "#7c3aed",
    icon: Globe,
  },
  {
    bg: "bg-amber-500",
    text: "text-amber-600",
    soft: "bg-amber-50",
    line: "#f59e0b",
    icon: Code2,
  },
];

function RepoSparkline({ tone, values }: { tone: RepoTone; values: number[] }) {
  const points = compressSeries(values, 11)
    .map((value, index) => `${index * 10},${46 - sparklineY(value, values)}`)
    .join(" ");
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 104 48"
      className="h-8 w-full overflow-visible"
    >
      <polyline
        points={points}
        fill="none"
        stroke={tone.line}
        strokeWidth="3"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function ContributionGraph({ evolutions }: { evolutions: EvolutionSummary[] }) {
  const graphMeasureRef = useRef<HTMLDivElement>(null);
  const [visibleWeeks, setVisibleWeeks] = useState(MIN_ACTIVITY_WEEKS);
  const counts = new Map<string, ActivityDay>();
  for (const evolution of evolutions) {
    const date = parseDate(evolution.updatedAt || evolution.createdAt);
    if (!date) continue;
    const key = isoDay(date);
    const current = counts.get(key) ?? { evolutionCount: 0, commitCount: 0 };
    counts.set(key, {
      evolutionCount: current.evolutionCount + 1,
      commitCount: current.commitCount + (evolution.commitCount ?? 0),
    });
  }

  useEffect(() => {
    const element = graphMeasureRef.current;
    if (!element) return;

    const updateVisibleWeeks = () => {
      const width = element.getBoundingClientRect().width;
      const usableWidth = Math.max(0, width - DAY_LABEL_WIDTH);
      const nextWeeks = clamp(
        Math.floor(
          (usableWidth + ACTIVITY_CELL_GAP) /
            (ACTIVITY_CELL_SIZE + ACTIVITY_CELL_GAP),
        ),
        MIN_ACTIVITY_WEEKS,
        MAX_ACTIVITY_WEEKS,
      );
      setVisibleWeeks((current) =>
        current === nextWeeks ? current : nextWeeks,
      );
    };

    updateVisibleWeeks();
    const observer = new ResizeObserver(updateVisibleWeeks);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  const today = new Date();
  const start = new Date(today);
  start.setDate(today.getDate() - (visibleWeeks - 1) * 7);
  start.setDate(start.getDate() - start.getDay());
  const weeks = Array.from({ length: visibleWeeks }, (_, week) =>
    Array.from({ length: 7 }, (_, day) => {
      const date = new Date(start);
      date.setDate(start.getDate() + week * 7 + day);
      const activity = counts.get(isoDay(date)) ?? {
        evolutionCount: 0,
        commitCount: 0,
      };
      return { date, ...activity };
    }),
  );
  const monthLabels = buildMonthLabels(weeks);
  const maxCount = Math.max(
    0,
    ...weeks.flat().map((day) => day.evolutionCount),
  );
  const peakCommits = Math.max(
    0,
    ...weeks.flat().map((day) => day.commitCount),
  );

  return (
    <div className="w-full rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
      <div ref={graphMeasureRef} className="w-full">
        <div
          className="mb-2 grid gap-x-[3px] text-xs text-muted-foreground"
          style={{
            gridTemplateColumns: `${DAY_LABEL_WIDTH}px repeat(${visibleWeeks}, ${ACTIVITY_CELL_SIZE}px)`,
            columnGap: ACTIVITY_CELL_GAP,
          }}
        >
          <span />
          {monthLabels.map((label) => (
            <span
              key={`${label.name}-${label.week}`}
              className="h-4 overflow-visible whitespace-nowrap"
              style={{ gridColumn: `${label.week + 2} / span ${label.span}` }}
            >
              {label.name}
            </span>
          ))}
        </div>
        <div
          className="grid"
          style={{
            gridTemplateColumns: `${DAY_LABEL_WIDTH}px repeat(${visibleWeeks}, ${ACTIVITY_CELL_SIZE}px)`,
            columnGap: ACTIVITY_CELL_GAP,
          }}
        >
          <div
            className="grid grid-rows-7 text-xs leading-3 text-muted-foreground"
            style={{ rowGap: ACTIVITY_CELL_GAP }}
          >
            {["", "Mon", "", "Wed", "", "Fri", ""].map((label, index) => (
              <span key={`${label}-${index}`} className="h-3">
                {label}
              </span>
            ))}
          </div>
          {weeks.map((week) => (
            <div
              key={isoDay(week[0].date)}
              className="grid grid-rows-7"
              style={{ rowGap: ACTIVITY_CELL_GAP }}
            >
              {week.map((day) => {
                const label = activityLabel(day);
                return (
                  <span
                    key={isoDay(day.date)}
                    aria-label={label}
                    title={label}
                    className={`group/day relative block size-3 rounded-[2px] ${heatClass(day.evolutionCount, maxCount)} ${day.date > today ? "opacity-40" : ""}`}
                  >
                    <span className="pointer-events-none absolute bottom-5 left-1/2 z-10 hidden -translate-x-1/2 whitespace-nowrap rounded-md bg-slate-950 px-2 py-1 text-[11px] font-medium text-white shadow-lg group-hover/day:block">
                      {label}
                    </span>
                  </span>
                );
              })}
            </div>
          ))}
        </div>
      </div>
      <div className="mt-4 flex flex-col gap-3 text-sm text-muted-foreground sm:flex-row sm:items-center sm:justify-between sm:gap-8">
        <span>
          {maxCount > 0
            ? `Peak day: ${maxCount} ${maxCount === 1 ? "Evolution" : "Evolutions"} · ${peakCommits} ${peakCommits === 1 ? "commit" : "commits"}`
            : "No activity in this range"}
        </span>
        <span className="flex items-center gap-1">
          {[0, 1, 2, 3, 4].map((value) => (
            <span
              key={value}
              className={`size-3 rounded-[3px] ${heatClass(value, 4)}`}
            />
          ))}
          More
        </span>
      </div>
    </div>
  );
}

function RecentActivityPanel({
  evolutions,
  repositories,
  details,
  selectedRepo,
}: {
  evolutions: EvolutionSummary[];
  repositories: RepositorySummary[];
  details: DetailResponse[];
  selectedRepo?: string;
}) {
  const [activeRepo, setActiveRepo] = useState<string | undefined>(
    selectedRepo,
  );

  useEffect(() => {
    setActiveRepo(selectedRepo);
  }, [selectedRepo]);

  const detailById = new Map(
    details.map((detail) => [detail.summary.id, detail]),
  );
  const filteredRows = activeRepo
    ? evolutions.filter(
        (evolution) =>
          repoForEvolution(
            evolution,
            detailById.get(evolution.id),
            repositories,
            selectedRepo,
          ) === activeRepo,
      )
    : evolutions;
  const visibleRows = filteredRows.slice(0, 6);

  return (
    <section className="space-y-3">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h2 className="text-lg font-semibold">Recent activity</h2>
        {/* <button className="inline-flex h-9 w-fit items-center gap-2 rounded-md bg-white px-3 text-xs font-semibold shadow-[0_0_0_1px_rgba(15,23,42,0.12)] transition-colors hover:bg-slate-50">
          All activity types <ChevronDown className="size-4" />
        </button> */}
      </div>
      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          className={filterPillClass(!activeRepo)}
          aria-pressed={!activeRepo}
          onClick={() => setActiveRepo(undefined)}
        >
          All repositories
        </button>
        {repositories.slice(0, 5).map((repo) => (
          <button
            key={repo.name}
            type="button"
            className={filterPillClass(activeRepo === repo.name)}
            aria-pressed={activeRepo === repo.name}
            onClick={() => setActiveRepo(repo.name)}
          >
            {repo.name}
          </button>
        ))}
      </div>
      <div className="overflow-hidden rounded-lg bg-white shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
        {visibleRows.length === 0 ? (
          <div className="p-6 text-sm text-muted-foreground">
            No Evolutions recorded for this repository.
          </div>
        ) : (
          visibleRows.map((evolution, index) => {
            const detail = detailById.get(evolution.id);
            const repo = repoForEvolution(
              evolution,
              detail,
              repositories,
              selectedRepo,
            );
            const repoIndex = Math.max(
              0,
              repositories.findIndex((row) => row.name === repo),
            );
            const tone = REPO_TONES[repoIndex % REPO_TONES.length];
            const Icon = tone.icon;
            return (
              <Link
                key={evolution.id}
                to="/snapshots/$id"
                params={{ id: evolution.id }}
                className={`grid grid-cols-[40px_minmax(0,1fr)_20px] items-center gap-3 px-3 py-3.5 transition-colors hover:bg-slate-50 sm:grid-cols-[44px_92px_minmax(0,1fr)_112px_24px] sm:gap-4 sm:px-4 ${index > 0 ? "border-t" : ""}`}
              >
                <span
                  className={`flex size-9 items-center justify-center rounded-full ${tone.soft} ${tone.text}`}
                >
                  <Icon className="size-5" />
                </span>
                <span className="hidden truncate text-sm font-semibold sm:block">
                  {repo}
                </span>
                <span className="min-w-0">
                  <span className="flex min-w-0 items-center gap-3">
                    <strong className="truncate text-sm font-semibold">
                      {evolution.title || "Untitled Snapshot"}
                    </strong>
                    <StatusBadge status={evolution.status} />
                  </span>
                  <span className="mt-1 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                    <span>{evolution.type}</span>
                    <span>by {primaryProvider(evolution, detail)}</span>
                  </span>
                </span>
                <span className="hidden text-right text-xs text-muted-foreground sm:block">
                  {compactDate(evolution.updatedAt || evolution.createdAt)}
                </span>
                <ArrowRight className="size-4 text-slate-500" />
              </Link>
            );
          })
        )}
      </div>
    </section>
  );
}

function filterPillClass(active: boolean) {
  return `inline-flex h-7 cursor-pointer appearance-none items-center justify-center rounded-full border-0 px-4 text-xs font-semibold shadow-[0_0_0_1px_rgba(15,23,42,0.12)] transition-colors hover:bg-slate-100 ${
    active
      ? "bg-slate-950 text-white hover:bg-slate-900"
      : "bg-white text-slate-600"
  }`;
}

type PlatformStats = {
  repositories: number;
  evolutions: number;
  snapshots: number;
  commits: number;
  artifacts: number;
  decisions: number;
  risks: number;
  failedVerifications: number;
  missingDecisions: number;
};

function PlatformOverview({ stats }: { stats: PlatformStats }) {
  const tiles = [
    ["Total snapshots", stats.evolutions],
    ["Snapshots", stats.snapshots],
    ["Commits", stats.commits],
    ["Artifacts", stats.artifacts],
    ["Decisions", stats.decisions],
    ["Risks", stats.risks],
  ] as const;
  return (
    <RailCard title="Platform overview" eyebrow="Last 12 months">
      <div className="grid grid-cols-2 gap-2.5">
        {tiles.map(([label, value]) => (
          <div
            key={label}
            className="min-w-0 rounded-lg bg-white px-3.5 py-3 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]"
          >
            <div className="text-xl font-semibold leading-6 tabular-nums">
              {value}
            </div>
            <div className="mt-1 max-w-full text-[11px] leading-3 text-muted-foreground">
              {label}
            </div>
          </div>
        ))}
      </div>
    </RailCard>
  );
}

function VelocityPanel({
  repositories,
  evolutions,
}: {
  repositories: RepositorySummary[];
  evolutions: EvolutionSummary[];
}) {
  return (
    <RailCard title="Evolution velocity" eyebrow="Last 12 months">
      <div className="space-y-4">
        {repositories.slice(0, 4).map((repo, index) => {
          const tone = REPO_TONES[index % REPO_TONES.length];
          const values = repositoryVelocityBuckets(
            repo,
            evolutions,
            repositories,
          );
          const count = values.reduce((total, value) => total + value, 0);
          return (
            <div
              key={repo.name}
              className="grid grid-cols-[72px_minmax(0,1fr)_24px] items-center gap-3"
            >
              <span className="flex items-center gap-2 text-sm font-semibold">
                <span
                  className={`flex size-4 items-center justify-center rounded-full ${tone.soft} ${tone.text}`}
                >
                  <Zap className="size-3" />
                </span>
                <span className="truncate">{repo.name}</span>
              </span>
              <MiniBars tone={tone} values={values} />
              <span className="text-right text-sm font-semibold tabular-nums">
                {count}
              </span>
            </div>
          );
        })}
      </div>
    </RailCard>
  );
}

function AgentContributionPanel({
  evolutions,
  details,
}: {
  evolutions: EvolutionSummary[];
  details: DetailResponse[];
}) {
  const rows = agentContributionRows(evolutions, details);
  const max = Math.max(1, ...rows.map((row) => row.count));
  return (
    <RailCard title="Agent contribution" eyebrow="Last 30 days">
      <div className="space-y-4">
        {rows.map((row) => (
          <div
            key={row.label}
            className="grid grid-cols-[108px_minmax(0,1fr)_74px] items-center gap-3"
          >
            <span className="flex min-w-0 items-center gap-2 text-sm font-semibold">
              <img
                src={agentAvatarPath(row.label)}
                alt=""
                className="size-6 shrink-0 rounded-lg"
              />
              <span className="min-w-0">{row.label}</span>
            </span>
            <span className="h-1.5 overflow-hidden rounded-full bg-slate-100">
              <span
                className={`block h-full rounded-full ${row.tone.bg}`}
                style={{
                  width:
                    row.count > 0
                      ? `${Math.max(7, (row.count / max) * 100)}%`
                      : "0%",
                }}
              />
            </span>
            <span className="whitespace-nowrap text-right text-sm text-muted-foreground tabular-nums">
              {row.count} ({row.percent}%)
            </span>
          </div>
        ))}
      </div>
    </RailCard>
  );
}

function agentAvatarPath(label: string) {
  if (label === "Codex") return "/agents/codex.svg";
  if (label === "Claude") return "/agents/claude.svg";
  if (label === "OpenCode") return "/agents/opencode.svg";
  return "/agents/other.svg";
}

function NeedsAttentionPanel({ stats }: { stats: PlatformStats }) {
  const rows = [
    {
      label: "Evolutions with risks",
      value: stats.risks,
      icon: AlertTriangle,
      tone: "text-red-500",
    },
    {
      label: "Evolutions with failed verifications",
      value: stats.failedVerifications,
      icon: ShieldAlert,
      tone: "text-red-500",
    },
    {
      label: "Evolutions missing decisions",
      value: stats.missingDecisions,
      icon: CircleHelp,
      tone: "text-amber-500",
    },
  ];
  return (
    <RailCard title="Needs attention" eyebrow="Across all repositories">
      <div className="divide-y">
        {rows.map((row) => {
          const Icon = row.icon;
          return (
            <div
              key={row.label}
              className="grid grid-cols-[22px_minmax(0,1fr)_20px_16px] items-center gap-3 py-3 first:pt-0 last:pb-0"
            >
              <Icon className={`size-4 ${row.tone}`} />
              <span className="truncate text-sm font-medium">{row.label}</span>
              <span className="text-right text-sm font-semibold tabular-nums">
                {row.value}
              </span>
              <ArrowRight className="size-4 text-muted-foreground" />
            </div>
          );
        })}
      </div>
    </RailCard>
  );
}

function RailCard({
  title,
  eyebrow,
  children,
}: {
  title: string;
  eyebrow: string;
  children: ReactNode;
}) {
  return (
    <section className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
      <div className="mb-5 flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold">{title}</h2>
        <span className="flex items-center gap-1 text-xs font-medium text-muted-foreground">
          <Package className="size-3" />
          {eyebrow}
        </span>
      </div>
      {children}
    </section>
  );
}

function MiniBars({ tone, values }: { tone: RepoTone; values: number[] }) {
  const max = Math.max(1, ...values);
  return (
    <span
      aria-hidden="true"
      className="flex h-8 items-end gap-[2px] overflow-hidden"
    >
      {values.map((value, index) => (
        <span
          key={index}
          className={`w-[3px] rounded-sm ${value > 0 ? tone.bg : "bg-slate-200"}`}
          style={{
            height: `${value > 0 ? Math.max(4, (value / max) * 24) : 3}px`,
            opacity: value > 0 ? 0.9 : 0.55,
          }}
        />
      ))}
    </span>
  );
}

function buildFallbackRepositories(
  evolutions: EvolutionSummary[],
  selectedRepo?: string,
): RepositorySummary[] {
  const name = selectedRepo ?? "eve";
  return [
    {
      name,
      evolutionCount: evolutions.length,
      snapshotCount: evolutions.filter((evolution) => evolution.snapshot)
        .length,
      commitCount: evolutions.reduce(
        (total, evolution) => total + (evolution.commitCount ?? 0),
        0,
      ),
      decisionCount: evolutions.reduce(
        (total, evolution) => total + (evolution.decisionCount ?? 0),
        0,
      ),
      riskCount: evolutions.reduce(
        (total, evolution) => total + (evolution.riskCount ?? 0),
        0,
      ),
      artifactCount: evolutions.reduce(
        (total, evolution) => total + (evolution.artifactCount ?? 0),
        0,
      ),
      latestAt: evolutions[0]?.updatedAt || evolutions[0]?.createdAt || "",
      latestEvolution: evolutions[0]?.id || "",
      latestTitle: evolutions[0]?.title || "",
      sessionProviders: Array.from(
        new Set(evolutions.flatMap((evolution) => evolution.sessionProviders)),
      ),
    },
  ];
}

function sortRepositoriesByLatestUse(repositories: RepositorySummary[]) {
  return [...repositories].sort((left, right) => {
    const latestDelta = timestamp(right.latestAt) - timestamp(left.latestAt);
    if (latestDelta !== 0) return latestDelta;
    if (left.evolutionCount !== right.evolutionCount) {
      return right.evolutionCount - left.evolutionCount;
    }
    return left.name.localeCompare(right.name);
  });
}

function buildPlatformStats(
  evolutions: EvolutionSummary[],
  details: DetailResponse[],
  repositories: RepositorySummary[],
): PlatformStats {
  const hasDetails = details.length > 0;
  const decisions = hasDetails
    ? details.reduce(
        (total, detail) => total + detail.snapshot.decisions.length,
        0,
      )
    : evolutions.reduce(
        (total, evolution) => total + (evolution.decisionCount ?? 0),
        0,
      );
  const risks = hasDetails
    ? details.reduce((total, detail) => total + detail.snapshot.risks.length, 0)
    : evolutions.reduce(
        (total, evolution) => total + (evolution.riskCount ?? 0),
        0,
      );
  const artifacts = hasDetails
    ? details.reduce(
        (total, detail) => total + detail.snapshot.artifacts.length,
        0,
      )
    : evolutions.reduce(
        (total, evolution) => total + (evolution.artifactCount ?? 0),
        0,
      );
  const failedVerifications = hasDetails
    ? details.reduce(
        (total, detail) =>
          total +
          detail.snapshot.validation.filter(
            (verification) => verification.status.toLowerCase() === "failed",
          ).length,
        0,
      )
    : evolutions.reduce(
        (total, evolution) => total + (evolution.failedValidationCount ?? 0),
        0,
      );
  return {
    repositories: repositories.length,
    evolutions: evolutions.length,
    snapshots: evolutions.filter((evolution) => evolution.snapshot).length,
    commits: evolutions.reduce(
      (total, evolution) => total + (evolution.commitCount ?? 0),
      0,
    ),
    artifacts,
    decisions,
    risks,
    failedVerifications,
    missingDecisions: hasDetails
      ? details.filter((detail) => detail.snapshot.decisions.length === 0)
          .length
      : evolutions.filter((evolution) => (evolution.decisionCount ?? 0) === 0)
          .length,
  };
}

function repositoryVelocityBuckets(
  repo: RepositorySummary,
  evolutions: EvolutionSummary[],
  repositories: RepositorySummary[],
) {
  const values = Array.from({ length: VELOCITY_BUCKETS }, () => 0);
  const now = new Date();
  const windowStart = new Date(now);
  windowStart.setDate(now.getDate() - VELOCITY_WINDOW_DAYS);
  const bucketMs =
    (now.getTime() - windowStart.getTime()) / VELOCITY_BUCKETS || 1;

  for (const evolution of evolutions) {
    if (!evolutionBelongsToRepository(evolution, repo, repositories)) continue;
    const date = parseDate(evolution.updatedAt || evolution.createdAt);
    if (!date) continue;
    const time = date.getTime();
    if (time < windowStart.getTime() || time > now.getTime()) continue;
    const index = clamp(
      Math.floor((time - windowStart.getTime()) / bucketMs),
      0,
      VELOCITY_BUCKETS - 1,
    );
    values[index] += 1;
  }

  return values;
}

function evolutionBelongsToRepository(
  evolution: EvolutionSummary,
  repo: RepositorySummary,
  repositories: RepositorySummary[],
) {
  if (evolution.repository) {
    return (
      evolution.repository === repo.name || evolution.repository === repo.id
    );
  }
  if (repositories.length <= 1) return true;
  return repo.latestEvolution === evolution.id;
}

function compressSeries(values: number[], length: number) {
  if (values.length <= length) return values;
  return Array.from({ length }, (_, index) => {
    const start = Math.floor((index / length) * values.length);
    const end = Math.max(
      start + 1,
      Math.floor(((index + 1) / length) * values.length),
    );
    return values.slice(start, end).reduce((total, value) => total + value, 0);
  });
}

function sparklineY(value: number, values: number[]) {
  const max = Math.max(1, ...values);
  return value > 0 ? Math.max(8, (value / max) * 32) : 8;
}

function repoForEvolution(
  evolution: EvolutionSummary,
  detail: DetailResponse | undefined,
  repositories: RepositorySummary[],
  selectedRepo?: string,
) {
  const detailRepo = Object.keys(
    detail?.evolution.implementation.repositories ?? {},
  )[0];
  if (detailRepo) return detailRepo;
  const summaryRepo = repositories.find(
    (repo) => repo.latestEvolution === evolution.id,
  );
  return summaryRepo?.name ?? selectedRepo ?? repositories[0]?.name ?? "eve";
}

function primaryProvider(evolution: EvolutionSummary, detail?: DetailResponse) {
  const provider =
    detail?.sessions[0]?.providerName ||
    detail?.sessions[0]?.provider ||
    evolution.sessionProviders[0] ||
    "codex";
  return providerLabel(provider).toLowerCase();
}

function agentContributionRows(
  evolutions: EvolutionSummary[],
  details: DetailResponse[],
) {
  const counts = new Map<string, number>();
  if (details.length > 0) {
    for (const detail of details) {
      const providers =
        detail.sessions.length > 0
          ? detail.sessions.map(
              (session) => session.providerName || session.provider,
            )
          : detail.summary.sessionProviders;
      for (const provider of providers) incrementProvider(counts, provider);
    }
  } else {
    for (const evolution of evolutions) {
      for (const provider of evolution.sessionProviders)
        incrementProvider(counts, provider);
    }
  }

  const total = Math.max(
    1,
    Array.from(counts.values()).reduce((sum, value) => sum + value, 0),
  );
  return ["Codex", "Claude", "OpenCode", "Other"].map((label, index) => {
    const count = counts.get(label) ?? 0;
    return {
      label,
      count,
      percent: Math.round((count / total) * 100),
      tone: REPO_TONES[index % REPO_TONES.length],
    };
  });
}

function incrementProvider(counts: Map<string, number>, provider?: string) {
  const label = providerLabel(provider);
  counts.set(label, (counts.get(label) ?? 0) + 1);
}

function providerLabel(provider?: string) {
  const value = (provider ?? "").toLowerCase();
  if (value.includes("codex")) return "Codex";
  if (value.includes("claude")) return "Claude";
  if (value.includes("opencode")) return "OpenCode";
  return "Other";
}

type ActivityDay = {
  evolutionCount: number;
  commitCount: number;
};

type ActivityCell = ActivityDay & {
  date: Date;
};

function buildMonthLabels(weeks: ActivityCell[][]) {
  const labels: Array<{ name: string; week: number; span: number }> = [];

  weeks.forEach((week, weekIndex) => {
    const firstOfMonth = week.find((day) => day.date.getDate() === 1);
    if (!firstOfMonth) return;
    labels.push({
      name: MONTHS[firstOfMonth.date.getMonth()],
      week: weekIndex,
      span: 1,
    });
  });

  return labels.map((label, index) => ({
    ...label,
    span: Math.max(1, (labels[index + 1]?.week ?? weeks.length) - label.week),
  }));
}

function activityLabel(day: ActivityCell) {
  const evolutions = `${day.evolutionCount} ${day.evolutionCount === 1 ? "Evolution" : "Evolutions"}`;
  const commits = `${day.commitCount} ${day.commitCount === 1 ? "commit" : "commits"}`;
  return `${compactDate(day.date.toISOString())}: ${evolutions}, ${commits}`;
}

function heatClass(count: number, maxCount: number) {
  if (count <= 0) return "bg-slate-100";
  const level = maxCount <= 1 ? 1 : Math.ceil((count / maxCount) * 4);
  if (level <= 1) return "bg-emerald-200";
  if (level === 2) return "bg-emerald-400";
  if (level === 3) return "bg-emerald-600";
  return "bg-emerald-800";
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value));
}

function parseDate(value?: string) {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : date;
}

function timestamp(value?: string) {
  return parseDate(value)?.getTime() ?? 0;
}

function isoDay(date: Date) {
  return date.toISOString().slice(0, 10);
}

const MONTHS = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
];
const ACTIVITY_CELL_SIZE = 12;
const ACTIVITY_CELL_GAP = 3;
const DAY_LABEL_WIDTH = 30;
const MIN_ACTIVITY_WEEKS = 12;
const MAX_ACTIVITY_WEEKS = 53;
const VELOCITY_BUCKETS = 32;
const VELOCITY_WINDOW_DAYS = 365;
