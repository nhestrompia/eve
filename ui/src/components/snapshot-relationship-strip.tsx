import { Link } from '@tanstack/react-router';
import { ArrowRight, GitFork } from 'lucide-react';
import { relationshipEntries } from '../lib/evolution-display';
import { cn } from '../lib/utils';
import type { Evolution } from '../types';

type SnapshotRelationshipStripProps = {
  relationships: Evolution['relationships'];
  snapshotId?: string;
  maxItems?: number;
  className?: string;
};

export function SnapshotRelationshipStrip({
  relationships,
  snapshotId,
  maxItems = 3,
  className
}: SnapshotRelationshipStripProps) {
  const entries = relationshipEntries(relationships);
  if (entries.length === 0) return null;

  const visibleEntries = entries.slice(0, maxItems);
  const hiddenCount = entries.length - visibleEntries.length;

  return (
    <aside
      className={cn(
        'flex flex-col gap-3 rounded-lg border bg-secondary/70 p-3 text-sm sm:flex-row sm:items-center sm:justify-between',
        className
      )}
      aria-label="Snapshot relationships"
    >
      <div className="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-center">
        <div className="flex shrink-0 items-center gap-2 text-muted-foreground">
          <span className="flex size-7 items-center justify-center rounded-md bg-white text-slate-600 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
            <GitFork className="size-3.5" />
          </span>
          <span className="font-medium">Connected states</span>
        </div>
        <ul className="flex min-w-0 flex-wrap gap-2">
          {visibleEntries.map((entry) => (
            <li key={`${entry.kind}-${entry.value}`}>
              <Link
                to="/snapshots/$id"
                params={{ id: entry.value }}
                className="group inline-flex max-w-full items-center gap-1.5 rounded-md bg-white px-2.5 py-1.5 text-xs shadow-[0_0_0_1px_rgba(15,23,42,0.08)] transition-[background-color,box-shadow] hover:bg-slate-50 hover:shadow-[0_0_0_1px_rgba(15,23,42,0.18)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                title={`${entry.label} ${entry.value}`}
              >
                <span className="shrink-0 font-medium text-muted-foreground">{entry.label}</span>
                <span className="min-w-0 truncate font-mono font-semibold text-slate-800 group-hover:text-slate-950">{entry.value}</span>
              </Link>
            </li>
          ))}
        </ul>
      </div>

      {snapshotId && hiddenCount > 0 ? (
        <Link
          to="/snapshots/$id/relationships"
          params={{ id: snapshotId }}
          className="inline-flex w-fit shrink-0 items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-blue-700 transition-colors hover:bg-blue-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          +{hiddenCount} more
          <ArrowRight className="size-3.5" />
        </Link>
      ) : null}
    </aside>
  );
}
