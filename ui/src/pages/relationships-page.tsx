import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { SnapshotRelationshipList } from '../components/snapshot-relationship-list';
import { Header } from './verification-page';

export function RelationshipsPage() {
  const { id } = useParams({ from: '/snapshots/$id/relationships' });
  const evolutions = useQuery({ queryKey: ['snapshots'], queryFn: api.snapshots });
  const detail = useQuery({ queryKey: ['snapshot-detail', id], queryFn: () => api.snapshotDetail(id) });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {detail.isLoading ? <LoadingState label="Loading relationships" /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {detail.data ? (
        <section className="space-y-6">
          <Header eyebrow={id} title="Relationships" subtitle="How this Snapshot connects to other product states." />
          <SnapshotRelationshipList relationships={detail.data.evolution.relationships} />
        </section>
      ) : null}
    </EvolutionShell>
  );
}
