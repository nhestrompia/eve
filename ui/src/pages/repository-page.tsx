import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from '@tanstack/react-router';
import {
  ArrowRight,
  BookOpen,
  Box,
  Calendar,
  CheckCircle2,
  CircleHelp,
  Code2,
  Copy,
  ExternalLink,
  FileText,
  GitBranch,
  HardDrive,
  History,
  Package,
  Sparkles,
  type LucideIcon
} from 'lucide-react';
import { useState } from 'react';
import { api } from '../api';
import { compactDate, shortCommit } from '../format';
import type { DetailResponse, EvolutionSummary, RepositorySummary } from '../types';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { StatusBadge } from '../components/status-badge';
import { Badge } from '../components/ui/badge';
import { Button } from '../components/ui/button';

export function RepositoryPage() {
  const { repo } = useParams({ from: '/repositories/$repo' });
  const allEvolutions = useQuery({ queryKey: ['snapshots'], queryFn: () => api.snapshots() });
  const evolutions = useQuery({ queryKey: ['snapshots', repo], queryFn: () => api.snapshots(repo) });
  const repositories = useQuery({ queryKey: ['repositories'], queryFn: api.repositories });
  const repository = useQuery({ queryKey: ['repository', repo], queryFn: () => api.repository(repo) });
  const details = useQuery({
    queryKey: ['repository-page-details', repo, evolutions.data?.map((evolution) => evolution.id).join(',') ?? ''],
    queryFn: () => Promise.all((evolutions.data ?? []).map((evolution) => api.snapshotDetail(evolution.id))),
    enabled: (evolutions.data?.length ?? 0) > 0,
    staleTime: 30_000
  });

  return (
    <EvolutionShell evolutions={allEvolutions.data ?? []} selectedId={undefined} showHistoryRail={false} contentClassName="p-0">
      {evolutions.isLoading || repositories.isLoading || repository.isLoading ? <LoadingState label={`Loading ${repo}`} /> : null}
      {evolutions.error ? <ErrorState error={evolutions.error} /> : null}
      {repositories.error ? <ErrorState error={repositories.error} /> : null}
      {repository.error ? <ErrorState error={repository.error} /> : null}
      {evolutions.data && repositories.data && repository.data ? (
        <RepositoryOverviewPage
          repository={repository.data}
          repositories={repositories.data}
          evolutions={evolutions.data}
          details={details.data ?? []}
        />
      ) : null}
    </EvolutionShell>
  );
}

