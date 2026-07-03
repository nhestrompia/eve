import { Copy, Download, Link as LinkIcon, MoreHorizontal, Terminal } from 'lucide-react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { toast } from 'sonner';
import { api } from '../api';
import type { SnapshotResponse } from '../types';
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

export function CheckoutActions({ snapshot }: { snapshot: SnapshotResponse }) {
  const [copied, setCopied] = useState(false);
  const [linkCopied, setLinkCopied] = useState(false);
  const queryClient = useQueryClient();
  const checkout = useMutation({
    mutationFn: () => api.checkout(snapshot.id),
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

  const copy = async () => {
    await navigator.clipboard.writeText(snapshot.checkoutCommand);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  };

  const copyLink = async () => {
    await navigator.clipboard.writeText(`${window.location.origin}/snapshots/${snapshot.id}/snapshot`);
    setLinkCopied(true);
    window.setTimeout(() => setLinkCopied(false), 1400);
  };

  return (
    <div className="flex w-full flex-col items-stretch gap-4 sm:w-auto sm:items-end">
      <div className="flex justify-end gap-3">
        <Button variant="outline" className="gap-2" onClick={copyLink}>
          <LinkIcon className="size-4" />
          {linkCopied ? 'Copied' : 'Copy link'}
        </Button>
        <Button asChild variant="outline" size="icon" aria-label="View implementation">
          <Link to="/snapshots/$id/implementation" params={{ id: snapshot.id }}>
            <MoreHorizontal className="size-4" />
          </Link>
        </Button>
      </div>
      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button className="h-12 w-full gap-3 bg-slate-950 text-white hover:bg-slate-900 sm:w-[250px]">
            <Download className="size-4" />
            Checkout snapshot
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Checkout {snapshot.id}?</AlertDialogTitle>
            <AlertDialogDescription>
              This runs <code className="font-mono">{snapshot.checkoutCommand}</code>. EVE will refuse if the working tree is dirty.
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
      <Button variant="outline" className="h-12 w-full gap-3 sm:w-[250px]" onClick={copy}>
        <Terminal className="size-4" />
        {copied ? 'Command copied' : 'Copy command'}
      </Button>
    </div>
  );
}
