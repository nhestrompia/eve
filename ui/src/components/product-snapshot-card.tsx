import { Copy, ImageIcon } from 'lucide-react';
import { useState } from 'react';
import type { DetailResponse, SnapshotResponse } from '../types';
import { shortCommit } from '../format';
import { SectionHeading } from './section-heading';
import { Button } from './ui/button';
import { Card, CardContent, CardFooter, CardHeader } from './ui/card';

export function ProductSnapshotCard({ detail, snapshot }: { detail: DetailResponse; snapshot?: SnapshotResponse }) {
  const commit = snapshot?.commit ?? detail.summary.snapshot;
  const [copied, setCopied] = useState(false);

  const copyCommit = async () => {
    if (!commit) return;
    await navigator.clipboard.writeText(commit);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  };

  return (
    <Card>
      <CardHeader>
        <SectionHeading icon={ImageIcon} title="Product Snapshot" />
      </CardHeader>
      <CardContent className="space-y-5">
        <dl className="grid grid-cols-[96px_minmax(0,1fr)] gap-4 text-sm">
          <dt className="text-muted-foreground">Snapshot</dt>
          <dd className="flex min-w-0 items-center gap-2">
            <code className="max-w-[94px] truncate rounded-md bg-slate-100 px-2 py-1 font-mono text-xs">{shortCommit(commit)}</code>
            <Button
              variant="ghost"
              size="icon"
              className="size-7 shrink-0"
              aria-label={copied ? 'Snapshot hash copied' : 'Copy snapshot hash'}
              title={copied ? 'Copied' : 'Copy snapshot hash'}
              onClick={copyCommit}
            >
              <Copy className="size-3.5" />
            </Button>
          </dd>
          <dt className="text-muted-foreground">Repository</dt>
          <dd className="font-semibold">{snapshot?.repository ?? Object.keys(detail.evolution.implementation.repositories ?? {})[0] ?? 'eve'}</dd>
          <dt className="text-muted-foreground">Committed</dt>
          <dd>{detail.summary.updatedAt ? new Date(detail.summary.updatedAt).toLocaleString() : 'Unknown'}</dd>
        </dl>
      </CardContent>
      <CardFooter className="border-t bg-slate-50/70 text-xs text-muted-foreground">
        <span className="mr-3 size-2 rounded-full bg-emerald-500" />
        Snapshot commit recorded in <span className="ml-1 font-mono">implementation.snapshot</span>.
      </CardFooter>
    </Card>
  );
}
