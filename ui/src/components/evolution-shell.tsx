import type { ReactNode } from 'react';
import type { EvolutionSummary } from '../types';
import { cn } from '../lib/utils';
import { EvolutionList } from './evolution-list';

export function EvolutionShell({
  evolutions,
  selectedId,
  showHistoryRail = true,
  contentClassName,
  children
}: {
  evolutions: EvolutionSummary[];
  selectedId?: string;
  showHistoryRail?: boolean;
  contentClassName?: string;
  children: ReactNode;
}) {
  return (
    <div className={showHistoryRail ? 'grid grid-cols-[260px_minmax(0,1fr)]' : 'min-w-0'}>
      {showHistoryRail ? <EvolutionList evolutions={evolutions} selectedId={selectedId} /> : null}
      <main className={cn('min-h-[calc(100dvh-76px)] min-w-0 p-11', contentClassName)}>{children}</main>
    </div>
  );
}
