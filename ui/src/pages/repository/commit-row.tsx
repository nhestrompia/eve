import { ExternalLink } from "lucide-react";

import { shortCommit } from "../../format";
import type { GitCommit } from "../../types";

export function CommitRow({
  commit,
  href,
}: {
  commit: GitCommit;
  href?: string;
}): React.JSX.Element {
  const content = (
    <>
      <span className="font-mono text-xs text-muted-foreground">
        {commit.shortHash || shortCommit(commit.hash)}
      </span>
      <span className="min-w-0 truncate text-sm font-medium">
        {commit.subject || "Untitled commit"}
      </span>
      {href ? (
        <ExternalLink className="ml-auto size-3.5 shrink-0 text-slate-500" />
      ) : null}
    </>
  );

  if (!href) {
    return <div className="flex min-h-11 items-center gap-3 px-3">{content}</div>;
  }

  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer"
      className="flex min-h-11 items-center gap-3 px-3 transition-colors hover:bg-slate-50"
    >
      {content}
    </a>
  );
}
