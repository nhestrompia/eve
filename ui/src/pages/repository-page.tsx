import { useMutation, useQuery } from "@tanstack/react-query";
import { Link, useParams } from "@tanstack/react-router";
import {
  ArrowRight,
  BookOpen,
  Box,
  Calendar,
  Code2,
  Copy,
  Edit3,
  ExternalLink,
  FileText,
  GitBranch,
  HardDrive,
  History,
  Image as ImageIcon,
  Package,
  Save,
  Sparkles,
  X,
  type LucideIcon,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { api } from "../api";
import { ErrorState } from "../components/error-state";
import { EvolutionShell } from "../components/evolution-shell";
import { LoadingState } from "../components/loading-state";
import { MarkdownViewer } from "../components/markdown-viewer";
import { StatusBadge } from "../components/status-badge";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { compactDate, shortCommit } from "../format";
import type {
  DetailResponse,
  EvolutionSummary,
  RepositorySummary,
  SnapshotArtifact,
} from "../types";

export function RepositoryPage() {
  const { repo } = useParams({ from: "/repositories/$repo" });
  const allEvolutions = useQuery({
    queryKey: ["snapshots"],
    queryFn: () => api.snapshots(),
  });
  const evolutions = useQuery({
    queryKey: ["snapshots", repo],
    queryFn: () => api.snapshots(repo),
  });
  const repositories = useQuery({
    queryKey: ["repositories"],
    queryFn: api.repositories,
  });
  const repository = useQuery({
    queryKey: ["repository", repo],
    queryFn: () => api.repository(repo),
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
          api.snapshotDetail(evolution.id),
        ),
      ),
    enabled: (evolutions.data?.length ?? 0) > 0,
    staleTime: 30_000,
  });

  return (
    <EvolutionShell
      evolutions={allEvolutions.data ?? []}
      selectedId={undefined}
      showHistoryRail={false}
      contentClassName="p-0 sm:p-0 lg:p-0"
    >
      {evolutions.isLoading ||
      repositories.isLoading ||
      repository.isLoading ? (
        <LoadingState label={`Loading ${repo}`} />
      ) : null}
      {evolutions.error ? <ErrorState error={evolutions.error} /> : null}
      {repositories.error ? <ErrorState error={repositories.error} /> : null}
      {repository.error ? <ErrorState error={repository.error} /> : null}
      {evolutions.data && repositories.data && repository.data ? (
        <RepositoryOverviewPage
          repository={repository.data}
          repositories={repositories.data}
          evolutions={evolutions.data}
          details={details.data ?? []}
        />
      ) : null}
    </EvolutionShell>
  );
}

