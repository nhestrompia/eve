import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { EmptyState } from '../components/empty-state';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { RepositoryActivityView } from '../components/repository-activity-view';
import { Skeleton } from '../components/ui/skeleton';

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
      <EvolutionShell evolutions={[]} selectedId={undefined} showHistoryRail={false} contentClassName="p-0">
        <TimelineSkeleton />
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
        <EmptyState title="No Snapshots found" detail="Committed records from .eve/snapshots will appear here." />
      </EvolutionShell>
    );
  }

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={undefined} showHistoryRail={false} contentClassName="p-0">
      <RepositoryActivityView repositories={repositories.data ?? []} evolutions={evolutions.data ?? []} />
    </EvolutionShell>
  );
}

function TimelineSkeleton() {
  return (
    <main
      className="min-h-dvh min-w-0 bg-background px-4 py-5 sm:px-6 sm:py-7 lg:px-8 lg:py-8"
      aria-label="Loading snapshots..."
      role="status"
    >
      <span className="sr-only">Loading snapshots...</span>
      <div className="grid grid-cols-1 gap-6 2xl:grid-cols-[minmax(0,1fr)_350px] 2xl:gap-7">
        <section className="min-w-0 space-y-7">
          <header className="space-y-5">
            <div>
              <Skeleton className="h-3 w-20" />
              <Skeleton className="mt-4 h-10 w-72 max-w-full" />
              <div className="mt-4 max-w-[54ch] space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
              </div>
            </div>

            <section className="min-w-0 space-y-2.5" aria-label="Loading recent repositories">
              <div className="flex items-center justify-between gap-3">
                <Skeleton className="h-4 w-36" />
                <Skeleton className="h-3 w-16" />
              </div>
              <div className="grid min-w-0 gap-3 md:grid-cols-2 xl:grid-cols-4">
                {[0, 1, 2, 3].map((item) => (
                  <div key={item} className="min-w-0 rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
                    <div className="flex items-center justify-between gap-3">
                      <div className="flex min-w-0 flex-1 items-center gap-2">
                        <Skeleton className="size-2.5 rounded-full" />
                        <Skeleton className="h-4 w-24" />
                      </div>
                      <Skeleton className="size-4" />
                    </div>
                    <Skeleton className="mt-4 h-8 w-full" />
                    <div className="mt-3 space-y-1.5">
                      <Skeleton className="h-3 w-28" />
                      <Skeleton className="h-3 w-20" />
                    </div>
                  </div>
                ))}
              </div>
            </section>
          </header>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <Skeleton className="h-6 w-40" />
              <Skeleton className="size-4 rounded-full" />
            </div>
            <div className="w-full rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
              <div className="grid grid-cols-[30px_repeat(12,12px)] gap-[3px] overflow-hidden sm:grid-cols-[30px_repeat(24,12px)] lg:grid-cols-[30px_repeat(36,12px)]">
                <Skeleton className="h-4 bg-transparent" />
                {Array.from({ length: 36 }).map((_, index) => (
                  <Skeleton key={`month-${index}`} className="h-3 w-8" />
                ))}
                {Array.from({ length: 259 }).map((_, index) => (
                  <Skeleton key={`day-${index}`} className="size-3 rounded-[2px]" />
                ))}
              </div>
              <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <Skeleton className="h-4 w-56 max-w-full" />
                <Skeleton className="h-4 w-28" />
              </div>
            </div>
          </section>

          <section className="space-y-3">
            <Skeleton className="h-6 w-32" />
            <div className="flex flex-wrap gap-2">
              {[0, 1, 2].map((item) => (
                <Skeleton key={item} className="h-7 w-28 rounded-full" />
              ))}
            </div>
            <div className="overflow-hidden rounded-lg bg-white shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
              {[0, 1, 2, 3, 4].map((item) => (
                <div
                  key={item}
                  className={`grid grid-cols-[40px_minmax(0,1fr)_20px] items-center gap-3 px-3 py-3.5 sm:grid-cols-[44px_92px_minmax(0,1fr)_112px_24px] sm:gap-4 sm:px-4 ${
                    item > 0 ? 'border-t' : ''
                  }`}
                >
                  <Skeleton className="size-9 rounded-full" />
                  <Skeleton className="hidden h-4 sm:block" />
                  <div className="min-w-0 space-y-2">
                    <Skeleton className="h-4 w-full max-w-[360px]" />
                    <Skeleton className="h-3 w-44 max-w-full" />
                  </div>
                  <Skeleton className="hidden h-3 sm:block" />
                  <Skeleton className="size-4" />
                </div>
              ))}
            </div>
          </section>
        </section>

        <aside className="grid min-w-0 grid-cols-1 gap-5 lg:grid-cols-2 2xl:block 2xl:space-y-5">
          {[0, 1, 2, 3].map((panel) => (
            <section key={panel} className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
              <div className="mb-5 flex items-center justify-between gap-3">
                <Skeleton className="h-5 w-36" />
                <Skeleton className="h-3 w-24" />
              </div>
              <div className="grid grid-cols-2 gap-2.5">
                {[0, 1, 2, 3, 4, 5].map((tile) => (
                  <div key={tile} className="min-w-0 rounded-lg bg-white px-3.5 py-3 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
                    <Skeleton className="h-6 w-10" />
                    <Skeleton className="mt-2 h-3 w-full" />
                  </div>
                ))}
              </div>
            </section>
          ))}
        </aside>
      </div>
    </main>
  );
}
