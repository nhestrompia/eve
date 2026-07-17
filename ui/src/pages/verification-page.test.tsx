import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it } from 'vitest';
import type { Snapshot } from '../types';
import { AggregatePanel, TrustBoundary } from './verification-page';

describe('verification evidence presentation', () => {
  it('discloses the local tamper-evident trust boundary', () => {
    const html = renderToStaticMarkup(<TrustBoundary />);
    expect(html).toContain('Tamper-evident local evidence');
    expect(html).toContain('does not protect against an adversarial actor');
  });

  it('shows policy reductions alongside a passing aggregate', () => {
    const snapshot = {
      implementation: { gitState: 'abc123', branch: 'main', commits: ['abc123'], dirty: false },
      verification: {
        status: 'required_checks_passed',
        profile: 'release',
        integrity: 'matched',
        selectedRunId: 'run_123',
        requiredChecks: ['unit'],
        ranChecks: ['unit'],
        policyChange: { changed: true, requirementsReduced: true, removedChecks: ['e2e'] }
      }
    } as Snapshot;
    const html = renderToStaticMarkup(<AggregatePanel snapshot={snapshot} />);
    expect(html).toContain('Required checks passed');
    expect(html).toContain('requirements were reduced');
    expect(html).toContain('e2e');
  });
});