function RepositoryOverviewPage({
  repository,
  repositories,
  evolutions,
  details,
}: {
  repository: RepositorySummary;
  repositories: RepositorySummary[];
  evolutions: EvolutionSummary[];
  details: DetailResponse[];
}) {
  const latest = evolutions[0];
  const stats = buildRepositoryStats(evolutions, details);
  const contributors = buildContributors(evolutions);
  const [activeTab, setActiveTab] = useState<RepositoryTab>("overview");
  const [description, setDescription] = useRepositoryDescription(repository);
  const tabs = repositoryTabs(evolutions.length);

  return (
    <main className="min-h-[calc(100dvh-76px)] min-w-0 bg-slate-50/45">
      <div className="grid min-h-[calc(100dvh-76px)] grid-cols-1 xl:grid-cols-[minmax(0,1fr)_350px]">
        <div className="min-w-0">
          <section className="bg-white px-4 pt-7 sm:px-6 lg:px-8">
            <div className="flex min-w-0 flex-col gap-5 sm:flex-row sm:items-start">
              <div className="flex size-[70px] shrink-0 items-center justify-center overflow-hidden rounded-lg bg-white ring-1 ring-inset ring-slate-200">
                <img src="/eve.svg" alt="" className="size-full object-cover" />
              </div>
              <div className="min-w-0 pb-6">
                <div className="flex min-w-0 flex-wrap items-center gap-3">
                  <h1 className="truncate text-[28px] font-semibold leading-tight tracking-normal text-slate-950">
                    {repository.name}
                  </h1>
                  <Badge
                    variant={repository.remoteUrl ? "success" : "secondary"}
                  >
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
                  onClick={() => setActiveTab(tab.id)}
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
          </section>

          <RepositoryTabPanel
            activeTab={activeTab}
            repository={repository}
            latest={latest}
            evolutions={evolutions}
            details={details}
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

type RepositoryTab =
  | "overview"
  | "snapshots"
  | "activity"
  | "artifacts"
  | "settings";

function repositoryTabs(
  snapshotCount: number,
): Array<{ id: RepositoryTab; label: string; count?: number }> {
  return [
    { id: "overview", label: "Overview" },
    { id: "snapshots", label: "Snapshots", count: snapshotCount },
    { id: "activity", label: "Activity" },
    { id: "artifacts", label: "Artifacts" },
    { id: "settings", label: "Settings" },
  ];
}

function useRepositoryDescription(repository: RepositorySummary) {
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

const DEFAULT_REPOSITORY_DESCRIPTION =
  "Track product states, snapshots, sessions, and verification recorded for this repository.";

function RepositoryTabPanel({
  activeTab,
  repository,
  latest,
  evolutions,
  details,
}: {
  activeTab: RepositoryTab;
  repository: RepositorySummary;
  latest?: EvolutionSummary;
  evolutions: EvolutionSummary[];
  details: DetailResponse[];
}) {
  return (
    <div
      id={`repository-tab-${activeTab}`}
      role="tabpanel"
      aria-labelledby={`repository-tab-trigger-${activeTab}`}
      className="px-4 py-6 sm:px-6 lg:px-8"
    >
      {activeTab === "overview" ? (
        <div className="space-y-5">
          <LatestSnapshotCard latest={latest} />

          <div className="grid grid-cols-1 items-start gap-5 xl:grid-cols-[minmax(0,1fr)_300px] 2xl:grid-cols-[minmax(0,1fr)_326px]">
            <ReadmePanel repository={repository} />
            <EvolutionTimelineCard evolutions={evolutions} />
          </div>
        </div>
      ) : null}

      {activeTab === "snapshots" ? (
        <EvolutionTimelineCard evolutions={evolutions} spacious />
      ) : null}

      {activeTab === "activity" ? (
        <div className="grid grid-cols-1 gap-5 2xl:grid-cols-[minmax(0,1fr)_326px]">
          <RecentActivityCard evolutions={evolutions} />
          <EvolutionTimelineCard evolutions={evolutions} />
        </div>
      ) : null}

      {activeTab === "artifacts" ? (
        <ArtifactsPanel repository={repository} details={details} />
      ) : null}

      {activeTab === "settings" ? (
        <div className="grid grid-cols-1 gap-5">
          <RepositoryLinksCard repository={repository} />
        </div>
      ) : null}
    </div>
  );
}

function RepositoryRightRail({
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
}) {
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

function ReadmePanel({ repository }: { repository: RepositorySummary }) {
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

function LatestSnapshotCard({ latest }: { latest?: EvolutionSummary }) {
  if (!latest) {
    return (
      <section
        id="snapshots"
        className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]"
      >
        <h2 className="flex items-center gap-2 text-base font-semibold">
          <BookOpen className="size-4 text-blue-600" />
          Latest snapshot
        </h2>
        <p className="mt-3 text-sm text-muted-foreground">
          No snapshots have been recorded for this repository.
        </p>
      </section>
    );
  }

  return (
    <section
      id="snapshots"
      className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]"
    >
      <h2 className="flex items-center gap-2 text-base font-semibold">
        <BookOpen className="size-4 text-blue-600" />
        Latest snapshot
      </h2>
      <div className="mt-5 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <div className="flex min-w-0 flex-wrap items-center gap-3">
            <h3 className="text-base font-semibold text-balance">
              {latest.title}
            </h3>
            <StatusBadge status={latest.status} />
          </div>
          <p className="mt-2 max-w-[70ch] text-sm leading-6 text-muted-foreground">
            {latest.outcome}
          </p>
          <div className="mt-4 flex flex-wrap gap-2">
            <MetaPill icon={Sparkles} label={latest.type} />
            <MetaPill
              icon={Code2}
              label={`${latest.commitCount} ${latest.commitCount === 1 ? "commit" : "commits"}`}
            />
            <MetaPill
              icon={Calendar}
              label={compactDate(latest.updatedAt || latest.createdAt)}
            />
          </div>
        </div>
        {/* <Button asChild variant="ghost" className="shrink-0 gap-2">
          <Link to="/snapshots/$id" params={{ id: latest.id }}>
            View snapshot
            <ArrowRight className="size-4" />
          </Link>
        </Button> */}
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
}) {
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
        <p className="text-sm text-muted-foreground">
          No timeline entries yet.
        </p>
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
                {evolution.type} ·{" "}
                {compactDate(evolution.updatedAt || evolution.createdAt)}
              </span>
            </Link>
          ))}
        </div>
      )}
    </section>
  );
}

