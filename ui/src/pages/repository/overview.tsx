import { Link } from "@tanstack/react-router";
import {
  BookOpen,
  Code2,
  Copy,
  ExternalLink,
  FileText,
} from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "../../components/ui/button";
import { MarkdownViewer } from "../../components/markdown-viewer";
import { SnapshotCodeBrowser } from "../../components/snapshot-code-browser";
import { StatusBadge } from "../../components/status-badge";
import { compactDate } from "../../format";
import type {
  DetailResponse,
  EvolutionSummary,
  PullRequestCollection,
  RepositorySummary,
} from "../../types";
import { ActivityAccordionCard } from "./activity";
import { ArtifactsPanel } from "./artifacts";
import { RepositoryComparePanel } from "./comparison";
import {
  RepositoryHeader,
  useRepositoryDescription,
} from "./header";
import {
  LatestSnapshotCard,
  PullRequestsPanel,
  ReadyPullRequestBanner,
  recentOpenPullRequest,
} from "./pull-requests";
import { RepositoryRightRail } from "./sidebar";
import { buildContributors, buildRepositoryStats, repositoryTabs, type RepositoryTab } from "./types";

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

function RepositoryTabPanel({
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

function ReadmePanel({
  repository,
}: {
  repository: RepositorySummary;
}): React.JSX.Element {
  const [copied, setCopied] = useState(false);
  const copyReadme = async () => {
    await navigator.clipboard.writeText(repository.readme || "");
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  };

  return (
    <section className="overflow-hidden rounded-lg bg-white shadow-[0_0_0_1px_rgba(15,23,42,0.1)] xl:flex xl:h-[576px] xl:flex-col">
      <div className="flex min-h-14 items-center justify-between gap-3 border-b px-5">
        <h2 className="flex min-w-0 items-center gap-2 text-sm font-semibold">
          <FileText className="size-4 text-slate-500" />
          README.md
        </h2>
        <div className="flex shrink-0 items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="gap-2"
            disabled={!repository.readme}
            onClick={copyReadme}
          >
            <Copy className="size-3.5" />
            {copied ? "Copied" : "Copy"}
          </Button>
          <Button asChild variant="outline" size="sm" className="gap-2">
            <a href="#readme-raw">
              View raw
              <ExternalLink className="size-3.5" />
            </a>
          </Button>
        </div>
      </div>
      <div
        id="readme-raw"
        className="max-h-[520px] overflow-y-auto px-5 py-5 sm:px-6 sm:py-6 xl:min-h-0 xl:flex-1"
      >
        {repository.readme ? (
          <MarkdownViewer
            content={repository.readme}
            surface="bare"
            className="pr-2"
          />
        ) : (
          <p className="text-sm text-muted-foreground">
            No README found in this repository.
          </p>
        )}
      </div>
    </section>
  );
}

function EvolutionTimelineCard({
  evolutions,
  spacious = false,
}: {
  evolutions: EvolutionSummary[];
  spacious?: boolean;
}): React.JSX.Element {
  return (
    <section
      className={`rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)] ${
        spacious ? "" : "xl:flex xl:h-[576px] xl:flex-col"
      }`}
    >
      <div className="mb-5 flex items-center gap-2">
        <h2 className="text-base font-semibold">Snapshot timeline</h2>
      </div>
      {evolutions.length === 0 ? (
        <p className="text-sm text-muted-foreground">No timeline entries yet.</p>
      ) : (
        <div
          className={`relative space-y-0 overflow-y-auto overscroll-contain pl-8 pr-1 ${
            spacious ? "max-h-[680px]" : "max-h-[520px] xl:min-h-0 xl:flex-1"
          }`}
        >
          {evolutions.map((evolution, index) => (
            <Link
              key={evolution.id}
              to="/snapshots/$id"
              params={{ id: evolution.id }}
              className="group relative block pb-7 last:pb-0"
            >
              {index < evolutions.length - 1 ? (
                <span className="absolute -left-[20px] top-5 h-full w-px bg-blue-600" />
              ) : null}
              <span className="absolute -left-[26px] top-1 flex size-3.5 rounded-full border-2 border-blue-600 bg-white ring-4 ring-blue-50" />
              <span className="flex min-w-0 flex-wrap items-start justify-between gap-3">
                <strong className="max-w-[26ch] text-sm font-semibold leading-5 text-balance group-hover:text-blue-700">
                  {evolution.title}
                </strong>
                <StatusBadge status={evolution.status} />
              </span>
              <span className="mt-2 block text-xs leading-5 text-muted-foreground">
                {evolution.type} · {compactDate(evolution.updatedAt || evolution.createdAt)}
              </span>
            </Link>
          ))}
        </div>
      )}
    </section>
  );
}

function RepositoryCodePanel({
  repository,
  evolutions,
}: {
  repository: RepositorySummary;
  evolutions: EvolutionSummary[];
}): React.JSX.Element {
  const [selectedId, setSelectedId] = useState(evolutions[0]?.id ?? "");

  useEffect(() => {
    if (evolutions.length === 0) {
      setSelectedId("");
      return;
    }
    if (!evolutions.some((evolution) => evolution.id === selectedId)) {
      setSelectedId(evolutions[0].id);
    }
  }, [evolutions, selectedId]);

  const selected = evolutions.find((evolution) => evolution.id === selectedId);

  if (evolutions.length === 0) {
    return (
      <section className="rounded-lg bg-card p-5 shadow-[var(--shadow-border)]">
        <h2 className="flex items-center gap-2 text-base font-semibold">
          <Code2 className="size-4 text-blue-600" />
          Code
        </h2>
        <p className="mt-3 text-sm text-muted-foreground">
          Record a Snapshot in this repository to inspect the code behind it.
        </p>
      </section>
    );
  }

  return (
    <section className="space-y-4">
      <section className="rounded-lg bg-card p-5 shadow-[var(--shadow-border)] sm:p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div className="max-w-[68ch]">
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              Snapshot code
            </p>
            <h2 className="mt-2 flex items-center gap-2 text-xl font-semibold text-balance">
              <Code2 className="size-5 text-blue-600" />
              Inspect code by Snapshot
            </h2>
            <p className="mt-3 text-sm leading-6 text-muted-foreground text-pretty">
              Choose a Snapshot from this repository and inspect the relevant files without leaving the repository view.
            </p>
          </div>
          {selected ? (
            <div className="flex flex-wrap gap-2">
              <Button asChild variant="outline" size="sm">
                <Link to="/snapshots/$id/code" params={{ id: selected.id }}>
                  <Code2 className="size-3.5" />
                  Full code view
                </Link>
              </Button>
              <Button asChild variant="ghost" size="sm">
                <Link to="/snapshots/$id" params={{ id: selected.id }}>
                  <ExternalLink className="size-3.5" />
                  Snapshot page
                </Link>
              </Button>
            </div>
          ) : null}
        </div>

        <div className="mt-5 grid gap-2 sm:max-w-xl">
          <label
            htmlFor="repository-code-snapshot"
            className="text-xs font-semibold uppercase tracking-wide text-muted-foreground"
          >
            Snapshot
          </label>
          <select
            id="repository-code-snapshot"
            value={selectedId}
            onChange={(event) => setSelectedId(event.target.value)}
            className="h-10 rounded-md border bg-background px-3 text-sm font-medium text-foreground shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {evolutions.map((evolution) => (
              <option key={evolution.id} value={evolution.id}>
                {evolution.title || evolution.id} - {compactDate(evolution.updatedAt || evolution.createdAt)}
              </option>
            ))}
          </select>
        </div>
      </section>

      {selected ? (
        <SnapshotCodeBrowser snapshotId={selected.id} repository={repository.name} />
      ) : null}
    </section>
  );
}
