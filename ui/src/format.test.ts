import { describe, expect, it } from 'vitest';
import { countBehavior, shortCommit, statusLabel } from './format';

describe('format helpers', () => {
  it('shortens snapshot commits without changing short refs', () => {
    expect(shortCommit('39e6f748ed5cc9a003c5d63b02ecd3cf69a5eb39')).toBe('39e6f748ed5c');
    expect(shortCommit('HEAD')).toBe('HEAD');
  });

  it('counts behavior claims across groups', () => {
    expect(
      countBehavior({
        added: [{ description: 'Add checkout' }],
        changed: [{ description: 'Update timeline' }],
        fixed: [{ description: 'Fix search' }]
      })
    ).toBe(3);
  });

  it('formats status labels', () => {
    expect(statusLabel('in_review')).toBe('in review');
  });
});
