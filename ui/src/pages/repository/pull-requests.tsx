import { Link } from "@tanstack/react-router";
import {
  AlertTriangle,
  ArrowRight,
  BookOpen,
  Calendar,
  Check,
  CheckCircle2,
  ChevronDown,
  CircleDashed,
  Code2,
  ExternalLink,
  GitPullRequest,
  Sparkles,
  X,
} from "lucide-react";
import { useState } from "react";

import { relativeTime } from "../../components/dashboard-chrome";
import { LoadingState } from "../../components/loading-state";
import { PullRequestBranchRoute } from "../../components/pull-request-branch-route";
import { StatusBadge } from "../../components/status-badge";
import { Badge } from "../../components/ui/badge";
import { Button } from "../../components/ui/button";
import { compactDate } from "../../format";
import type {
  EvolutionSummary,
  PullRequestCollection,
  PullRequestSummary,
  RepositorySummary,
} from "../../types";
import { MetaPill } from "./header";

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

export function pullRequestBannerSignals(
  pullRequest: PullRequestSummary,
): Array<{
  label: string;
  detail: string;
  tone: "success" | "warning" | "neutral";
}> {
  const exactHeadSnapshot = hasCurrentSnapshotContext(pullRequest);
  const checkSignal =
    pullRequest.checksTotal === 0
      ? {
          label: "No GitHub checks",
          detail: "No check runs have been reported",
          tone: "neutral" as const,
        }
      : pullRequest.checksFailed > 0
        ? {
            label: `${pullRequest.checksFailed} ${
              pullRequest.checksFailed === 1 ? "check" : "checks"
            } failed`,
            detail: `${pullRequest.checksPassed}/${pullRequest.checksTotal} GitHub checks passed`,
            tone: "warning" as const,
          }
        : pullRequest.checksPending > 0
          ? {
              label: `${pullRequest.checksPending} ${
                pullRequest.checksPending === 1 ? "check" : "checks"
              } pending`,
              detail: `${pullRequest.checksPassed}/${pullRequest.checksTotal} GitHub checks passed`,
              tone: "warning" as const,
            }
          : {
              label: `${pullRequest.checksPassed}/${pullRequest.checksTotal} checks passed`,
              detail: "GitHub checks successful",
              tone: "success" as const,
            };
  const mergeSignal = pullRequest.draft
    ? {
        label: "Draft pull request",
        detail: "Mark ready before requesting merge",
        tone: "neutral" as const,
      }
    : pullRequest.reviewDecision === "changes_requested"
      ? {
          label: "Changes requested",
          detail: "A reviewer is waiting for updates",
          tone: "warning" as const,
        }
      : pullRequest.mergeability === "conflicting"
        ? {
            label: "Merge conflict",
            detail: "Resolve branch conflicts before merging",
            tone: "warning" as const,
          }
        : pullRequest.githubReady
          ? {
              label: "GitHub permits merge",
              detail: "Reviews and mergeability are clear",
              tone: "success" as const,
            }
          : pullRequest.reviewDecision === "review_required"
            ? {
                label: "Review required",
                detail: "Waiting for an approving review",
                tone: "warning" as const,
              }
            : {
                label: "GitHub blocks merge",
                detail: "Reviews or mergeability need attention",
                tone: "warning" as const,
              };

  return [
    {
      label: exactHeadSnapshot
        ? "Snapshot linked"
        : pullRequest.snapshotId
          ? "Snapshot out of date"
          : "Snapshot required",
      detail: exactHeadSnapshot
        ? "Current product changes are recorded"
        : pullRequest.snapshotId
          ? "Linked evidence does not match the PR head"
          : "No current product record is linked",
      tone: exactHeadSnapshot ? "success" : "warning",
    },
    checkSignal,
    {
      label: !exactHeadSnapshot
        ? "Scope not evaluated"
        : !pullRequest.planValid
          ? "Plan revision invalid"
          : pullRequest.scopeDrift
            ? "Scope drift detected"
            : !pullRequest.planAligned
              ? "Scope not confirmed"
              : "Within plan scope",
      detail: !exactHeadSnapshot
        ? "Requires current product evidence"
        : !pullRequest.planValid
          ? "The locked plan record is no longer valid"
          : pullRequest.planAligned
            ? "Compared with the declared plan"
            : "Plan conformance evidence is incomplete",
      tone: !exactHeadSnapshot
        ? "neutral"
        : !pullRequest.planValid ||
            pullRequest.scopeDrift ||
            !pullRequest.planAligned
          ? "warning"
          : "success",
    },
    mergeSignal,
  ];
}

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
            <MetaPill icon={Code2} label={`${latest.commitCount} ${latest.commitCount === 1 ? "commit" : "commits"}`} />
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

