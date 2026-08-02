import { ExternalLink, FileText } from "lucide-react";
import { useState } from "react";

import { ArtifactLogContent } from "./artifact-log-content";
import { ArtifactUnavailable } from "./artifact-unavailable";
import type { ArtifactCardRow } from "./artifact-types";

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
