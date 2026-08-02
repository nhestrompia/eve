import { agentAvatarPath, REPOSITORY_TONES, type ContributorRow } from "./types";
import { RailCard } from "./rail-card";

export function ContributorCard({
  rows,
}: {
  rows: ContributorRow[];
}): React.JSX.Element {
  const max = Math.max(1, ...rows.map((row) => row.count));

  return (
    <RailCard title="Top contributors" eyebrow="Last 30 days">
      <div className="space-y-4">
        {rows.map((row, index) => {
          const tone = REPOSITORY_TONES[index % REPOSITORY_TONES.length];
          return (
            <div
              key={row.label}
              className="grid grid-cols-[88px_minmax(0,1fr)_54px] items-center gap-3"
            >
              <span className="flex min-w-0 items-center gap-2 text-sm font-semibold">
                <img
                  src={agentAvatarPath(row.label)}
                  alt=""
                  className="size-6 rounded-lg"
                />
                <span className="truncate">{row.label}</span>
              </span>
              <span className="h-1.5 overflow-hidden rounded-full bg-slate-100">
                <span
                  className={`block h-full rounded-full ${tone.bg}`}
                  style={{ width: `${Math.max(8, (row.count / max) * 100)}%` }}
                />
              </span>
              <span className="text-right text-sm text-muted-foreground tabular-nums">
                {row.count}
              </span>
            </div>
          );
        })}
      </div>
    </RailCard>
  );
}