export function PullRequestsPanel({
  collection,
  loading,
  repository,
}: {
  collection: PullRequestCollection;
  loading: boolean;
  repository: RepositorySummary;
}): React.JSX.Element {
  const [showOtherPullRequests, setShowOtherPullRequests] = useState(false);

  if (loading) return <LoadingState label="Loading pull requests" />;
  if (!collection.connected) {
    return (
      <section className="rounded-lg bg-white p-6 shadow-[var(--shadow-border)] sm:p-8">
        <div className="grid size-10 place-items-center rounded-full bg-slate-100 text-slate-600">
          <GitPullRequest className="size-5" />
        </div>
        <h2 className="mt-5 text-lg font-semibold">Connect GitHub pull requests</h2>
        <p className="mt-2 max-w-[62ch] text-sm leading-6 text-muted-foreground">
          {collection.reason ||
            "eve could not read pull requests for this repository."}
        </p>
        {repository.remoteUrl ? (
          <Button asChild variant="outline" className="mt-5">
            <a href={repository.remoteUrl} target="_blank" rel="noreferrer">
              Open repository on GitHub
              <ExternalLink className="size-4" />
            </a>
          </Button>
        ) : null}
      </section>
    );
  }

  const open = collection.pullRequests.filter(
    (pullRequest) => pullRequest.state === "open",
  );
  if (open.length === 0 && collection.openCount === 0) {
    return (
      <section className="rounded-lg bg-white p-6 shadow-[var(--shadow-border)] sm:p-8">
        <GitPullRequest className="size-5 text-slate-500" />
        <h2 className="mt-4 text-lg font-semibold">No pull requests yet</h2>
        <p className="mt-2 max-w-[62ch] text-sm text-muted-foreground">
          Pull requests will appear here once a branch is opened against this
          repository.
        </p>
      </section>
    );
  }

  const { contextual, hidden } = partitionOpenPullRequests(open);
  const truncated = open.length < collection.openCount;
  const disclosureCopy = pullRequestDisclosureCopy(
    hidden.length,
    collection.openCount,
    truncated,
    showOtherPullRequests,
  );
  return (
    <div className="space-y-4">
      <PullRequestGroup
        title="eve review queue"
        detail={`${contextual.length}${truncated ? " loaded" : ""} ${
          contextual.length === 1 ? "pull request" : "pull requests"
        } with context`}
        pullRequests={contextual}
        repository={repository.name}
        empty={
          truncated
            ? `None of the ${open.length} loaded open pull requests has a current Snapshot. eve has not classified the remaining ${
                collection.openCount - open.length
              }.`
            : `${collection.openCount} open ${
                collection.openCount === 1
                  ? "pull request is"
                  : "pull requests are"
              } waiting for a current Snapshot before eve can review them.`
        }
      />
      {hidden.length > 0 ? (
        <>
          <button
            type="button"
            aria-expanded={showOtherPullRequests}
            aria-controls="other-open-pull-requests"
            onClick={() => setShowOtherPullRequests((current) => !current)}
            className="group flex w-full items-center gap-4 rounded-lg border bg-white px-5 py-4 text-left transition-colors hover:border-slate-300 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-600 focus-visible:ring-offset-2"
          >
            <span className="grid size-9 shrink-0 place-items-center rounded-full bg-slate-100 text-slate-600">
              <GitPullRequest className="size-4" />
            </span>
            <span className="min-w-0 flex-1">
              <span className="block text-sm font-semibold text-slate-950">
                {disclosureCopy.label}
              </span>
              <span className="mt-0.5 block text-xs leading-5 text-muted-foreground">
                {disclosureCopy.detail}
              </span>
            </span>
            <span className="inline-flex shrink-0 items-center gap-2 text-xs font-medium text-blue-700">
              {showOtherPullRequests ? "Hide" : "Show"}
              <ChevronDown
                className={`size-4 transition-transform ${
                  showOtherPullRequests ? "rotate-180" : ""
                }`}
              />
            </span>
          </button>
          <div id="other-open-pull-requests" hidden={!showOtherPullRequests}>
            <PullRequestGroup
              title="Other open pull requests"
              detail={`${hidden.length} without current eve context`}
              pullRequests={hidden}
              repository={repository.name}
              empty="No other open pull requests."
            />
          </div>
        </>
      ) : null}
      {truncated && repository.remoteUrl ? (
        <p className="flex flex-wrap items-center justify-between gap-3 rounded-lg border bg-white px-5 py-4 text-sm text-muted-foreground">
          eve shows the 50 most recently updated pull requests here.
          <a
            href={`${repository.remoteUrl}/pulls`}
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center gap-2 font-medium text-blue-700 hover:text-blue-800"
          >
            Browse all {collection.openCount} on GitHub
            <ExternalLink className="size-3.5" />
          </a>
        </p>
      ) : null}
    </div>
  );
}

