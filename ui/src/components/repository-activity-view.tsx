import { Link } from '@tanstack/react-router';
import { ArrowRight, GitCommitHorizontal } from 'lucide-react';
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
              <h2 className="text-xl font-semibold">{total} {total === 1 ? 'Evolution' : 'Evolutions'} in the last year</h2>
              <span className="text-sm text-muted-foreground">{new Date().getFullYear()}</span>
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
  const counts = new Map<string, number>();
  for (const evolution of evolutions) {
    const date = parseDate(evolution.updatedAt || evolution.createdAt);
    if (!date) continue;
    const key = isoDay(date);
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  const today = new Date();
  const start = new Date(today);
  start.setDate(today.getDate() - 52 * 7);
  const weeks = Array.from({ length: 53 }, (_, week) =>
    Array.from({ length: 7 }, (_, day) => {
      const date = new Date(start);
      date.setDate(start.getDate() + week * 7 + day);
      const count = counts.get(isoDay(date)) ?? 0;
      return { date, count };
    })
  );

  return (
    <div className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
      <div className="grid grid-flow-col grid-rows-7 gap-1 overflow-hidden">
        {weeks.flat().map((day) => (
          <span
            key={isoDay(day.date)}
            title={`${compactDate(day.date.toISOString())}: ${day.count} Evolutions`}
            className={`size-3 rounded-[3px] ${heatClass(day.count)}`}
          />
        ))}
      </div>
      <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
        <span>Less</span>
        <span className="flex items-center gap-1">
          {[0, 1, 2, 3, 4].map((value) => (
            <span key={value} className={`size-3 rounded-[3px] ${heatClass(value)}`} />
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

function heatClass(count: number) {
  if (count <= 0) return 'bg-slate-100';
  if (count === 1) return 'bg-emerald-200';
  if (count === 2) return 'bg-emerald-400';
  if (count === 3) return 'bg-emerald-600';
  return 'bg-emerald-800';
}

function parseDate(value?: string) {
  if (!value) return undefined;
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? undefined : date;
}

function isoDay(date: Date) {
  return date.toISOString().slice(0, 10);
}
