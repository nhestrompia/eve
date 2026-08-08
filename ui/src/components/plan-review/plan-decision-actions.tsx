import { CheckCircle2 } from 'lucide-react';
import type { ReactNode } from 'react';
import { Button } from '../ui/button';

export function PlanDecisionActions({ rejecting, edited, busy, approvalDisabled, onCancelReject, onOpenReject, onApprove, rejectAction }: {
  rejecting: boolean;
  edited: boolean;
  busy: boolean;
  approvalDisabled: boolean;
  onCancelReject: () => void;
  onOpenReject: () => void;
  onApprove: () => void;
  rejectAction: ReactNode;
}) {
  return (
    <footer className="phone-decision-bar">
      {rejecting ? (
        <>
          <Button variant="outline" onClick={onCancelReject} disabled={busy}>Cancel</Button>
          {rejectAction}
        </>
      ) : (
        <>
          <Button variant="outline" onClick={onOpenReject} disabled={busy}>Request changes</Button>
          <Button onClick={onApprove} disabled={busy || approvalDisabled}>
            <CheckCircle2 aria-hidden="true" /> {edited ? 'Approve edits' : 'Approve'}
          </Button>
        </>
      )}
    </footer>
  );
}
