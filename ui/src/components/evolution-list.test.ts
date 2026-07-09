import { describe, expect, it } from 'vitest';
import { snapshotRailRouteForTarget } from './evolution-list';

describe('snapshot rail routing', () => {
  it('keeps snapshot selection in code view when the rail targets code', () => {
    expect(snapshotRailRouteForTarget('code')).toBe('/snapshots/$id/code');
    expect(snapshotRailRouteForTarget('snapshot')).toBe('/snapshots/$id');
  });
});