function RecentActivityCard({
  evolutions,
  title = "Recent activity",
}: {
  evolutions: EvolutionSummary[];
  title?: string;
}) {
  return (
    <section id="activity" className="space-y-3">
      <div className="flex items-center justify-between gap-4">
        <h2 className="text-base font-semibold">{title}</h2>
        <Button variant="outline" size="sm" className="gap-2">
          All activity types
          <History className="size-3.5" />
        </Button>
      </div>
      <div className="overflow-hidden rounded-lg bg-white shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
        {evolutions.length === 0 ? (
          <div className="p-5 text-sm text-muted-foreground">
            No activity has been recorded.
          </div>
        ) : (
          evolutions.slice(0, 6).map((evolution, index) => (
            <Link
              key={evolution.id}
              to="/snapshots/$id"
              params={{ id: evolution.id }}
              className={`grid grid-cols-[36px_minmax(0,1fr)_20px] items-center gap-3 px-4 py-3.5 transition-colors hover:bg-slate-50 sm:grid-cols-[44px_minmax(0,1fr)_112px_20px] ${index > 0 ? "border-t" : ""}`}
            >
              <span className="flex size-9 items-center justify-center rounded-full bg-blue-50 text-blue-700">
                <BookOpen className="size-5" />
              </span>
              <span className="min-w-0">
                <span className="flex min-w-0 items-center gap-3">
                  <strong className="truncate text-sm font-semibold">
                    {evolution.title}
                  </strong>
                  <StatusBadge status={evolution.status} />
                </span>
                <span className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
                  <span>{evolution.type}</span>
                  {evolution.snapshot ? (
                    <span className="font-mono">
                      {shortCommit(evolution.snapshot)}
                    </span>
                  ) : null}
                </span>
              </span>
              <span className="hidden text-right text-xs text-muted-foreground sm:block">
                {compactDate(evolution.updatedAt || evolution.createdAt)}
              </span>
              <ArrowRight className="size-4 text-slate-500" />
            </Link>
          ))
        )}
      </div>
    </section>
  );
}

