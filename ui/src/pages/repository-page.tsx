import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { RepositoryActivityView } from '../components/repository-activity-view';

export function RepositoryPage() {
  const { repo } = useParams({ from: '/repositories/$repo' });
  const allEvolutions = useQuery({ queryKey: ['evolutions'], queryFn: () => api.evolutions() });
  const evolutions = useQuery({ queryKey: ['evolutions', repo], queryFn: () => api.evolutions(repo) });
  const repositories = useQuery({ queryKey: ['repositories'], queryFn: api.repositories });

  return (
    <EvolutionShell evolutions={allEvolutions.data ?? []} selectedId={undefined} showHistoryRail={false} contentClassName="p-0">
      {evolutions.isLoading || repositories.isLoading ? <LoadingState label={`Loading ${repo}`} /> : null}
      {evolutions.error ? <ErrorState error={evolutions.error} /> : null}
      {repositories.error ? <ErrorState error={repositories.error} /> : null}
      {evolutions.data && repositories.data ? (
        <RepositoryActivityView repositories={repositories.data} evolutions={evolutions.data} selectedRepo={repo} />
      ) : null}
    </EvolutionShell>
  );
}
