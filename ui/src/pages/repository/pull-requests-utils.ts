import type { PullRequestSummary } from "../../types";

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
