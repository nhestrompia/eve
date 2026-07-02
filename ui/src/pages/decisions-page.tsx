import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { EmptyPanel, Header } from './verification-page';

export function DecisionsPage() {
  const { id } = useParams({ from: '/evolutions/$id/decisions' });
  const evolutions = useQuery({ queryKey: ['evolutions'], queryFn: api.evolutions });
  const detail = useQuery({ queryKey: ['evolution', id], queryFn: () => api.evolution(id) });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {detail.isLoading ? <LoadingState label="Loading decisions" /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {detail.data ? (
        <section className="space-y-6">
          <Header eyebrow={id} title="Decisions" subtitle="Recorded product or implementation decisions for this Evolution." />
          {detail.data.evolution.decisions.length === 0 ? <EmptyPanel text="No decisions are recorded in this Evolution." /> : null}
          <div className="grid gap-4">
            {detail.data.evolution.decisions.map((decision, index) => (
              <article key={index} className="rounded-lg border bg-white p-5">
                <pre className="whitespace-pre-wrap font-mono text-sm">{JSON.stringify(decision, null, 2)}</pre>
              </article>
            ))}
          </div>
        </section>
      ) : null}
    </EvolutionShell>
  );
}
