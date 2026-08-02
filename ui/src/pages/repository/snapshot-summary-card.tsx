import { RailCard } from "./rail-card";
import type { RepositoryStats } from "./types";

export function SnapshotSummaryCard({
  stats,
}: {
  stats: RepositoryStats;
}): React.JSX.Element {
  const tiles = [
    ["Snapshots", stats.snapshots],
    ["Features", stats.features],
    ["Bug fixes", stats.bugfixes],
    ["Refactor", stats.refactors],
    ["Commits", stats.commits],
    ["Decisions", stats.decisions],
    ["Validated", stats.validated],
    ["Risks", stats.risks],
  ] as const;

  return (
    <RailCard title="Snapshot summary">
      <div className="grid grid-cols-2 gap-2.5">
        {tiles.map(([label, value]) => (
          <div
            key={label}
            className="rounded-lg bg-white px-3 py-2.5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]"
          >
            <div className="text-xl font-semibold leading-6 tabular-nums">
              {value}
            </div>
            <div className="mt-1 text-xs text-muted-foreground">{label}</div>
          </div>
        ))}
      </div>
    </RailCard>
  );
}
