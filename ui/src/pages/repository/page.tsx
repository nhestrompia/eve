import { useQuery } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { useEffect, useState } from "react";

import { api } from "../../api";
import { EvolutionShell } from "../../components/evolution-shell";
import { ErrorState } from "../../components/error-state";
import { LoadingState } from "../../components/loading-state";
import type { RepositoryTab } from "./types";
import { repositoryTabFromHash, shouldLoadRepositoryDetails } from "./types";
import { RepositoryOverviewPage } from "./overview";

export function RepositoryPage(): React.JSX.Element {
  const { repo } = useParams({ from: "/repositories/$repo" });
  const [activeTab, setActiveTab] = useState<RepositoryTab>(() =>
    repositoryTabFromHash(),
  );
  const evolutions = useQuery({
    queryKey: ["snapshots", repo],
    queryFn: () => api.snapshots(repo),
    staleTime: 30_000,
  });
  const repository = useQuery({
    queryKey: ["repository", repo],
    queryFn: () => api.repository(repo),
    staleTime: 30_000,
  });
  const pullRequests = useQuery({
    queryKey: ["pull-requests", repo],
    queryFn: () => api.pullRequests(repo),
    staleTime: 30_000,
  });
  const details = useQuery({
    queryKey: [
      "repository-page-details",
      repo,
      evolutions.data?.map((evolution) => evolution.id).join(",") ?? "",
    ],
    queryFn: () =>
      Promise.all(
        (evolutions.data ?? []).map((evolution) =>
          api.snapshotDetail(evolution.id, repo),
        ),
      ),
    enabled: shouldLoadRepositoryDetails(
      activeTab,
      evolutions.data?.length ?? 0,
    ),
    staleTime: 30_000,
  });

  useEffect(() => {
    const syncTab = () => setActiveTab(repositoryTabFromHash());
    window.addEventListener("hashchange", syncTab);
    return () => window.removeEventListener("hashchange", syncTab);
  }, []);

  return (
    <EvolutionShell
      evolutions={[]}
      selectedId={undefined}
      showHistoryRail={false}
      contentClassName="p-0 sm:p-0 lg:p-0"
    >
      {evolutions.isLoading || repository.isLoading ? (
        <LoadingState label={`Loading ${repo}`} />
      ) : null}
      {evolutions.error ? <ErrorState error={evolutions.error} /> : null}
      {repository.error ? <ErrorState error={repository.error} /> : null}
      {evolutions.data && repository.data ? (
        <RepositoryOverviewPage
          repository={repository.data}
          evolutions={evolutions.data}
          details={details.data ?? []}
          activeTab={activeTab}
          onActiveTabChange={setActiveTab}
          pullRequests={
            pullRequests.data ?? {
              connected: false,
              repository: repo,
              openCount: 0,
              reason: pullRequests.isLoading
                ? "Loading pull requests…"
                : "Pull requests are unavailable.",
              pullRequests: [],
            }
          }
          pullRequestsLoading={pullRequests.isLoading}
        />
      ) : null}
    </EvolutionShell>
  );
}
