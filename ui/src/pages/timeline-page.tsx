import { Navigate } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { EmptyState } from '../components/empty-state';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';

export function TimelinePage() {
  const config = useQuery({ queryKey: ['config'], queryFn: api.config });
  const evolutions = useQuery({ queryKey: ['evolutions'], queryFn: api.evolutions });

  if (config.data && !config.data.initialized) {
    return (
      <EvolutionShell evolutions={[]} selectedId={undefined}>
        <EmptyState title="EVE is not initialized" detail="Run `eve init` in this repository, then refresh." />
      </EvolutionShell>
    );
  }

  if (evolutions.isLoading) {
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

  const first = evolutions.data?.[0];
  if (!first) {
    return (
      <EvolutionShell evolutions={[]} selectedId={undefined}>
        <EmptyState title="No Evolutions found" detail="Committed records from .eve/evolutions will appear here." />
      </EvolutionShell>
    );
  }

  return <Navigate to="/evolutions/$id" params={{ id: first.id }} />;
}
