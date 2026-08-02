import {
  artifactHref,
  artifactKind,
  artifactSource,
} from "../../lib/artifacts";
import type { DetailResponse, RepositorySummary } from "../../types";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "../../components/ui/dialog";
import { useState } from "react";
import { ArtifactCard } from "./artifact-card";
import { ArtifactLogContent } from "./artifact-log-content";
import type { ArtifactCardRow } from "./artifact-types";

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
