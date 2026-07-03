import { Link } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import {
  AlertTriangle,
  ArrowRight,
  Bot,
  BookOpen,
  ChevronDown,
  CircleHelp,
  Code2,
  Globe,
  Package,
  ShieldAlert,
  Zap,
  type LucideIcon
} from 'lucide-react';
import type { ReactNode } from 'react';
import { useEffect, useRef, useState } from 'react';
import { compactDate, shortCommit } from '../format';
import { api } from '../api';
import type { DetailResponse, EvolutionSummary, RepositorySummary } from '../types';
import { StatusBadge } from './status-badge';

export function RepositoryActivityView({
  repositories,
  evolutions,
  selectedRepo
}: {
  repositories: RepositorySummary[];
  evolutions: EvolutionSummary[];
  selectedRepo?: string;
}) {
  const title = selectedRepo ?? 'Product evolution';
  const details = useQuery({
    queryKey: ['activity-overview-details', evolutions.map((evolution) => evolution.id).join(',')],
    queryFn: () => Promise.all(evolutions.map((evolution) => api.evolution(evolution.id))),
    enabled: evolutions.length > 0,
    staleTime: 30_000
  });
  const rows = [...evolutions].sort((left, right) => timestamp(right.updatedAt || right.createdAt) - timestamp(left.updatedAt || left.createdAt));
  const repoRows = repositories.length > 0 ? repositories : buildFallbackRepositories(evolutions, selectedRepo);
  const detailRows = details.data ?? [];
  const stats = buildPlatformStats(evolutions, detailRows);

  return (
    <main className="min-h-[calc(100dvh-76px)] min-w-0 bg-slate-50/35 px-8 py-8">
      <div className="grid grid-cols-1 gap-7 xl:grid-cols-[minmax(0,1fr)_350px]">
        <section className="min-w-0 space-y-7">
          <header className="grid grid-cols-1 items-end gap-7 xl:grid-cols-[minmax(400px,0.8fr)_minmax(0,1.2fr)]">
            <div>
              <p className="font-mono text-xs font-semibold uppercase tracking-wide text-slate-600">Overview</p>
              <h1 className="mt-4 text-[34px] font-semibold leading-tight tracking-[-0.01em] text-balance">{title}</h1>
              <p className="mt-3 max-w-[54ch] text-sm leading-6 text-muted-foreground text-pretty">
                {selectedRepo
                  ? 'Track product states, snapshots, sessions, and verification recorded for this repository.'
                  : 'Track and understand how your products are evolving across repositories.'}
              </p>
            </div>
            <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
              {repoRows.slice(0, 4).map((repo, index) => (
                <RepositoryOverviewCard key={repo.name} repo={repo} tone={REPO_TONES[index % REPO_TONES.length]} selected={selectedRepo === repo.name} />
              ))}
            </div>
          </header>

          <section className="space-y-3">
            <div className="flex items-center justify-between gap-4">
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-semibold">Evolution activity</h2>
                <CircleHelp className="size-4 text-muted-foreground" />
              </div>
              <button className="inline-flex h-9 items-center gap-2 rounded-md bg-white px-3 text-xs font-semibold shadow-[0_0_0_1px_rgba(15,23,42,0.12)] transition-colors hover:bg-slate-50">
                Last 12 months <ChevronDown className="size-4" />
              </button>
            </div>
            <ContributionGraph evolutions={evolutions} />
          </section>

          <RecentActivityPanel evolutions={rows} repositories={repoRows} details={detailRows} selectedRepo={selectedRepo} />
        </section>

        <aside className="space-y-5">
          <PlatformOverview stats={stats} />
          <VelocityPanel repositories={repoRows} evolutions={evolutions} />
          <AgentContributionPanel evolutions={evolutions} details={detailRows} />
          <NeedsAttentionPanel stats={stats} />
        </aside>
      </div>
    </main>
  );
}

