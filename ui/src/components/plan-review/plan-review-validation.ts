import type { PlanProposal, PlanRequest, PlanRevision } from '../../types';

export function currentRevision(plan: PlanRequest): PlanRevision | undefined {
  return plan.revisions.find((revision) => revision.revision === plan.currentRevision);
}

export function planToProposal(plan?: PlanRequest): PlanProposal {
  const revision = plan ? currentRevision(plan) : undefined;
  return {
    goal: revision?.goal ?? '',
    acceptanceCriteria: revision?.acceptanceCriteria ?? '',
    allowedPathGlobs: revision?.allowedPathGlobs ?? [],
    milestones: revision?.milestones ?? [],
    requiredSuite: revision?.configuredSuite
  };
}

export function proposalValidationMessage(proposal: PlanProposal) {
	return proposalFieldValidationMessage(proposal, 'goal')
		|| proposalFieldValidationMessage(proposal, 'acceptanceCriteria')
		|| proposalFieldValidationMessage(proposal, 'allowedPathGlobs');
}

export function proposalFieldValidationMessage(proposal: PlanProposal, field: 'goal' | 'acceptanceCriteria' | 'allowedPathGlobs') {
	if (field === 'goal' && !proposal.goal.trim()) return 'Goal is required.';
	if (field === 'acceptanceCriteria' && !proposal.acceptanceCriteria.trim()) return 'Acceptance criteria are required.';
	if (field === 'allowedPathGlobs' && (proposal.allowedPathGlobs.length === 0 || proposal.allowedPathGlobs.some((glob) => !glob.trim()))) {
		return 'At least one allowed path glob is required.';
	}
	return '';
}

export function proposalsMatch(left: PlanProposal, right: PlanProposal) {
  return JSON.stringify(normalizeProposal(left)) === JSON.stringify(normalizeProposal(right));
}

function normalizeProposal(proposal: PlanProposal): PlanProposal {
  return {
    goal: proposal.goal.trim(),
    acceptanceCriteria: proposal.acceptanceCriteria.trim(),
    allowedPathGlobs: proposal.allowedPathGlobs.map((glob) => glob.trim()).filter(Boolean),
    milestones: proposal.milestones,
    requiredSuite: proposal.requiredSuite?.trim() || undefined
  };
}
