import { Link } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { GitCompareArrows } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { defaultComparisonPair } from '../lib/comparison';
import { compactDate, humanDate, statusLabel } from '../format';
import type {
  ComparisonChange,
  ComparisonCheck,
  ComparisonDecision,
  ComparisonRisk,
  ComparisonTimelineItem,
  EvolutionSummary
} from '../types';

export function ComparePage() {
  const evolutions = useQuery({ queryKey: ['snapshots'], queryFn: api.snapshots });
  const [fromId, setFromId] = useState('');
  const [toId, setToId] = useState('');

  const options = useMemo(() => [...(evolutions.data ?? [])].sort((left, right) => right.createdAt.localeCompare(left.createdAt)), [evolutions.data]);

  useEffect(() => {
    if (fromId || toId || !evolutions.data?.length) return;
    const pair = defaultComparisonPair(evolutions.data);
    if (!pair) return;
    setFromId(pair.from);
    setToId(pair.to);
  }, [evolutions.data, fromId, toId]);

  const comparison = useQuery({
    queryKey: ['compare', fromId, toId],
    queryFn: () => api.compare(fromId, toId),
    enabled: Boolean(fromId && toId)
  });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={undefined} showHistoryRail={false}>
      <section className="mx-auto flex max-w-6xl flex-col gap-7">
        <header className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div className="max-w-3xl">
            <p className="text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Snapshot comparison</p>
            <h1 className="mt-2 text-3xl font-semibold tracking-tight text-slate-950">Compare product states</h1>
            <p className="mt-3 max-w-2xl text-sm leading-6 text-muted-foreground">
              Review user-visible changes, decisions, risks, validation, and timeline entries between two completed Snapshots.
            </p>
          </div>
          <div className="flex size-12 items-center justify-center rounded-lg bg-slate-100 text-blue-600">
            <GitCompareArrows className="size-5" />
          </div>
        </header>

        {evolutions.isLoading ? <LoadingState label="Loading Snapshots" /> : null}
        {evolutions.error ? <ErrorState error={evolutions.error} /> : null}

        {options.length < 2 && !evolutions.isLoading ? (
          <div className="rounded-lg border bg-white p-6">
            <h2 className="font-semibold">Need at least two Snapshots</h2>
            <p className="mt-2 text-sm text-muted-foreground">Comparison becomes available after this repository records another product state.</p>
          </div>
        ) : null}

        {options.length >= 2 ? (
          <>
            <div className="grid gap-4 rounded-lg border bg-white p-4 sm:grid-cols-[1fr_1fr_auto] sm:items-end">
              <SnapshotSelect label="From" value={fromId} evolutions={options} onChange={setFromId} />
              <SnapshotSelect label="To" value={toId} evolutions={options} onChange={setToId} />
              <div className="rounded-lg bg-slate-50 px-4 py-3 text-sm text-muted-foreground">
                {comparison.data ? `${comparison.data.range.length} ${comparison.data.range.length === 1 ? 'Snapshot' : 'Snapshots'} in range` : 'Select a range'}
              </div>
            </div>

            {comparison.isLoading ? <LoadingState label="Comparing Snapshots" /> : null}
            {comparison.error ? <ErrorState error={comparison.error} /> : null}
            {comparison.data ? (
              <div className="space-y-5">
                <div className="grid gap-4 md:grid-cols-2">
                  <BoundaryCard label="From" snapshot={comparison.data.from} />
                  <BoundaryCard label="To" snapshot={comparison.data.to} />
                </div>
                <ChangeSection title="Added" items={comparison.data.added} />
                <ChangeSection title="Changed" items={comparison.data.changed} />
                <ChangeSection title="Fixed" items={comparison.data.fixed} />
                <DecisionSection items={comparison.data.decisions} />
                <RiskSection items={comparison.data.risks} />
                <ValidationSection items={comparison.data.validation} />
                <TimelineSection items={comparison.data.timeline} />
              </div>
            ) : null}
          </>
        ) : null}
      </section>
    </EvolutionShell>
  );
}

function SnapshotSelect({
  label,
  value,
  evolutions,
  onChange
}: {
  label: string;
  value: string;
  evolutions: EvolutionSummary[];
  onChange: (value: string) => void;
}) {
  return (
    <label className="grid gap-2">
      <span className="text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">{label}</span>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="h-11 min-w-0 rounded-md border border-input bg-card px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      >
        <option value="">Select Snapshot</option>
        {evolutions.map((evolution) => (
          <option key={`${label}-${evolution.repository}-${evolution.id}`} value={evolution.id}>
            {evolution.repository ? `${evolution.repository} / ` : ''}
            {evolution.title} ({compactDate(evolution.createdAt)})
          </option>
        ))}
      </select>
    </label>
  );
}

