import type { EvolutionSummary } from '../types';

export function defaultComparisonPair(evolutions: EvolutionSummary[]) {
  const byRepository = new Map<string, EvolutionSummary[]>();
  for (const evolution of evolutions) {
    const repository = evolution.repository || '';
    const rows = byRepository.get(repository) ?? [];
    rows.push(evolution);
    byRepository.set(repository, rows);
  }

  for (const rows of byRepository.values()) {
    const sorted = [...rows].sort((left, right) => right.createdAt.localeCompare(left.createdAt));
    if (sorted.length >= 2) {
      return { from: sorted[1].id, to: sorted[0].id };
    }
  }

  return undefined;
}
