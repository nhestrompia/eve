import { useEffect, useMemo, useState } from "react";
import {
  Calendar,
  Code2,
  GitBranch,
  type LucideIcon,
} from "lucide-react";

import { Badge } from "../../components/ui/badge";
import { compactDate, shortCommit } from "../../format";
import type { RepositorySummary } from "../../types";
import type { RepositoryTab } from "./types";

export const DEFAULT_REPOSITORY_DESCRIPTION =
  "Track product states, snapshots, sessions, and verification recorded for this repository.";

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

function RepositoryTabNavigation({
  activeTab,
  tabs,
  onActiveTabChange,
}: {
  activeTab: RepositoryTab;
  tabs: Array<{ id: RepositoryTab; label: string; count?: number }>;
  onActiveTabChange: (tab: RepositoryTab) => void;
}): React.JSX.Element {
  return (
    <div
      className="flex gap-7 overflow-x-auto overflow-y-hidden text-sm font-medium text-muted-foreground"
      role="tablist"
      aria-label="Repository sections"
    >
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={activeTab === tab.id}
          aria-controls={`repository-tab-${tab.id}`}
          id={`repository-tab-trigger-${tab.id}`}
          data-state={activeTab === tab.id ? "active" : "inactive"}
          onClick={() => {
            onActiveTabChange(tab.id);
            window.history.replaceState(
              null,
              "",
              tab.id === "overview"
                ? window.location.pathname
                : `#${tab.id}`,
            );
          }}
          className={`relative inline-flex min-h-12 shrink-0 items-center gap-2 rounded-t-lg px-3 text-left transition-colors after:absolute after:inset-x-0 after:bottom-0 after:h-0.5 after:rounded-full hover:bg-slate-50 hover:text-foreground data-[state=active]:bg-white data-[state=active]:after:bg-blue-600 ${
            activeTab === tab.id
              ? "text-blue-700"
              : "text-muted-foreground after:bg-transparent"
          }`}
        >
          <span>{tab.label}</span>
          {tab.count !== undefined ? (
            <span
              className={`rounded-full px-2 py-0.5 text-xs ${
                activeTab === tab.id
                  ? "bg-blue-50 text-blue-700"
                  : "bg-slate-100 text-slate-500"
              }`}
            >
              {tab.count}
            </span>
          ) : null}
        </button>
      ))}
    </div>
  );
}

export function RepositoryHeaderMark({
  repository,
}: {
  repository: RepositorySummary;
}): React.JSX.Element {
  if (repository.name.toLowerCase() === "eve") {
    return (
      <div className="flex h-[70px] w-[170px] shrink-0 items-center rounded-lg bg-white px-3 ring-1 ring-inset ring-slate-200">
        <img
          src="/eve.svg"
          alt="eve"
          className="eve-logo h-full w-full object-contain object-left"
        />
      </div>
    );
  }

  return (
    <div
      className="grid h-[70px] w-[170px] shrink-0 place-items-center rounded-lg bg-slate-950 text-white ring-1 ring-inset ring-slate-800"
      aria-label={`${repository.name} repository mark`}
    >
      <span className="text-[34px] font-semibold uppercase leading-none tracking-normal">
        {repositoryInitial(repository.name)}
      </span>
    </div>
  );
}

export function repositoryInitial(name: string): string {
  const firstLetter = name.trim().match(/[A-Za-z0-9]/)?.[0];
  return (firstLetter ?? "?").toUpperCase();
}

export function useRepositoryDescription(
  repository: RepositorySummary,
): readonly [string, (value: string) => void] {
  const storageKey = useMemo(
    () => `eve:repository-description:${repository.name}`,
    [repository.name],
  );
  const [description, setDescription] = useState(
    DEFAULT_REPOSITORY_DESCRIPTION,
  );

  useEffect(() => {
    const saved = window.localStorage.getItem(storageKey);
    setDescription(saved?.trim() || DEFAULT_REPOSITORY_DESCRIPTION);
  }, [storageKey]);

  const saveDescription = (value: string) => {
    const next = value.trim() || DEFAULT_REPOSITORY_DESCRIPTION;
    window.localStorage.setItem(storageKey, next);
    setDescription(next);
  };

  return [description, saveDescription] as const;
}

export function MetaPill({
  icon: Icon,
  label,
  tone,
}: {
  icon: LucideIcon;
  label: string;
  tone?: "success" | "warning";
}): React.JSX.Element {
  return (
    <span
      className={`inline-flex h-8 max-w-full items-center gap-2 rounded-md bg-white px-3 text-xs font-medium shadow-[0_0_0_1px_rgba(15,23,42,0.12)] ${
        tone === "success"
          ? "text-emerald-700"
          : tone === "warning"
            ? "text-orange-700"
            : "text-slate-600"
      }`}
    >
      <Icon className="size-3.5 shrink-0" />
      <span className="truncate">{label}</span>
    </span>
  );
}
