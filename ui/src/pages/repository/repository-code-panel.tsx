import { Link } from "@tanstack/react-router";
import { Code2, ExternalLink } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "../../components/ui/button";
import { SnapshotCodeBrowser } from "../../components/snapshot-code-browser";
import { compactDate } from "../../format";
import type { EvolutionSummary, RepositorySummary } from "../../types";

export function RepositoryCodePanel({
  repository,
  evolutions,
}: {
  repository: RepositorySummary;
  evolutions: EvolutionSummary[];
}): React.JSX.Element {
  const [selectedId, setSelectedId] = useState(evolutions[0]?.id ?? "");

  useEffect(() => {
    if (evolutions.length === 0) {
      setSelectedId("");
      return;
    }
    if (!evolutions.some((evolution) => evolution.id === selectedId)) {
      setSelectedId(evolutions[0].id);
    }
  }, [evolutions, selectedId]);

  const selected = evolutions.find((evolution) => evolution.id === selectedId);

  if (evolutions.length === 0) {
    return (
      <section className="rounded-lg bg-card p-5 shadow-[var(--shadow-border)]">
        <h2 className="flex items-center gap-2 text-base font-semibold">
          <Code2 className="size-4 text-blue-600" />
          Code
        </h2>
        <p className="mt-3 text-sm text-muted-foreground">
          Record a Snapshot in this repository to inspect the code behind it.
        </p>
      </section>
    );
  }

  return (
    <section className="space-y-4">
      <section className="rounded-lg bg-card p-5 shadow-[var(--shadow-border)] sm:p-6">
        <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div className="max-w-[68ch]">
            <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              Snapshot code
            </p>
            <h2 className="mt-2 flex items-center gap-2 text-xl font-semibold text-balance">
              <Code2 className="size-5 text-blue-600" />
              Inspect code by Snapshot
            </h2>
            <p className="mt-3 text-sm leading-6 text-muted-foreground text-pretty">
              Choose a Snapshot from this repository and inspect the relevant files without leaving the repository view.
            </p>
          </div>
          {selected ? (
            <div className="flex flex-wrap gap-2">
              <Button asChild variant="outline" size="sm">
                <Link to="/snapshots/$id/code" params={{ id: selected.id }}>
                  <Code2 className="size-3.5" />
                  Full code view
                </Link>
              </Button>
              <Button asChild variant="ghost" size="sm">
                <Link to="/snapshots/$id" params={{ id: selected.id }}>
                  <ExternalLink className="size-3.5" />
                  Snapshot page
                </Link>
              </Button>
            </div>
          ) : null}
        </div>

        <div className="mt-5 grid gap-2 sm:max-w-xl">
          <label
            htmlFor="repository-code-snapshot"
            className="text-xs font-semibold uppercase tracking-wide text-muted-foreground"
          >
            Snapshot
          </label>
          <select
            id="repository-code-snapshot"
            value={selectedId}
            onChange={(event) => setSelectedId(event.target.value)}
            className="h-10 rounded-md border bg-background px-3 text-sm font-medium text-foreground shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            {evolutions.map((evolution) => (
              <option key={evolution.id} value={evolution.id}>
                {evolution.title || evolution.id} - {compactDate(evolution.updatedAt || evolution.createdAt)}
              </option>
            ))}
          </select>
        </div>
      </section>

      {selected ? (
        <SnapshotCodeBrowser snapshotId={selected.id} repository={repository.name} />
      ) : null}
    </section>
  );
}
