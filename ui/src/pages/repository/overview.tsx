import type {
  DetailResponse,
  EvolutionSummary,
  PullRequestCollection,
  RepositorySummary,
} from "../../types";
import { RepositoryHeader } from "./header";
import { useRepositoryDescription } from "./repository-description";
import { RepositoryTabPanel } from "./repository-tab-panel";
import { RepositoryRightRail } from "./sidebar";
import {
  buildContributors,
  buildRepositoryStats,
  repositoryTabs,
  type RepositoryTab,
} from "./types";

export function RepositoryOverviewPage({
  repository,
  evolutions,
  details,
  activeTab,
  onActiveTabChange,
  pullRequests,
  pullRequestsLoading,
}: {
  repository: RepositorySummary;
  evolutions: EvolutionSummary[];
  details: DetailResponse[];
  activeTab: RepositoryTab;
  onActiveTabChange: (tab: RepositoryTab) => void;
  pullRequests: PullRequestCollection;
  pullRequestsLoading: boolean;
}): React.JSX.Element {
  const latest = evolutions[0];
  const stats = buildRepositoryStats(evolutions);
  const contributors = buildContributors(evolutions);
  const [description, setDescription] = useRepositoryDescription(repository);
  const tabs = repositoryTabs(evolutions.length, pullRequests.openCount);

  return (
    <main className="min-h-[calc(100dvh-76px)] min-w-0 bg-background">
      <div className="grid min-h-[calc(100dvh-76px)] grid-cols-1 xl:grid-cols-[minmax(0,1fr)_350px]">
        <div className="min-w-0">
          <RepositoryHeader
            repository={repository}
            description={description}
            activeTab={activeTab}
            tabs={tabs}
            onActiveTabChange={onActiveTabChange}
          />
          <RepositoryTabPanel
            activeTab={activeTab}
            repository={repository}
            latest={latest}
            evolutions={evolutions}
            details={details}
            pullRequests={pullRequests}
            pullRequestsLoading={pullRequestsLoading}
          />
        </div>

        <RepositoryRightRail
          repository={repository}
          description={description}
          onDescriptionChange={setDescription}
          stats={stats}
          contributors={contributors}
        />
      </div>
    </main>
  );
}
