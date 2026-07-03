import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { displayDecision } from '../lib/evolution-display';
import { EmptyPanel, Header } from './verification-page';

export function DecisionsPage() {
  const { id } = useParams({ from: '/snapshots/$id/decisions' });
  const evolutions = useQuery({ queryKey: ['snapshots'], queryFn: api.snapshots });
  const detail = useQuery({ queryKey: ['snapshot-detail', id], queryFn: () => api.snapshotDetail(id) });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {detail.isLoading ? <LoadingState label="Loading decisions" /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {detail.data ? (
        <section className="space-y-6">
          <Header eyebrow={id} title="Decisions" subtitle="Recorded product or implementation decisions for this Evolution." />
          {detail.data.evolution.decisions.length === 0 ? <EmptyPanel text="No decisions are recorded in this Evolution." /> : null}
          <div className="grid gap-4">
            {detail.data.evolution.decisions.map((decision, index) => {
              const record = displayDecision(decision);
              return (
                <article key={index} className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
                  <h2 className="text-lg font-semibold text-balance">{record.title}</h2>
                  {record.body ? <p className="mt-3 max-w-3xl text-muted-foreground text-pretty">{record.body}</p> : null}
                  {record.meta && record.meta.length > 0 ? (
                    <dl className="mt-5 grid grid-cols-1 gap-3 sm:grid-cols-3">
                      {record.meta.map((item) => (
                        <div key={`${item.label}-${item.value}`} className="rounded-md bg-secondary px-3 py-2">
                          <dt className="text-xs text-muted-foreground">{item.label}</dt>
                          <dd className="mt-1 font-medium">{item.value}</dd>
                        </div>
                      ))}
                    </dl>
                  ) : null}
                </article>
              );
            })}
          </div>
        </section>
      ) : null}
    </EvolutionShell>
  );
}
