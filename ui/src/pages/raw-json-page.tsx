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
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <JsonSection id="verification" title="Verification" value={detail.data.evolution.verification} />
            <JsonSection id="decisions" title="Decisions" value={detail.data.evolution.decisions} />
            <JsonSection id="risks" title="Risks" value={detail.data.evolution.risks} />
            <JsonSection id="implementation" title="Implementation" value={detail.data.evolution.implementation} />
            <JsonSection id="relationships" title="Relationships" value={detail.data.evolution.relationships} />
            <JsonSection id="sessions" title="Sessions" value={{ references: detail.data.evolution.sessions, artifacts: detail.data.sessions }} />
          </div>
          <pre className="overflow-auto rounded-lg border bg-slate-950 p-6 font-mono text-xs text-white">
            {JSON.stringify(detail.data.rawJson, null, 2)}
          </pre>
        </div>
      ) : null}
    </EvolutionShell>
  );
}

function JsonSection({ id, title, value }: { id: string; title: string; value: unknown }) {
  return (
    <section id={id} className="scroll-mt-24 rounded-lg border bg-white p-5">
      <h2 className="text-sm font-semibold">{title}</h2>
      <pre className="mt-3 max-h-56 overflow-auto rounded-md bg-slate-950 p-4 font-mono text-xs text-white">
        {JSON.stringify(value, null, 2)}
      </pre>
    </section>
  );
}
