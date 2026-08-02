import { compactDate } from "../../format";
import type { EvolutionSummary } from "../../types";

export function comparisonPairByCount(
  evolutions: EvolutionSummary[],
  count: number,
): { from: string; to: string } | undefined {
  if (evolutions.length < 2) return undefined;
  const size = Math.max(2, Math.min(count, evolutions.length));
  const to = evolutions[evolutions.length - 1];
  const from = evolutions[evolutions.length - size];
  return { from: from.id, to: to.id };
}

export function snapshotMatchesQuery(
  snapshot: EvolutionSummary,
  query: string,
): boolean {
  const haystack = [
    snapshot.title,
    snapshot.id,
    snapshot.type,
    snapshot.status,
    snapshot.repository,
    snapshot.createdAt,
    compactDate(snapshot.createdAt),
  ]
    .join(" ")
    .toLowerCase();
  return haystack.includes(query);
}
