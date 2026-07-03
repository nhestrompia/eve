import { Link } from '@tanstack/react-router';
import { Box, CalendarDays, Download, Tag, Users } from 'lucide-react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { api } from '../api';
import { humanDate } from '../format';
import type { DetailResponse, SnapshotResponse } from '../types';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger
} from './ui/alert-dialog';
import { Button } from './ui/button';
import { StatusBadge } from './status-badge';

export function EvolutionHero({ detail, snapshot }: { detail: DetailResponse; snapshot?: SnapshotResponse }) {
  const queryClient = useQueryClient();
  const checkout = useMutation({
    mutationFn: () => api.checkout(detail.summary.id),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['config'] });
      if (result.exitCode === 0) {
        toast.success('Snapshot checked out', {
          description: `${result.repository || 'Repository'} is now at ${result.commit.slice(0, 12)}.`
        });
        return;
      }
      toast.error('Checkout failed', {
        description: (result.stderr || result.stdout || 'EVE could not checkout this snapshot.').trim()
      });
    },
    onError: (error) => {
      toast.error('Checkout failed', {
        description: error instanceof Error ? error.message : 'EVE could not checkout this snapshot.'
      });
    }
  });
  const author = detailAuthor(detail);

  return (
    <section className="grid grid-cols-1 gap-6 py-8 sm:py-10 lg:grid-cols-[minmax(0,1fr)_260px] lg:gap-8 lg:py-14">
      <div className="min-w-0">
        <StatusBadge status={detail.summary.status} />
        <div className="mt-6">
          <h1 className="text-2xl font-semibold leading-tight tracking-[-0.01em] text-balance sm:text-[34px]">
            {detail.summary.title || 'Untitled Snapshot'}
          </h1>
        </div>
        <p className="mt-4 max-w-[68ch] text-[15px] leading-6 text-pretty">
          {detail.summary.outcome || detail.evolution.intent || 'No outcome recorded.'}
        </p>
        <div className="mt-6 flex flex-wrap items-center gap-x-5 gap-y-2 text-sm text-muted-foreground">
          <span className="inline-flex items-center gap-2">
            <CalendarDays className="size-4" />
            {humanDate(detail.summary.updatedAt || detail.summary.createdAt)}
          </span>
          <span className="inline-flex items-center gap-2">
            <Users className="size-4" />
            Recorded by {author}
          </span>
          <span className="inline-flex items-center gap-2">
            <Tag className="size-4" />
            Snapshot {detail.summary.id}
          </span>
        </div>
      </div>

      <div className="flex flex-col justify-center gap-3 sm:max-w-sm lg:max-w-none">
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button className="h-12 justify-start gap-3 rounded-lg bg-slate-950 pl-5 text-white shadow-[0_8px_18px_-14px_rgba(15,23,42,0.7)] hover:bg-slate-900">
              <Download className="size-4" />
              Checkout snapshot
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Checkout {detail.summary.id}?</AlertDialogTitle>
              <AlertDialogDescription>
                This runs <code className="font-mono">{snapshot?.checkoutCommand ?? `eve checkout ${detail.summary.id}`}</code>.
                EVE will refuse if the working tree is dirty.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction disabled={checkout.isPending} onClick={() => checkout.mutate()}>
                {checkout.isPending ? 'Checking out...' : 'Run checkout'}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
        <Button asChild variant="outline" className="h-12 justify-start gap-3 rounded-lg pl-5">
          <Link to="/snapshots/$id/snapshot" params={{ id: detail.summary.id }}>
            <Box className="size-4" />
            View snapshot
          </Link>
        </Button>
      </div>
    </section>
  );
}

function detailAuthor(detail: DetailResponse) {
  const providers = Array.from(
    new Set(
      [
        ...detail.sessions.map((session) => session.providerName || session.provider),
        ...detail.summary.sessionProviders
      ].filter(Boolean)
    )
  );
  if (providers.length > 0) return providers.join(' & ');
  return detail.commits.find((commit) => commit.authorName)?.authorName || 'Unknown author';
}