const RECENT_PULL_REQUEST_DAYS = 14;

export function hasCurrentSnapshotContext(
  pullRequest: PullRequestSummary,
): boolean {
  return (
    Boolean(pullRequest.snapshotId) &&
    pullRequest.snapshotHeadMatch &&
    (pullRequest.planRevision ?? 0) > 0
  );
}

export function partitionOpenPullRequests(
  pullRequests: PullRequestSummary[],
): {
  contextual: PullRequestSummary[];
  hidden: PullRequestSummary[];
} {
  return pullRequests.reduce<{
    contextual: PullRequestSummary[];
    hidden: PullRequestSummary[];
  }>(
    (groups, pullRequest) => {
      if (hasCurrentSnapshotContext(pullRequest)) {
        groups.contextual.push(pullRequest);
      } else {
        groups.hidden.push(pullRequest);
      }
      return groups;
    },
    { contextual: [], hidden: [] },
  );
}

export function pullRequestDisclosureCopy(
  hiddenCount: number,
  openCount: number,
  truncated: boolean,
  expanded: boolean,
): { label: string; detail: string } {
  const singular = hiddenCount === 1;

  if (expanded) {
    return {
      label: `Hide ${hiddenCount} other ${
        singular ? "pull request" : "pull requests"
      }`,
      detail: `${hiddenCount} shown for reference; ${
        singular ? "it does" : "they do"
      } not have current eve context.`,
    };
  }

  return {
    label: truncated
      ? `Show ${hiddenCount} other loaded ${
          singular ? "pull request" : "pull requests"
        }`
      : openCount === 1
        ? "Show 1 open pull request"
        : `Show all ${openCount} open pull requests`,
    detail: `${hiddenCount} ${singular ? "is" : "are"} hidden because ${
      singular ? "it does" : "they do"
    } not have current eve context.`,
  };
}

export function recentOpenPullRequest(
  pullRequests: PullRequestSummary[],
  now = Date.now(),
): PullRequestSummary | undefined {
  const cutoff = now - RECENT_PULL_REQUEST_DAYS * 24 * 60 * 60 * 1000;
  return pullRequests
    .filter((pullRequest) => {
      if (
        pullRequest.state !== "open" ||
        !hasCurrentSnapshotContext(pullRequest)
      ) {
        return false;
      }
      const createdAt = new Date(pullRequest.createdAt).getTime();
      return Number.isFinite(createdAt) && createdAt >= cutoff;
    })
    .sort((left, right) => right.createdAt.localeCompare(left.createdAt))[0];
}

function PullRequestGroup({
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

function PullRequestStateBadge({
  pullRequest,
}: {
  pullRequest: PullRequestSummary;
}): React.JSX.Element {
  if (pullRequest.draft) return <Badge variant="secondary">Draft</Badge>;
  return <Badge variant="success">Open</Badge>;
}
