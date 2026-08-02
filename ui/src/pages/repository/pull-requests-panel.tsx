import {
  ChevronDown,
  ExternalLink,
  GitPullRequest,
} from "lucide-react";
import { useState } from "react";

import { LoadingState } from "../../components/loading-state";
import { Button } from "../../components/ui/button";
import type {
  PullRequestCollection,
  RepositorySummary,
} from "../../types";
import { PullRequestGroup } from "./pull-request-group";
import {
  partitionOpenPullRequests,
  pullRequestDisclosureCopy,
} from "./pull-requests-utils";

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
