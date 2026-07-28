import { describe, expect, it } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import type { PullRequestSummary, RepositorySummary } from '../types';
import {
  hasExactHeadSnapshot,
  partitionOpenPullRequests,
  pullRequestBannerSignals,
  pullRequestDisclosureCopy,
  recentOpenPullRequest,
  RepositoryHeaderMark,
  repositoryInitial,
  repositoryTabs,
} from './repository-page';

describe('EVE pull request context', () => {
  it('keeps exact-head Snapshot PRs visible and partitions the rest', () => {
    const contextual = {
      number: 32,
      state: 'open',
      snapshotId: 'snap_32',
      snapshotHeadMatch: true,
    } as PullRequestSummary;
    const stale = {
      number: 31,
      state: 'open',
      snapshotId: 'snap_31',
      snapshotHeadMatch: false,
    } as PullRequestSummary;
    const unlinked = {
      number: 19,
      state: 'open',
      snapshotHeadMatch: false,
    } as PullRequestSummary;

    expect(hasExactHeadSnapshot(contextual)).toBe(true);
    expect(hasExactHeadSnapshot(stale)).toBe(false);
    expect(partitionOpenPullRequests([contextual, stale, unlinked])).toEqual({
      contextual: [contextual],
      hidden: [stale, unlinked],
    });
  });

  it('describes hidden and expanded open PR counts truthfully', () => {
    expect(pullRequestDisclosureCopy(13, 14, false, false)).toEqual({
      label: 'Show all 14 open pull requests',
      detail: '13 are hidden because they do not have current EVE context.',
    });
    expect(pullRequestDisclosureCopy(13, 14, false, true)).toEqual({
      label: 'Hide 13 other pull requests',
      detail: '13 shown for reference; they do not have current EVE context.',
    });
    expect(pullRequestDisclosureCopy(49, 88, true, false).label).toBe(
      'Show 49 other loaded pull requests',
    );
    expect(pullRequestDisclosureCopy(1, 1, false, false).label).toBe(
      'Show 1 open pull request',
    );
  });
});

describe('pullRequestBannerSignals', () => {
  it('shows one Snapshot blocker and leaves scope unevaluated without evidence', () => {
    const signals = pullRequestBannerSignals({
      snapshotId: undefined,
      snapshotHeadMatch: false,
      checksPassed: 4,
      checksTotal: 4,
      checksFailed: 0,
      checksPending: 0,
      scopeDrift: false,
      githubReady: true,
      readyToMerge: false,
    } as PullRequestSummary);

    expect(signals.filter((signal) => signal.label.includes('Snapshot'))).toHaveLength(1);
    expect(signals).toContainEqual({
      label: 'Scope not evaluated',
      detail: 'Requires exact-head product evidence',
      tone: 'neutral',
    });
    expect(signals.map((signal) => signal.label)).not.toContain('No scope drift');
    expect(signals.at(-1)?.label).toBe('GitHub permits merge');
  });

  it('reports scope positively only for an exact-head Snapshot', () => {
    const signals = pullRequestBannerSignals({
      snapshotId: 'EV-032',
      snapshotHeadMatch: true,
      checksPassed: 4,
      checksTotal: 4,
      checksFailed: 0,
      checksPending: 0,
      scopeDrift: false,
      planValid: true,
      planAligned: true,
      githubReady: true,
      readyToMerge: true,
    } as PullRequestSummary);

    expect(signals[0]?.label).toBe('Snapshot linked');
    expect(signals[2]).toMatchObject({
      label: 'Within plan scope',
      tone: 'success',
    });
    expect(signals[3]?.label).toBe('GitHub permits merge');
  });

  it('does not call failed or missing GitHub checks successful', () => {
    const failed = pullRequestBannerSignals({
      checksPassed: 3,
      checksTotal: 4,
      checksFailed: 1,
      checksPending: 0,
    } as PullRequestSummary);
    const missing = pullRequestBannerSignals({
      checksPassed: 0,
      checksTotal: 0,
      checksFailed: 0,
      checksPending: 0,
    } as PullRequestSummary);

    expect(failed[1]).toEqual({
      label: '1 check failed',
      detail: '3/4 GitHub checks passed',
      tone: 'warning',
    });
    expect(missing[1]).toEqual({
      label: 'No GitHub checks',
      detail: 'No check runs have been reported',
      tone: 'neutral',
    });
  });
});

describe('repository tabs', () => {
  it('includes code inspection after snapshots', () => {
    expect(repositoryTabs(46, 2)).toEqual([
      { id: 'overview', label: 'Overview' },
      { id: 'snapshots', label: 'Snapshots', count: 46 },
      { id: 'pull-requests', label: 'Pull requests', count: 2 },
      { id: 'code', label: 'Code' },
      { id: 'compare', label: 'Compare' },
      { id: 'activity', label: 'Activity' },
      { id: 'artifacts', label: 'Artifacts' }
    ]);
    expect(repositoryTabs(46, 2).map((tab) => tab.id)).toEqual([
      'overview',
      'snapshots',
      'pull-requests',
      'code',
      'compare',
      'activity',
      'artifacts'
    ]);
  });
});

describe('recentOpenPullRequest', () => {
  it('uses opened time and ignores older recently-updated pull requests', () => {
    const now = Date.parse('2026-07-28T12:00:00Z');
    const result = recentOpenPullRequest(
      [
        {
          number: 1,
          state: 'open',
          snapshotId: 'snap_1',
          snapshotHeadMatch: true,
          createdAt: '2026-06-01T00:00:00Z',
          updatedAt: '2026-07-28T11:59:00Z',
        },
        {
          number: 2,
          state: 'open',
          snapshotId: 'snap_2',
          snapshotHeadMatch: true,
          createdAt: '2026-07-27T00:00:00Z',
          updatedAt: '2026-07-27T01:00:00Z',
        },
        {
          number: 3,
          state: 'open',
          snapshotHeadMatch: false,
          createdAt: '2026-07-28T00:00:00Z',
          updatedAt: '2026-07-28T01:00:00Z',
        },
      ] as PullRequestSummary[],
      now,
    );

    expect(result?.number).toBe(2);
  });
});

describe('repositoryInitial', () => {
  it('uses the first readable character from repository names', () => {
    expect(repositoryInitial('chart-performance-demo')).toBe('C');
    expect(repositoryInitial('  _mobile-app')).toBe('M');
    expect(repositoryInitial('2026-reports')).toBe('2');
  });

  it('falls back when no readable initial exists', () => {
    expect(repositoryInitial('')).toBe('?');
    expect(repositoryInitial('---')).toBe('?');
  });
});

describe('RepositoryHeaderMark', () => {
  it('keeps the eve wordmark for the eve repository', () => {
    const html = renderToStaticMarkup(
      React.createElement(RepositoryHeaderMark, { repository: repository('eve') }),
    );

    expect(html).toContain('src="/eve.svg"');
  });

  it('uses an initial instead of the eve wordmark for unrelated repositories', () => {
    const html = renderToStaticMarkup(
      React.createElement(RepositoryHeaderMark, { repository: repository('chart-performance-demo') }),
    );

    expect(html).not.toContain('src="/eve.svg"');
    expect(html).toContain('C');
    expect(html).toContain('chart-performance-demo repository mark');
  });
});

function repository(name: string): RepositorySummary {
  return {
    name,
    evolutionCount: 0,
    snapshotCount: 0,
    commitCount: 0,
    latestAt: '',
    latestEvolution: '',
    latestTitle: '',
    sessionProviders: [],
  };
}
