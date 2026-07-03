import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { BehaviorSummarySection } from '../components/behavior-summary-section';
import { DetailActionTiles } from '../components/detail-action-tiles';
import { EmptyState } from '../components/empty-state';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { EvolutionHero } from '../components/evolution-hero';
import { ImplementationRail } from '../components/implementation-rail';
import { LoadingState } from '../components/loading-state';
import { VerificationSummarySection } from '../components/verification-summary-section';

export function EvolutionDetailPage() {
  const { id } = useParams({ from: '/snapshots/$id' });
  const evolutions = useQuery({ queryKey: ['snapshots'], queryFn: api.snapshots });
  const detail = useQuery({ queryKey: ['snapshot-detail', id], queryFn: () => api.snapshotDetail(id) });
  const snapshot = useQuery({ queryKey: ['snapshot', id], queryFn: () => api.snapshot(id), retry: false });

  const rows = evolutions.data ?? [];

  return (
    <EvolutionShell evolutions={rows} selectedId={id} showHistoryRail={false} contentClassName="p-0">
      {detail.isLoading ? <LoadingState label={`Loading ${id}`} /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {detail.data ? (
        <div className="min-h-[calc(100dvh-76px)] xl:pr-[460px]">
          <main className="min-w-0 px-5 sm:px-7 lg:px-9">
            <EvolutionHero detail={detail.data} snapshot={snapshot.data} />
            <BehaviorSummarySection behavior={detail.data.evolution.behavior} />
            <VerificationSummarySection values={detail.data.evolution.verification} evolutionId={detail.data.summary.id} />
            <DetailActionTiles detail={detail.data} />
          </main>
          <ImplementationRail detail={detail.data} evolutions={rows} snapshot={snapshot.data} />
        </div>
      ) : null}
      {!detail.isLoading && !detail.error && !detail.data ? <EmptyState title="Snapshot not found" detail={`${id} is not available.`} /> : null}
    </EvolutionShell>
  );
}
