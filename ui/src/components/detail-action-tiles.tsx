import { CheckCircle2, Clock3, Code2, FileText, GitFork, Scale } from 'lucide-react';
import type * as React from 'react';
import type { DetailResponse } from '../types';
import { compactDate, humanDate } from '../format';
import { activityEntries, displayDecision, displayRisk, titleCase } from '../lib/evolution-display';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from './ui/dialog';

export function DetailActionTiles({ detail }: { detail: DetailResponse }) {
  const tiles = [
    {
      title: 'Implementation',
      subtitle: `${detail.commits.length} commits · ${detail.snapshot.artifacts.length} artifacts`,
      icon: Code2,
      description: 'Git state, contributed commits, and Snapshot artifacts.',
      content: <ImplementationDialogContent detail={detail} />
    },
    {
      title: 'Decisions & Risks',
      subtitle: `${detail.evolution.decisions.length} decisions · ${detail.evolution.risks.length} risks`,
      icon: Scale,
      description: 'Recorded product decisions and known risks for this Snapshot.',
      content: <DecisionsRisksDialogContent detail={detail} />
    },
    {
      title: 'Related Snapshots',
      subtitle: relationshipSummary(detail),
      icon: GitFork,
      description: 'How this Snapshot connects to other product states.',
      content: <RelationshipsDialogContent detail={detail} />
    },
    {
      title: 'Snapshot Activity',
      subtitle: `${activityEntries(detail.evolution).length} events`,
      icon: Clock3,
      description: 'Recorded lifecycle events for this product state.',
      content: <ActivityDialogContent detail={detail} />
    }
  ];

  return (
    <section className="border-t py-8" aria-label="Snapshot detail sections">
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {tiles.map((tile, index) => {
          const Icon = tile.icon;
          return (
            <Dialog key={tile.title}>
              <DialogTrigger asChild>
                <button
                  className={`group grid min-h-[74px] grid-cols-[28px_minmax(0,1fr)] items-center gap-3 rounded-lg bg-white px-4 py-3 text-left transition-[background-color,box-shadow,scale] duration-150 hover:bg-slate-50 hover:shadow-[0_0_0_1px_rgba(15,23,42,0.16),0_10px_22px_-18px_rgba(15,23,42,0.45)] active:scale-[0.96] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${index === 0 ? 'shadow-[0_0_0_1px_rgba(15,23,42,0.28)]' : 'shadow-[0_0_0_1px_rgba(15,23,42,0.1)]'}`}
                >
                  <Icon className="size-5 text-slate-600" />
                  <span className="min-w-0">
                    <span className="block truncate text-xs font-semibold leading-4">{tile.title}</span>
                    <span className="mt-0.5 block truncate text-[11px] leading-4 text-muted-foreground">{tile.subtitle}</span>
                  </span>
                </button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>{tile.title}</DialogTitle>
                  <DialogDescription>{tile.description}</DialogDescription>
                </DialogHeader>
                <div className="min-h-0 overflow-y-auto pr-1">{tile.content}</div>
              </DialogContent>
            </Dialog>
          );
        })}
      </div>
    </section>
  );
}

function relationshipSummary(detail: DetailResponse): string {
  const entries = Object.entries(detail.evolution.relationships)
    .flatMap(([kind, values]) => (values ?? []).map((value) => `${kind.replaceAll('_', ' ')} ${value}`));
  return entries.length > 0 ? entries.slice(0, 2).join(' · ') : 'No relationships';
}

function ImplementationDialogContent({ detail }: { detail: DetailResponse }) {
  const repositories = Object.entries(detail.evolution.implementation.repositories ?? {});
  const snapshot = detail.evolution.implementation.snapshot;

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <article className="rounded-lg bg-secondary p-4">
          <p className="text-sm text-muted-foreground">Snapshot commit</p>
          <p className="mt-2 break-all font-mono text-base font-semibold">{snapshot || 'None recorded'}</p>
        </article>
        <article className="rounded-lg bg-secondary p-4">
          <p className="text-sm text-muted-foreground">Repositories</p>
          <div className="mt-3 space-y-2">
            {repositories.length === 0 ? <p className="text-sm text-muted-foreground">No repositories recorded.</p> : null}
            {repositories.map(([name, repo]) => (
              <div key={name} className="flex flex-col gap-1 rounded-md bg-white px-3 py-2 shadow-[0_0_0_1px_rgba(15,23,42,0.06)] sm:flex-row sm:items-center sm:justify-between sm:gap-4">
                <span className="font-semibold">{name}</span>
                <span className="text-muted-foreground">{repo.status || 'unknown'}</span>
              </div>
            ))}
          </div>
        </article>
      </div>

      <DialogSection title={`Commits (${detail.commits.length})`}>
        {detail.commits.length === 0 ? <EmptyDialogState text="No contributed commits are recorded for this Snapshot." /> : null}
        <div className="space-y-3">
          {detail.commits.map((commit) => (
            <article key={commit.hash} className="grid grid-cols-1 gap-3 rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.08)] sm:grid-cols-[96px_minmax(0,1fr)_112px] sm:gap-4">
              <code className="font-mono font-semibold text-blue-700">{commit.shortHash}</code>
              <div className="min-w-0">
                <h3 className="truncate font-semibold">{commit.subject}</h3>
                <p className="mt-1 text-sm text-muted-foreground">{commit.authorName || 'Unknown author'}</p>
              </div>
              <span className="text-sm text-muted-foreground sm:text-right">{compactDate(commit.committedAt || commit.authoredAt)}</span>
            </article>
          ))}
        </div>
      </DialogSection>

      <DialogSection title={`Artifacts (${detail.snapshot.artifacts.length})`}>
        {detail.snapshot.artifacts.length === 0 ? <EmptyDialogState text="No artifacts are recorded for this Snapshot." /> : null}
        <div className="space-y-3">
          {detail.snapshot.artifacts.map((artifact, index) => (
            <article key={`${artifact.type}-${artifact.path ?? artifact.url ?? artifact.uri ?? index}`} className="grid grid-cols-[40px_minmax(0,1fr)] items-start gap-3 rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.08)] sm:grid-cols-[40px_minmax(0,1fr)_110px]">
              <span className="flex size-10 items-center justify-center rounded-lg bg-violet-50 text-violet-700">
                <FileText className="size-4" />
              </span>
              <div className="min-w-0">
                <h3 className="truncate font-semibold capitalize">{artifact.type}</h3>
                <p className="mt-1 truncate text-sm text-muted-foreground">
                  {artifact.description || artifact.path || artifact.url || artifact.uri || 'Artifact reference'}
                </p>
              </div>
              <span className="col-start-2 w-fit rounded-md bg-emerald-50 px-2 py-1 text-center text-xs font-medium text-emerald-700 sm:col-start-auto sm:w-auto">{artifact.mimeType || 'reference'}</span>
            </article>
          ))}
        </div>
      </DialogSection>
    </div>
  );
}