function RepositoryOverviewPage({
  repository,
  repositories,
  evolutions,
  details
}: {
  repository: RepositorySummary;
  repositories: RepositorySummary[];
  evolutions: EvolutionSummary[];
  details: DetailResponse[];
}) {
  const latest = evolutions[0];
  const stats = buildRepositoryStats(evolutions, details);
  const contributors = buildContributors(evolutions);
  const repoIndex = Math.max(0, repositories.findIndex((row) => row.name === repository.name));
  const tone = REPOSITORY_TONES[repoIndex % REPOSITORY_TONES.length];

  return (
    <main className="min-h-[calc(100dvh-76px)] min-w-0 bg-slate-50/45">
      <section className="border-b bg-white px-4 py-6 sm:px-6 lg:px-8">
        <div className="flex flex-col gap-6 xl:flex-row xl:items-end xl:justify-between">
          <div className="flex min-w-0 flex-col gap-5 sm:flex-row sm:items-start">
            <div className={`flex size-[70px] shrink-0 items-center justify-center rounded-lg ${tone.soft} ${tone.text} ring-1 ring-inset ring-current/10`}>
              <BookOpen className="size-9" />
            </div>
            <div className="min-w-0">
              <div className="flex min-w-0 flex-wrap items-center gap-3">
                <h1 className="truncate text-[28px] font-semibold leading-tight tracking-normal text-slate-950">{repository.name}</h1>
                <Badge variant={repository.remoteUrl ? 'success' : 'secondary'}>{repository.remoteUrl ? 'Remote' : 'Local'}</Badge>
              </div>
              <p className="mt-2 max-w-[62ch] text-sm leading-6 text-muted-foreground">
                Track product states, snapshots, sessions, and verification recorded for this repository.
              </p>
              <div className="mt-5 flex flex-wrap gap-2">
                <MetaPill icon={GitBranch} label={repository.branch || 'branch unknown'} />
                <MetaPill icon={Code2} label={shortCommit(repository.head)} />
                <MetaPill icon={CheckCircle2} label={repository.dirty ? 'Dirty' : 'Clean'} tone={repository.dirty ? 'warning' : 'success'} />
                <MetaPill icon={Calendar} label={repository.latestAt ? `Updated ${compactDate(repository.latestAt)}` : 'No snapshots'} />
              </div>
            </div>
          </div>
        </div>
        <nav className="mt-7 flex gap-7 overflow-x-auto text-sm font-medium text-muted-foreground">
          {[
            ['Overview', '#overview'],
            [`Snapshots ${evolutions.length}`, '#snapshots'],
            ['Activity', '#activity'],
            ['Artifacts', '#artifacts'],
            ['Settings', '#links']
          ].map(([label, href], index) => (
            <a
              key={label}
              href={href}
              className={`border-b-2 px-0.5 pb-4 transition-colors hover:text-foreground ${index === 0 ? 'border-blue-600 text-blue-700' : 'border-transparent'}`}
            >
              {label}
            </a>
          ))}
        </nav>
      </section>

      <div className="grid grid-cols-1 gap-6 px-4 py-6 sm:px-6 lg:px-8 xl:grid-cols-[minmax(0,1fr)_330px]">
        <div id="overview" className="grid min-w-0 grid-cols-1 gap-6 2xl:grid-cols-[minmax(0,1.2fr)_minmax(330px,0.8fr)]">
          <div className="min-w-0 space-y-5">
            <ReadmePanel repository={repository} />
            <LatestSnapshotCard latest={latest} />
            <RecentActivityCard evolutions={evolutions} />
          </div>
          <EvolutionTimelineCard evolutions={evolutions} />
        </div>

        <aside className="space-y-5">
          <RepositoryFactsCard repository={repository} />
          <SnapshotSummaryCard stats={stats} />
          <ContributorCard rows={contributors} />
          <RepositoryLinksCard repository={repository} />
        </aside>
      </div>
    </main>
  );
}

function ReadmePanel({ repository }: { repository: RepositorySummary }) {
  return (
    <section className="overflow-hidden rounded-lg bg-white shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
      <div className="flex min-h-14 items-center justify-between gap-3 border-b px-5">
        <h2 className="flex min-w-0 items-center gap-2 text-sm font-semibold">
          <FileText className="size-4 text-slate-500" />
          README.md
        </h2>
        <Button asChild variant="outline" size="sm" className="gap-2">
          <a href="#readme-raw">
            View raw
            <ExternalLink className="size-3.5" />
          </a>
        </Button>
      </div>
      <div id="readme-raw" className="px-5 py-5 sm:px-6 sm:py-6">
        {repository.readme ? (
          <MarkdownPreview content={repository.readme} />
        ) : (
          <p className="text-sm text-muted-foreground">No README found in this repository.</p>
        )}
      </div>
    </section>
  );
}

function MarkdownPreview({ content }: { content: string }) {
  const lines = content.split(/\r?\n/).slice(0, 80);
  return (
    <div className="space-y-4 text-sm leading-6 text-slate-800">
      {lines.map((line, index) => {
        if (line.startsWith('# ')) {
          return (
            <h3 key={index} className="text-2xl font-semibold leading-tight text-slate-950">
              {line.slice(2)}
            </h3>
          );
        }
        if (line.startsWith('## ')) {
          return (
            <h4 key={index} className="pt-4 text-lg font-semibold leading-tight text-slate-950">
              {line.slice(3)}
            </h4>
          );
        }
        if (line.startsWith('- ')) {
          return (
            <p key={index} className="pl-3 font-mono text-xs text-muted-foreground">
              {line}
            </p>
          );
        }
        if (line.startsWith('```')) return null;
        if (!line.trim()) return null;
        return (
          <p key={index} className="max-w-[76ch] text-pretty">
            {line}
          </p>
        );
      })}
    </div>
  );
}

