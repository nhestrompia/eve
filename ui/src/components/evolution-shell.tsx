import type { ReactNode } from 'react';
import type { EvolutionSummary } from '../types';
import { EvolutionList } from './evolution-list';

export function EvolutionShell({
  evolutions,
  selectedId,
  children
}: {
  evolutions: EvolutionSummary[];
  selectedId?: string;
  children: ReactNode;
}) {
  return (
    <div className="grid grid-cols-[260px_minmax(0,1fr)]">
      <EvolutionList evolutions={evolutions} selectedId={selectedId} />
      <main className="min-h-[calc(100dvh-76px)] min-w-0 p-11">{children}</main>
    </div>
  );
}
