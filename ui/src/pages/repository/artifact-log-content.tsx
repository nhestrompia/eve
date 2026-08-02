import { useQuery } from "@tanstack/react-query";
import { Terminal } from "lucide-react";

import { ArtifactUnavailable } from "./artifact-unavailable";
import type { ArtifactCardRow } from "./artifact-types";

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
