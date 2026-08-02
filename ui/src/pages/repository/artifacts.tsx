import { useQuery } from "@tanstack/react-query";
import {
  ExternalLink,
  FileText,
  Image as ImageIcon,
  Terminal,
} from "lucide-react";
import { useState } from "react";

import {
  artifactHref,
  artifactKind,
  artifactSource,
  type ArtifactKind,
} from "../../lib/artifacts";
import type { DetailResponse, RepositorySummary } from "../../types";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "../../components/ui/dialog";

export function ArtifactsPanel({
  repository,
  details,
}: {
  repository: RepositorySummary;
  details: DetailResponse[];
}): React.JSX.Element {
  const [selectedArtifact, setSelectedArtifact] =
    useState<ArtifactCardRow | null>(null);
  const artifacts: ArtifactCardRow[] = details.flatMap((detail) =>
    detail.snapshot.artifacts.map((artifact, index) => {
      const kind = artifactKind(artifact);
      const href = artifactHref(repository.name, artifact);
      return {
        id: `${detail.snapshot.id}-${index}`,
        type: artifact.type,
        description:
          artifact.description || artifactSource(artifact) || "Artifact",
        href,
        imageSrc: kind === "image" ? href : undefined,
        source: artifactSource(artifact),
        kind,
      };
    }),
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
            <ArtifactCard
              key={artifact.id}
              artifact={artifact}
              onPreview={setSelectedArtifact}
            />
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
              {selectedArtifact.kind === "image" ? (
                <div className="max-h-[78dvh] overflow-auto bg-slate-950 p-3">
                  <img
                    src={selectedArtifact.imageSrc ?? ""}
                    alt={selectedArtifact.description}
                    className="mx-auto max-h-[72dvh] w-auto max-w-full rounded-md object-contain outline outline-1 -outline-offset-1 outline-white/10"
                  />
                </div>
              ) : (
                <ArtifactLogContent artifact={selectedArtifact} expanded />
              )}
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </section>
  );
}

export type ArtifactCardRow = {
  id: string;
  type: string;
  description: string;
  href?: string;
  imageSrc?: string;
  source?: string;
  kind: ArtifactKind;
};

export function ArtifactCard({
  artifact,
  onPreview,
}: {
  artifact: ArtifactCardRow;
  onPreview: (artifact: ArtifactCardRow) => void;
}): React.JSX.Element {
  const [imageUnavailable, setImageUnavailable] = useState(false);
  const canPreviewImage =
    artifact.kind === "image" &&
    Boolean(artifact.imageSrc) &&
    !imageUnavailable;
  const openArtifact = () => onPreview(artifact);
  const handleCardKeyDown = (event: React.KeyboardEvent<HTMLElement>) => {
    if (event.target !== event.currentTarget) return;
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      openArtifact();
    }
  };

  return (
    <article
      role="button"
      tabIndex={0}
      aria-label={`Open ${artifact.description}`}
      onClick={openArtifact}
      onKeyDown={handleCardKeyDown}
      className="cursor-pointer overflow-hidden rounded-lg bg-slate-50/70 shadow-[0_0_0_1px_rgba(15,23,42,0.1)] outline-none transition-shadow focus-visible:ring-2 focus-visible:ring-blue-600"
    >
      {artifact.kind === "image" ? (
        canPreviewImage ? (
          <button
            type="button"
            className="block aspect-video w-full overflow-hidden bg-white text-left active:scale-[0.96] transition-transform"
            onClick={(event) => {
              event.stopPropagation();
              openArtifact();
            }}
            aria-label={`Open ${artifact.description}`}
          >
            <img
              src={artifact.imageSrc}
              alt={artifact.description}
              className="size-full object-cover outline outline-1 -outline-offset-1 outline-black/10 transition-transform duration-150 hover:scale-[1.02]"
              loading="lazy"
              onError={() => setImageUnavailable(true)}
            />
          </button>
        ) : (
          <ArtifactUnavailable kind="image" />
        )
      ) : artifact.kind === "log" ? (
        <ArtifactLogContent artifact={artifact} />
      ) : (
        <div className="flex min-h-28 items-center gap-3 bg-white px-4 py-5 text-slate-500">
          <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-slate-50 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
            <FileText className="size-5" />
          </span>
          <span className="min-w-0 text-sm">Recorded file evidence</span>
        </div>
      )}
      <div className="flex items-start justify-between gap-4 p-4">
        <div className="min-w-0">
          <p className="text-sm font-semibold capitalize">{artifact.type}</p>
          <p className="mt-1 text-sm text-muted-foreground text-pretty">
            {artifact.description}
          </p>
        </div>
        {canPreviewImage || artifact.kind === "log" ? (
          <button
            type="button"
            className="inline-flex size-10 shrink-0 items-center justify-center rounded-lg bg-white text-slate-600 shadow-[0_0_0_1px_rgba(15,23,42,0.1)] transition-transform hover:text-slate-950 active:scale-[0.96]"
            aria-label={`Preview ${artifact.type} artifact`}
            onClick={(event) => {
              event.stopPropagation();
              openArtifact();
            }}
          >
            <ExternalLink className="size-4" />
          </button>
        ) : artifact.href ? (
          <a
            href={artifact.href}
            target="_blank"
            rel="noreferrer"
            className="inline-flex size-10 shrink-0 items-center justify-center rounded-lg bg-white text-slate-600 shadow-[0_0_0_1px_rgba(15,23,42,0.1)] transition-transform hover:text-slate-950 active:scale-[0.96]"
            aria-label="Open artifact"
            onClick={(event) => event.stopPropagation()}
          >
            <ExternalLink className="size-4" />
          </a>
        ) : null}
      </div>
    </article>
  );
}

