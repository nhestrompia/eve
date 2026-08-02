import { Link } from "@tanstack/react-router";
import {
  ArrowRight,
  BookOpen,
  CheckCircle2,
  GitPullRequest,
} from "lucide-react";

import { relativeTime } from "../../components/dashboard-chrome";
import { PullRequestBranchRoute } from "../../components/pull-request-branch-route";
import { Badge } from "../../components/ui/badge";
import type { PullRequestSummary } from "../../types";
import { PullRequestStateBadge } from "./pull-request-state-badge";

export function PullRequestGroup({
  title,
  detail,
  pullRequests,
  repository,
  empty,
}: {
  title: string;
  detail: string;
  pullRequests: PullRequestSummary[];
  repository: string;
  empty: string;
}): React.JSX.Element {
  return (
    <section className="overflow-hidden rounded-lg bg-white shadow-[var(--shadow-border)]">
      <header className="flex min-h-14 items-center justify-between gap-4 border-b px-5">
        <h2 className="font-semibold">{title}</h2>
        <span className="text-xs font-medium text-muted-foreground">
          {detail}
        </span>
      </header>
      {pullRequests.length === 0 ? (
        <p className="px-5 py-8 text-sm text-muted-foreground">{empty}</p>
      ) : (
        <div className="divide-y">
          {pullRequests.map((pullRequest) => (
            <Link
              key={pullRequest.number}
              to="/repositories/$repo/pull-requests/$number"
              params={{
                repo: repository,
                number: String(pullRequest.number),
              }}
              className="group grid gap-4 px-5 py-5 transition-colors hover:bg-slate-50 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
            >
              <div className="flex min-w-0 gap-4">
                <span
                  className={`mt-0.5 grid size-9 shrink-0 place-items-center rounded-full ${
                    pullRequest.readyToMerge
                      ? "bg-emerald-100 text-emerald-700"
                      : "bg-slate-100 text-slate-600"
                  }`}
                >
                  {pullRequest.readyToMerge ? (
                    <CheckCircle2 className="size-4.5" />
                  ) : (
                    <GitPullRequest className="size-4.5" />
                  )}
                </span>
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <h3 className="font-semibold group-hover:text-blue-700">
                      {pullRequest.title}
                    </h3>
                    {pullRequest.readyToMerge ? (
                      <Badge variant="success">Ready to merge</Badge>
                    ) : null}
                  </div>
                  <p className="mt-1 text-sm text-muted-foreground">
                    #{pullRequest.number} opened by {pullRequest.author || "unknown"} · updated {relativeTime(pullRequest.updatedAt)}
                  </p>
                  <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                    <PullRequestBranchRoute
                      headBranch={pullRequest.headBranch}
                      baseBranch={pullRequest.baseBranch}
                    />
                    {pullRequest.snapshotId ? (
                      <span className="inline-flex items-center gap-1 rounded-md bg-blue-50 px-2 py-1 text-blue-700">
                        <BookOpen className="size-3" />
                        {pullRequest.snapshotTitle || pullRequest.snapshotId}
                      </span>
                    ) : null}
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-4 sm:justify-end">
                <div className="text-right text-xs text-muted-foreground">
                  <p>{pullRequest.changedFiles} files</p>
                  <p className="mt-1">
                    <span className="text-emerald-700">+{pullRequest.additions}</span>{" "}
                    <span className="text-red-700">−{pullRequest.deletions}</span>
                  </p>
                </div>
                <PullRequestStateBadge pullRequest={pullRequest} />
                <ArrowRight className="size-4 text-slate-400 transition-transform group-hover:translate-x-0.5 group-hover:text-blue-700" />
              </div>
            </Link>
          ))}
        </div>
      )}
    </section>
  );
}
