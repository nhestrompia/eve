import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { EmptyState } from '../components/empty-state';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { RepositoryActivityView } from '../components/repository-activity-view';

export function TimelinePage() {
  const config = useQuery({ queryKey: ['config'], queryFn: api.config });
  const evolutions = useQuery({ queryKey: ['snapshots'], queryFn: api.snapshots });
  const repositories = useQuery({ queryKey: ['repositories'], queryFn: api.repositories });

  if (config.data && !config.data.initialized) {
    return (
      <EvolutionShell evolutions={[]} selectedId={undefined}>
        <EmptyState title="EVE is not initialized" detail="Run `eve init` in this repository, then refresh." />
      </EvolutionShell>
    );
  }

  if (evolutions.isLoading || repositories.isLoading) {
    return (
      <EvolutionShell evolutions={[]} selectedId={undefined}>
        <LoadingState label="Loading Evolutions" />
      </EvolutionShell>
    );
  }

  if (evolutions.error) {
    return (
      <EvolutionShell evolutions={[]} selectedId={undefined}>
        <ErrorState error={evolutions.error} />
      </EvolutionShell>
    );
  }
  if (repositories.error) {
    return (
      <EvolutionShell evolutions={[]} selectedId={undefined}>
        <ErrorState error={repositories.error} />
      </EvolutionShell>
    );
  }

  if (!evolutions.data?.length) {
    return (
      <EvolutionShell evolutions={[]} selectedId={undefined}>
        <EmptyState title="No Evolutions found" detail="Committed records from .eve/evolutions will appear here." />
      </EvolutionShell>
    );
  }

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={undefined} showHistoryRail={false} contentClassName="p-0">
      <RepositoryActivityView repositories={repositories.data ?? []} evolutions={evolutions.data ?? []} />
    </EvolutionShell>
  );
}