function ArtifactsPanel({
  repository,
  details,
}: {
  repository: RepositorySummary;
  details: DetailResponse[];
}) {
  const [selectedArtifact, setSelectedArtifact] =
    useState<ArtifactCardRow | null>(null);
  const artifacts: ArtifactCardRow[] = details.flatMap((detail) =>
    detail.snapshot.artifacts.map((artifact, index) => ({
      id: `${detail.snapshot.id}-${index}`,
      type: artifact.type,
      description:
        artifact.description ||
        artifact.path ||
        artifact.url ||
        artifact.uri ||
        "Artifact",
      href:
        artifact.url ||
        artifact.uri ||
        localArtifactHref(repository.name, artifact.path),
      imageSrc: artifactImageSrc(repository.name, artifact),
      source: artifact.path || artifact.url || artifact.uri,
      isImage: isImageArtifact(artifact),
    })),
  );

  return (
    <section className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]">
      <div className="mb-5 flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold">Artifacts</h2>
        <span className="rounded-md bg-secondary px-2 py-1 text-xs font-medium text-muted-foreground">
          {artifacts.length} {artifacts.length === 1 ? "file" : "files"}
        </span>
      </div>
      {artifacts.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No artifacts have been recorded for this repository.
        </p>
      ) : (
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          {artifacts.map((artifact) => (
            <article
              key={artifact.id}
              className="overflow-hidden rounded-lg border bg-slate-50/70"
            >
              {artifact.isImage && artifact.imageSrc ? (
                <button
                  type="button"
                  className="block aspect-video w-full overflow-hidden bg-white text-left"
                  onClick={() => setSelectedArtifact(artifact)}
                  aria-label={`Open ${artifact.description}`}
                >
                  <img
                    src={artifact.imageSrc}
                    alt={artifact.description}
                    className="size-full object-cover transition-transform duration-150 hover:scale-[1.02]"
                    loading="lazy"
                  />
                </button>
              ) : (
                <div className="flex aspect-video items-center justify-center bg-white text-slate-400">
                  <ImageIcon className="size-8" />
                </div>
              )}
              <div className="flex items-start justify-between gap-4 p-4">
                <div className="min-w-0">
                  <p className="text-sm font-semibold capitalize">
                    {artifact.type}
                  </p>
                  <p className="mt-1 text-sm text-muted-foreground text-pretty">
                    {artifact.description}
                  </p>
                </div>
                {artifact.isImage ? (
                  <button
                    type="button"
                    className="inline-flex size-8 shrink-0 items-center justify-center rounded-md bg-white text-slate-600 shadow-[0_0_0_1px_rgba(15,23,42,0.1)] hover:text-slate-950"
                    aria-label="Preview artifact"
                    onClick={() => setSelectedArtifact(artifact)}
                  >
                    <ExternalLink className="size-4" />
                  </button>
                ) : artifact.href ? (
                  <a
                    href={artifact.href}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex size-8 shrink-0 items-center justify-center rounded-md bg-white text-slate-600 shadow-[0_0_0_1px_rgba(15,23,42,0.1)] hover:text-slate-950"
                    aria-label="Open artifact"
                  >
                    <ExternalLink className="size-4" />
                  </a>
                ) : null}
              </div>
            </article>
          ))}
        </div>
      )}
      <Dialog
        open={Boolean(selectedArtifact)}
        onOpenChange={(open) => {
          if (!open) setSelectedArtifact(null);
        }}
      >
        <DialogContent className="max-w-[min(980px,calc(100vw-24px))] p-0">
          {selectedArtifact ? (
            <div>
              <DialogHeader className="border-b px-5 py-4">
                <DialogTitle className="text-base">
                  {selectedArtifact.description}
                </DialogTitle>
                <DialogDescription className="truncate text-xs">
                  {selectedArtifact.source}
                </DialogDescription>
              </DialogHeader>
              <div className="max-h-[78dvh] overflow-auto bg-slate-950 p-3">
                <img
                  src={selectedArtifact.imageSrc ?? ""}
                  alt={selectedArtifact.description}
                  className="mx-auto max-h-[72dvh] w-auto max-w-full rounded-md object-contain"
                />
              </div>
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </section>
  );
}

type ArtifactCardRow = {
  id: string;
  type: string;
  description: string;
  href?: string;
  imageSrc?: string;
  source?: string;
  isImage: boolean;
};

