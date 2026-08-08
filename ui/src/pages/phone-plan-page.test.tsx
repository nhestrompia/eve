// @vitest-environment jsdom

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { ComponentProps, ReactNode } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { api } from '../api';
import type { PlanRequest } from '../types';
import { PhonePlanPage } from './phone-plan-page';

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual<typeof import('@tanstack/react-router')>('@tanstack/react-router');
  return {
    ...actual,
    useParams: () => ({ planRequestId: 'planreq_phone_review_12345678' }),
    Link: ({ children, to, params, ...props }: ComponentProps<'a'> & { children: ReactNode; to: string; params?: Record<string, string> }) => (
      <a href={params ? to.replace('$planRequestId', params.planRequestId) : to} {...props}>{children}</a>
    )
  };
});

afterEach(() => vi.restoreAllMocks());

function renderPlan() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<QueryClientProvider client={client}><PhonePlanPage /></QueryClientProvider>);
}

describe('PhonePlanPage', () => {
  it('labels editable controls and submits a complete human proposal', async () => {
    const plan = phonePlan();
    vi.spyOn(api, 'planRequest').mockResolvedValue(plan);
    vi.spyOn(api, 'planRequests').mockResolvedValue([]);
    const approve = vi.spyOn(api, 'approvePlanRequest').mockResolvedValue({ ...plan, state: 'locked', lockedRevision: 2, currentRevision: 2 });
    renderPlan();

    fireEvent.click(await screen.findByRole('button', { name: 'Edit plan' }));
    const goal = screen.getByRole('textbox', { name: 'Goal' });
    expect(screen.getByRole('textbox', { name: 'Acceptance criteria' })).toBeTruthy();
    expect(screen.getByRole('textbox', { name: 'Allowed paths' })).toBeTruthy();
    expect(screen.getByRole('combobox', { name: 'Required suite' })).toBeTruthy();
    fireEvent.change(goal, { target: { value: 'Approve the exact mobile revision safely' } });
    fireEvent.click(screen.getByRole('button', { name: 'Approve edits' }));
    expect(await screen.findByRole('alertdialog', { name: /Approve your edited plan/ })).toBeTruthy();
    fireEvent.click(screen.getByRole('button', { name: 'Approve plan' }));

	await waitFor(() => expect(approve).toHaveBeenCalledWith(plan, expect.objectContaining({
	  goal: 'Approve the exact mobile revision safely',
	  milestones: plan.revisions[0].milestones
	})));
  });

  it('requires labelled non-whitespace rejection feedback', async () => {
    const plan = phonePlan();
    vi.spyOn(api, 'planRequest').mockResolvedValue(plan);
    vi.spyOn(api, 'planRequests').mockResolvedValue([]);
    const reject = vi.spyOn(api, 'rejectPlanRequest').mockResolvedValue({ ...plan, state: 'rejected', rejectionFeedback: 'Narrow the scope.' });
    renderPlan();

    fireEvent.click(await screen.findByRole('button', { name: 'Request changes' }));
    const feedback = screen.getByRole('textbox', { name: 'Requested changes' });
    const submit = screen.getByRole('button', { name: 'Send feedback' });
	expect((submit as HTMLButtonElement).disabled).toBe(true);
    fireEvent.change(feedback, { target: { value: '  Narrow the scope.  ' } });
    fireEvent.click(submit);
    await waitFor(() => expect(reject).toHaveBeenCalledWith(plan, 'Narrow the scope.'));
  });

	it('associates the shared validation alert only with the active invalid field', async () => {
		vi.spyOn(api, 'planRequest').mockResolvedValue(phonePlan());
		vi.spyOn(api, 'planRequests').mockResolvedValue([]);
		renderPlan();

		fireEvent.click(await screen.findByRole('button', { name: 'Edit plan' }));
		const goal = screen.getByRole('textbox', { name: 'Goal' });
		const criteria = screen.getByRole('textbox', { name: 'Acceptance criteria' });
		fireEvent.change(goal, { target: { value: '' } });
		fireEvent.change(criteria, { target: { value: '' } });

		expect(goal.getAttribute('aria-invalid')).toBe('true');
		expect(goal.getAttribute('aria-describedby')).toBe('proposal-validation');
		expect(criteria.getAttribute('aria-invalid')).toBe('false');
		expect(criteria.getAttribute('aria-describedby')).toBeNull();
		expect(screen.getByRole('alert').textContent).toBe('Goal is required.');
	});
});

function phonePlan(): PlanRequest {
  return {
    planRequestId: 'planreq_phone_review_12345678',
    repository: 'eve',
    repositoryRoot: '/tmp/eve',
    branch: 'main',
    state: 'pending_approval',
    currentRevision: 1,
    availableSuites: ['default', 'release'],
    staleReasons: [],
    createdAt: '2026-08-08T12:00:00Z',
    updatedAt: '2026-08-08T12:00:00Z',
    revisions: [{
      revision: 1,
      source: 'agent',
      goal: 'Approve phone plans safely',
      acceptanceCriteria: '- Exact revision is visible',
      allowedPathGlobs: ['ui/**'],
      milestones: [{ title: 'Build the PWA', goal: 'Keep approval focused.' }],
      configuredSuite: 'default',
      resolvedSuite: 'default',
      resolvedCheckIds: ['ui'],
      policyHash: 'sha256:policy',
      checkDefinitionsHash: 'sha256:checks',
      suiteDigest: 'sha256:checks',
      baseCommit: 'abc123',
      branch: 'main',
      createdAt: '2026-08-08T12:00:00Z'
    }]
  };
}
