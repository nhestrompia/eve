import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from '@tanstack/react-router';
import { ExternalLink } from 'lucide-react';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { SnapshotCodeBrowser } from '../components/snapshot-code-browser';
import { Button } from '../components/ui/button';
import { Header } from './verification-page';

export function CodePage() {
  const { id } = useParams({ from: '/snapshots/$id/code' });
  const evolutions = useQuery({ queryKey: ['snapshots'], queryFn: api.snapshots });
  const detail = useQuery({ queryKey: ['snapshot-detail', id], queryFn: () => api.snapshotDetail(id) });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id} historyRailTarget="code" showSelectedSnapshotLink>
      {detail.isLoading ? <LoadingState label="Loading code" /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {detail.data ? (
        <section className="space-y-6">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <Header
              eyebrow="Code"
              title={detail.data.summary.title || detail.data.snapshot.title || id}
              subtitle="Relevant code behind this Snapshot, shown from the recorded Git state."
            />
            <Button asChild variant="outline" size="sm" className="shrink-0">
              <Link to="/snapshots/$id" params={{ id }}>
                <ExternalLink className="size-3.5" />
                Open Snapshot
              </Link>
            </Button>
          </div>
          <SnapshotCodeBrowser snapshotId={id} repository={detail.data.repository} />
        </section>
      ) : null}
    </EvolutionShell>
  );
}
