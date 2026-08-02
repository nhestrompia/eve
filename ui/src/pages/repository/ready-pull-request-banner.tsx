import { Link } from "@tanstack/react-router";
import {
  AlertTriangle,
  ArrowRight,
  Check,
  CheckCircle2,
  CircleDashed,
  GitPullRequest,
  X,
} from "lucide-react";
import { useState } from "react";

import { relativeTime } from "../../components/dashboard-chrome";
import { Button } from "../../components/ui/button";
import type { PullRequestSummary } from "../../types";
import { pullRequestBannerSignals } from "./pull-requests-utils";

export function ReadyPullRequestBanner({
  pullRequest,
  repository,
}: {
  pullRequest: PullRequestSummary;
  repository: string;
}): React.JSX.Element | null {
  const [dismissed, setDismissed] = useState(false);
  if (dismissed) return null;

  const ready = pullRequest.readyToMerge;
  const signals = pullRequestBannerSignals(pullRequest);

  return (
    <section
      className={`overflow-hidden rounded-lg border shadow-[0_0_0_1px_rgba(15,23,42,0.03)] ${
        ready
          ? "border-emerald-200 bg-emerald-50/55"
          : "border-blue-200 bg-blue-50/55"
      }`}
    >
      <div className="flex flex-col gap-5 px-5 py-5 sm:px-6 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex min-w-0 gap-4">
          <span
            className={`grid size-10 shrink-0 place-items-center rounded-full ${
              ready
                ? "bg-emerald-100 text-emerald-700"
                : "bg-blue-100 text-blue-700"
            }`}
          >
            {ready ? (
              <CheckCircle2 className="size-5" />
            ) : (
              <GitPullRequest className="size-5" />
            )}
          </span>
          <div className="min-w-0">
            <h2
              className={`text-base font-semibold ${
                ready ? "text-emerald-950" : "text-blue-950"
              }`}
            >
              {ready
                ? "1 pull request ready to merge"
                : "Recent pull request ready for review"}
            </h2>
            <p
              className={`mt-1 max-w-[70ch] text-sm leading-6 ${
                ready ? "text-emerald-900/75" : "text-blue-900/75"
              }`}
            >
              {ready
                ? `${pullRequest.title} matches its verified Snapshot and GitHub reports no merge blockers.`
                : `${pullRequest.title} is open. Review its product evidence and remaining blockers.`}
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Button
            asChild
            className={
              ready ? "bg-emerald-700 text-white hover:bg-emerald-800" : ""
            }
          >
            <Link
              to="/repositories/$repo/pull-requests/$number"
              params={{
                repo: repository,
                number: String(pullRequest.number),
              }}
            >
              Review PR #{pullRequest.number}
              <ArrowRight className="size-4" />
            </Link>
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Dismiss pull request banner"
            onClick={() => setDismissed(true)}
          >
            <X className="size-4" />
          </Button>
        </div>
      </div>
      <div
        className={`grid border-y sm:grid-cols-2 xl:grid-cols-4 ${
          ready ? "border-emerald-200/70" : "border-blue-200/70"
        }`}
      >
        {signals.map((signal) => (
          <div
            key={signal.label}
            className={`flex gap-3 px-5 py-4 sm:[&:nth-child(even)]:border-l xl:[&:not(:first-child)]:border-l ${
              ready ? "border-emerald-200/70" : "border-blue-200/70"
            }`}
          >
            {signal.tone === "success" ? (
              <Check className="mt-0.5 size-4 shrink-0 text-emerald-700" />
            ) : signal.tone === "warning" ? (
              <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-700" />
            ) : (
              <CircleDashed className="mt-0.5 size-4 shrink-0 text-slate-500" />
            )}
            <div>
              <p className="text-sm font-semibold text-slate-950">
                {signal.label}
              </p>
              <p className="mt-0.5 text-xs text-slate-600">
                {signal.detail}
              </p>
            </div>
          </div>
        ))}
      </div>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 px-5 py-3 text-xs text-slate-700 sm:px-6">
        <GitPullRequest className="size-3.5" />
        <span className="font-semibold">PR #{pullRequest.number}</span>
        <span>
          {pullRequest.headBranch} → {pullRequest.baseBranch}
        </span>
        <span>Updated {relativeTime(pullRequest.updatedAt)}</span>
      </div>
    </section>
  );
}
