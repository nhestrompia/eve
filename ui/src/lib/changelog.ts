import type { EvolutionSummary } from '../types';
import { compareEvolutionOrder } from './comparison';

export type ChangelogGroup = {
  title: string;
  items: Array<{
    snapshotId: string;
    snapshotTitle: string;
    text: string;
    createdAt: string;
  }>;
};

const changelogGroupOrder = ['Features', 'Improvements', 'Fixes', 'Other'];

export function changelogCandidatesForSnapshot(
  evolutions: EvolutionSummary[],
  current: EvolutionSummary,
) {
  const currentRepository = current.repository ?? '';
  return [...evolutions]
    .filter((evolution) => {
      const sameRepository = (evolution.repository ?? '') === currentRepository;
      return sameRepository && compareEvolutionOrder(evolution, current) <= 0;
    })
    .sort(compareEvolutionOrder);
}

export function changelogGroupTitle(snapshotType: string) {
  switch (snapshotType) {
    case 'feature':
      return 'Features';
    case 'bugfix':
      return 'Fixes';
    case 'refactor':
    case 'experiment':
      return 'Improvements';
    default:
      return 'Other';
  }
}

export function snapshotChangelogText(snapshot: EvolutionSummary) {
  return (snapshot.userVisibleChange || snapshot.title).trim();
}

export function buildChangelogGroups(snapshots: EvolutionSummary[]) {
  const grouped = new Map<string, ChangelogGroup['items']>();
  for (const snapshot of snapshots) {
    const title = changelogGroupTitle(snapshot.type);
    const items = grouped.get(title) ?? [];
    items.push({
      snapshotId: snapshot.id,
      snapshotTitle: snapshot.title,
      text: snapshotChangelogText(snapshot),
      createdAt: snapshot.createdAt,
    });
    grouped.set(title, items);
  }

  return changelogGroupOrder.flatMap((title) => {
    const items = grouped.get(title) ?? [];
    return items.length > 0 ? [{ title, items }] : [];
  });
}

export function formatChangelogMarkdown(groups: ChangelogGroup[]) {
  if (groups.length === 0) return 'No snapshot changes found.';
  return [
    '# Release Notes',
    ...groups.flatMap((group) => [
      '',
      `## ${group.title}`,
      ...group.items.map((item) => `- ${item.text}`),
    ]),
  ].join('\n');
}
