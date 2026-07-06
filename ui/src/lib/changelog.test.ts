import { describe, expect, it } from 'vitest';
import {
  buildChangelogGroups,
  changelogCandidatesForSnapshot,
  formatChangelogMarkdown,
  snapshotChangelogText,
} from './changelog';
import type { EvolutionSummary } from '../types';

const base: EvolutionSummary = {
  id: 'snap_1',
  repository: 'eve',
  title: 'Add login',
  type: 'feature',
  status: 'completed',
  outcome: 'Added login.',
  snapshot: 'abc',
  commitCount: 1,
  decisionCount: 0,
  riskCount: 0,
  artifactCount: 0,
  failedValidationCount: 0,
  verificationState: 'passed',
  verificationSummary: 'passed',
  sessionProviders: [],
  createdAt: '2026-07-01T00:00:00Z',
  updatedAt: '2026-07-01T00:00:00Z',
};

describe('changelog helpers', () => {
  it('prefers user-visible change text', () => {
    expect(
      snapshotChangelogText({
        ...base,
        userVisibleChange: 'Users can sign in with GitHub.',
      }),
    ).toBe('Users can sign in with GitHub.');
    expect(snapshotChangelogText(base)).toBe('Add login');
  });

  it('groups snapshots using CLI changelog sections', () => {
    const groups = buildChangelogGroups([
      { ...base, id: 'snap_feature', type: 'feature', userVisibleChange: 'Added password login.' },
      { ...base, id: 'snap_refactor', type: 'refactor', title: 'Simplify auth routing' },
      { ...base, id: 'snap_bug', type: 'bugfix', title: 'Fixed redirect loop.' },
      { ...base, id: 'snap_release', type: 'release', title: 'Release prep' },
    ]);

    expect(groups.map((group) => group.title)).toEqual([
      'Features',
      'Improvements',
      'Fixes',
      'Other',
    ]);
    expect(formatChangelogMarkdown(groups)).toContain('- Added password login.');
    expect(formatChangelogMarkdown(groups)).toContain('## Fixes');
  });

  it('selects same-repository candidates through the current snapshot', () => {
    const candidates = changelogCandidatesForSnapshot(
      [
        { ...base, id: 'snap_3', createdAt: '2026-07-03T00:00:00Z' },
        { ...base, id: 'other', repository: 'other', createdAt: '2026-07-02T00:00:00Z' },
        { ...base, id: 'snap_1', createdAt: '2026-07-01T00:00:00Z' },
        { ...base, id: 'snap_2', createdAt: '2026-07-02T00:00:00Z' },
      ],
      { ...base, id: 'snap_2', createdAt: '2026-07-02T00:00:00Z' },
    );

    expect(candidates.map((candidate) => candidate.id)).toEqual(['snap_1', 'snap_2']);
  });
});
