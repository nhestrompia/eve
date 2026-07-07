import { useQuery } from '@tanstack/react-query';
import { AlertCircle } from 'lucide-react';
import { api } from '../api';
import type { PendingSnapshot } from '../types';

export function PendingSnapshotBanner() {
  const config = useQuery({
    queryKey: ['config'],
    queryFn: api.config,
    refetchInterval: 10_000
  });
  const pending = config.data?.pendingSnapshot;

  if (!pending) return null;

  return (
    <section className="pending-snapshot-banner border-b px-4 py-3 md:px-8" aria-label="Pending Snapshot">
      <div className="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 items-start gap-3">
          <AlertCircle className="pending-snapshot-icon mt-0.5 size-4 shrink-0" aria-hidden="true" />
          <div className="min-w-0">
            <p className="font-semibold text-balance">Pending Snapshot</p>
            <p className="pending-snapshot-copy mt-0.5 text-sm leading-5 text-pretty">
              {pending.repoId} on {pending.branch} has {commitCountLabel(pending)} of committed work waiting for a Snapshot or Skip.
            </p>
          </div>
        </div>
        <dl className="pending-snapshot-meta grid shrink-0 grid-cols-3 gap-3 text-xs sm:text-right">
          <PendingDatum label="Trigger" value={triggerLabel(pending.trigger)} />
          <PendingDatum label="To" value={shortHash(pending.range.to)} mono />
          <PendingDatum label="Trunk" value={pending.trunkBranch} />
        </dl>
      </div>
    </section>
  );
}

function PendingDatum({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="min-w-0">
      <dt className="font-medium">{label}</dt>
      <dd className={`mt-0.5 truncate font-semibold ${mono ? 'font-mono tabular-nums' : ''}`}>{value}</dd>
    </div>
  );
}

function commitCountLabel(pending: PendingSnapshot) {
  const count = pending.range.commits.length;
  return `${count} ${count === 1 ? 'commit' : 'commits'}`;
}

function shortHash(hash: string) {
  return hash ? hash.slice(0, 7) : 'unknown';
}

function triggerLabel(trigger?: string) {
  if (trigger === 'merge') return 'Merge';
  if (trigger === 'idle') return 'Idle';
  return 'Pending';
}