function LatestSnapshotCard({ latest }: { latest?: EvolutionSummary }) {
  if (!latest) {
    return (
      <section id="snapshots" className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
        <h2 className="flex items-center gap-2 text-base font-semibold">
          <BookOpen className="size-4 text-blue-600" />
          Latest snapshot
        </h2>
        <p className="mt-3 text-sm text-muted-foreground">No snapshots have been recorded for this repository.</p>
      </section>
    );
  }

  return (
    <section id="snapshots" className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
      <h2 className="flex items-center gap-2 text-base font-semibold">
        <BookOpen className="size-4 text-blue-600" />
        Latest snapshot
      </h2>
      <div className="mt-5 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-3">
            <h3 className="text-base font-semibold text-balance">{latest.title}</h3>
            <StatusBadge status={latest.status} />
          </div>
          <p className="mt-2 max-w-[70ch] text-sm leading-6 text-muted-foreground">{latest.outcome}</p>
          <div className="mt-4 flex flex-wrap gap-2">
            <MetaPill icon={Sparkles} label={latest.type} />
            <MetaPill icon={Code2} label={`${latest.commitCount} ${latest.commitCount === 1 ? 'commit' : 'commits'}`} />
            <MetaPill icon={Calendar} label={compactDate(latest.updatedAt || latest.createdAt)} />
          </div>
        </div>
        <Button asChild variant="ghost" className="shrink-0 gap-2">
          <Link to="/snapshots/$id" params={{ id: latest.id }}>
            View snapshot
            <ArrowRight className="size-4" />
          </Link>
        </Button>
      </div>
    </section>
  );
}

function EvolutionTimelineCard({ evolutions }: { evolutions: EvolutionSummary[] }) {
  return (
    <section className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
      <div className="mb-5 flex items-center gap-2">
        <h2 className="text-base font-semibold">Evolution timeline</h2>
        <CircleHelp className="size-4 text-muted-foreground" />
      </div>
      {evolutions.length === 0 ? (
        <p className="text-sm text-muted-foreground">No timeline entries yet.</p>
      ) : (
        <div className="relative space-y-0 pl-8">
          <span className="absolute bottom-4 left-[14px] top-2 w-px bg-blue-600" />
          {evolutions.slice(0, 5).map((evolution) => (
            <Link
              key={evolution.id}
              to="/snapshots/$id"
              params={{ id: evolution.id }}
              className="group relative block pb-7 last:pb-0"
            >
              <span className="absolute -left-[26px] top-1 flex size-3.5 rounded-full border-2 border-blue-600 bg-white ring-4 ring-blue-50" />
              <span className="flex min-w-0 flex-wrap items-start justify-between gap-3">
                <strong className="max-w-[26ch] text-sm font-semibold leading-5 text-balance group-hover:text-blue-700">{evolution.title}</strong>
                <StatusBadge status={evolution.status} />
              </span>
              <span className="mt-2 block text-xs leading-5 text-muted-foreground">
                {evolution.type} · {compactDate(evolution.updatedAt || evolution.createdAt)}
              </span>
            </Link>
          ))}
        </div>
      )}
      {evolutions.length > 5 ? (
        <Button asChild variant="outline" className="mt-6 w-full gap-2">
          <a href="#activity">
            View all snapshots
            <ArrowRight className="size-4" />
          </a>
        </Button>
      ) : null}
    </section>
  );
}

function RecentActivityCard({ evolutions }: { evolutions: EvolutionSummary[] }) {
  return (
    <section id="activity" className="space-y-3">
      <div className="flex items-center justify-between gap-4">
        <h2 className="text-base font-semibold">Recent activity</h2>
        <Button variant="outline" size="sm" className="gap-2">
          All activity types
          <History className="size-3.5" />
        </Button>
      </div>
      <div className="overflow-hidden rounded-lg bg-white shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
        {evolutions.length === 0 ? (
          <div className="p-5 text-sm text-muted-foreground">No activity has been recorded.</div>
        ) : (
          evolutions.slice(0, 6).map((evolution, index) => (
            <Link
              key={evolution.id}
              to="/snapshots/$id"
              params={{ id: evolution.id }}
              className={`grid grid-cols-[36px_minmax(0,1fr)_20px] items-center gap-3 px-4 py-3.5 transition-colors hover:bg-slate-50 sm:grid-cols-[44px_minmax(0,1fr)_112px_20px] ${index > 0 ? 'border-t' : ''}`}
            >
              <span className="flex size-9 items-center justify-center rounded-full bg-blue-50 text-blue-700">
                <BookOpen className="size-5" />
              </span>
              <span className="min-w-0">
                <span className="flex min-w-0 items-center gap-3">
                  <strong className="truncate text-sm font-semibold">{evolution.title}</strong>
                  <StatusBadge status={evolution.status} />
                </span>
                <span className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
                  <span className="font-mono">{evolution.id}</span>
                  <span>{evolution.type}</span>
                  {evolution.snapshot ? <span className="font-mono">{shortCommit(evolution.snapshot)}</span> : null}
                </span>
              </span>
              <span className="hidden text-right text-xs text-muted-foreground sm:block">{compactDate(evolution.updatedAt || evolution.createdAt)}</span>
              <ArrowRight className="size-4 text-slate-500" />
            </Link>
          ))
        )}
      </div>
    </section>
  );
}

