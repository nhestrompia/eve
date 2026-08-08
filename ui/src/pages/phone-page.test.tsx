// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import type { ComponentProps, ReactNode } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { api } from '../api';
import type { PlanRequest } from '../types';
import { PhonePage } from './phone-page';

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual<typeof import('@tanstack/react-router')>('@tanstack/react-router');
  return {
    ...actual,
    Link: ({ children, to, params, ...props }: ComponentProps<'a'> & { children: ReactNode; to: string; params?: Record<string, string> }) => (
      <a href={params ? to.replace('$planRequestId', params.planRequestId) : to} {...props}>{children}</a>
    )
  };
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

function renderPhonePage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}><PhonePage /></QueryClientProvider>);
}

describe('PhonePage', () => {
  it('gives the exact Mac setup command when the feature is disabled', async () => {
	vi.spyOn(api, 'phoneStatus').mockResolvedValue({ enabled: false, origin: null, tailscaleLogin: null, vapidPublicKey: null, pendingPlanCount: 0, devices: [] });
    renderPhonePage();
    expect(await screen.findByText('Phone approvals are not enabled')).toBeTruthy();
    expect(screen.getByText('eve phone setup')).toBeTruthy();
  });

  it('shows repository context and the waiting goal in the focused queue', async () => {
    vi.spyOn(api, 'phoneStatus').mockResolvedValue({
      enabled: true,
	  origin: 'https://mac.example.ts.net:8443',
      tailscaleLogin: 'owner@example.com',
      vapidPublicKey: 'AQID',
      pendingPlanCount: 1,
      devices: []
    });
    vi.spyOn(api, 'planRequests').mockResolvedValue([phonePlan()]);
    renderPhonePage();
    expect(await screen.findByText('Approve phone plans safely')).toBeTruthy();
    expect(screen.getByText('eve')).toBeTruthy();
    expect(screen.getByText('main')).toBeTruthy();
    expect(screen.getByText('Connected as owner@example.com')).toBeTruthy();
  });
});

function phonePlan(): PlanRequest {
  return {
    planRequestId: 'planreq_phone_12345678',
    repository: 'eve',
    repositoryRoot: '/tmp/eve',
    branch: 'main',
    state: 'pending_approval',
    currentRevision: 1,
    availableSuites: ['default'],
    createdAt: '2026-08-08T12:00:00Z',
    updatedAt: '2026-08-08T12:00:00Z',
    revisions: [{
      revision: 1,
      source: 'agent',
      goal: 'Approve phone plans safely',
      acceptanceCriteria: 'The phone can approve.',
      allowedPathGlobs: ['ui/**'],
      milestones: [{ title: 'Build the PWA' }],
      configuredSuite: 'default',
      resolvedCheckIds: ['ui'],
      policyHash: '',
      checkDefinitionsHash: '',
      suiteDigest: '',
      baseCommit: 'abc123',
      branch: 'main',
      createdAt: '2026-08-08T12:00:00Z'
    }]
  };
}