function ArtifactUnavailable({ kind }: { kind: "image" | "log" }): React.JSX.Element {
  const Icon = kind === "image" ? ImageIcon : Terminal;
  return (
    <div className="flex min-h-36 flex-col items-center justify-center gap-2 bg-white px-5 text-center text-slate-400">
      <Icon className="size-7" />
      <p className="text-xs font-medium">
        {kind === "image"
          ? "Image unavailable in this checkout"
          : "Log preview unavailable"}
      </p>
    </div>
  );
}

export function ArtifactLogContent({
  artifact,
  expanded = false,
}: {
  artifact: ArtifactCardRow;
  expanded?: boolean;
}): React.JSX.Element {
  const canLoad = Boolean(artifact.href?.startsWith("/api/"));
  const content = useQuery({
    queryKey: ["artifact-text", artifact.href],
    queryFn: async () => {
      const response = await fetch(artifact.href!, {
        headers: { Range: "bytes=0-131071" },
      });
      if (response.status === 416) {
        const fallback = await fetch(artifact.href!);
        if (!fallback.ok) {
          throw new Error(`Artifact returned ${fallback.status}`);
        }
        return fallback.text();
      }
      if (!response.ok) throw new Error(`Artifact returned ${response.status}`);
      return response.text();
    },
    enabled: canLoad,
    retry: false,
    staleTime: Number.POSITIVE_INFINITY,
  });

  if (!canLoad || content.isError) {
    return <ArtifactUnavailable kind="log" />;
  }

  return (
    <div
      className={
        expanded
          ? "max-h-[72dvh] min-h-64 overflow-auto bg-slate-950 p-5"
          : "h-44 overflow-hidden bg-slate-950 p-4"
      }
    >
      <div className="mb-3 flex items-center gap-2 text-xs font-medium text-slate-400">
        <Terminal className="size-4" />
        <span>{content.isPending ? "Loading log…" : "Log output"}</span>
      </div>
      {content.data ? (
        <pre className="whitespace-pre-wrap break-words font-mono text-xs leading-5 text-slate-200">
          {expanded ? content.data : content.data.slice(0, 4_000)}
        </pre>
      ) : content.isPending ? (
        <div className="space-y-2" aria-label="Loading log preview">
          <div className="h-2 w-4/5 animate-pulse rounded bg-slate-800" />
          <div className="h-2 w-3/5 animate-pulse rounded bg-slate-800" />
          <div className="h-2 w-2/3 animate-pulse rounded bg-slate-800" />
        </div>
      ) : (
        <p className="font-mono text-xs text-slate-400">The log is empty.</p>
      )}
    </div>
  );
}