function RepositoryFactsCard({ repository }: { repository: RepositorySummary }) {
  const rows = [
    ['Description', 'Track product states, snapshots, sessions, and verification recorded for this repository.', Box],
    ['Language', repository.primaryLanguage || 'Unknown', Code2],
    ['Size', formatBytes(repository.sizeBytes), HardDrive],
    ['Created', compactDate(repository.createdAt), Calendar]
  ] as const;
  return (
    <RailCard title="Repository overview">
      <div className="space-y-4">
        {rows.map(([label, value, Icon]) => (
          <div key={label} className="grid grid-cols-[18px_minmax(0,1fr)] gap-3">
            <Icon className="mt-0.5 size-4 text-slate-500" />
            <div className="min-w-0">
              <p className="text-xs font-medium text-muted-foreground">{label}</p>
              <p className="mt-1 text-sm leading-5 text-slate-700 text-pretty">{value}</p>
            </div>
          </div>
        ))}
      </div>
    </RailCard>
  );
}

function SnapshotSummaryCard({ stats }: { stats: RepositoryStats }) {
  const tiles = [
    ['Snapshots', stats.snapshots],
    ['Features', stats.features],
    ['Bug fixes', stats.bugfixes],
    ['Refactor', stats.refactors],
    ['Commits', stats.commits],
    ['Decisions', stats.decisions],
    ['Validated', stats.validated],
    ['Risks', stats.risks]
  ] as const;
  return (
    <RailCard title="Snapshot summary" eyebrow="Last 12 months">
      <div className="grid grid-cols-2 gap-2.5">
        {tiles.map(([label, value]) => (
          <div key={label} className="rounded-lg bg-white px-3 py-2.5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
            <div className="text-xl font-semibold leading-6 tabular-nums">{value}</div>
            <div className="mt-1 text-xs text-muted-foreground">{label}</div>
          </div>
        ))}
      </div>
    </RailCard>
  );
}

function ContributorCard({ rows }: { rows: ContributorRow[] }) {
  const max = Math.max(1, ...rows.map((row) => row.count));
  return (
    <RailCard title="Top contributors" eyebrow="Last 30 days">
      <div className="space-y-4">
        {rows.map((row, index) => {
          const tone = REPOSITORY_TONES[index % REPOSITORY_TONES.length];
          return (
            <div key={row.label} className="grid grid-cols-[88px_minmax(0,1fr)_54px] items-center gap-3">
              <span className="flex min-w-0 items-center gap-2 text-sm font-semibold">
                <img src={agentAvatarPath(row.label)} alt="" className="size-6 rounded-lg" />
                <span className="truncate">{row.label}</span>
              </span>
              <span className="h-1.5 overflow-hidden rounded-full bg-slate-100">
                <span className={`block h-full rounded-full ${tone.bg}`} style={{ width: `${Math.max(8, (row.count / max) * 100)}%` }} />
              </span>
              <span className="text-right text-sm text-muted-foreground tabular-nums">{row.count}</span>
            </div>
          );
        })}
      </div>
    </RailCard>
  );
}

function RepositoryLinksCard({ repository }: { repository: RepositorySummary }) {
  const [copied, setCopied] = useState(false);
  const copyPath = async () => {
    await navigator.clipboard.writeText(repository.root || repository.name);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  };
  return (
    <RailCard id="links" title="Repository links">
      <div className="space-y-3">
        {repository.remoteUrl ? (
          <a
            className="flex min-h-9 items-center gap-3 rounded-md px-1 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50 hover:text-slate-950"
            href={repository.remoteUrl}
            target="_blank"
            rel="noreferrer"
          >
            <ExternalLink className="size-4" />
            Open in GitHub
            <ExternalLink className="ml-auto size-4 text-slate-500" />
          </a>
        ) : null}
        <button
          className="flex min-h-9 w-full items-center gap-3 rounded-md px-1 text-left text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50 hover:text-slate-950"
          onClick={copyPath}
        >
          <Copy className="size-4" />
          {copied ? 'Copied path' : 'Copy local path'}
          <span className="ml-auto max-w-[150px] truncate font-mono text-xs text-muted-foreground">{repository.root}</span>
        </button>
      </div>
    </RailCard>
  );
}

