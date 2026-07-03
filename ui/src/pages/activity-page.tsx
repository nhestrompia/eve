import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { activityEntries, titleCase } from '../lib/evolution-display';
import { humanDate } from '../format';
import { Header } from './verification-page';

export function ActivityPage() {
  const { id } = useParams({ from: '/snapshots/$id/activity' });
  const evolutions = useQuery({ queryKey: ['snapshots'], queryFn: api.snapshots });
  const detail = useQuery({ queryKey: ['snapshot-detail', id], queryFn: () => api.snapshotDetail(id) });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {detail.isLoading ? <LoadingState label="Loading activity" /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {detail.data ? (
        <section className="space-y-6">
          <Header eyebrow={id} title="Evolution Activity" subtitle="Recorded lifecycle events for this product state." />
          <ol className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
            {activityEntries(detail.data.evolution).map((entry, index, entries) => (
              <li key={`${entry.event}-${entry.timestamp}-${index}`} className="grid grid-cols-[24px_minmax(0,1fr)] gap-4 pb-6 last:pb-0">
                <span className="relative flex justify-center">
                  <span className="z-10 mt-1 size-3 rounded-full bg-emerald-500 shadow-[0_0_0_3px_rgba(16,185,129,0.14)]" />
                  {index < entries.length - 1 ? <span className="absolute top-4 h-full w-px bg-emerald-200" /> : null}
                </span>
                <span className="min-w-0">
                  <span className="block font-semibold">{titleCase(entry.event || 'event')}</span>
                  <span className="mt-1 block text-muted-foreground">
                    {entry.description || 'No event description.'}
                    {entry.actor?.provider ? ` · ${entry.actor.provider}` : ''}
                  </span>
                  <span className="mt-1 block text-sm text-muted-foreground">{humanDate(entry.timestamp)}</span>
                </span>
              </li>
            ))}
          </ol>
        </section>
      ) : null}
    </EvolutionShell>
  );
}
