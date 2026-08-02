import { humanDate, statusLabel } from "../../format";
import type { EvolutionSummary } from "../../types";

export function ComparisonEndpointCard({
  label,
  snapshot,
  active,
  onClick,
}: {
  label: string;
  snapshot?: EvolutionSummary;
  active: boolean;
  onClick: () => void;
}): React.JSX.Element {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`min-h-[96px] rounded-lg bg-card p-4 text-left shadow-[var(--shadow-border)] transition-[scale,background-color,box-shadow] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.98] ${
        active ? "bg-accent text-accent-foreground" : "hover:bg-secondary"
      }`}
    >
      <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        {label}
      </p>
      <h3 className="mt-2 truncate text-base font-semibold">
        {snapshot?.title ?? "Select snapshot"}
      </h3>
      <p className="mt-1 text-sm text-muted-foreground">
        {snapshot
          ? `${statusLabel(snapshot.type)} · ${humanDate(snapshot.createdAt)}`
          : "Choose a snapshot"}
      </p>
    </button>
  );
}
