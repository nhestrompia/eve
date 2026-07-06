import { describe, expect, it } from 'vitest';
import { defaultComparisonPair } from './comparison';
import type { EvolutionSummary } from '../types';

const base: EvolutionSummary = {
  id: 'snap_base',
  repository: 'eve',
  title: 'Snapshot',
  type: 'feature',
  status: 'completed',
  outcome: 'Recorded.',
  snapshot: 'abc',
  commitCount: 1,
  decisionCount: 0,
  riskCount: 0,
  artifactCount: 0,
  failedValidationCount: 0,
  verificationState: 'passed',
  verificationSummary: 'passed',
  sessionProviders: [],
  createdAt: '2026-07-01T09:00:00Z',
  updatedAt: '2026-07-01T09:00:00Z'
};

describe('comparison helpers', () => {
  it('chooses the two newest snapshots from the same repository', () => {
    expect(
      defaultComparisonPair([
        { ...base, id: 'other_new', repository: 'other', createdAt: '2026-07-04T09:00:00Z' },
        { ...base, id: 'eve_old', createdAt: '2026-07-01T09:00:00Z' },
        { ...base, id: 'eve_new', createdAt: '2026-07-03T09:00:00Z' },
        { ...base, id: 'eve_middle', createdAt: '2026-07-02T09:00:00Z' }
      ])
    ).toEqual({ from: 'eve_middle', to: 'eve_new' });
  });

  it('returns undefined when no repository has two snapshots', () => {
    expect(defaultComparisonPair([{ ...base, id: 'only' }])).toBeUndefined();
  });
});
