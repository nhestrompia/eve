import { useMutation } from "@tanstack/react-query";
import {
  Box,
  Calendar,
  Code2,
  Copy,
  Edit3,
  ExternalLink,
  GitBranch,
  HardDrive,
  Package,
  Save,
  X,
} from "lucide-react";
import { useEffect, useState } from "react";

import { api } from "../../api";
import { Button } from "../../components/ui/button";
import { compactDate } from "../../format";
import type { RepositorySummary } from "../../types";
import { DEFAULT_REPOSITORY_DESCRIPTION } from "./header";
import {
  agentAvatarPath,
  formatBytes,
  REPOSITORY_TONES,
  type ContributorRow,
  type RailCardProps,
  type RepositoryStats,
} from "./types";

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

function RepositoryFactsCard({
  repository,
  description,
  onDescriptionChange,
}: {
  repository: RepositorySummary;
  description: string;
  onDescriptionChange: (value: string) => void;
}): React.JSX.Element {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(description);
  useEffect(() => setDraft(description), [description]);
  const rows = [
    ["Language", repository.primaryLanguage || "Unknown", Code2],
    ["Size", formatBytes(repository.sizeBytes), HardDrive],
    ["Created", compactDate(repository.createdAt), Calendar],
  ] as const;

  return (
    <RailCard title="Repository overview">
      <div className="space-y-5">
        <div className="grid grid-cols-[18px_minmax(0,1fr)] gap-3">
          <Box className="mt-0.5 size-4 text-slate-500" />
          <div className="min-w-0">
            <div className="flex items-center justify-between gap-3">
              <p className="text-xs font-medium text-muted-foreground">
                Description
              </p>
              <button
                type="button"
                className="inline-flex size-7 items-center justify-center rounded-md text-slate-500 hover:bg-slate-50 hover:text-slate-950"
                onClick={() => setEditing((value) => !value)}
                aria-label={
                  editing ? "Cancel description edit" : "Edit description"
                }
              >
                {editing ? (
                  <X className="size-3.5" />
                ) : (
                  <Edit3 className="size-3.5" />
                )}
              </button>
            </div>
            {editing ? (
              <form
                className="mt-2 space-y-2"
                onSubmit={(event) => {
                  event.preventDefault();
                  const value = draft.trim() || DEFAULT_REPOSITORY_DESCRIPTION;
                  onDescriptionChange(value);
                  setEditing(false);
                }}
              >
                <textarea
                  value={draft}
                  onChange={(event) => setDraft(event.target.value)}
                  className="min-h-24 w-full resize-y rounded-md border bg-white px-3 py-2 text-sm leading-5 text-slate-700 outline-none focus:ring-2 focus:ring-blue-500"
                />
                <div className="flex justify-end gap-2">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setDraft(description);
                      setEditing(false);
                    }}
                  >
                    Cancel
                  </Button>
                  <Button type="submit" size="sm" className="gap-2">
                    <Save className="size-3.5" />
                    Save
                  </Button>
                </div>
              </form>
            ) : (
              <p className="mt-1 text-sm leading-5 text-slate-700 text-pretty">
                {description}
              </p>
            )}
          </div>
        </div>
        {rows.map(([label, value, Icon]) => (
          <div
            key={label}
            className="grid grid-cols-[18px_minmax(0,1fr)] gap-3"
          >
            <Icon className="mt-0.5 size-4 text-slate-500" />
            <div className="min-w-0">
              <p className="text-xs font-medium text-muted-foreground">
                {label}
              </p>
              <p className="mt-1 text-sm leading-5 text-slate-700 text-pretty">
                {value}
              </p>
            </div>
          </div>
        ))}
      </div>
    </RailCard>
  );
}

