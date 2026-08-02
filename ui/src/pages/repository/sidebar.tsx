import type { RepositorySummary } from "../../types";
import { ContributorCard } from "./contributor-card";
import { RepositoryFactsCard } from "./repository-facts-card";
import { RepositoryLinksCard } from "./repository-links-card";
import { SnapshotSummaryCard } from "./snapshot-summary-card";
import type { ContributorRow, RepositoryStats } from "./types";

export function RepositoryRightRail({
  repository,
  description,
  onDescriptionChange,
  stats,
  contributors,
}: {
  repository: RepositorySummary;
  description: string;
  onDescriptionChange: (value: string) => void;
  stats: RepositoryStats;
  contributors: ContributorRow[];
}): React.JSX.Element {
  return (
    <aside className="space-y-4 border-t px-4 py-6 sm:px-6 lg:px-8 xl:border-l xl:border-t-0 xl:px-6 xl:py-7">
      <RepositoryFactsCard
        repository={repository}
        description={description}
        onDescriptionChange={onDescriptionChange}
      />
      <SnapshotSummaryCard stats={stats} />
      <ContributorCard rows={contributors} />
      <RepositoryLinksCard repository={repository} />
    </aside>
  );
}
