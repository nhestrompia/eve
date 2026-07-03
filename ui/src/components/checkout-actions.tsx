import { Copy, Download, Link as LinkIcon, MoreHorizontal, Terminal } from 'lucide-react';
import { useMutation } from '@tanstack/react-query';
import { useState } from 'react';
import { Link } from '@tanstack/react-router';
import { api } from '../api';
import type { CheckoutResponse, SnapshotResponse } from '../types';
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
  const checkout = useMutation({ mutationFn: () => api.checkout(snapshot.id) });

  const copy = async () => {
    await navigator.clipboard.writeText(snapshot.checkoutCommand);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  };

  const copyLink = async () => {
    await navigator.clipboard.writeText(`${window.location.origin}/evolutions/${snapshot.id}/snapshot`);
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
          <Link to="/evolutions/$id/implementation" params={{ id: snapshot.id }}>
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
            <AlertDialogAction onClick={() => checkout.mutate()}>Run checkout</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <Button variant="outline" className="h-12 w-full gap-3 sm:w-[250px]" onClick={copy}>
        <Terminal className="size-4" />
        {copied ? 'Command copied' : 'Copy command'}
      </Button>
      <CheckoutResult result={checkout.data} error={checkout.error} />
    </div>
  );
}

function CheckoutResult({ result, error }: { result?: CheckoutResponse; error: unknown }) {
  if (error instanceof Error) {
    return <pre className="w-full whitespace-pre-wrap rounded-lg border bg-red-50 p-3 font-mono text-xs text-red-700 sm:w-[250px]">{error.message}</pre>;
  }
  if (!result) return null;
  return (
    <pre className="w-full whitespace-pre-wrap rounded-lg border bg-slate-950 p-3 font-mono text-xs text-white sm:w-[250px]">
      {result.exitCode === 0 ? 'Product snapshot restored\n' : ''}
      {result.stdout}
      {result.stderr}
    </pre>
  );
}