function BoundaryCard({ label, snapshot }: { label: string; snapshot: { id: string; title: string; type: string; createdAt: string; repository?: string } }) {
  return (
    <Link to="/snapshots/$id" params={{ id: snapshot.id }} className="rounded-lg border bg-white p-5 transition-colors hover:bg-slate-50">
      <p className="text-xs font-semibold uppercase tracking-[0.14em] text-muted-foreground">{label}</p>
      <h2 className="mt-2 text-lg font-semibold">{snapshot.title}</h2>
      <p className="mt-2 text-sm text-muted-foreground">
        {snapshot.repository ? `${snapshot.repository} · ` : ''}
        {statusLabel(snapshot.type)} · {humanDate(snapshot.createdAt)}
      </p>
    </Link>
  );
}

function ChangeSection({ title, items }: { title: string; items: ComparisonChange[] }) {
  return (
    <Section title={title} count={items.length} empty={`No ${title.toLowerCase()} changes in this range.`}>
      {items.map((item) => (
        <LinkedItem key={`${title}-${item.snapshotId}-${item.text}`} snapshotId={item.snapshotId} title={item.snapshotTitle} meta={humanDate(item.createdAt)}>
          {item.text}
        </LinkedItem>
      ))}
    </Section>
  );
}

function DecisionSection({ items }: { items: ComparisonDecision[] }) {
  return (
    <Section title="Decisions" count={items.length} empty="No decisions were recorded in this range.">
      {items.map((item) => (
        <LinkedItem key={`${item.snapshotId}-${item.title}`} snapshotId={item.snapshotId} title={item.snapshotTitle} meta={item.rationale}>
          {item.title}
        </LinkedItem>
      ))}
    </Section>
  );
}

function RiskSection({ items }: { items: ComparisonRisk[] }) {
  return (
    <Section title="Risks" count={items.length} empty="No risks were recorded in this range.">
      {items.map((item) => (
        <LinkedItem
          key={`${item.snapshotId}-${item.title}`}
          snapshotId={item.snapshotId}
          title={item.snapshotTitle}
          meta={`${statusLabel(item.severity)}${item.mitigation ? ` · ${item.mitigation}` : ''}`}
        >
          {item.title}
        </LinkedItem>
      ))}
    </Section>
  );
}

function ValidationSection({ items }: { items: ComparisonCheck[] }) {
  return (
    <Section title="Validation" count={items.length} empty="No validation was recorded in this range.">
      {items.map((item) => (
        <LinkedItem key={`${item.snapshotId}-${item.command}`} snapshotId={item.snapshotId} title={item.snapshotTitle} meta={statusLabel(item.status)}>
          {item.command}
        </LinkedItem>
      ))}
    </Section>
  );
}

function TimelineSection({ items }: { items: ComparisonTimelineItem[] }) {
  return (
    <Section title="Timeline" count={items.length} empty="No timeline entries were recorded in this range.">
      {items.map((item) => (
        <LinkedItem
          key={`${item.snapshotId}-${item.phase}-${item.title}-${item.occurredAt}`}
          snapshotId={item.snapshotId}
          title={item.snapshotTitle}
          meta={`${statusLabel(item.phase)} · ${humanDate(item.occurredAt)}`}
        >
          {item.title}
        </LinkedItem>
      ))}
    </Section>
  );
}

function Section({ title, count, empty, children }: { title: string; count: number; empty: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border bg-white">
      <div className="flex items-center justify-between border-b px-5 py-4">
        <h2 className="font-semibold">{title}</h2>
        <span className="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-muted-foreground">{count}</span>
      </div>
      <div className="divide-y">
        {count === 0 ? <p className="p-5 text-sm text-muted-foreground">{empty}</p> : children}
      </div>
    </section>
  );
}

function LinkedItem({
  snapshotId,
  title,
  meta,
  children
}: {
  snapshotId: string;
  title: string;
  meta?: string;
  children: React.ReactNode;
}) {
  return (
    <Link to="/snapshots/$id" params={{ id: snapshotId }} className="block px-5 py-4 transition-colors hover:bg-slate-50">
      <p className="font-medium">{children}</p>
      <p className="mt-1 text-sm text-muted-foreground">
        {title} · {snapshotId}
        {meta ? ` · ${meta}` : ''}
      </p>
    </Link>
  );
}
