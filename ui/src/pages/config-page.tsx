import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';

export function ConfigPage() {
  const config = useQuery({ queryKey: ['config'], queryFn: api.config });
  const evolutions = useQuery({ queryKey: ['snapshots'], queryFn: api.snapshots });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={undefined}>
      {config.isLoading ? <LoadingState label="Loading config" /> : null}
      {config.error ? <ErrorState error={config.error} /> : null}
      {config.data ? (
        <section className="space-y-6">
          <div>
            <p className="font-mono text-sm text-muted-foreground">Repository Config</p>
            <h1 className="mt-2 text-3xl font-semibold text-balance">EVE UI source</h1>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="rounded-lg border bg-white p-5">
              <p className="text-sm text-muted-foreground">Repository</p>
              <p className="mt-2 break-all font-mono text-lg">{config.data.repository}</p>
            </div>
            <div className="rounded-lg border bg-white p-5">
              <p className="text-sm text-muted-foreground">EVE directory</p>
              <p className="mt-2 break-all font-mono text-lg">{config.data.eveDir}</p>
            </div>
            <div className="rounded-lg border bg-white p-5">
              <p className="text-sm text-muted-foreground">CLI version</p>
              <p className="mt-2 font-mono text-lg">{config.data.cliVersion}</p>
            </div>
            <div className="rounded-lg border bg-white p-5">
              <p className="text-sm text-muted-foreground">Snapshot schema</p>
              <p className="mt-2 font-mono text-lg">{config.data.snapshotSchemaVersion}</p>
            </div>
          </div>
          <pre className="overflow-auto rounded-lg border bg-slate-950 p-6 font-mono text-xs text-white">
            {JSON.stringify(config.data, null, 2)}
          </pre>
        </section>
      ) : null}
    </EvolutionShell>
  );
}
