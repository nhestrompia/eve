import { Link } from '@tanstack/react-router';
import { ArrowRight } from 'lucide-react';
import { relationshipEntries } from '../lib/evolution-display';
import type { Evolution } from '../types';

type SnapshotRelationshipListProps = {
  relationships: Evolution['relationships'];
  emptyText?: string;
};

export function SnapshotRelationshipList({
  relationships,
  emptyText = 'No relationships are recorded in this Snapshot.'
}: SnapshotRelationshipListProps) {
  const entries = relationshipEntries(relationships);
  if (entries.length === 0) {
    return <p className="rounded-lg bg-secondary p-4 text-sm text-muted-foreground">{emptyText}</p>;
  }

  const groups = entries.reduce<Array<{ kind: string; label: string; values: string[] }>>((result, entry) => {
    const group = result.find((item) => item.kind === entry.kind);
    if (group) {
      group.values.push(entry.value);
      return result;
    }
    return [...result, { kind: entry.kind, label: entry.label, values: [entry.value] }];
  }, []);

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      {groups.map((group) => (
        <section key={group.kind} className="rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
          <div className="flex items-center justify-between gap-3">
            <h3 className="text-sm font-semibold">{group.label}</h3>
            <span className="rounded-md bg-secondary px-2 py-1 text-xs font-medium text-muted-foreground">
              {group.values.length}
            </span>
          </div>
          <div className="mt-3 grid gap-2">
            {group.values.map((value) => (
              <Link
                key={value}
                to="/snapshots/$id"
                params={{ id: value }}
                className="group flex min-w-0 items-center justify-between gap-3 rounded-md bg-secondary px-3 py-2 text-sm transition-[background-color,box-shadow] hover:bg-slate-50 hover:shadow-[0_0_0_1px_rgba(15,23,42,0.12)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                <span className="min-w-0 truncate font-mono font-semibold">{value}</span>
                <ArrowRight className="size-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
              </Link>
            ))}
          </div>
        </section>
      ))}
    </div>
  );
}
