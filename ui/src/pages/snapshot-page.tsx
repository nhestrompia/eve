import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { GitCommitHorizontal } from 'lucide-react';
import { api } from '../api';
import { BehaviorCard } from '../components/behavior-card';
import { Button } from '../components/ui/button';
import { CheckoutActions } from '../components/checkout-actions';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '../components/ui/dialog';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { SnapshotTimeline } from '../components/snapshot-timeline';
import { VerificationCard } from '../components/verification-card';
import { compactDate } from '../format';
import type { GitCommit } from '../types';

export function SnapshotPage() {
  const { id } = useParams({ from: '/evolutions/$id/snapshot' });
  const evolutions = useQuery({ queryKey: ['evolutions'], queryFn: api.evolutions });
  const snapshot = useQuery({ queryKey: ['snapshot', id], queryFn: () => api.snapshot(id), retry: false });
  const detail = useQuery({ queryKey: ['evolution', id], queryFn: () => api.evolution(id), retry: false });

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
                  <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                    <div className="min-w-0">
                      <p className="text-sm text-muted-foreground">Snapshot commit</p>
                      <p className="mt-2 break-all font-mono text-xl font-semibold">{snapshot.data.commit}</p>
                    </div>
                    <SnapshotCommitsDialog commits={detail.data?.commits ?? []} loading={detail.isLoading} />
                  </div>
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

function SnapshotCommitsDialog({ commits, loading }: { commits: GitCommit[]; loading: boolean }) {
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="outline" className="w-fit shrink-0 gap-2">
          <GitCommitHorizontal className="size-4" />
          View commits
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-[720px]">
        <DialogHeader>
          <DialogTitle>Snapshot commits</DialogTitle>
          <DialogDescription>Commits recorded for this Evolution implementation.</DialogDescription>
        </DialogHeader>
        <div className="max-h-[54dvh] overflow-auto pr-1">
          {loading ? <p className="rounded-lg border bg-slate-50 p-4 text-sm text-muted-foreground">Loading commits...</p> : null}
          {!loading && commits.length === 0 ? (
            <p className="rounded-lg border bg-slate-50 p-4 text-sm text-muted-foreground">No contributed commits are recorded for this snapshot.</p>
          ) : null}
          <div className="space-y-3">
            {commits.map((commit) => (
              <article key={commit.hash} className="rounded-lg border bg-white p-4">
                <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0">
                    <p className="truncate font-semibold">{commit.subject || 'Commit'}</p>
                    <p className="mt-1 break-all font-mono text-xs text-muted-foreground">{commit.hash}</p>
                  </div>
                  <span className="w-fit rounded-md bg-secondary px-2 py-1 font-mono text-xs">{commit.shortHash}</span>
                </div>
                <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                  <span>{commit.authorName || 'Unknown author'}</span>
                  <span>{compactDate(commit.committedAt || commit.authoredAt)}</span>
                </div>
              </article>
            ))}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
