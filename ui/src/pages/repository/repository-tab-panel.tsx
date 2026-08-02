import type {
  DetailResponse,
  EvolutionSummary,
  PullRequestCollection,
  RepositorySummary,
} from "../../types";
import { ActivityAccordionCard } from "./activity-accordion-card";
import { ArtifactsPanel } from "./artifacts-panel";
import { EvolutionTimelineCard } from "./evolution-timeline-card";
import { LatestSnapshotCard } from "./latest-snapshot-card";
import { PullRequestsPanel } from "./pull-requests-panel";
import { ReadmePanel } from "./readme-panel";
import { ReadyPullRequestBanner } from "./ready-pull-request-banner";
import { RepositoryCodePanel } from "./repository-code-panel";
import { RepositoryComparePanel } from "./comparison-panel";
import { recentOpenPullRequest } from "./pull-requests-utils";
import type { RepositoryTab } from "./types";

export function RepositoryTabPanel({
  activeTab,
  repository,
  latest,
  evolutions,
  details,
  pullRequests,
  pullRequestsLoading,
}: {
  activeTab: RepositoryTab;
  repository: RepositorySummary;
  latest?: EvolutionSummary;
  evolutions: EvolutionSummary[];
  details: DetailResponse[];
  pullRequests: PullRequestCollection;
  pullRequestsLoading: boolean;
}): React.JSX.Element {
  const recentPullRequest = recentOpenPullRequest(pullRequests.pullRequests);
  const latestPullRequest = latest
    ? pullRequests.pullRequests.find(
        (pullRequest) => pullRequest.snapshotId === latest.id,
      )
    : undefined;

  return (
    <div
      id={`repository-tab-${activeTab}`}
      role="tabpanel"
      aria-labelledby={`repository-tab-trigger-${activeTab}`}
      className="px-4 py-6 sm:px-6 lg:px-8"
    >
      {activeTab === "overview" ? (
        <div className="space-y-5">
          {recentPullRequest ? (
            <ReadyPullRequestBanner
              pullRequest={recentPullRequest}
              repository={repository.name}
            />
          ) : null}
          <LatestSnapshotCard
            latest={latest}
            pullRequest={latestPullRequest}
            repository={repository.name}
          />

          <div className="grid grid-cols-1 items-start gap-5 xl:grid-cols-[minmax(0,1fr)_300px] 2xl:grid-cols-[minmax(0,1fr)_326px]">
            <ReadmePanel repository={repository} />
            <EvolutionTimelineCard evolutions={evolutions} />
          </div>
        </div>
      ) : null}

      {activeTab === "snapshots" ? (
        <EvolutionTimelineCard evolutions={evolutions} spacious />
      ) : null}

      {activeTab === "pull-requests" ? (
        <PullRequestsPanel
          collection={pullRequests}
          loading={pullRequestsLoading}
          repository={repository}
        />
      ) : null}

      {activeTab === "compare" ? (
        <RepositoryComparePanel evolutions={evolutions} />
      ) : null}

      {activeTab === "code" ? (
        <RepositoryCodePanel repository={repository} evolutions={evolutions} />
      ) : null}

      {activeTab === "activity" ? (
        <ActivityAccordionCard
          repository={repository}
          evolutions={evolutions}
          details={details}
        />
      ) : null}

      {activeTab === "artifacts" ? (
        <ArtifactsPanel repository={repository} details={details} />
      ) : null}
    </div>
  );
}
