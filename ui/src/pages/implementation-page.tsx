import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { compactDate } from '../format';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { EmptyPanel, Header } from './verification-page';

export function ImplementationPage() {
  const { id } = useParams({ from: '/evolutions/$id/implementation' });
  const evolutions = useQuery({ queryKey: ['evolutions'], queryFn: api.evolutions });
  const detail = useQuery({ queryKey: ['evolution', id], queryFn: () => api.evolution(id) });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {detail.isLoading ? <LoadingState label="Loading implementation" /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {detail.data ? (
        <section className="space-y-6">
          <Header eyebrow={id} title="Implementation" subtitle="Git repositories, snapshot commit, and contributed commits for this Evolution." />
          <div className="grid grid-cols-2 gap-4">
            <article className="rounded-lg border bg-white p-5">
              <p className="text-sm text-muted-foreground">Snapshot commit</p>
              <p className="mt-2 break-all font-mono text-lg font-semibold">{detail.data.evolution.implementation.snapshot || 'None recorded'}</p>
            </article>
            <article className="rounded-lg border bg-white p-5">
              <p className="text-sm text-muted-foreground">Repositories</p>
              <div className="mt-3 space-y-2">
                {Object.entries(detail.data.evolution.implementation.repositories ?? {}).map(([name, repo]) => (
                  <div key={name} className="flex justify-between rounded-md bg-slate-50 px-3 py-2">
                    <span className="font-semibold">{name}</span>
                    <span className="text-muted-foreground">{repo.status || 'unknown'}</span>
                  </div>
                ))}
              </div>
            </article>
          </div>
          {detail.data.commits.length === 0 ? <EmptyPanel text="No contributed commits are recorded for this Evolution." /> : null}
          <div className="grid gap-4">
            {detail.data.commits.map((commit) => (
              <article key={commit.hash} className="rounded-lg border bg-white p-5">
                <div className="grid grid-cols-[120px_minmax(0,1fr)_160px] gap-4">
                  <code className="font-mono font-semibold text-blue-700">{commit.shortHash}</code>
                  <div>
                    <h2 className="font-semibold">{commit.subject}</h2>
                    <p className="mt-1 text-sm text-muted-foreground">{commit.authorName || 'Unknown author'}</p>
                  </div>
                  <span className="text-right text-muted-foreground">{compactDate(commit.committedAt || commit.authoredAt)}</span>
                </div>
              </article>
            ))}
          </div>
        </section>
      ) : null}
    </EvolutionShell>
  );
}
