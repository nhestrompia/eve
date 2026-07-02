import { Link, useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { LockKeyhole, MoreHorizontal } from 'lucide-react';
import { api } from '../api';
import { ActivityCard } from '../components/activity-card';
import { BehaviorCard } from '../components/behavior-card';
import { CheckoutActions } from '../components/checkout-actions';
import { DecisionsCard } from '../components/decisions-card';
import { EmptyState } from '../components/empty-state';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { ImplementationCard } from '../components/implementation-card';
import { JourneyCard } from '../components/journey-card';
import { LoadingState } from '../components/loading-state';
import { ProductSnapshotCard } from '../components/product-snapshot-card';
import { RelatedEvolutions } from '../components/related-evolutions';
import { RisksCard } from '../components/risks-card';
import { StatusBadge } from '../components/status-badge';
import { VerificationCard } from '../components/verification-card';
import { Button } from '../components/ui/button';

export function EvolutionDetailPage() {
  const { id } = useParams({ from: '/evolutions/$id' });
  const evolutions = useQuery({ queryKey: ['evolutions'], queryFn: api.evolutions });
  const detail = useQuery({ queryKey: ['evolution', id], queryFn: () => api.evolution(id) });
  const snapshot = useQuery({ queryKey: ['snapshot', id], queryFn: () => api.snapshot(id), retry: false });

  const rows = evolutions.data ?? [];

  return (
    <EvolutionShell evolutions={rows} selectedId={id}>
      {detail.isLoading ? <LoadingState label={`Loading ${id}`} /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {detail.data ? (
        <div className="space-y-4">
          <section className="grid grid-cols-[minmax(0,1fr)_250px] gap-8">
            <div className="flex gap-6">
              <div className="flex size-24 shrink-0 items-center justify-center rounded-2xl bg-emerald-50 text-emerald-700">
                <LockKeyhole className="size-12" />
              </div>
              <div className="min-w-0">
                <div className="flex items-center gap-3">
                  <h1 className="text-3xl font-semibold text-balance">{detail.data.summary.title || 'Untitled Evolution'}</h1>
                  <StatusBadge status={detail.data.summary.status} />
                </div>
                <p className="mt-3 text-muted-foreground">
                  Evolution <span className="font-mono">#{detail.data.summary.id.replace('EV-', '')}</span>
                  <span className="mx-2">•</span>
                  {detail.data.summary.updatedAt ? new Date(detail.data.summary.updatedAt).toLocaleString() : 'Unknown date'}
                  <span className="mx-2">•</span>
                  by <span className="font-medium text-blue-700">{detail.data.summary.sessionProviders.join(' & ') || 'EVE'}</span>
                </p>
                <p className="mt-5 max-w-3xl text-pretty">{detail.data.summary.outcome || 'No outcome recorded.'}</p>
                <RelationshipChips relationships={detail.data.evolution.relationships} />
              </div>
            </div>
            {snapshot.data ? (
              <CheckoutActions snapshot={snapshot.data} />
            ) : (
              <div className="flex justify-end">
                <Button asChild variant="outline" size="icon" aria-label="View implementation">
                  <Link to="/evolutions/$id/implementation" params={{ id }}>
                    <MoreHorizontal className="size-4" />
                  </Link>
                </Button>
              </div>
            )}
          </section>

          <section className="grid grid-cols-3 gap-4">
            <ProductSnapshotCard detail={detail.data} snapshot={snapshot.data} />
            <BehaviorCard behavior={detail.data.evolution.behavior} />
            <VerificationCard values={detail.data.evolution.verification} evolutionId={detail.data.summary.id} />
          </section>
          <section className="grid grid-cols-3 gap-4">
            <DecisionsCard decisions={detail.data.evolution.decisions} evolutionId={detail.data.summary.id} />
            <RisksCard risks={detail.data.evolution.risks} evolutionId={detail.data.summary.id} />
            <ImplementationCard evolution={detail.data.evolution} commits={detail.data.commits} />
          </section>
          <section className="grid grid-cols-[minmax(0,1fr)_450px] gap-4">
            <JourneyCard detail={detail.data} />
            <ActivityCard evolution={detail.data.evolution} />
          </section>
          <RelatedEvolutions evolution={detail.data.evolution} />
          <div className="flex gap-3">
            <Button asChild variant="outline">
              <Link to="/evolutions/$id/snapshot" params={{ id }}>
                Snapshot view
              </Link>
            </Button>
            <Button asChild variant="outline">
              <Link to="/evolutions/$id/sessions" params={{ id }}>
                AI sessions
              </Link>
            </Button>
          </div>
        </div>
      ) : null}
      {!detail.isLoading && !detail.error && !detail.data ? <EmptyState title="Evolution not found" detail={`${id} is not available.`} /> : null}
    </EvolutionShell>
  );
}

function RelationshipChips({ relationships }: { relationships: Record<string, string[] | undefined> }) {
  const entries = Object.entries(relationships)
    .flatMap(([kind, values]) => (values ?? []).map((value) => [kind.replaceAll('_', ' '), value] as const));

  if (entries.length === 0) {
    return <p className="mt-6 text-sm text-muted-foreground">No relationships recorded.</p>;
  }

  return (
    <div className="mt-6 flex flex-wrap items-center gap-2">
      <span className="rounded-md bg-secondary px-2 py-0.5 text-xs font-medium">Relationships:</span>
      {entries.map(([kind, value]) => (
        <span key={`${kind}-${value}`} className="rounded-md border px-2 py-0.5 text-xs capitalize">
          {kind} {value}
        </span>
      ))}
    </div>
  );
}
