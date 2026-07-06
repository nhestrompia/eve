import type { EvolutionSummary } from '../types';

type ComparisonPair = {
  from: string;
  to: string;
};

export function defaultComparisonPair(evolutions: EvolutionSummary[]) {
  const byRepository = new Map<string, EvolutionSummary[]>();
  for (const evolution of evolutions) {
    const repository = evolution.repository || '';
    const rows = byRepository.get(repository) ?? [];
    rows.push(evolution);
    byRepository.set(repository, rows);
  }

  for (const rows of byRepository.values()) {
    const sorted = [...rows].sort((left, right) =>
      right.createdAt.localeCompare(left.createdAt),
    );
    if (sorted.length >= 2) {
      return { from: sorted[1].id, to: sorted[0].id };
    }
  }

  return undefined;
}

export function orderedComparisonPair(
  evolutions: EvolutionSummary[],
  firstId: string,
  secondId: string,
): ComparisonPair | undefined {
  if (!firstId || !secondId || firstId === secondId) return undefined;
  const first = evolutions.find((evolution) => evolution.id === firstId);
  const second = evolutions.find((evolution) => evolution.id === secondId);
  if (!first || !second) return undefined;

  return compareEvolutionOrder(first, second) <= 0
    ? { from: first.id, to: second.id }
    : { from: second.id, to: first.id };
}

export function updateComparisonRange(
  evolutions: EvolutionSummary[],
  current: { from?: string; to?: string },
  clickedId: string,
): ComparisonPair | { from: string; to: '' } | undefined {
  const clicked = evolutions.find((evolution) => evolution.id === clickedId);
  if (!clicked) return undefined;

  if (!current.from || !current.to || current.from === current.to) {
    if (current.from && current.from !== clickedId) {
      return orderedComparisonPair(evolutions, current.from, clickedId);
    }
    if (current.to && current.to !== clickedId) {
      return orderedComparisonPair(evolutions, current.to, clickedId);
    }
    return { from: clickedId, to: '' };
  }

  const ordered = [...evolutions].sort(compareEvolutionOrder);
  const fromIndex = ordered.findIndex(
    (evolution) => evolution.id === current.from,
  );
  const toIndex = ordered.findIndex((evolution) => evolution.id === current.to);
  const clickedIndex = ordered.findIndex(
    (evolution) => evolution.id === clickedId,
  );
  if (fromIndex < 0 || toIndex < 0 || clickedIndex < 0) {
    return { from: clickedId, to: '' };
  }
  if (clickedId === current.from || clickedId === current.to) {
    return { from: current.from, to: current.to };
  }
  if (clickedIndex < fromIndex) return { from: clickedId, to: current.to };
  if (clickedIndex > toIndex) return { from: current.from, to: clickedId };

  const distanceFromStart = clickedIndex - fromIndex;
  const distanceFromEnd = toIndex - clickedIndex;
  if (distanceFromStart <= distanceFromEnd) {
    return orderedComparisonPair(evolutions, clickedId, current.to);
  }
  return orderedComparisonPair(evolutions, current.from, clickedId);
}

export function compareEvolutionOrder(
  left: EvolutionSummary,
  right: EvolutionSummary,
) {
  const timeComparison = left.createdAt.localeCompare(right.createdAt);
  if (timeComparison !== 0) return timeComparison;
  return left.id.localeCompare(right.id);
}
