import { Link } from '@tanstack/react-router';
import { ArrowRight, GitCommitHorizontal } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { compactDate, monthYear, shortCommit } from '../format';
import type { EvolutionSummary, RepositorySummary } from '../types';
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
  const title = selectedRepo ?? 'Repositories';
  const total = evolutions.length;

  return (
    <main className="min-h-[calc(100dvh-76px)] min-w-0 px-9 py-10">
      <div className="grid grid-cols-[minmax(0,1fr)_280px] gap-10">
        <section className="min-w-0 space-y-10">
          <header>
            <p className="font-mono text-sm font-semibold text-blue-700">{selectedRepo ? 'Repository' : 'EVE activity'}</p>
            <h1 className="mt-2 text-[32px] font-semibold leading-tight text-balance">{title}</h1>
            <p className="mt-3 max-w-[72ch] text-muted-foreground text-pretty">
              {selectedRepo
                ? 'Product states, snapshots, sessions, and verification recorded for this repository.'
                : 'Latest repositories with committed EVE product states.'}
            </p>
          </header>

          <section className="space-y-4">
            <div className="flex items-center justify-between gap-4">
              <h2 className="text-xl font-semibold">Recent activity</h2>
              <span className="text-sm text-muted-foreground">
                {total} {total === 1 ? 'Evolution' : 'Evolutions'}
              </span>
            </div>
            <ContributionGraph evolutions={evolutions} />
          </section>

          <section className="space-y-6">
            <h2 className="text-xl font-semibold">Contribution activity</h2>
            <ActivityList evolutions={evolutions} />
          </section>
        </section>

        <aside className="space-y-4">
          <h2 className="text-sm font-semibold text-muted-foreground">Latest repositories</h2>
          {repositories.map((repo) => (
            <Link
              key={repo.name}
              to="/repositories/$repo"
              params={{ repo: repo.name }}
              className={`block rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.08)] transition-[box-shadow,background-color,scale] duration-150 hover:bg-slate-50 active:scale-[0.96] ${
                selectedRepo === repo.name ? 'ring-2 ring-blue-500/20' : ''
              }`}
            >
              <div className="flex items-center justify-between gap-3">
                <span className="font-semibold">{repo.name}</span>
                <ArrowRight className="size-4 text-muted-foreground" />
              </div>
              <p className="mt-2 text-sm text-muted-foreground">
                {repo.evolutionCount} EVs · {repo.snapshotCount} snapshots · {repo.commitCount} commits
              </p>
              {repo.latestTitle ? <p className="mt-3 truncate text-sm">{repo.latestTitle}</p> : null}
            </Link>
          ))}
        </aside>
      </div>
    </main>
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

function ActivityList({ evolutions }: { evolutions: EvolutionSummary[] }) {
  const grouped = new Map<string, EvolutionSummary[]>();
  for (const evolution of evolutions) {
    const group = monthYear(evolution.updatedAt || evolution.createdAt);
    grouped.set(group, [...(grouped.get(group) ?? []), evolution]);
  }
  if (evolutions.length === 0) {
    return <div className="rounded-lg bg-white p-6 text-muted-foreground shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">No Evolutions recorded for this repository.</div>;
  }
  return (
    <div className="space-y-8">
      {Array.from(grouped.entries()).map(([month, rows]) => (
        <section key={month}>
          <div className="grid grid-cols-[130px_minmax(0,1fr)] items-center gap-4">
            <h3 className="font-semibold">{month}</h3>
            <div className="h-px bg-border" />
          </div>
          <div className="mt-5 space-y-3">
            {rows.map((evolution) => (
              <Link
                key={evolution.id}
                to="/evolutions/$id"
                params={{ id: evolution.id }}
                className="grid grid-cols-[36px_minmax(0,1fr)_120px] items-start gap-4 rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.08)] transition-[box-shadow,background-color,scale] duration-150 hover:bg-slate-50 active:scale-[0.96]"
              >
                <span className="flex size-9 items-center justify-center rounded-full bg-slate-50 text-slate-600">
                  <GitCommitHorizontal className="size-4" />
                </span>
                <span className="min-w-0">
                  <span className="flex items-center gap-3">
                    <strong className="truncate">{evolution.title || 'Untitled Evolution'}</strong>
                    <StatusBadge status={evolution.status} />
                  </span>
                  <span className="mt-1 block truncate text-sm text-muted-foreground">{evolution.outcome || 'No outcome recorded.'}</span>
                  <span className="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
                    <span className="font-mono">{evolution.id}</span>
                    {evolution.snapshot ? <span className="font-mono">{shortCommit(evolution.snapshot)}</span> : null}
                    {evolution.sessionProviders.length > 0 ? <span>{evolution.sessionProviders.join(' & ')}</span> : null}
                  </span>
                </span>
                <span className="text-right text-sm text-muted-foreground">{compactDate(evolution.updatedAt || evolution.createdAt)}</span>
              </Link>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
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

function isoDay(date: Date) {
  return date.toISOString().slice(0, 10);
}

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const ACTIVITY_CELL_SIZE = 12;
const ACTIVITY_CELL_GAP = 3;
const DAY_LABEL_WIDTH = 30;
const MIN_ACTIVITY_WEEKS = 18;
const MAX_ACTIVITY_WEEKS = 53;
