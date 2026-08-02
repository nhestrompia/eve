import { compactDate } from "../../format";
import type { EvolutionSummary } from "../../types";

export function ComparisonSnapshotResult({
  snapshot,
  selectedAs,
  onClick,
}: {
  snapshot: EvolutionSummary;
  selectedAs?: "Earlier" | "Later";
  onClick: () => void;
}): React.JSX.Element {
  return (
    <button
      type="button"
      onClick={onClick}
      className="grid min-h-14 w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-3 rounded-md bg-card px-3 py-2.5 text-left shadow-[var(--shadow-border)] transition-[scale,background-color,box-shadow] hover:bg-background active:scale-[0.98]"
    >
      <span className="min-w-0">
        <span className="block truncate text-sm font-semibold">{snapshot.title}</span>
        <span className="mt-1 block truncate text-xs text-muted-foreground">
          {snapshot.type} · {compactDate(snapshot.createdAt)} · {snapshot.id}
        </span>
      </span>
      {selectedAs ? (
        <span className="rounded-md bg-accent px-2 py-1 text-xs font-semibold text-accent-foreground">
          {selectedAs}
        </span>
      ) : (
        <span className="text-xs font-medium text-muted-foreground">Pick</span>
      )}
    </button>
  );
}
