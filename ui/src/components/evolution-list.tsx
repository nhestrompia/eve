import { Link } from '@tanstack/react-router';
import { CheckCircle2, ExternalLink, ListFilter } from 'lucide-react';
import { type Ref, useEffect, useMemo, useRef, useState } from 'react';
import { compactDate, monthYear } from '../format';
import type { EvolutionSummary } from '../types';
import { Button } from './ui/button';

export function EvolutionList({
  evolutions,
  selectedId,
  linkTarget = 'snapshot',
  showSnapshotLink = false
}: {
  evolutions: EvolutionSummary[];
  selectedId?: string;
  linkTarget?: 'snapshot' | 'code';
  showSnapshotLink?: boolean;
}) {
  const [ascending, setAscending] = useState(false);
  const selectedRef = useRef<HTMLAnchorElement | null>(null);
  const sorted = useMemo(() => {
    return [...evolutions].sort((left, right) => (ascending ? left.id.localeCompare(right.id) : right.id.localeCompare(left.id)));
  }, [ascending, evolutions]);
  const selected = selectedId ? evolutions.find((evolution) => evolution.id === selectedId) : undefined;
  const groupLabel = monthYear(sorted[0]?.updatedAt || sorted[0]?.createdAt);

  useEffect(() => {
    selectedRef.current?.scrollIntoView({ block: 'center' });
  }, [selectedId, sorted.length]);

  return (
    <aside className="border-b bg-white/72 lg:sticky lg:top-[76px] lg:h-[calc(100dvh-76px)] lg:overflow-hidden lg:border-b-0 lg:border-r">
      <div className="border-b px-5 py-4 lg:px-7">
        <div className="flex items-center justify-between gap-3">
          <h2 className="font-semibold">{evolutions.length} {evolutions.length === 1 ? 'Snapshot' : 'Snapshots'}</h2>
          <Button
            variant="ghost"
            size="icon"
            aria-label={ascending ? 'Show newest Snapshots first' : 'Show oldest Snapshots first'}
            title={ascending ? 'Newest first' : 'Oldest first'}
            aria-pressed={ascending}
            onClick={() => setAscending((value) => !value)}
          >
            <ListFilter className="size-4" />
          </Button>
        </div>
        {selected ? (
          <div className="mt-2 space-y-2">
            <p className="truncate text-xs text-muted-foreground">
              Selected: <span className="font-medium text-foreground">{selected.title || selected.id}</span>
            </p>
            {showSnapshotLink ? (
              <Link
                to="/snapshots/$id"
                params={{ id: selected.id }}
                className="inline-flex h-8 items-center gap-1.5 rounded-md border bg-card px-2.5 text-xs font-medium text-foreground transition-colors hover:bg-secondary"
              >
                <ExternalLink className="size-3.5" />
                Open Snapshot
              </Link>
            ) : null}
          </div>
        ) : null}
      </div>
      <div className="max-h-72 overflow-auto px-3 py-4 lg:h-[calc(100%-92px)] lg:max-h-none lg:px-4 lg:py-5">
        <p className="mb-4 px-3 text-xs font-medium text-muted-foreground">{groupLabel}</p>
        <div className="space-y-2">
          {sorted.map((evolution) => (
            <SnapshotListLink
              key={evolution.id}
              evolution={evolution}
              selected={selectedId === evolution.id}
              linkTarget={linkTarget}
              selectedRef={selectedId === evolution.id ? selectedRef : undefined}
            />
          ))}
        </div>
      </div>
    </aside>
  );
}

function SnapshotListLink({
  evolution,
  selected,
  linkTarget,
  selectedRef
}: {
  evolution: EvolutionSummary;
  selected: boolean;
  linkTarget: 'snapshot' | 'code';
  selectedRef?: Ref<HTMLAnchorElement>;
}) {
  const route = snapshotRailRouteForTarget(linkTarget);
  const className = `snapshot-list-link grid grid-cols-[24px_minmax(0,1fr)] items-center gap-3 rounded-lg px-3 py-4 ${
    selected ? 'is-active bg-blue-50 shadow-sm ring-1 ring-blue-100' : 'hover:bg-slate-50'
  }`;
  const content = (
    <>
      <CheckCircle2 className="size-4 text-emerald-600" />
      <span className="min-w-0">
        <span className="block truncate font-semibold">{evolution.title || 'Untitled Snapshot'}</span>
        <span className="block text-sm text-muted-foreground">{compactDate(evolution.updatedAt || evolution.createdAt)}</span>
      </span>
    </>
  );

  if (route === '/snapshots/$id/code') {
    return (
      <Link
        to="/snapshots/$id/code"
        params={{ id: evolution.id }}
        aria-current={selected ? 'page' : undefined}
        ref={selectedRef}
        className={className}
      >
        {content}
      </Link>
    );
  }

  return (
    <Link
      to="/snapshots/$id"
      params={{ id: evolution.id }}
      aria-current={selected ? 'page' : undefined}
      ref={selectedRef}
      className={className}
    >
      {content}
    </Link>
  );
}

export function snapshotRailRouteForTarget(linkTarget: 'snapshot' | 'code') {
  return linkTarget === 'code' ? '/snapshots/$id/code' : '/snapshots/$id';
}
