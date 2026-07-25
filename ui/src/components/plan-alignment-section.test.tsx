import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type { PlanRecord, Snapshot } from '../types';
import { PlanAlignmentSection } from './plan-alignment-section';

const plan: PlanRecord = {
  id: 'plan_123',
  schemaVersion: '0.1.0',
  planRequestId: 'planreq_12345678',
  repository: 'eve',
  status: 'fulfilled',
  lockedRevision: 2,
  lockedAt: '2026-07-24T00:00:00Z',
  approvedBy: 'local_ui',
  revisions: [{
    revision: 2,
    source: 'human',
    goal: 'Add a durable plan gate',
    acceptanceCriteria: '- Requests resume',
    allowedPathGlobs: ['cmd/**'],
    milestones: [],
    resolvedCheckIds: ['unit'],
    policyHash: 'sha256:policy',
    checkDefinitionsHash: 'sha256:checks',
    suiteDigest: 'sha256:checks',
    baseCommit: 'abc',
    branch: 'main',
    createdAt: '2026-07-24T00:00:00Z'
  }]
};

function snapshot(status: 'matched' | 'failed' | 'incomplete' | 'no_plan', paths: string[] = []) {
  return {
    plan: status === 'no_plan' ? undefined : { id: plan.id, revision: 2 },
    planConformance: {
      status,
      noPlanOnFile: status === 'no_plan',
      requiredChecksStatus: status === 'matched' ? 'passed' : 'incomplete',
      policyMatched: status === 'matched',
      checkDefinitionsMatch: status === 'matched',
      scopeDrift: paths.length > 0,
      changedPaths: paths,
      outOfScopePaths: paths
    }
  } as Snapshot;
}

describe('Plan Alignment', () => {
  it('shows a matched locked revision and its declared scope', () => {
    const html = renderToStaticMarkup(<PlanAlignmentSection snapshot={snapshot('matched')} plan={plan} />);
    expect(html).toContain('Implementation matches the locked plan');
    expect(html).toContain('cmd/**');
    expect(html).toContain('unit');
  });

  it('shows policy/check failure and offending paths', () => {
    const html = renderToStaticMarkup(<PlanAlignmentSection snapshot={snapshot('failed', ['secret.txt'])} plan={plan} />);
    expect(html).toContain('Plan conformance failed');
    expect(html).toContain('Out-of-scope paths');
    expect(html).toContain('secret.txt');
  });

  it('shows incomplete evidence', () => {
    const html = renderToStaticMarkup(<PlanAlignmentSection snapshot={snapshot('incomplete')} plan={plan} />);
    expect(html).toContain('Plan evidence is incomplete');
  });

  it('shows a prominent unplanned-work warning', () => {
    const html = renderToStaticMarkup(<PlanAlignmentSection snapshot={snapshot('no_plan')} />);
    expect(html).toContain('Unplanned work');
    expect(html).toContain('without a valid locked Plan reference');
  });
});
