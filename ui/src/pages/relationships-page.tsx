import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { EmptyPanel, Header } from './verification-page';

export function RelationshipsPage() {
  const { id } = useParams({ from: '/evolutions/$id/relationships' });
  const evolutions = useQuery({ queryKey: ['evolutions'], queryFn: api.evolutions });
  const detail = useQuery({ queryKey: ['evolution', id], queryFn: () => api.evolution(id) });

  const entries = Object.entries(detail.data?.evolution.relationships ?? {}).flatMap(([kind, values]) =>
    (values ?? []).map((value) => ({ kind, value }))
  );

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {detail.isLoading ? <LoadingState label="Loading relationships" /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {detail.data ? (
        <section className="space-y-6">
          <Header eyebrow={id} title="Relationships" subtitle="How this Evolution connects to other product states." />
          {entries.length === 0 ? <EmptyPanel text="No relationships are recorded in this Evolution." /> : null}
          <div className="grid grid-cols-2 gap-4">
            {entries.map((entry) => (
              <article key={`${entry.kind}-${entry.value}`} className="rounded-lg border bg-white p-5">
                <p className="text-sm capitalize text-muted-foreground">{entry.kind.replaceAll('_', ' ')}</p>
                <p className="mt-2 font-mono text-lg font-semibold">{entry.value}</p>
              </article>
            ))}
          </div>
        </section>
      ) : null}
    </EvolutionShell>
  );
}
