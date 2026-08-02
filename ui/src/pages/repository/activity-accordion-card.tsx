import { Link } from "@tanstack/react-router";
import {
  ArrowRight,
  BookOpen,
  ChevronDown,
  History,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { Button } from "../../components/ui/button";
import { StatusBadge } from "../../components/status-badge";
import { compactDate } from "../../format";
import type {
  DetailResponse,
  EvolutionSummary,
  RepositorySummary,
} from "../../types";
import { CommitRow } from "./commit-row";
import { commitsForDetail, githubCommitUrl } from "./activity-utils";

export function ActivityAccordionCard({
  repository,
  evolutions,
  details,
}: {
  repository: RepositorySummary;
  evolutions: EvolutionSummary[];
  details: DetailResponse[];
}): React.JSX.Element {
  const [openId, setOpenId] = useState<string | null>(evolutions[0]?.id ?? null);
  const detailById = useMemo(
    () => new Map(details.map((detail) => [detail.summary.id, detail])),
    [details],
  );

  useEffect(() => {
    if (evolutions.length === 0) {
      setOpenId(null);
      return;
    }
    if (!evolutions.some((evolution) => evolution.id === openId)) {
      setOpenId(evolutions[0].id);
    }
  }, [evolutions, openId]);

  return (
    <section id="activity" className="space-y-3">
      <div className="flex items-center justify-between gap-4">
        <h2 className="text-base font-semibold">Recent activity</h2>
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
          evolutions.slice(0, 12).map((evolution, index) => {
            const detail = detailById.get(evolution.id);
            const commits = detail ? commitsForDetail(detail) : undefined;
            const commitCount = commits?.length ?? evolution.commitCount;
            const open = openId === evolution.id;
            return (
              <article key={evolution.id} className={index > 0 ? "border-t" : ""}>
                <button
                  type="button"
                  aria-expanded={open}
                  className="grid w-full grid-cols-[36px_minmax(0,1fr)_20px] items-center gap-3 px-4 py-3.5 text-left transition-colors hover:bg-slate-50 sm:grid-cols-[44px_minmax(0,1fr)_112px_20px]"
                  onClick={() => setOpenId(open ? null : evolution.id)}
                >
                  <span className="flex size-9 items-center justify-center rounded-full bg-blue-50 text-blue-700">
                    <BookOpen className="size-5" />
                  </span>
                  <span className="min-w-0">
                    <span className="flex min-w-0 flex-wrap items-center gap-2">
                      <strong className="max-w-[72ch] truncate text-sm font-semibold">
                        {evolution.title || "Untitled snapshot"}
                      </strong>
                      <StatusBadge status={evolution.status} />
                    </span>
                    <span className="mt-1 flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
                      <span>{evolution.type}</span>
                      <span>
                        {commitCount} {commitCount === 1 ? "commit" : "commits"}
                      </span>
                    </span>
                  </span>
                  <span className="hidden text-right text-xs text-muted-foreground sm:block">
                    {compactDate(evolution.updatedAt || evolution.createdAt)}
                  </span>
                  <ChevronDown
                    className={`size-4 text-slate-500 transition-transform ${open ? "rotate-180" : ""}`}
                  />
                </button>
                {open ? (
                  <div className="border-t bg-slate-50/70 px-4 py-4 sm:px-[72px]">
                    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                        Commits in this snapshot
                      </p>
                      <Button asChild variant="outline" size="sm" className="w-fit gap-2">
                        <Link to="/snapshots/$id" params={{ id: evolution.id }}>
                          Open snapshot
                          <ArrowRight className="size-3.5" />
                        </Link>
                      </Button>
                    </div>
                    <div className="mt-3 divide-y rounded-lg border bg-white">
                      {commits ? (
                        commits.length > 0 ? (
                          commits.map((commit) => (
                            <CommitRow
                              key={commit.hash}
                              commit={commit}
                              href={githubCommitUrl(repository.remoteUrl, commit.hash)}
                            />
                          ))
                        ) : (
                          <p className="p-4 text-sm text-muted-foreground">
                            No commits were recorded for this snapshot.
                          </p>
                        )
                      ) : (
                        <p className="p-4 text-sm text-muted-foreground">
                          Commit details are loading.
                        </p>
                      )}
                    </div>
                  </div>
                ) : null}
              </article>
            );
          })
        )}
      </div>
    </section>
  );
}