function SnapshotSummaryCard({ stats }: { stats: RepositoryStats }): React.JSX.Element {
  const tiles = [
    ["Snapshots", stats.snapshots],
    ["Features", stats.features],
    ["Bug fixes", stats.bugfixes],
    ["Refactor", stats.refactors],
    ["Commits", stats.commits],
    ["Decisions", stats.decisions],
    ["Validated", stats.validated],
    ["Risks", stats.risks],
  ] as const;
  return (
    <RailCard title="Snapshot summary">
      <div className="grid grid-cols-2 gap-2.5">
        {tiles.map(([label, value]) => (
          <div
            key={label}
            className="rounded-lg bg-white px-3 py-2.5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]"
          >
            <div className="text-xl font-semibold leading-6 tabular-nums">
              {value}
            </div>
            <div className="mt-1 text-xs text-muted-foreground">{label}</div>
          </div>
        ))}
      </div>
    </RailCard>
  );
}

function ContributorCard({ rows }: { rows: ContributorRow[] }): React.JSX.Element {
  const max = Math.max(1, ...rows.map((row) => row.count));
  return (
    <RailCard title="Top contributors" eyebrow="Last 30 days">
      <div className="space-y-4">
        {rows.map((row, index) => {
          const tone = REPOSITORY_TONES[index % REPOSITORY_TONES.length];
          return (
            <div
              key={row.label}
              className="grid grid-cols-[88px_minmax(0,1fr)_54px] items-center gap-3"
            >
              <span className="flex min-w-0 items-center gap-2 text-sm font-semibold">
                <img
                  src={agentAvatarPath(row.label)}
                  alt=""
                  className="size-6 rounded-lg"
                />
                <span className="truncate">{row.label}</span>
              </span>
              <span className="h-1.5 overflow-hidden rounded-full bg-slate-100">
                <span
                  className={`block h-full rounded-full ${tone.bg}`}
                  style={{ width: `${Math.max(8, (row.count / max) * 100)}%` }}
                />
              </span>
              <span className="text-right text-sm text-muted-foreground tabular-nums">
                {row.count}
              </span>
            </div>
          );
        })}
      </div>
    </RailCard>
  );
}

function RepositoryLinksCard({
  repository,
}: {
  repository: RepositorySummary;
}): React.JSX.Element {
  const [copied, setCopied] = useState(false);
  const openEditor = useMutation({
    mutationFn: () => api.openRepositoryInEditor(repository.name),
  });
  const copyPath = async () => {
    await navigator.clipboard.writeText(repository.root || repository.name);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  };
  return (
    <RailCard id="links" title="Repository links">
      <div className="space-y-3">
        {repository.remoteUrl ? (
          <a
            className="flex min-h-9 items-center gap-3 rounded-md px-1 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50 hover:text-slate-950"
            href={repository.remoteUrl}
            target="_blank"
            rel="noreferrer"
          >
            <GitBranch className="size-4" />
            Open in GitHub
            <ExternalLink className="ml-auto size-4 text-slate-500" />
          </a>
        ) : null}
        <button
          className="flex min-h-9 w-full items-center gap-3 rounded-md px-1 text-left text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50 hover:text-slate-950 disabled:cursor-not-allowed disabled:opacity-60"
          disabled={openEditor.isPending}
          onClick={() => openEditor.mutate()}
          title={openEditor.data?.stderr || "Open repository in editor"}
        >
          <Code2 className="size-4" />
          {openEditor.isPending ? "Opening in editor" : "Open in editor"}
          <ExternalLink className="ml-auto size-4 text-slate-500" />
        </button>
        <button
          className="flex min-h-9 w-full items-center gap-3 rounded-md px-1 text-left text-sm font-medium text-slate-700 transition-colors hover:bg-slate-50 hover:text-slate-950"
          onClick={copyPath}
        >
          <Copy className="size-4" />
          {copied ? "Copied path" : "Copy local path"}
          <span className="ml-auto max-w-[150px] truncate font-mono text-xs text-muted-foreground">
            {repository.root}
          </span>
        </button>
      </div>
    </RailCard>
  );
}

export function RailCard({
  id,
  title,
  eyebrow,
  children,
}: RailCardProps): React.JSX.Element {
  return (
    <section
      id={id}
      className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]"
    >
      <div className="mb-5 flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold">{title}</h2>
        {eyebrow ? (
          <span className="flex items-center gap-1 text-xs font-medium text-muted-foreground">
            <Package className="size-3" />
            {eyebrow}
          </span>
        ) : null}
      </div>
      {children}
    </section>
  );
}
