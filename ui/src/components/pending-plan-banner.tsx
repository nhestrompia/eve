import { useMutation, useQueryClient } from "@tanstack/react-query";
import { BellRing, CheckCircle2, ChevronRight, Edit3, GitBranch, Trash2, XCircle } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { api } from "../api";
import type { PlanProposal, PlanRequest, PlanRevision } from "../types";
import { Button } from "./ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "./ui/dialog";

export function PendingPlanBanner({ plans }: { plans: PlanRequest[] }) {
  const pendingPlans = plans.filter((plan) => plan.state === "pending_approval" || isStalePlan(plan));
  const [open, setOpen] = useState(false);

  if (pendingPlans.length === 0) return null;

  return (
    <>
      <section
        aria-live="polite"
        aria-label={`${pendingPlans.length} ${pendingPlans.length === 1 ? "plan" : "plans"} awaiting approval`}
        className="rounded-xl bg-blue-50/80 p-4 shadow-[inset_0_0_0_1px_rgba(37,99,235,0.2)] sm:p-5"
      >
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="flex min-w-0 gap-3">
            <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-blue-600 text-white shadow-sm">
              <BellRing className="size-5" aria-hidden="true" />
            </span>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h2 className="text-sm font-semibold text-slate-950">
                  {pendingPlans.length} {pendingPlans.length === 1 ? "plan is" : "plans are"} waiting for you
                </h2>
                <span className="rounded-full bg-white px-2 py-0.5 text-[11px] font-semibold text-blue-700 shadow-[0_0_0_1px_rgba(37,99,235,0.16)]">
                  Agents paused
                </span>
              </div>
              <p className="mt-1 text-sm leading-5 text-slate-600">
                Review, edit, approve, or reject the queued plans directly from this UI.
              </p>
            </div>
          </div>

          <Button type="button" size="sm" onClick={() => setOpen(true)}>
            Review plans
            <ChevronRight className="size-4" aria-hidden="true" />
          </Button>
        </div>

        <div className="mt-4 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
          {pendingPlans.slice(0, 3).map((plan) => (
            <button
              type="button"
              key={plan.planRequestId}
              onClick={() => setOpen(true)}
              className="min-w-0 rounded-lg bg-white/90 px-3 py-2.5 text-left shadow-[0_0_0_1px_rgba(15,23,42,0.08)] transition-colors hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
            >
              <div className="flex items-center gap-2 text-xs text-slate-500">
                <strong className="truncate font-semibold text-slate-800">{plan.repository}</strong>
                <span aria-hidden="true">·</span>
                <span className="truncate font-mono">{plan.branch}</span>
              </div>
              <p className="mt-1 truncate text-sm font-medium text-slate-950">
                {currentRevision(plan)?.goal || "Plan awaiting review"}
              </p>
            </button>
          ))}
        </div>

        {pendingPlans.length > 3 ? (
          <p className="mt-3 text-xs font-medium text-blue-800">
            +{pendingPlans.length - 3} more in the approval queue
          </p>
        ) : null}
      </section>

      <PlanApprovalDialog plans={pendingPlans} open={open} onOpenChange={setOpen} />
    </>
  );
}

