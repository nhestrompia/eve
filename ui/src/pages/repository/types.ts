import type { ReactNode } from "react";

import type { EvolutionSummary } from "../../types";

export type RepositoryTab =
  | "overview"
  | "snapshots"
  | "pull-requests"
  | "code"
  | "compare"
  | "activity"
  | "artifacts";

export type RepositoryStats = {
  snapshots: number;
  features: number;
  bugfixes: number;
  refactors: number;
  commits: number;
  decisions: number;
  validated: number;
  risks: number;
};

export type ContributorRow = {
  label: string;
  count: number;
};

export type RailCardProps = {
  id?: string;
  title: string;
  eyebrow?: string;
  children: ReactNode;
};

export function shouldLoadRepositoryDetails(
  activeTab: RepositoryTab,
  snapshotCount: number,
): boolean {
  return (
    snapshotCount > 0 &&
    (activeTab === "activity" || activeTab === "artifacts")
  );
}

export function repositoryTabs(
  snapshotCount: number,
  pullRequestCount = 0,
): Array<{ id: RepositoryTab; label: string; count?: number }> {
  return [
    { id: "overview", label: "Overview" },
    { id: "snapshots", label: "Snapshots", count: snapshotCount },
    {
      id: "pull-requests",
      label: "Pull requests",
      count: pullRequestCount,
    },
    { id: "code", label: "Code" },
    { id: "compare", label: "Compare" },
    { id: "activity", label: "Activity" },
    { id: "artifacts", label: "Artifacts" },
  ];
}

export function repositoryTabFromHash(): RepositoryTab {
  if (typeof window === "undefined") return "overview";
  const value = window.location.hash.slice(1) as RepositoryTab;
  return [
    "snapshots",
    "pull-requests",
    "code",
    "compare",
    "activity",
    "artifacts",
  ].includes(value)
    ? value
    : "overview";
}

export function buildRepositoryStats(
  evolutions: EvolutionSummary[],
): RepositoryStats {
  return {
    snapshots: evolutions.length,
    features: evolutions.filter((evolution) => evolution.type === "feature")
      .length,
    bugfixes: evolutions.filter((evolution) => evolution.type === "bugfix")
      .length,
    refactors: evolutions.filter((evolution) => evolution.type === "refactor")
      .length,
    commits: evolutions.reduce(
      (total, evolution) => total + (evolution.commitCount ?? 0),
      0,
    ),
    decisions: evolutions.reduce(
      (total, evolution) => total + evolution.decisionCount,
      0,
    ),
    validated: evolutions.filter(
      (evolution) => evolution.verificationState === "passed",
    ).length,
    risks: evolutions.reduce(
      (total, evolution) => total + evolution.riskCount,
      0,
    ),
  };
}

export function buildContributors(
  evolutions: EvolutionSummary[],
): ContributorRow[] {
  const counts = new Map<string, number>();
  for (const evolution of evolutions) {
    const providers =
      evolution.sessionProviders.length > 0
        ? evolution.sessionProviders
        : ["Codex"];
    for (const provider of providers) {
      const label = normalizeProvider(provider);
      counts.set(label, (counts.get(label) ?? 0) + 1);
    }
  }
  return ["Codex", "Claude", "OpenCode", "Other"]
    .map((label) => ({ label, count: counts.get(label) ?? 0 }))
    .filter((row) => row.count > 0 || row.label === "Codex");
}

function normalizeProvider(value: string): string {
  const normalized = value.toLowerCase();
  if (normalized.includes("codex")) return "Codex";
  if (normalized.includes("claude")) return "Claude";
  if (normalized.includes("opencode")) return "OpenCode";
  return "Other";
}

export function agentAvatarPath(label: string): string {
  if (label === "Codex") return "/agents/codex.svg";
  if (label === "Claude") return "/agents/claude.svg";
  if (label === "OpenCode") return "/agents/opencode.svg";
  return "/agents/other.svg";
}

export function formatBytes(value?: number): string {
  if (!value) return "Unknown";
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

export type RepositoryTone = {
  bg: string;
  text: string;
  soft: string;
};

export const REPOSITORY_TONES: RepositoryTone[] = [
  { bg: "bg-blue-600", text: "text-blue-700", soft: "bg-blue-50" },
  { bg: "bg-emerald-500", text: "text-emerald-700", soft: "bg-emerald-50" },
  { bg: "bg-violet-600", text: "text-violet-700", soft: "bg-violet-50" },
  { bg: "bg-amber-500", text: "text-amber-700", soft: "bg-amber-50" },
];