function RailCard({
  id,
  title,
  eyebrow,
  children
}: {
  id?: string;
  title: string;
  eyebrow?: string;
  children: React.ReactNode;
}) {
  return (
    <section id={id} className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
      <div className="mb-5 flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold">{title}</h2>
        {eyebrow ? (
          <span className="flex items-center gap-1 text-xs font-medium text-muted-foreground">
            <Package className="size-3" />
            {eyebrow}
          </span>
        ) : null}
      </div>
      {children}
    </section>
  );
}

function MetaPill({ icon: Icon, label, tone }: { icon: LucideIcon; label: string; tone?: 'success' | 'warning' }) {
  return (
    <span
      className={`inline-flex h-8 max-w-full items-center gap-2 rounded-md bg-white px-3 text-xs font-medium shadow-[0_0_0_1px_rgba(15,23,42,0.12)] ${
        tone === 'success' ? 'text-emerald-700' : tone === 'warning' ? 'text-orange-700' : 'text-slate-600'
      }`}
    >
      <Icon className="size-3.5 shrink-0" />
      <span className="truncate">{label}</span>
    </span>
  );
}

type RepositoryStats = {
  snapshots: number;
  features: number;
  bugfixes: number;
  refactors: number;
  commits: number;
  decisions: number;
  validated: number;
  risks: number;
};

function buildRepositoryStats(evolutions: EvolutionSummary[], details: DetailResponse[]): RepositoryStats {
  return {
    snapshots: evolutions.length,
    features: evolutions.filter((evolution) => evolution.type === 'feature').length,
    bugfixes: evolutions.filter((evolution) => evolution.type === 'bugfix').length,
    refactors: evolutions.filter((evolution) => evolution.type === 'refactor').length,
    commits: evolutions.reduce((total, evolution) => total + (evolution.commitCount ?? 0), 0),
    decisions: details.reduce((total, detail) => total + detail.evolution.decisions.length, 0),
    validated: evolutions.filter((evolution) => evolution.verificationState === 'passed').length,
    risks: details.reduce((total, detail) => total + detail.evolution.risks.length, 0)
  };
}

type ContributorRow = {
  label: string;
  count: number;
};

function buildContributors(evolutions: EvolutionSummary[]): ContributorRow[] {
  const counts = new Map<string, number>();
  for (const evolution of evolutions) {
    const providers = evolution.sessionProviders.length > 0 ? evolution.sessionProviders : ['Codex'];
    for (const provider of providers) {
      const label = normalizeProvider(provider);
      counts.set(label, (counts.get(label) ?? 0) + 1);
    }
  }
  return ['Codex', 'Claude', 'OpenCode', 'Other']
    .map((label) => ({ label, count: counts.get(label) ?? 0 }))
    .filter((row) => row.count > 0 || row.label === 'Codex');
}

function normalizeProvider(value: string) {
  const normalized = value.toLowerCase();
  if (normalized.includes('codex')) return 'Codex';
  if (normalized.includes('claude')) return 'Claude';
  if (normalized.includes('opencode')) return 'OpenCode';
  return 'Other';
}

function agentAvatarPath(label: string) {
  if (label === 'Codex') return '/agents/codex.svg';
  if (label === 'Claude') return '/agents/claude.svg';
  if (label === 'OpenCode') return '/agents/opencode.svg';
  return '/agents/other.svg';
}

function formatBytes(value?: number) {
  if (!value) return 'Unknown';
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

type RepositoryTone = {
  bg: string;
  text: string;
  soft: string;
};

const REPOSITORY_TONES: RepositoryTone[] = [
  { bg: 'bg-blue-600', text: 'text-blue-700', soft: 'bg-blue-50' },
  { bg: 'bg-emerald-500', text: 'text-emerald-700', soft: 'bg-emerald-50' },
  { bg: 'bg-violet-600', text: 'text-violet-700', soft: 'bg-violet-50' },
  { bg: 'bg-amber-500', text: 'text-amber-700', soft: 'bg-amber-50' }
];
