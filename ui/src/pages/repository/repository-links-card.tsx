import { useMutation } from "@tanstack/react-query";
import {
  Code2,
  Copy,
  ExternalLink,
  GitBranch,
} from "lucide-react";
import { useState } from "react";

import { api } from "../../api";
import type { RepositorySummary } from "../../types";
import { RailCard } from "./rail-card";

export function RepositoryLinksCard({
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
