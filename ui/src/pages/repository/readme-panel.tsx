import { Button } from "../../components/ui/button";
import { MarkdownViewer } from "../../components/markdown-viewer";
import type { RepositorySummary } from "../../types";
import { Copy, ExternalLink, FileText } from "lucide-react";
import { useState } from "react";

export function ReadmePanel({
  repository,
}: {
  repository: RepositorySummary;
}): React.JSX.Element {
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
