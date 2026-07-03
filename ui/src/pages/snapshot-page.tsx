import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { BehaviorCard } from '../components/behavior-card';
import { CheckoutActions } from '../components/checkout-actions';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { SnapshotTimeline } from '../components/snapshot-timeline';
import { VerificationCard } from '../components/verification-card';

export function SnapshotPage() {
  const { id } = useParams({ from: '/evolutions/$id/snapshot' });
  const evolutions = useQuery({ queryKey: ['evolutions'], queryFn: api.evolutions });
  const snapshot = useQuery({ queryKey: ['snapshot', id], queryFn: () => api.snapshot(id), retry: false });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {snapshot.isLoading ? <LoadingState label={`Resolving ${id}`} /> : null}
      {snapshot.error ? <ErrorState error={snapshot.error} /> : null}
      {snapshot.data ? (
        <div className="grid grid-cols-1 gap-6 xl:grid-cols-[minmax(0,1fr)_360px] xl:gap-7">
          <div className="space-y-6">
            <section className="grid grid-cols-1 gap-6 rounded-lg border bg-white p-5 sm:p-8 lg:grid-cols-[minmax(0,1fr)_250px] lg:gap-8">
              <div>
                <p className="font-mono text-sm font-semibold text-blue-700">{snapshot.data.id}</p>
                <h1 className="mt-3 text-3xl font-semibold text-balance">{snapshot.data.title}</h1>
                <p className="mt-4 max-w-3xl text-muted-foreground text-pretty">{snapshot.data.outcome}</p>
                <div className="mt-8 rounded-lg border bg-slate-50 p-5">
                  <p className="text-sm text-muted-foreground">Snapshot commit</p>
                  <p className="mt-2 break-all font-mono text-xl font-semibold">{snapshot.data.commit}</p>
                </div>
              </div>
              <CheckoutActions snapshot={snapshot.data} />
            </section>
            <section className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <BehaviorCard behavior={snapshot.data.behavior} />
              <VerificationCard values={snapshot.data.verification} evolutionId={snapshot.data.id} />
            </section>
          </div>
          <aside className="rounded-lg border bg-white p-5">
            <SnapshotTimeline evolutions={evolutions.data ?? []} selectedId={id} route="snapshot" />
          </aside>
        </div>
      ) : null}
    </EvolutionShell>
  );
}
