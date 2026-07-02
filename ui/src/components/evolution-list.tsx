import { Link } from '@tanstack/react-router';
import { CheckCircle2, ListFilter } from 'lucide-react';
import { useMemo, useState } from 'react';
import { compactDate, monthYear } from '../format';
import type { EvolutionSummary } from '../types';
import { Button } from './ui/button';

export function EvolutionList({ evolutions, selectedId }: { evolutions: EvolutionSummary[]; selectedId?: string }) {
  const [ascending, setAscending] = useState(false);
  const sorted = useMemo(() => {
    return [...evolutions].sort((left, right) => (ascending ? left.id.localeCompare(right.id) : right.id.localeCompare(left.id)));
  }, [ascending, evolutions]);
  const groupLabel = monthYear(sorted[0]?.updatedAt || sorted[0]?.createdAt);

  return (
    <aside className="sticky top-[76px] h-[calc(100dvh-76px)] overflow-hidden border-r bg-white/72">
      <div className="flex h-16 items-center justify-between border-b px-7">
        <h2 className="font-semibold">{evolutions.length} {evolutions.length === 1 ? 'Evolution' : 'Evolutions'}</h2>
        <Button
          variant="ghost"
          size="icon"
          aria-label={ascending ? 'Show newest Evolutions first' : 'Show oldest Evolutions first'}
          title={ascending ? 'Newest first' : 'Oldest first'}
          aria-pressed={ascending}
          onClick={() => setAscending((value) => !value)}
        >
          <ListFilter className="size-4" />
        </Button>
      </div>
      <div className="h-[calc(100%-64px)] overflow-auto px-4 py-5">
        <p className="mb-4 px-3 text-xs font-medium text-muted-foreground">{groupLabel}</p>
        <div className="space-y-2">
          {sorted.map((evolution) => (
            <Link
              key={evolution.id}
              to="/evolutions/$id"
              params={{ id: evolution.id }}
              className={`grid grid-cols-[24px_minmax(0,1fr)_auto] items-center gap-3 rounded-lg px-3 py-4 ${
                selectedId === evolution.id ? 'bg-blue-50 shadow-sm ring-1 ring-blue-100' : 'hover:bg-slate-50'
              }`}
            >
              <CheckCircle2 className="size-4 text-emerald-600" />
              <span className="min-w-0">
                <span className="block truncate font-semibold">{evolution.title || 'Untitled Evolution'}</span>
                <span className="block text-sm text-muted-foreground">{compactDate(evolution.updatedAt || evolution.createdAt)}</span>
              </span>
              <span className="font-mono text-sm text-muted-foreground">#{evolution.id.replace('EV-', '')}</span>
            </Link>
          ))}
        </div>
      </div>
    </aside>
  );
}
