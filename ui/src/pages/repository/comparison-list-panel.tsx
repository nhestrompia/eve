import { ComparisonCompactItem } from "./comparison-compact-item";
import type { ComparisonPanelItem } from "./comparison-types";

export function ComparisonListPanel({
  title,
  count,
  empty,
  items,
  compact = false,
}: {
  title: string;
  count: number;
  empty: string;
  items: ComparisonPanelItem[];
  compact?: boolean;
}): React.JSX.Element {
  return (
    <section className="overflow-hidden rounded-lg bg-card shadow-[var(--shadow-border)]">
      <div className="flex min-h-12 items-center justify-between gap-3 border-b px-4">
        <h3 className="text-sm font-semibold">{title}</h3>
        <span className="rounded-md bg-secondary px-2 py-1 text-xs font-medium text-muted-foreground tabular-nums">
          {count}
        </span>
      </div>
      <div className={`${compact ? "max-h-[224px]" : "max-h-[300px]"} overflow-y-auto`}>
        {items.length === 0 ? (
          <p className="p-4 text-sm text-muted-foreground">{empty}</p>
        ) : (
          <div className="divide-y">
            {items.map((item) => (
              <ComparisonCompactItem key={item.key} item={item} />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
