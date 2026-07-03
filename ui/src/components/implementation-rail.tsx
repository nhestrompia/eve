import { Link } from '@tanstack/react-router';
import { Copy, ExternalLink, FileText, X } from 'lucide-react';
import { useState } from 'react';
import { humanDate, shortCommit } from '../format';
import { displayDecision, displayRisk } from '../lib/evolution-display';
import type { DetailResponse, EvolutionSummary, SessionRecord, SnapshotResponse } from '../types';
import { SnapshotTimeline } from './snapshot-timeline';
import { Button } from './ui/button';

export function ImplementationRail({
  detail,
  evolutions,
  snapshot
}: {
  detail: DetailResponse;
  evolutions: EvolutionSummary[];
  snapshot?: SnapshotResponse;
}) {
  const id = detail.summary.id;
  const commit = snapshot?.commit || detail.evolution.implementation.snapshot || '';
  const [copied, setCopied] = useState(false);

  const copyCommit = async () => {
    if (!commit) return;
    await navigator.clipboard.writeText(commit);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  };

  return (
    <aside className="scrollbar-none border-t bg-white px-5 py-6 sm:px-7 sm:py-8 xl:fixed xl:right-0 xl:top-0 xl:z-30 xl:h-dvh xl:w-[460px] xl:overflow-y-auto xl:border-l xl:border-t-0">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <FileText className="size-4" />
          <h2 className="text-lg font-semibold">Implementation</h2>
        </div>
        <Button asChild variant="ghost" size="icon" aria-label="Return to history">
          <Link to="/">
            <X className="size-4" />
          </Link>
        </Button>
      </div>

      <section className="mt-4">
        <p className="text-sm font-medium">Snapshot</p>
        <div className="mt-3 flex h-12 items-center justify-between gap-3 rounded-lg bg-white px-4 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
          <code className="min-w-0 truncate font-mono font-semibold tabular-nums">{commit ? shortCommit(commit) : 'None recorded'}</code>
          <Button variant="ghost" size="icon" aria-label={copied ? 'Snapshot copied' : 'Copy snapshot commit'} onClick={copyCommit} disabled={!commit}>
            <Copy className="size-4" />
          </Button>
        </div>
        <p className="mt-3 text-sm text-muted-foreground">
          Committed {humanDate(detail.summary.updatedAt || detail.summary.createdAt)}
        </p>
      </section>

      <SnapshotTimeline evolutions={evolutions} selectedId={id} className="mt-6" />

      <section className="mt-8">
        <h3 className="font-semibold">Implementation Sessions ({detail.sessions.length})</h3>
        <div className="mt-5 space-y-6">
          {detail.sessions.length === 0 ? <p className="text-sm text-muted-foreground">No AI sessions are recorded for this Evolution.</p> : null}
          {detail.sessions.slice(0, 4).map((session, index) => (
            <SessionRailItem key={session.key} session={session} index={index} isLast={index === detail.sessions.length - 1} evolutionId={id} />
          ))}
        </div>
        <Link
          to="/snapshots/$id/sessions"
          params={{ id }}
          className="mt-6 inline-flex min-h-10 items-center gap-2 rounded-md px-2 text-sm font-medium text-blue-700 hover:bg-blue-50"
        >
          View conversation threads <ExternalLink className="size-3.5" />
        </Link>
      </section>

      <div className="mt-8 space-y-6 border-t pt-6">
        <RailRecordGroup title="Decisions" records={detail.evolution.decisions.map(displayDecision)} emptyText="No decisions recorded." />
        <RailRecordGroup title="Risks" records={detail.evolution.risks.map(displayRisk)} emptyText="No risks recorded." />
      </div>
    </aside>
  );
}

type RailRecord = ReturnType<typeof displayDecision>;

function RailRecordGroup({ title, records, emptyText }: { title: string; records: RailRecord[]; emptyText: string }) {
  return (
    <section>
      <h3 className="font-semibold">
        {title} ({records.length})
      </h3>
      {records.length === 0 ? <p className="mt-3 rounded-lg bg-slate-50 p-3 text-sm text-muted-foreground">{emptyText}</p> : null}
      <div className="mt-3 space-y-3">
        {records.map((record, index) => (
          <article key={`${title}-${record.title}-${index}`} className="rounded-lg bg-slate-50 p-3">
            <h4 className="text-sm font-semibold text-balance">{record.title || `${title.slice(0, -1)} ${index + 1}`}</h4>
            {record.body ? <p className="mt-2 text-sm text-muted-foreground text-pretty">{record.body}</p> : null}
            {(record.meta ?? []).length > 0 ? (
              <dl className="mt-3 grid grid-cols-1 gap-2">
                {(record.meta ?? []).map((item) => (
                  <div key={`${item.label}-${item.value}`} className="rounded-md bg-white px-2.5 py-2 shadow-[0_0_0_1px_rgba(15,23,42,0.06)]">
                    <dt className="text-xs text-muted-foreground">{item.label}</dt>
                    <dd className="mt-0.5 text-sm font-medium">{item.value}</dd>
                  </div>
                ))}
              </dl>
            ) : null}
          </article>
        ))}
      </div>
    </section>
  );
}

function SessionRailItem({
  session,
  index,
  isLast,
  evolutionId
}: {
  session: SessionRecord;
  index: number;
  isLast: boolean;
  evolutionId: string;
}) {
  const content = (
    <>
      <div className="relative flex justify-center">
        <span className="z-10 flex size-6 items-center justify-center rounded-full bg-white font-mono text-xs shadow-[0_0_0_1px_rgba(15,23,42,0.18)]">
          {index + 1}
        </span>
        {!isLast ? <span className="absolute top-6 h-16 w-px bg-slate-200" /> : null}
      </div>
      <div className="flex size-9 items-center justify-center rounded-lg bg-violet-50 text-violet-700">
        <FileText className="size-4" />
      </div>
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <p className="truncate font-semibold">{session.providerName}</p>
          <span className="rounded-md bg-emerald-50 px-1.5 py-0.5 text-[11px] font-medium text-emerald-700">{session.status}</span>
        </div>
        <p className="mt-1 text-sm text-muted-foreground text-pretty">
          {session.hasTranscript
            ? session.title || `${session.preview.messageCount} messages captured`
            : session.localSources[0]?.title || `Reference only: ${session.id}`}
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          {session.preview.messageCount
            ? `${session.preview.messageCount} messages`
            : session.localSources[0]?.match
              ? `Matched by ${session.localSources[0].match}`
              : session.captureHint}
        </p>
      </div>
    </>
  );

  if (session.hasTranscript || session.localSources.length > 0) {
    return (
      <Link
        to="/snapshots/$id/session/$sessionId"
        params={{ id: evolutionId, sessionId: session.key }}
        className="grid grid-cols-[24px_40px_minmax(0,1fr)] gap-4 rounded-lg py-1 hover:text-blue-700"
      >
        {content}
      </Link>
    );
  }

  return <div className="grid grid-cols-[24px_40px_minmax(0,1fr)] gap-4 py-1">{content}</div>;
}