function isImageArtifact(artifact: SnapshotArtifact) {
  const source = artifact.path || artifact.url || artifact.uri || "";
  const lower = source.toLowerCase();
  return (
    artifact.mimeType?.startsWith("image/") ||
    artifact.type.toLowerCase().includes("screenshot") ||
    artifact.type.toLowerCase().includes("image") ||
    /\.(png|jpe?g|gif|webp|avif|svg)$/i.test(lower)
  );
}

function artifactImageSrc(repo: string, artifact: SnapshotArtifact) {
  if (!isImageArtifact(artifact)) return undefined;
  return artifact.url || artifact.uri || localArtifactHref(repo, artifact.path);
}

function localArtifactHref(repo: string, artifactPath?: string) {
  if (!artifactPath) return undefined;
  if (/^https?:\/\//i.test(artifactPath)) return artifactPath;
  const normalized = artifactPath.replace(/^\/+/, "");
  const prefix = ".eve/artifacts/";
  if (!normalized.startsWith(prefix)) return undefined;
  const relative = normalized.slice(prefix.length);
  return `/api/repos/${encodeURIComponent(repo)}/artifacts/${relative
    .split("/")
    .map(encodeURIComponent)
    .join("/")}`;
}

function RepositoryFactsCard({
  repository,
  description,
  onDescriptionChange,
}: {
  repository: RepositorySummary;
  description: string;
  onDescriptionChange: (value: string) => void;
}) {
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

function SnapshotSummaryCard({ stats }: { stats: RepositoryStats }) {
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

function ContributorCard({ rows }: { rows: ContributorRow[] }) {
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
}) {
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

function RailCard({
  id,
  title,
  eyebrow,
  children,
}: {
  id?: string;
  title: string;
  eyebrow?: string;
  children: React.ReactNode;
}) {
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

function MetaPill({
  icon: Icon,
  label,
  tone,
}: {
  icon: LucideIcon;
  label: string;
  tone?: "success" | "warning";
}) {
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

type RepositoryStats = {
  snapshots: number;
  features: number;
  bugfixes: number;
  refactors: number;
  commits: number;
  decisions: number;
  validated: number;
  risks: number;
};

function buildRepositoryStats(
  evolutions: EvolutionSummary[],
  details: DetailResponse[],
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
    decisions: details.reduce(
      (total, detail) => total + detail.evolution.decisions.length,
      0,
    ),
    validated: evolutions.filter(
      (evolution) => evolution.verificationState === "passed",
    ).length,
    risks: details.reduce(
      (total, detail) => total + detail.evolution.risks.length,
      0,
    ),
  };
}

type ContributorRow = {
  label: string;
  count: number;
};

function buildContributors(evolutions: EvolutionSummary[]): ContributorRow[] {
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

function normalizeProvider(value: string) {
  const normalized = value.toLowerCase();
  if (normalized.includes("codex")) return "Codex";
  if (normalized.includes("claude")) return "Claude";
  if (normalized.includes("opencode")) return "OpenCode";
  return "Other";
}

function agentAvatarPath(label: string) {
  if (label === "Codex") return "/agents/codex.svg";
  if (label === "Claude") return "/agents/claude.svg";
  if (label === "OpenCode") return "/agents/opencode.svg";
  return "/agents/other.svg";
}

function formatBytes(value?: number) {
  if (!value) return "Unknown";
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

type RepositoryTone = {
  bg: string;
  text: string;
  soft: string;
};

const REPOSITORY_TONES: RepositoryTone[] = [
  { bg: "bg-blue-600", text: "text-blue-700", soft: "bg-blue-50" },
  { bg: "bg-emerald-500", text: "text-emerald-700", soft: "bg-emerald-50" },
  { bg: "bg-violet-600", text: "text-violet-700", soft: "bg-violet-50" },
  { bg: "bg-amber-500", text: "text-amber-700", soft: "bg-amber-50" },
];