export function PlanApprovalDialog({
  plans,
  open,
  onOpenChange,
}: {
  plans: PlanRequest[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const [selectedID, setSelectedID] = useState(plans[0]?.planRequestId ?? "");
  const [editing, setEditing] = useState(false);
  const [rejecting, setRejecting] = useState(false);
  const [feedback, setFeedback] = useState("");
  const selected = plans.find((plan) => plan.planRequestId === selectedID) ?? plans[0];
  const [proposal, setProposal] = useState<PlanProposal>(() => planToProposal(selected));

  useEffect(() => {
    if (!plans.some((plan) => plan.planRequestId === selectedID)) {
      setSelectedID(plans[0]?.planRequestId ?? "");
    }
  }, [plans, selectedID]);

  useEffect(() => {
    setProposal(planToProposal(selected));
    setEditing(false);
    setRejecting(false);
    setFeedback("");
  }, [selected?.planRequestId, selected?.currentRevision]);

  const finishMutation = async (message: string) => {
    toast.success(message);
    await queryClient.invalidateQueries({ queryKey: ["pending-plan-requests"] });
    if (plans.length <= 1) {
      onOpenChange(false);
    }
  };
  const approve = useMutation({
    mutationFn: () => api.approvePlanRequest(selected, editing ? proposal : undefined),
    onSuccess: () => finishMutation(editing ? "Edited plan approved" : "Plan approved"),
    onError: (error) => toast.error("Approval failed", { description: errorMessage(error) }),
  });
  const reject = useMutation({
    mutationFn: () => api.rejectPlanRequest(selected, feedback),
    onSuccess: () => finishMutation("Plan rejected"),
    onError: (error) => toast.error("Rejection failed", { description: errorMessage(error) }),
  });
  const remove = useMutation({
    mutationFn: () => api.removePlanRequest(selected),
    onSuccess: () => finishMutation("Plan removed"),
    onError: (error) => toast.error("Remove failed", { description: errorMessage(error) }),
  });
  const validationMessage = proposalValidationMessage(proposal);
  const feedbackMessage = feedback.trim() ? "" : "Rejection feedback is required.";
  const busy = approve.isPending || reject.isPending || remove.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[min(1080px,calc(100vw-24px))] p-0">
        <div className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)_auto]">
          <DialogHeader className="border-b px-5 py-4">
            <DialogTitle className="text-base">Plan approvals</DialogTitle>
            <DialogDescription>
              {plans.length} {plans.length === 1 ? "plan is" : "plans are"} waiting across your repositories.
            </DialogDescription>
          </DialogHeader>

          <div className="grid min-h-0 grid-cols-1 overflow-hidden md:grid-cols-[240px_minmax(0,1fr)]">
            <PlanQueueList plans={plans} selectedID={selected?.planRequestId} onSelect={setSelectedID} />
            {selected ? (
              <PlanApprovalContent
                plan={selected}
                proposal={proposal}
                onProposalChange={setProposal}
                editing={editing}
                rejecting={rejecting}
                feedback={feedback}
                onFeedbackChange={setFeedback}
              />
            ) : (
              <div className="p-5 text-sm text-muted-foreground">No plans are waiting.</div>
            )}
          </div>

          {selected ? (
            <div className="flex flex-col gap-3 border-t bg-white px-5 py-4 sm:flex-row sm:items-center">
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setEditing((value) => !value);
                  setRejecting(false);
                }}
                disabled={busy || isStalePlan(selected)}
              >
                <Edit3 className="size-4" aria-hidden="true" />
                {editing ? "Cancel edits" : "Edit plan"}
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setRejecting((value) => !value);
                  setEditing(false);
                }}
                disabled={busy || isStalePlan(selected)}
              >
                <XCircle className="size-4" aria-hidden="true" />
                Reject
              </Button>
              <div className="min-h-5 flex-1 text-xs text-red-600">
                {editing && validationMessage ? validationMessage : rejecting ? feedbackMessage : ""}
              </div>
              {isStalePlan(selected) ? (
                <Button
                  type="button"
                  variant="destructive"
                  onClick={() => remove.mutate()}
                  disabled={busy}
                >
                  <Trash2 className="size-4" aria-hidden="true" />
                  Remove stale plan
                </Button>
              ) : rejecting ? (
                <Button
                  type="button"
                  variant="destructive"
                  onClick={() => reject.mutate()}
                  disabled={busy || Boolean(feedbackMessage)}
                >
                  Send rejection
                </Button>
              ) : (
                <Button
                  type="button"
                  onClick={() => approve.mutate()}
                  disabled={busy || selected.state !== "pending_approval" || (editing && Boolean(validationMessage))}
                >
                  <CheckCircle2 className="size-4" aria-hidden="true" />
                  {editing ? "Approve edited plan" : "Approve plan"}
                </Button>
              )}
            </div>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function PlanQueueList({
  plans,
  selectedID,
  onSelect,
}: {
  plans: PlanRequest[];
  selectedID?: string;
  onSelect: (id: string) => void;
}) {
  return (
    <aside className="min-h-0 overflow-auto border-b bg-slate-50 p-2 md:border-b-0 md:border-r">
      <div className="px-2 py-2 text-[11px] font-semibold uppercase tracking-wide text-slate-500">
        Waiting queue
      </div>
      <div className="space-y-1">
        {plans.map((plan) => (
          <button
            type="button"
            key={plan.planRequestId}
            onClick={() => onSelect(plan.planRequestId)}
            className={`w-full min-w-0 rounded-md px-3 py-2.5 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 ${
              plan.planRequestId === selectedID ? "bg-white shadow-[0_0_0_1px_rgba(37,99,235,0.22)]" : "hover:bg-white"
            }`}
          >
            <div className="flex min-w-0 items-center gap-2 text-xs text-slate-500">
              <span className={`size-2 rounded-full ${isStalePlan(plan) ? "bg-orange-500" : "bg-blue-600"}`} aria-hidden="true" />
              <span className="truncate font-semibold text-slate-800">{plan.repository}</span>
            </div>
            <div className="mt-1 truncate text-xs font-mono text-slate-500">{plan.branch}</div>
            <div className="mt-1 line-clamp-2 text-sm font-medium text-slate-950">
              {currentRevision(plan)?.goal || "Plan awaiting review"}
            </div>
          </button>
        ))}
      </div>
    </aside>
  );
}