function RepositoryOverviewCard({ repo, tone, selected }: { repo: RepositorySummary; tone: RepoTone; selected: boolean }) {
  const Icon = tone.icon;
  return (
    <Link
      to="/repositories/$repo"
      params={{ repo: repo.name }}
      className={`min-w-0 rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.1)] transition-[background-color,box-shadow,scale] duration-150 hover:bg-slate-50 active:scale-[0.96] ${selected ? 'ring-2 ring-blue-500/20' : ''}`}
    >
      <div className="flex items-center justify-between gap-3">
        <span className="flex min-w-0 items-center gap-2">
          <span className={`size-2.5 rounded-full ${tone.bg}`} />
          <span className="truncate font-semibold">{repo.name}</span>
        </span>
        <Icon className={`size-4 shrink-0 ${tone.text}`} />
      </div>
      <div className="mt-4">
        <RepoSparkline tone={tone} seed={repo.name} />
      </div>
      <p className="mt-3 truncate text-xs text-muted-foreground">
        {repo.evolutionCount} EVs · {repo.snapshotCount} snapshots · {repo.commitCount} {repo.commitCount === 1 ? 'commit' : 'commits'}
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
  { bg: 'bg-blue-600', text: 'text-blue-600', soft: 'bg-blue-50', line: '#2563eb', icon: BookOpen },
  { bg: 'bg-emerald-500', text: 'text-emerald-600', soft: 'bg-emerald-50', line: '#10b981', icon: BookOpen },
  { bg: 'bg-violet-600', text: 'text-violet-600', soft: 'bg-violet-50', line: '#7c3aed', icon: Globe },
  { bg: 'bg-amber-500', text: 'text-amber-600', soft: 'bg-amber-50', line: '#f59e0b', icon: Code2 }
];

function RepoSparkline({ tone, seed }: { tone: RepoTone; seed: string }) {
  const values = seededSeries(seed, 11, 10, 38);
  const points = values.map((value, index) => `${index * 10},${46 - value}`).join(' ');
  return (
    <svg aria-hidden="true" viewBox="0 0 104 48" className="h-8 w-full overflow-visible">
      <polyline points={points} fill="none" stroke={tone.line} strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
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
      commitCount: current.commitCount + (evolution.commitCount ?? 0)
    });
  }

  useEffect(() => {
    const element = graphMeasureRef.current;
    if (!element) return;

    const updateVisibleWeeks = () => {
      const width = element.getBoundingClientRect().width;
      const usableWidth = Math.max(0, width - DAY_LABEL_WIDTH);
      const nextWeeks = clamp(
        Math.floor((usableWidth + ACTIVITY_CELL_GAP) / (ACTIVITY_CELL_SIZE + ACTIVITY_CELL_GAP)),
        MIN_ACTIVITY_WEEKS,
        MAX_ACTIVITY_WEEKS
      );
      setVisibleWeeks((current) => (current === nextWeeks ? current : nextWeeks));
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
      const activity = counts.get(isoDay(date)) ?? { evolutionCount: 0, commitCount: 0 };
      return { date, ...activity };
    })
  );
  const monthLabels = buildMonthLabels(weeks);
  const maxCount = Math.max(0, ...weeks.flat().map((day) => day.evolutionCount));
  const peakCommits = Math.max(0, ...weeks.flat().map((day) => day.commitCount));

  return (
    <div className="w-full rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
      <div ref={graphMeasureRef} className="w-full">
        <div
          className="mb-2 grid gap-x-[3px] text-xs text-muted-foreground"
          style={{
            gridTemplateColumns: `${DAY_LABEL_WIDTH}px repeat(${visibleWeeks}, ${ACTIVITY_CELL_SIZE}px)`,
            columnGap: ACTIVITY_CELL_GAP
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
            columnGap: ACTIVITY_CELL_GAP
          }}
        >
          <div className="grid grid-rows-7 text-xs leading-3 text-muted-foreground" style={{ rowGap: ACTIVITY_CELL_GAP }}>
            {['', 'Mon', '', 'Wed', '', 'Fri', ''].map((label, index) => (
              <span key={`${label}-${index}`} className="h-3">
                {label}
              </span>
            ))}
          </div>
          {weeks.map((week) => (
            <div key={isoDay(week[0].date)} className="grid grid-rows-7" style={{ rowGap: ACTIVITY_CELL_GAP }}>
              {week.map((day) => {
                const label = activityLabel(day);
                return (
                  <span
                    key={isoDay(day.date)}
                    aria-label={label}
                    title={label}
                    className={`group/day relative block size-3 rounded-[2px] ${heatClass(day.evolutionCount, maxCount)} ${day.date > today ? 'opacity-40' : ''}`}
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
      <div className="mt-4 flex items-center justify-between gap-8 text-sm text-muted-foreground">
        <span>
          {maxCount > 0
            ? `Peak day: ${maxCount} ${maxCount === 1 ? 'Evolution' : 'Evolutions'} · ${peakCommits} ${peakCommits === 1 ? 'commit' : 'commits'}`
            : 'No activity in this range'}
        </span>
        <span className="flex items-center gap-1">
          {[0, 1, 2, 3, 4].map((value) => (
            <span key={value} className={`size-3 rounded-[3px] ${heatClass(value, 4)}`} />
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
  selectedRepo
}: {
  evolutions: EvolutionSummary[];
  repositories: RepositorySummary[];
  details: DetailResponse[];
  selectedRepo?: string;
}) {
  const visibleRows = evolutions.slice(0, 4);
  const detailById = new Map(details.map((detail) => [detail.summary.id, detail]));

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-lg font-semibold">Recent activity</h2>
        <button className="inline-flex h-9 items-center gap-2 rounded-md bg-white px-3 text-xs font-semibold shadow-[0_0_0_1px_rgba(15,23,42,0.12)] transition-colors hover:bg-slate-50">
          All activity types <ChevronDown className="size-4" />
        </button>
      </div>
      <div className="flex flex-wrap gap-2">
        <FilterPill active={!selectedRepo}>All repositories</FilterPill>
        {repositories.slice(0, 5).map((repo) => (
          <FilterPill key={repo.name} active={selectedRepo === repo.name}>
            {repo.name}
          </FilterPill>
        ))}
      </div>
      <div className="overflow-hidden rounded-lg bg-white shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
        {visibleRows.length === 0 ? (
          <div className="p-6 text-sm text-muted-foreground">No Evolutions recorded for this repository.</div>
        ) : (
          visibleRows.map((evolution, index) => {
            const detail = detailById.get(evolution.id);
            const repo = repoForEvolution(evolution, detail, repositories, selectedRepo);
            const repoIndex = Math.max(0, repositories.findIndex((row) => row.name === repo));
            const tone = REPO_TONES[repoIndex % REPO_TONES.length];
            const Icon = tone.icon;
            return (
              <Link
                key={evolution.id}
                to="/evolutions/$id"
                params={{ id: evolution.id }}
                className={`grid grid-cols-[44px_92px_minmax(0,1fr)_112px_24px] items-center gap-4 px-4 py-3.5 transition-colors hover:bg-slate-50 ${index > 0 ? 'border-t' : ''}`}
              >
                <span className={`flex size-9 items-center justify-center rounded-full ${tone.soft} ${tone.text}`}>
                  <Icon className="size-5" />
                </span>
                <span className="truncate text-sm font-semibold">{repo}</span>
                <span className="min-w-0">
                  <span className="flex min-w-0 items-center gap-3">
                    <strong className="truncate text-sm font-semibold">{evolution.title || 'Untitled Evolution'}</strong>
                    <StatusBadge status={evolution.status} />
                  </span>
                  <span className="mt-1 flex min-w-0 flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
                    <span className="font-mono">{evolution.id}</span>
                    {evolution.snapshot ? <span className="font-mono">Snapshot {shortCommit(evolution.snapshot)}</span> : null}
                    <span>by {primaryProvider(evolution, detail)}</span>
                  </span>
                </span>
                <span className="text-right text-xs text-muted-foreground">{compactDate(evolution.updatedAt || evolution.createdAt)}</span>
                <ArrowRight className="size-4 text-slate-500" />
              </Link>
            );
          })
        )}
        {evolutions.length > 4 ? (
          <Link
            to="/"
            className="mx-3 mb-3 mt-1 flex h-11 items-center justify-center gap-2 rounded-md text-sm font-semibold shadow-[0_0_0_1px_rgba(15,23,42,0.08)] transition-colors hover:bg-slate-50"
          >
            View all activity <ArrowRight className="size-4" />
          </Link>
        ) : null}
      </div>
    </section>
  );
}

function FilterPill({ active, children }: { active: boolean; children: ReactNode }) {
  return (
    <span className={`inline-flex h-7 items-center rounded-full px-4 text-xs font-semibold shadow-[0_0_0_1px_rgba(15,23,42,0.12)] ${active ? 'bg-slate-950 text-white' : 'bg-white text-slate-600'}`}>
      {children}
    </span>
  );
}

type PlatformStats = {
  evolutions: number;
  snapshots: number;
  commits: number;
  sessions: number;
  decisions: number;
  risks: number;
  failedVerifications: number;
  missingDecisions: number;
};

function PlatformOverview({ stats }: { stats: PlatformStats }) {
  const tiles = [
    ['Total evolutions', stats.evolutions],
    ['Snapshots', stats.snapshots],
    ['Commits', stats.commits],
    ['Implementation sessions', stats.sessions],
    ['Decisions', stats.decisions],
    ['Risks', stats.risks]
  ] as const;
  return (
    <RailCard title="Platform overview" eyebrow="Last 12 months">
      <div className="grid grid-cols-3 gap-3">
        {tiles.map(([label, value]) => (
          <div key={label} className="rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
            <div className="text-2xl font-semibold tabular-nums">{value}</div>
            <div className="mt-1 text-xs leading-4 text-muted-foreground">{label}</div>
          </div>
        ))}
      </div>
    </RailCard>
  );
}

function VelocityPanel({ repositories, evolutions }: { repositories: RepositorySummary[]; evolutions: EvolutionSummary[] }) {
  return (
    <RailCard title="Evolution velocity" eyebrow="Last 12 months">
      <div className="space-y-4">
        {repositories.slice(0, 4).map((repo, index) => {
          const tone = REPO_TONES[index % REPO_TONES.length];
          return (
            <div key={repo.name} className="grid grid-cols-[72px_minmax(0,1fr)_24px] items-center gap-3">
              <span className="flex items-center gap-2 text-sm font-semibold">
                <span className={`flex size-4 items-center justify-center rounded-full ${tone.soft} ${tone.text}`}>
                  <Zap className="size-3" />
                </span>
                <span className="truncate">{repo.name}</span>
              </span>
              <MiniBars tone={tone} seed={`${repo.name}-${evolutions.length}`} />
              <span className="text-right text-sm font-semibold tabular-nums">{repo.evolutionCount}</span>
            </div>
          );
        })}
      </div>
    </RailCard>
  );
}

function AgentContributionPanel({ evolutions, details }: { evolutions: EvolutionSummary[]; details: DetailResponse[] }) {
  const rows = agentContributionRows(evolutions, details);
  const max = Math.max(1, ...rows.map((row) => row.count));
  return (
    <RailCard title="Agent contribution" eyebrow="Last 30 days">
      <div className="space-y-4">
        {rows.map((row) => (
          <div key={row.label} className="grid grid-cols-[82px_minmax(0,1fr)_62px] items-center gap-3">
            <span className="flex min-w-0 items-center gap-2 text-sm font-semibold">
              <span className={`flex size-5 items-center justify-center rounded-full ${row.tone.soft} ${row.tone.text}`}>
                <Bot className="size-3.5" />
              </span>
              <span className="truncate">{row.label}</span>
            </span>
            <span className="h-1.5 overflow-hidden rounded-full bg-slate-100">
              <span className={`block h-full rounded-full ${row.tone.bg}`} style={{ width: `${Math.max(7, (row.count / max) * 100)}%` }} />
            </span>
            <span className="text-right text-sm text-muted-foreground tabular-nums">
              {row.count} ({row.percent}%)
            </span>
          </div>
        ))}
      </div>
    </RailCard>
  );
}

function NeedsAttentionPanel({ stats }: { stats: PlatformStats }) {
  const rows = [
    { label: 'Evolutions with risks', value: stats.risks, icon: AlertTriangle, tone: 'text-red-500' },
    { label: 'Evolutions with failed verifications', value: stats.failedVerifications, icon: ShieldAlert, tone: 'text-red-500' },
    { label: 'Evolutions missing decisions', value: stats.missingDecisions, icon: CircleHelp, tone: 'text-amber-500' }
  ];
  return (
    <RailCard title="Needs attention" eyebrow="Across all repositories">
      <div className="divide-y">
        {rows.map((row) => {
          const Icon = row.icon;
          return (
            <div key={row.label} className="grid grid-cols-[22px_minmax(0,1fr)_20px_16px] items-center gap-3 py-3 first:pt-0 last:pb-0">
              <Icon className={`size-4 ${row.tone}`} />
              <span className="truncate text-sm font-medium">{row.label}</span>
              <span className="text-right text-sm font-semibold tabular-nums">{row.value}</span>
              <ArrowRight className="size-4 text-muted-foreground" />
            </div>
          );
        })}
      </div>
    </RailCard>
  );
}

function RailCard({ title, eyebrow, children }: { title: string; eyebrow: string; children: ReactNode }) {
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

function MiniBars({ tone, seed }: { tone: RepoTone; seed: string }) {
  const values = seededSeries(seed, 32, 1, 18);
  const max = Math.max(1, ...values);
  return (
    <span aria-hidden="true" className="flex h-8 items-end gap-[2px] overflow-hidden">
      {values.map((value, index) => (
        <span
          key={`${seed}-${index}`}
          className={`w-[3px] rounded-sm ${index > values.length - 8 ? tone.bg : 'bg-slate-200'}`}
          style={{ height: `${Math.max(3, (value / max) * 24)}px`, opacity: index > values.length - 8 ? 0.9 : 0.55 }}
        />
      ))}
    </span>
  );
}

function buildFallbackRepositories(evolutions: EvolutionSummary[], selectedRepo?: string): RepositorySummary[] {
  const name = selectedRepo ?? 'eve';
  return [
    {
      name,
      evolutionCount: evolutions.length,
      snapshotCount: evolutions.filter((evolution) => evolution.snapshot).length,
      commitCount: evolutions.reduce((total, evolution) => total + (evolution.commitCount ?? 0), 0),
      latestAt: evolutions[0]?.updatedAt || evolutions[0]?.createdAt || '',
      latestEvolution: evolutions[0]?.id || '',
      latestTitle: evolutions[0]?.title || '',
      sessionProviders: Array.from(new Set(evolutions.flatMap((evolution) => evolution.sessionProviders)))
    }
  ];
}

function buildPlatformStats(evolutions: EvolutionSummary[], details: DetailResponse[]): PlatformStats {
  const decisions = details.reduce((total, detail) => total + detail.evolution.decisions.length, 0);
  const risks = details.reduce((total, detail) => total + detail.evolution.risks.length, 0);
  const failedVerifications = details.reduce(
    (total, detail) => total + detail.evolution.verification.filter((verification) => verification.status.toLowerCase() === 'failed').length,
    0
  );
  return {
    evolutions: evolutions.length,
    snapshots: evolutions.filter((evolution) => evolution.snapshot).length,
    commits: evolutions.reduce((total, evolution) => total + (evolution.commitCount ?? 0), 0),
    sessions:
      details.length > 0
        ? details.reduce((total, detail) => total + detail.sessions.length, 0)
        : evolutions.reduce((total, evolution) => total + evolution.sessionProviders.length, 0),
    decisions,
    risks,
    failedVerifications,
    missingDecisions: details.length > 0 ? details.filter((detail) => detail.evolution.decisions.length === 0).length : 0
  };
}

function repoForEvolution(
  evolution: EvolutionSummary,
  detail: DetailResponse | undefined,
  repositories: RepositorySummary[],
  selectedRepo?: string
) {
  const detailRepo = Object.keys(detail?.evolution.implementation.repositories ?? {})[0];
  if (detailRepo) return detailRepo;
  const summaryRepo = repositories.find((repo) => repo.latestEvolution === evolution.id);
  return summaryRepo?.name ?? selectedRepo ?? repositories[0]?.name ?? 'eve';
}

function primaryProvider(evolution: EvolutionSummary, detail?: DetailResponse) {
  const provider = detail?.sessions[0]?.providerName || detail?.sessions[0]?.provider || evolution.sessionProviders[0] || 'codex';
  return providerLabel(provider).toLowerCase();
}

function agentContributionRows(evolutions: EvolutionSummary[], details: DetailResponse[]) {
  const counts = new Map<string, number>();
  if (details.length > 0) {
    for (const detail of details) {
      const providers = detail.sessions.length > 0 ? detail.sessions.map((session) => session.providerName || session.provider) : detail.summary.sessionProviders;
      for (const provider of providers) incrementProvider(counts, provider);
    }
  } else {
    for (const evolution of evolutions) {
      for (const provider of evolution.sessionProviders) incrementProvider(counts, provider);
    }
  }

  const total = Math.max(1, Array.from(counts.values()).reduce((sum, value) => sum + value, 0));
  return ['Codex', 'Claude', 'OpenCode', 'Other'].map((label, index) => {
    const count = counts.get(label) ?? 0;
    return {
      label,
      count,
      percent: Math.round((count / total) * 100),
      tone: REPO_TONES[index % REPO_TONES.length]
    };
  });
}

function incrementProvider(counts: Map<string, number>, provider?: string) {
  const label = providerLabel(provider);
  counts.set(label, (counts.get(label) ?? 0) + 1);
}

function providerLabel(provider?: string) {
  const value = (provider ?? '').toLowerCase();
  if (value.includes('codex')) return 'Codex';
  if (value.includes('claude')) return 'Claude';
  if (value.includes('opencode')) return 'OpenCode';
  return 'Other';
}

function seededSeries(seed: string, length: number, min: number, max: number) {
  let state = seed.split('').reduce((total, char) => total + char.charCodeAt(0), 17);
  return Array.from({ length }, (_, index) => {
    state = (state * 9301 + 49297 + index * 233) % 233280;
    const wave = Math.sin((index + state / 1000) * 0.85) * 0.5 + 0.5;
    return Math.round(min + (max - min) * (0.35 * (state / 233280) + 0.65 * wave));
  });
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
      span: 1
    });
  });

  return labels.map((label, index) => ({
    ...label,
    span: Math.max(1, (labels[index + 1]?.week ?? weeks.length) - label.week)
  }));
}

function activityLabel(day: ActivityCell) {
  const evolutions = `${day.evolutionCount} ${day.evolutionCount === 1 ? 'Evolution' : 'Evolutions'}`;
  const commits = `${day.commitCount} ${day.commitCount === 1 ? 'commit' : 'commits'}`;
  return `${compactDate(day.date.toISOString())}: ${evolutions}, ${commits}`;
}

function heatClass(count: number, maxCount: number) {
  if (count <= 0) return 'bg-slate-100';
  const level = maxCount <= 1 ? 1 : Math.ceil((count / maxCount) * 4);
  if (level <= 1) return 'bg-emerald-200';
  if (level === 2) return 'bg-emerald-400';
  if (level === 3) return 'bg-emerald-600';
  return 'bg-emerald-800';
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

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const ACTIVITY_CELL_SIZE = 12;
const ACTIVITY_CELL_GAP = 3;
const DAY_LABEL_WIDTH = 30;
const MIN_ACTIVITY_WEEKS = 18;
const MAX_ACTIVITY_WEEKS = 53;
