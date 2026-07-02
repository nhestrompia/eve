import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';

export function RawJsonPage() {
  const { id } = useParams({ from: '/json/$id' });
  const evolutions = useQuery({ queryKey: ['evolutions'], queryFn: api.evolutions });
  const detail = useQuery({ queryKey: ['evolution', id], queryFn: () => api.evolution(id) });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {detail.isLoading ? <LoadingState label={`Loading ${id} JSON`} /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {detail.data ? (
        <div className="space-y-5">
          <div>
            <p className="font-mono text-sm font-semibold text-blue-700">{id}</p>
            <h1 className="mt-2 text-3xl font-semibold text-balance">Raw canonical JSON</h1>
          </div>
          <pre className="overflow-auto rounded-lg border bg-slate-950 p-6 font-mono text-xs text-white">
            {JSON.stringify(detail.data.rawJson, null, 2)}
          </pre>
        </div>
      ) : null}
    </EvolutionShell>
  );
}
