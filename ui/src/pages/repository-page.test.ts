import { describe, expect, it } from 'vitest';
import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import type { RepositorySummary } from '../types';
import { RepositoryHeaderMark, repositoryInitial, repositoryTabs } from './repository-page';

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
