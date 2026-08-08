import { useQuery } from '@tanstack/react-query';
import { Link, useParams } from '@tanstack/react-router';
import { ArrowLeft, CheckCircle2, Edit3, GitBranch, RefreshCw, ShieldAlert, XCircle } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { api } from '../api';
import { PlanDecisionActions } from '../components/plan-review/plan-decision-actions';
import { RequiredSuiteSelector } from '../components/plan-review/plan-edit-form';
import { PlanReviewSection, PlanReviewText } from '../components/plan-review/plan-review-content';
import {
  currentRevision,
  planToProposal,
	proposalFieldValidationMessage,
  proposalValidationMessage,
  proposalsMatch
} from '../components/plan-review/plan-review-validation';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '../components/ui/alert-dialog';
import { Button } from '../components/ui/button';
import { usePlanDecision } from '../hooks/use-plan-decision';
import type { PlanProposal, PlanRequest } from '../types';

export function PhonePlanPage() {
  const { planRequestId } = useParams({ from: '/phone/plans/$planRequestId' });
  const planQuery = useQuery({
    queryKey: ['plan-request', planRequestId],
    queryFn: () => api.planRequest(planRequestId),
    retry: false
  });
	const reviewQueueQuery = useQuery({
		queryKey: ['plan-requests', 'phone-review'],
		queryFn: () => api.planRequests('review'),
		retry: false
	});
  const [resolvedPlan, setResolvedPlan] = useState<PlanRequest>();
  const plan = resolvedPlan ?? planQuery.data;
  const revision = plan ? currentRevision(plan) : undefined;
  const [proposal, setProposal] = useState<PlanProposal>(() => planToProposal());
  const [editing, setEditing] = useState(false);
  const [rejecting, setRejecting] = useState(false);
  const [feedback, setFeedback] = useState('');
  const [confirmOpen, setConfirmOpen] = useState(false);
  const decision = usePlanDecision(plan, (result) => {
    setResolvedPlan(result);
    setEditing(false);
    setRejecting(false);
    setConfirmOpen(false);
  });

  useEffect(() => {
    if (!planQuery.data) return;
    setProposal(planToProposal(planQuery.data));
    setResolvedPlan(undefined);
    setEditing(false);
    setRejecting(false);
    setFeedback('');
  }, [planQuery.data?.planRequestId, planQuery.data?.currentRevision]);

  useEffect(() => {
    if (decision.conflict) void planQuery.refetch();
  }, [decision.conflict]);

  const original = useMemo(() => planToProposal(plan), [plan?.planRequestId, plan?.currentRevision]);
  const edited = !proposalsMatch(original, proposal);
  const validation = proposalValidationMessage(proposal);
  const stale = plan?.state === 'stale' || Boolean(plan?.staleReasons?.length);
  const terminal = plan && !['pending_approval', 'stale'].includes(plan.state);
	const nextPlan = reviewQueueQuery.data?.find((candidate) => candidate.planRequestId !== plan?.planRequestId);

  if (planQuery.isLoading) {
    return <PlanRouteMessage icon={<RefreshCw className="animate-spin" />} title="Opening the plan" detail="Checking its latest revision…" />;
  }
  if (planQuery.error || !plan || !revision) {
    return <PlanRouteMessage icon={<ShieldAlert />} title="This plan is unavailable" detail="It may have been removed or the Mac may be offline." />;
  }
  if (terminal) {
    return (
      <main className="phone-resolution">
		{plan.state === 'rejected' ? <XCircle aria-hidden="true" /> : <CheckCircle2 aria-hidden="true" />}
        <h1>{plan.state === 'rejected' ? 'Changes requested' : 'Plan approved'}</h1>
        <p>{plan.repository} · revision {plan.lockedRevision || plan.currentRevision}</p>
        {plan.rejectionFeedback ? <blockquote>{plan.rejectionFeedback}</blockquote> : null}
		<Button asChild size="lg">
			{nextPlan ? (
				<Link to="/phone/plans/$planRequestId" params={{ planRequestId: nextPlan.planRequestId }}>Review next plan</Link>
			) : <Link to="/phone">Return to approval queue</Link>}
		</Button>
      </main>
    );
  }

  return (
    <main className="phone-review-page">
      <header className="phone-review-header">
        <Link to="/phone" aria-label="Back to approval queue"><ArrowLeft aria-hidden="true" /></Link>
        <div className="min-w-0 flex-1">
          <strong>{plan.repository}</strong>
          <span><GitBranch aria-hidden="true" /> {plan.branch}</span>
        </div>
        <div className="phone-revision">Rev {plan.currentRevision}</div>
      </header>

      <article className="phone-review-document">
        <div className="phone-review-title">
          <div className="phone-state-label" data-stale={stale}>{stale ? 'Stale plan' : 'Awaiting approval'}</div>
          <h1>{editing ? 'Edit the proposed plan' : revision.goal}</h1>
          <p>{editing ? 'Your changes become a new human revision before EVE locks it.' : 'Read through the complete proposal before deciding.'}</p>
          {!stale ? (
            <Button
              variant="outline"
              onClick={() => {
                setEditing((value) => !value);
                setRejecting(false);
                setProposal(planToProposal(plan));
              }}
            >
              <Edit3 aria-hidden="true" /> {editing ? 'Cancel edits' : 'Edit plan'}
            </Button>
          ) : null}
        </div>

        {stale ? (
          <section className="phone-stale" aria-labelledby="stale-plan-title">
            <ShieldAlert aria-hidden="true" />
            <div>
              <h2 id="stale-plan-title">Approval is disabled</h2>
              <ul>{(plan.staleReasons?.length ? plan.staleReasons : ['Repository context changed.']).map((reason) => <li key={reason}>{reason}</li>)}</ul>
            </div>
          </section>
        ) : null}

        <PlanReviewText
          title="Goal"
          value={editing ? proposal.goal : revision.goal}
          editing={editing}
          rows={4}
		  validationMessage={editing && validation === proposalFieldValidationMessage(proposal, 'goal') ? validation : undefined}
          onChange={(goal) => setProposal({ ...proposal, goal })}
        />
        <PlanReviewText
          title="Acceptance criteria"
          value={editing ? proposal.acceptanceCriteria : revision.acceptanceCriteria}
          editing={editing}
          rows={8}
		  validationMessage={editing && validation === proposalFieldValidationMessage(proposal, 'acceptanceCriteria') ? validation : undefined}
          onChange={(acceptanceCriteria) => setProposal({ ...proposal, acceptanceCriteria })}
        />
        <PlanReviewText
          title="Allowed paths"
          value={(editing ? proposal.allowedPathGlobs : revision.allowedPathGlobs).join('\n')}
          editing={editing}
          rows={6}
          monospaced
		  validationMessage={editing && validation === proposalFieldValidationMessage(proposal, 'allowedPathGlobs') ? validation : undefined}
          onChange={(value) => setProposal({ ...proposal, allowedPathGlobs: value.split(/\r?\n/) })}
        />

        <PlanReviewSection title="Milestones">
          {revision.milestones.length ? (
            <ol className="phone-milestones">
              {revision.milestones.map((milestone) => (
                <li key={`${milestone.title}-${milestone.goal}`}>
                  <strong>{milestone.title}</strong>
                  {milestone.goal ? <p>{milestone.goal}</p> : null}
                </li>
              ))}
            </ol>
          ) : <p className="phone-muted">No milestones were declared.</p>}
        </PlanReviewSection>

        <PlanReviewSection title="Verification">
          {editing ? (
			<RequiredSuiteSelector
			  value={proposal.requiredSuite}
			  suites={plan.availableSuites ?? []}
			  onChange={(requiredSuite) => setProposal({ ...proposal, requiredSuite })}
			/>
          ) : <p>{revision.configuredSuite || 'Branch default suite'}</p>}
          <div className="phone-check-list">
            {(revision.resolvedCheckIds.length ? revision.resolvedCheckIds : ['No deterministic checks resolved']).map((check) => <code key={check}>{check}</code>)}
          </div>
        </PlanReviewSection>

        <PlanReviewSection title="Review context">
          <dl className="phone-context-list">
            <div><dt>Base commit</dt><dd><code>{revision.baseCommit || 'Uncommitted context'}</code></dd></div>
            <div><dt>Plan source</dt><dd>{revision.source === 'human' ? 'Human revision' : 'Agent proposal'}</dd></div>
            <div><dt>Repository</dt><dd>{plan.repositoryRoot}</dd></div>
          </dl>
        </PlanReviewSection>

        {rejecting ? (
          <section className="phone-feedback" aria-labelledby="feedback-title">
            <h2 id="feedback-title">What should the agent change?</h2>
            <p>Specific feedback returns this plan to the agent without approving any work.</p>
            <textarea
              value={feedback}
              onChange={(event) => setFeedback(event.target.value)}
              rows={6}
              autoFocus
			  aria-label="Requested changes"
			  aria-required="true"
              placeholder="Explain the missing constraint, risky scope, or desired correction."
            />
          </section>
        ) : null}

		{editing && validation ? <p id="proposal-validation" className="phone-error" role="alert">{validation}</p> : null}
        {decision.error ? (
          <div className="phone-error-block" role="alert">
            <strong>{decision.conflict ? 'The plan changed on the Mac.' : 'Your decision was not saved.'}</strong>
            <p>{decision.conflict ? 'The latest revision has been reloaded. Review it before trying again.' : errorMessage(decision.error)}</p>
          </div>
        ) : null}
      </article>

      {!stale ? (
		<PlanDecisionActions
		  rejecting={rejecting}
		  edited={editing && edited}
		  busy={decision.busy}
		  approvalDisabled={Boolean(editing && validation)}
		  onCancelReject={() => setRejecting(false)}
		  onOpenReject={() => { setRejecting(true); setEditing(false); }}
		  onApprove={() => setConfirmOpen(true)}
		  rejectAction={(
			<Button variant="destructive" onClick={() => decision.reject.mutate(feedback.trim())} disabled={decision.busy || !feedback.trim()}>
			  <XCircle aria-hidden="true" /> Send feedback
			</Button>
		  )}
		/>
      ) : null}

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{editing && edited ? 'Approve your edited plan?' : `Approve revision ${plan.currentRevision}?`}</AlertDialogTitle>
            <AlertDialogDescription>
              {editing && edited
                ? `EVE will create human revision ${plan.currentRevision + 1}, lock it, and let the agent begin.`
                : 'EVE will lock this exact revision and let the agent begin implementation.'}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
			<AlertDialogCancel className="min-h-11">Review again</AlertDialogCancel>
			<AlertDialogAction className="min-h-11" onClick={() => decision.approve.mutate(editing && edited ? proposal : undefined)}>
              Approve plan
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </main>
  );
}

function PlanRouteMessage({ icon, title, detail }: { icon: ReactNode; title: string; detail: string }) {
  return (
    <main className="phone-message">
      <Link to="/phone" className="phone-back-link"><ArrowLeft aria-hidden="true" /> Approval queue</Link>
      <div className="phone-message-icon">{icon}</div>
      <h1>{title}</h1>
      <p>{detail}</p>
    </main>
  );
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}
