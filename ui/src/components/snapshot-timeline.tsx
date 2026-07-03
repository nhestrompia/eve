import { Link } from "@tanstack/react-router";
import { Box, CheckCircle2, GitCommitHorizontal } from "lucide-react";
import { compactDate, shortCommit } from "../format";
import { cn } from "../lib/utils";
import type { EvolutionSummary } from "../types";

type SnapshotTimelineProps = {
  evolutions: EvolutionSummary[];
  selectedId?: string;
  route?: "detail" | "snapshot";
  className?: string;
};

export function SnapshotTimeline({
  evolutions,
  selectedId,
  route = "detail",
  className,
}: SnapshotTimelineProps) {
  const snapshots = [...evolutions]
    .filter((evolution) => evolution.snapshot)
    .sort((a, b) => {
      const left = timestamp(b.updatedAt || b.createdAt);
      const right = timestamp(a.updatedAt || a.createdAt);
      if (left !== right) return left - right;
      return b.id.localeCompare(a.id);
    });
  return (
    <section className={cn("space-y-4", className)}>
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Box className="size-4 text-slate-700" />
          <h3 className="font-semibold">Snapshot timeline</h3>
        </div>
        <span className="text-xs text-muted-foreground">
          {snapshots.length}
        </span>
      </div>

      {snapshots.length === 0 ? (
        <div className="rounded-lg bg-white p-4 text-sm text-muted-foreground shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
          No repository snapshots are recorded yet.
        </div>
      ) : (
        <div className="max-h-[26rem] space-y-1 overflow-y-auto overscroll-contain pr-1">
          {snapshots.map((evolution, index) => (
            <SnapshotTimelineLink
              key={evolution.id}
              evolution={evolution}
              isSelected={evolution.id === selectedId}
              isLast={index === snapshots.length - 1}
              route={route}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function SnapshotTimelineLink({
  evolution,
  isSelected,
  isLast,
  route,
}: {
  evolution: EvolutionSummary;
  isSelected: boolean;
  isLast: boolean;
  route: "detail" | "snapshot";
}) {
  const content = (
    <SnapshotTimelineContent
      evolution={evolution}
      isSelected={isSelected}
      isLast={isLast}
    />
  );

  if (route === "snapshot") {
    return (
      <Link
        to="/snapshots/$id/snapshot"
        params={{ id: evolution.id }}
        className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
      >
        {content}
      </Link>
    );
  }

  return (
    <Link
      to="/snapshots/$id"
      params={{ id: evolution.id }}
      className="block rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
    >
      {content}
    </Link>
  );
}

function SnapshotTimelineContent({
  evolution,
  isSelected,
  isLast,
}: {
  evolution: EvolutionSummary;
  isSelected: boolean;
  isLast: boolean;
}) {
  return (
    <div
      className={cn(
        "grid grid-cols-[28px_minmax(0,1fr)] gap-3 rounded-lg px-2 py-2 transition-[background-color,scale] duration-150 hover:bg-slate-50 active:scale-[0.98]",
        isSelected && "bg-blue-50 hover:bg-blue-50",
      )}
    >
      <div className="relative flex justify-center">
        <span
          className={cn(
            "z-10 mt-1 flex size-5 items-center justify-center rounded-full bg-white shadow-[0_0_0_1px_rgba(15,23,42,0.18)]",
            isSelected && "bg-blue-600 text-white shadow-none",
          )}
        >
          {isSelected ? (
            <CheckCircle2 className="size-3.5" />
          ) : (
            <GitCommitHorizontal className="size-3 text-slate-500" />
          )}
        </span>
        {!isLast ? (
          <span className="absolute top-7 h-[calc(100%+4px)] w-px bg-slate-200" />
        ) : null}
      </div>
      <div className="min-w-0">
        <div className="flex items-center justify-between gap-3">
          <p className="truncate font-medium">
            {evolution.title || "Untitled Snapshot"}
          </p>
          <code
            className={cn(
              "shrink-0 font-mono text-xs tabular-nums text-muted-foreground",
              isSelected && "text-blue-700",
            )}
          >
            {evolution.id}
          </code>
        </div>
        <div className="mt-1 flex items-center justify-between gap-3 text-xs text-muted-foreground">
          <code className="truncate rounded-md bg-secondary px-1.5 py-0.5 font-mono tabular-nums">
            {shortCommit(evolution.snapshot)}
          </code>
          <span className="shrink-0">
            {compactDate(evolution.updatedAt || evolution.createdAt)}
          </span>
        </div>
      </div>
    </div>
  );
}

function timestamp(value?: string) {
  const date = value ? new Date(value) : undefined;
  return date && !Number.isNaN(date.getTime()) ? date.getTime() : 0;
}
