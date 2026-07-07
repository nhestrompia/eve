import type { ReactNode } from 'react';
import type { EvolutionSummary } from '../types';
import { cn } from '../lib/utils';
import { EvolutionList } from './evolution-list';

export function EvolutionShell({
  evolutions,
  selectedId,
  showHistoryRail = true,
  historyRailTarget = 'snapshot',
  showSelectedSnapshotLink = false,
  contentClassName,
  children
}: {
  evolutions: EvolutionSummary[];
  selectedId?: string;
  showHistoryRail?: boolean;
  historyRailTarget?: 'snapshot' | 'code';
  showSelectedSnapshotLink?: boolean;
  contentClassName?: string;
  children: ReactNode;
}) {
  return (
    <div className={showHistoryRail ? 'grid min-w-0 grid-cols-1 lg:grid-cols-[260px_minmax(0,1fr)]' : 'min-w-0'}>
      {showHistoryRail ? (
        <EvolutionList
          evolutions={evolutions}
          selectedId={selectedId}
          linkTarget={historyRailTarget}
          showSnapshotLink={showSelectedSnapshotLink}
        />
      ) : null}
      <main className={cn('min-h-[calc(100dvh-76px)] min-w-0 p-5 sm:p-7 lg:p-11', contentClassName)}>{children}</main>
    </div>
  );
}
