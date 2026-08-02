import { Calendar, Code2, GitBranch } from "lucide-react";

import { Badge } from "../../components/ui/badge";
import { compactDate, shortCommit } from "../../format";
import type { RepositorySummary } from "../../types";
import { MetaPill } from "./meta-pill";
import { RepositoryHeaderMark } from "./repository-header-mark";
import { RepositoryTabNavigation } from "./repository-tab-navigation";
import type { RepositoryTab } from "./types";

export function RepositoryHeader({
  repository,
  description,
  activeTab,
  tabs,
  onActiveTabChange,
}: {
  repository: RepositorySummary;
  description: string;
  activeTab: RepositoryTab;
  tabs: Array<{ id: RepositoryTab; label: string; count?: number }>;
  onActiveTabChange: (tab: RepositoryTab) => void;
}): React.JSX.Element {
  return (
    <section className="bg-white px-4 pt-7 sm:px-6 lg:px-8">
      <div className="flex min-w-0 flex-col gap-5 sm:flex-row sm:items-start">
        <RepositoryHeaderMark repository={repository} />
        <div className="min-w-0 pb-6">
          <div className="flex min-w-0 flex-wrap items-center gap-3">
            <h1 className="truncate text-[28px] font-semibold leading-tight tracking-normal text-slate-950">
              {repository.name}
            </h1>
            <Badge variant={repository.remoteUrl ? "success" : "secondary"}>
              {repository.remoteUrl ? "Remote" : "Local"}
            </Badge>
          </div>
          <p className="mt-2 max-w-[62ch] text-sm leading-6 text-muted-foreground">
            {description}
          </p>
          <div className="mt-5 flex flex-wrap gap-2">
            <MetaPill
              icon={GitBranch}
              label={repository.branch || "branch unknown"}
            />
            <MetaPill icon={Code2} label={shortCommit(repository.head)} />
            <MetaPill
              icon={Calendar}
              label={
                repository.latestAt
                  ? `Updated ${compactDate(repository.latestAt)}`
                  : "No snapshots"
              }
            />
          </div>
        </div>
      </div>
      <RepositoryTabNavigation
        activeTab={activeTab}
        tabs={tabs}
        onActiveTabChange={onActiveTabChange}
      />
    </section>
  );
}
