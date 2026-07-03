import { Link } from '@tanstack/react-router';
import { Box, CalendarDays, Download, Tag, Users } from 'lucide-react';
import { useMutation } from '@tanstack/react-query';
import { api } from '../api';
import { humanDate, shortCommit } from '../format';
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
  const checkout = useMutation({ mutationFn: () => api.checkout(detail.summary.id) });
  const providers = detail.summary.sessionProviders.join(' & ') || 'EVE';

  return (
    <section className="grid grid-cols-[minmax(0,1fr)_260px] gap-8 py-14">
      <div className="min-w-0">
        <StatusBadge status={detail.summary.status} />
        <div className="mt-6 flex flex-wrap items-center gap-3">
          <h1 className="text-[34px] font-semibold leading-tight tracking-[-0.01em] text-balance">
            {detail.summary.title || 'Untitled Evolution'}
          </h1>
          <span className="rounded-lg bg-secondary px-3 py-1 font-mono text-lg font-semibold tabular-nums text-muted-foreground">
            #{detail.summary.id.replace('EV-', '')}
          </span>
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
            by {providers}
          </span>
          {detail.summary.snapshot ? (
            <span className="inline-flex items-center gap-2">
              <Tag className="size-4" />
              Snapshot {shortCommit(detail.summary.snapshot)}
            </span>
          ) : null}
        </div>
      </div>

      <div className="flex flex-col justify-center gap-3">
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
              <AlertDialogAction onClick={() => checkout.mutate()}>Run checkout</AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
        <Button asChild variant="outline" className="h-12 justify-start gap-3 rounded-lg pl-5">
          <Link to="/evolutions/$id/snapshot" params={{ id: detail.summary.id }}>
            <Box className="size-4" />
            View snapshot
          </Link>
        </Button>
        {checkout.error instanceof Error ? (
          <pre className="whitespace-pre-wrap rounded-lg bg-red-50 p-3 font-mono text-xs text-red-700">{checkout.error.message}</pre>
        ) : null}
        {checkout.data ? (
          <pre className="whitespace-pre-wrap rounded-lg bg-slate-950 p-3 font-mono text-xs text-white">
            {checkout.data.exitCode === 0 ? 'Product snapshot restored\n' : ''}
            {checkout.data.stdout}
            {checkout.data.stderr}
          </pre>
        ) : null}
      </div>
    </section>
  );
}
