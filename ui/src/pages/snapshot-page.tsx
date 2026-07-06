import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { GitCommitHorizontal, ImageIcon, Maximize2 } from 'lucide-react';
import { useState } from 'react';
import { api } from '../api';
import { BehaviorCard } from '../components/behavior-card';
import { Button } from '../components/ui/button';
import { CheckoutActions } from '../components/checkout-actions';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '../components/ui/dialog';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { SnapshotRelationshipStrip } from '../components/snapshot-relationship-strip';
import { SnapshotTimeline } from '../components/snapshot-timeline';
import { VerificationCard } from '../components/verification-card';
import { compactDate } from '../format';
import type { GitCommit, SnapshotImage } from '../types';

export function SnapshotPage() {
  const { id } = useParams({ from: '/snapshots/$id/snapshot' });
  const evolutions = useQuery({ queryKey: ['snapshots'], queryFn: api.snapshots });
  const snapshot = useQuery({ queryKey: ['snapshot', id], queryFn: () => api.snapshot(id), retry: false });
  const detail = useQuery({ queryKey: ['snapshot-detail', id], queryFn: () => api.snapshotDetail(id), retry: false });

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
                {detail.data ? (
                  <SnapshotRelationshipStrip
                    relationships={detail.data.evolution.relationships}
                    snapshotId={snapshot.data.id}
                    className="mt-5"
                  />
                ) : null}
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
            {snapshot.data.snapshotImages.length > 0 ? <SnapshotImagesSection images={snapshot.data.snapshotImages} /> : null}
            <section className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <BehaviorCard behavior={snapshot.data.behavior} />
              <VerificationCard values={snapshot.data.verification} evolutionId={snapshot.data.id} showLink={false} />
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

function SnapshotImagesSection({ images }: { images: SnapshotImage[] }) {
  const [selectedImage, setSelectedImage] = useState<SnapshotImage | null>(null);

  return (
    <section className="rounded-lg border bg-white p-5 sm:p-6">
      <div className="mb-5 flex items-center gap-2">
        <ImageIcon className="size-4 text-blue-600" />
        <h2 className="text-lg font-semibold">Snapshots</h2>
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {images.map((image) => (
          <button
            type="button"
            key={image.id}
            onClick={() => setSelectedImage(image)}
            className="group overflow-hidden rounded-lg border bg-slate-50 text-left transition-colors hover:bg-slate-100"
          >
            <span className="block aspect-video overflow-hidden bg-white">
              <img src={image.url} alt={image.title || 'Snapshot image'} loading="lazy" className="h-full w-full object-contain" />
            </span>
            <span className="flex items-start justify-between gap-3 border-t bg-white px-4 py-3">
              <span className="min-w-0">
                <span className="block truncate text-sm font-semibold">{image.title || image.id}</span>
                {image.source ? <span className="mt-1 block truncate text-xs text-muted-foreground">{image.source}</span> : null}
              </span>
              <Maximize2 className="mt-0.5 size-4 shrink-0 text-slate-500 transition-colors group-hover:text-slate-950" />
            </span>
          </button>
        ))}
      </div>
      <Dialog open={Boolean(selectedImage)} onOpenChange={(open) => !open && setSelectedImage(null)}>
        <DialogContent className="max-w-[min(1080px,calc(100vw-24px))] p-0">
          {selectedImage ? (
            <div>
              <DialogHeader className="border-b px-5 py-4">
                <DialogTitle className="text-base">{selectedImage.title || selectedImage.id}</DialogTitle>
                <DialogDescription className="truncate text-xs">{selectedImage.source}</DialogDescription>
              </DialogHeader>
              <div className="max-h-[78dvh] overflow-auto bg-slate-950 p-3">
                <img
                  src={selectedImage.url}
                  alt={selectedImage.title || 'Snapshot image'}
                  className="mx-auto max-h-[72dvh] w-auto max-w-full rounded-md object-contain"
                />
              </div>
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </section>
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
          <DialogDescription>Commits recorded for this Snapshot implementation.</DialogDescription>
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
