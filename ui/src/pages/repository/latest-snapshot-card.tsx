import { Link } from "@tanstack/react-router";
import {
  ArrowRight,
  BookOpen,
  Calendar,
  Code2,
  Sparkles,
} from "lucide-react";

import { relativeTime } from "../../components/dashboard-chrome";
import { StatusBadge } from "../../components/status-badge";
import { compactDate } from "../../format";
import type {
  EvolutionSummary,
  PullRequestSummary,
} from "../../types";
import { MetaPill } from "./meta-pill";
import { PullRequestStateBadge } from "./pull-request-state-badge";

export function LatestSnapshotCard({
  latest,
  pullRequest,
  repository,
}: {
  latest?: EvolutionSummary;
  pullRequest?: PullRequestSummary;
  repository: string;
}): React.JSX.Element {
  if (!latest) {
    return (
      <section
        id="snapshots"
        className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]"
      >
        <h2 className="flex items-center gap-2 text-base font-semibold">
          <BookOpen className="size-4 text-blue-600" />
          Latest snapshot
        </h2>
        <p className="mt-3 text-sm text-muted-foreground">
          No snapshots have been recorded for this repository.
        </p>
      </section>
    );
  }

  return (
    <section
      id="snapshots"
      className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]"
    >
      <h2 className="flex items-center gap-2 text-base font-semibold">
        <BookOpen className="size-4 text-blue-600" />
        Latest snapshot
      </h2>
      <div
        className={`mt-5 grid gap-5 ${
          pullRequest
            ? "lg:grid-cols-[minmax(0,1fr)_300px] lg:divide-x"
            : "grid-cols-1"
        }`}
      >
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-3">
            <h3 className="text-base font-semibold text-balance">
              {latest.title}
            </h3>
            <StatusBadge status={latest.status} />
          </div>
          <p className="mt-2 max-w-[70ch] text-sm leading-6 text-muted-foreground">
            {latest.outcome}
          </p>
          <div className="mt-4 flex flex-wrap gap-2">
            <MetaPill icon={Sparkles} label={latest.type} />
            <MetaPill
              icon={Code2}
              label={`${latest.commitCount} ${latest.commitCount === 1 ? "commit" : "commits"}`}
            />
            <MetaPill
              icon={Calendar}
              label={compactDate(latest.updatedAt || latest.createdAt)}
            />
          </div>
        </div>
        {pullRequest ? (
          <div className="min-w-0 lg:pl-5">
            <div className="flex items-center justify-between gap-3">
              <div>
                <p className="text-xs font-medium text-muted-foreground">
                  Pull request
                </p>
                <Link
                  to="/repositories/$repo/pull-requests/$number"
                  params={{
                    repo: repository,
                    number: String(pullRequest.number),
                  }}
                  className="mt-1 inline-flex items-center gap-2 text-lg font-semibold text-blue-700 hover:text-blue-800"
                >
                  #{pullRequest.number}
                  <ArrowRight className="size-4" />
                </Link>
              </div>
              <PullRequestStateBadge pullRequest={pullRequest} />
            </div>
            <dl className="mt-4 grid grid-cols-[1fr_auto] gap-x-4 gap-y-2 text-sm">
              <dt className="text-muted-foreground">Mergeable</dt>
              <dd className="font-medium">
                {pullRequest.mergeability === "mergeable" ? "Yes" : "No"}
              </dd>
              <dt className="text-muted-foreground">Checks</dt>
              <dd className="font-medium">{pullRequest.checksPassed} passed</dd>
              <dt className="text-muted-foreground">Updated</dt>
              <dd className="font-medium">
                {relativeTime(pullRequest.updatedAt)}
              </dd>
            </dl>
          </div>
        ) : null}
      </div>
    </section>
  );
}
