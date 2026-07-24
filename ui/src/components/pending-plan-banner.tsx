import { BellRing, ShieldCheck } from "lucide-react";
import type { PlanRequest } from "../types";

export function PendingPlanBanner({ plans }: { plans: PlanRequest[] }) {
  if (plans.length === 0) return null;

  return (
    <section
      aria-live="polite"
      aria-label={`${plans.length} ${plans.length === 1 ? "plan" : "plans"} awaiting approval`}
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
                {plans.length} {plans.length === 1 ? "plan is" : "plans are"} waiting for you
              </h2>
              <span className="rounded-full bg-white px-2 py-0.5 text-[11px] font-semibold text-blue-700 shadow-[0_0_0_1px_rgba(37,99,235,0.16)]">
                Agents paused
              </span>
            </div>
            <p className="mt-1 text-sm leading-5 text-slate-600">
              Review the requests in the EVE approval window. It opens automatically when a new plan arrives.
            </p>
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-2 text-xs font-medium text-blue-800">
          <ShieldCheck className="size-4" aria-hidden="true" />
          EVE menu bar
        </div>
      </div>

      <div className="mt-4 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
        {plans.slice(0, 3).map((plan) => {
          const revision = plan.revisions.find(
            (candidate) => candidate.revision === plan.currentRevision,
          );
          return (
            <div
              key={plan.planRequestId}
              className="min-w-0 rounded-lg bg-white/90 px-3 py-2.5 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]"
            >
              <div className="flex items-center gap-2 text-xs text-slate-500">
                <strong className="truncate font-semibold text-slate-800">
                  {plan.repository}
                </strong>
                <span aria-hidden="true">·</span>
                <span className="truncate font-mono">{plan.branch}</span>
              </div>
              <p className="mt-1 truncate text-sm font-medium text-slate-950">
                {revision?.goal || "Plan awaiting review"}
              </p>
            </div>
          );
        })}
      </div>

      {plans.length > 3 ? (
        <p className="mt-3 text-xs font-medium text-blue-800">
          +{plans.length - 3} more in the approval queue
        </p>
      ) : null}
    </section>
  );
}