export function PlanApprovalContent({
  plan,
  proposal,
  onProposalChange,
  editing,
  rejecting,
  feedback,
  onFeedbackChange,
}: {
  plan: PlanRequest;
  proposal: PlanProposal;
  onProposalChange: (proposal: PlanProposal) => void;
  editing: boolean;
  rejecting: boolean;
  feedback: string;
  onFeedbackChange: (feedback: string) => void;
}) {
  const revision = currentRevision(plan);
  return (
    <section className="min-h-0 overflow-auto p-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-2 text-xs text-slate-500">
            <strong className="text-slate-800">{plan.repository}</strong>
            <GitBranch className="size-3" aria-hidden="true" />
            <span className="font-mono">{plan.branch}</span>
            <span>Revision {plan.currentRevision}</span>
          </div>
          <h3 className="mt-2 text-lg font-semibold text-slate-950">
            {revision?.goal || "Plan awaiting review"}
          </h3>
        </div>
        <div className={`flex shrink-0 items-center gap-2 rounded-full px-3 py-1 text-xs font-semibold ${
          isStalePlan(plan) ? "bg-orange-50 text-orange-700" : "bg-blue-50 text-blue-700"
        }`}>
          {isStalePlan(plan) ? "Stale" : "Awaiting approval"}
        </div>
      </div>

      <div className="mt-5 grid gap-5">
        {isStalePlan(plan) ? (
          <div className="rounded-lg border border-orange-200 bg-orange-50 p-4 text-sm text-orange-950">
            <p className="font-semibold">Approval disabled</p>
            <ul className="mt-2 list-disc space-y-1 pl-5">
              {(plan.staleReasons?.length ? plan.staleReasons : ["Repository context changed"]).map((reason) => (
                <li key={reason}>{reason}</li>
              ))}
            </ul>
          </div>
        ) : null}
        <PlanTextField
          label="Goal"
          value={editing ? proposal.goal : revision?.goal ?? ""}
          editing={editing}
          rows={3}
          onChange={(value) => onProposalChange({ ...proposal, goal: value })}
        />
        <PlanTextField
          label="Acceptance criteria"
          value={editing ? proposal.acceptanceCriteria : revision?.acceptanceCriteria ?? ""}
          editing={editing}
          rows={6}
          onChange={(value) => onProposalChange({ ...proposal, acceptanceCriteria: value })}
        />
        <PlanTextField
          label="Declared scope"
          value={editing ? proposal.allowedPathGlobs.join("\n") : revision?.allowedPathGlobs.join("\n") ?? ""}
          editing={editing}
          rows={4}
          monospaced
          onChange={(value) => onProposalChange({ ...proposal, allowedPathGlobs: value.split(/\r?\n/) })}
        />
        <div>
          <label className="text-xs font-semibold uppercase tracking-wide text-slate-500">
            Verification suite
          </label>
          {editing ? (
            <select
              value={proposal.requiredSuite ?? ""}
              onChange={(event) => onProposalChange({ ...proposal, requiredSuite: event.target.value || undefined })}
              className="mt-2 h-10 w-full rounded-md border border-input bg-card px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <option value="">Use branch default</option>
              {(plan.availableSuites ?? []).map((suite) => (
                <option key={suite} value={suite}>
                  {suite}
                </option>
              ))}
            </select>
          ) : (
            <p className="mt-2 text-sm text-slate-900">{revision?.configuredSuite || "Branch default"}</p>
          )}
        </div>
        <ReadOnlyList
          title="Milestones"
          values={revision?.milestones.map((milestone) => milestone.goal ? `${milestone.title}: ${milestone.goal}` : milestone.title) ?? []}
          empty="No milestones"
        />
        <ReadOnlyList
          title="Resolved checks"
          values={revision?.resolvedCheckIds ?? []}
          empty="No configured checks"
          monospaced
        />
        {rejecting ? (
          <div>
            <label className="text-xs font-semibold uppercase tracking-wide text-slate-500">
              Rejection feedback
            </label>
            <textarea
              value={feedback}
              onChange={(event) => onFeedbackChange(event.target.value)}
              rows={4}
              placeholder="Tell the agent what needs to change."
              className="mt-2 w-full rounded-md border border-input bg-card px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            />
          </div>
        ) : null}
      </div>
    </section>
  );
}

