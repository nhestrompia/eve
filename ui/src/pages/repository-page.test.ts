import { describe, expect, it } from 'vitest';
import { repositoryTabs } from './repository-page';

describe('repository tabs', () => {
  it('includes code inspection after snapshots', () => {
    expect(repositoryTabs(46).map((tab) => tab.id)).toEqual([
      'overview',
      'snapshots',
      'code',
      'compare',
      'activity',
      'artifacts',
      'settings'
    ]);
  });
});