function DecisionsRisksDialogContent({ detail }: { detail: DetailResponse }) {
  return (
    <div className="grid gap-6">
      <DialogSection title={`Decisions (${detail.evolution.decisions.length})`}>
        {detail.evolution.decisions.length === 0 ? <EmptyDialogState text="No decisions are recorded in this Snapshot." /> : null}
        <RecordList records={detail.evolution.decisions.map(displayDecision)} emptyPrefix="Decision" />
      </DialogSection>

      <DialogSection title={`Risks (${detail.evolution.risks.length})`}>
        {detail.evolution.risks.length === 0 ? <EmptyDialogState text="No risks are recorded in this Snapshot." /> : null}
        <RecordList records={detail.evolution.risks.map(displayRisk)} emptyPrefix="Risk" />
      </DialogSection>
    </div>
  );
}

function RelationshipsDialogContent({ detail }: { detail: DetailResponse }) {
  const entries = Object.entries(detail.evolution.relationships).flatMap(([kind, values]) =>
    (values ?? []).map((value) => ({ kind, value }))
  );

  if (entries.length === 0) {
    return <EmptyDialogState text="No relationships are recorded in this Snapshot." />;
  }

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      {entries.map((entry) => (
        <article key={`${entry.kind}-${entry.value}`} className="rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
          <p className="text-sm capitalize text-muted-foreground">{entry.kind.replaceAll('_', ' ')}</p>
          <p className="mt-2 break-all font-mono text-lg font-semibold">{entry.value}</p>
        </article>
      ))}
    </div>
  );
}

function ActivityDialogContent({ detail }: { detail: DetailResponse }) {
  const entries = activityEntries(detail.evolution);

  return (
    <ol className="space-y-0">
      {entries.map((entry, index) => (
        <li key={`${entry.event}-${entry.timestamp}-${index}`} className="grid grid-cols-[28px_minmax(0,1fr)] gap-4 pb-6 last:pb-0">
          <span className="relative flex justify-center">
            <span className="z-10 mt-1 flex size-6 items-center justify-center rounded-full bg-emerald-50 text-emerald-600 shadow-[0_0_0_1px_rgba(16,185,129,0.22)]">
              <CheckCircle2 className="size-3.5" />
            </span>
            {index < entries.length - 1 ? <span className="absolute top-7 h-full w-px bg-emerald-100" /> : null}
          </span>
          <span className="min-w-0">
            <span className="block font-semibold">{titleCase(entry.event || 'event')}</span>
            <span className="mt-1 block text-muted-foreground text-pretty">
              {entry.description || 'No event description.'}
              {entry.actor?.provider ? ` · ${entry.actor.provider}` : ''}
            </span>
            <span className="mt-1 block text-sm text-muted-foreground">{humanDate(entry.timestamp)}</span>
          </span>
        </li>
      ))}
    </ol>
  );
}

function DialogSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section>
      <h3 className="mb-3 font-semibold">{title}</h3>
      {children}
    </section>
  );
}

function RecordList({ records, emptyPrefix }: { records: ReturnType<typeof displayDecision>[]; emptyPrefix: string }) {
  if (records.length === 0) return null;

  return (
    <div className="grid gap-3">
      {records.map((record, index) => (
        <article key={`${record.title}-${index}`} className="rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
          <h4 className="font-semibold text-balance">{record.title || `${emptyPrefix} ${index + 1}`}</h4>
          {record.body ? <p className="mt-2 text-sm text-muted-foreground text-pretty">{record.body}</p> : null}
          {record.meta && record.meta.length > 0 ? (
            <dl className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
              {record.meta.map((item) => (
                <div key={`${item.label}-${item.value}`} className="rounded-md bg-secondary px-3 py-2">
                  <dt className="text-xs text-muted-foreground">{item.label}</dt>
                  <dd className="mt-1 font-medium">{item.value}</dd>
                </div>
              ))}
            </dl>
          ) : null}
        </article>
      ))}
    </div>
  );
}

function EmptyDialogState({ text }: { text: string }) {
  return <p className="rounded-lg bg-secondary p-4 text-sm text-muted-foreground">{text}</p>;
}