function PlanTextField({
  label,
  value,
  editing,
  rows,
  monospaced = false,
  onChange,
}: {
  label: string;
  value: string;
  editing: boolean;
  rows: number;
  monospaced?: boolean;
  onChange: (value: string) => void;
}) {
  return (
    <div>
      <label className="text-xs font-semibold uppercase tracking-wide text-slate-500">{label}</label>
      {editing ? (
        <textarea
          value={value}
          onChange={(event) => onChange(event.target.value)}
          rows={rows}
          className={`mt-2 w-full rounded-md border border-input bg-card px-3 py-2 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${monospaced ? "font-mono" : ""}`}
        />
      ) : (
        <p className={`mt-2 whitespace-pre-wrap text-sm leading-6 text-slate-900 ${monospaced ? "font-mono" : ""}`}>
          {value}
        </p>
      )}
    </div>
  );
}

function ReadOnlyList({
  title,
  values,
  empty,
  monospaced = false,
}: {
  title: string;
  values: string[];
  empty: string;
  monospaced?: boolean;
}) {
  return (
    <div>
      <div className="text-xs font-semibold uppercase tracking-wide text-slate-500">{title}</div>
      <div className="mt-2 grid gap-1.5">
        {(values.length > 0 ? values : [empty]).map((value, index) => (
          <div
            key={`${value}-${index}`}
            className={`rounded-md bg-slate-50 px-3 py-2 text-sm text-slate-800 ${monospaced ? "font-mono" : ""}`}
          >
            {value}
          </div>
        ))}
      </div>
    </div>
  );
}

export function currentRevision(plan: PlanRequest): PlanRevision | undefined {
  return plan.revisions.find((revision) => revision.revision === plan.currentRevision);
}

function isStalePlan(plan?: PlanRequest): boolean {
  return Boolean(plan && (plan.state === "stale" || (plan.staleReasons?.length ?? 0) > 0));
}

export function planToProposal(plan?: PlanRequest): PlanProposal {
  const revision = plan ? currentRevision(plan) : undefined;
  return {
    goal: revision?.goal ?? "",
    acceptanceCriteria: revision?.acceptanceCriteria ?? "",
    allowedPathGlobs: revision?.allowedPathGlobs ?? [],
    milestones: revision?.milestones ?? [],
    requiredSuite: revision?.configuredSuite,
  };
}

export function proposalValidationMessage(proposal: PlanProposal) {
  if (!proposal.goal.trim()) return "Goal is required.";
  if (!proposal.acceptanceCriteria.trim()) return "Acceptance criteria are required.";
  if (proposal.allowedPathGlobs.length === 0 || proposal.allowedPathGlobs.some((glob) => !glob.trim())) {
    return "At least one allowed path glob is required.";
  }
  return "";
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}
