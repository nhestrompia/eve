import { useMutation, useQueryClient } from '@tanstack/react-query';
import { APIRequestError, api } from '../api';
import type { PlanProposal, PlanRequest } from '../types';

export function usePlanDecision(plan?: PlanRequest, onSuccess?: (plan: PlanRequest) => void) {
  const queryClient = useQueryClient();
  const settle = async (result: PlanRequest) => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['plan-requests'] }),
      queryClient.invalidateQueries({ queryKey: ['pending-plan-requests'] }),
      queryClient.invalidateQueries({ queryKey: ['phone-status'] })
    ]);
    onSuccess?.(result);
  };
  const approve = useMutation({
    mutationFn: (proposal?: PlanProposal) => {
      if (!plan) throw new Error('Plan is unavailable.');
      return api.approvePlanRequest(plan, proposal);
    },
    onSuccess: settle
  });
  const reject = useMutation({
    mutationFn: (feedback: string) => {
      if (!plan) throw new Error('Plan is unavailable.');
      return api.rejectPlanRequest(plan, feedback);
    },
    onSuccess: settle
  });
  const error = approve.error ?? reject.error;
  return {
    approve,
    reject,
    busy: approve.isPending || reject.isPending,
    error,
    conflict: error instanceof APIRequestError && error.status === 409
  };
}
