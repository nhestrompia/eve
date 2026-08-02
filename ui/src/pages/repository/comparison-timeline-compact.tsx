import { Link } from "@tanstack/react-router";

import { compactDate, statusLabel } from "../../format";
import type { ComparisonTimelineItem } from "../../types";

export function ComparisonTimelineCompact({
  items,
}: {
  items: ComparisonTimelineItem[];
}): React.JSX.Element {
  return (
    <section className="rounded-lg bg-card p-4 shadow-[var(--shadow-border)]">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 className="text-sm font-semibold">Timeline</h3>
        <span className="rounded-md bg-secondary px-2 py-1 text-xs font-medium text-muted-foreground tabular-nums">
          {items.length}
        </span>
      </div>
      {items.length === 0 ? (
        <p className="mt-3 text-sm text-muted-foreground">
          No timeline entries recorded.
        </p>
      ) : (
        <div className="mt-4 overflow-x-auto overscroll-x-contain pb-2">
          <div className="relative min-w-max">
            {items.length > 1 ? (
              <span className="absolute left-[15px] right-[15px] top-[7px] h-px bg-blue-600" />
            ) : null}
            <div className="flex gap-3">
              {items.map((item) => (
                <Link
                  key={`${item.snapshotId}-${item.phase}-${item.title}-${item.occurredAt}`}
                  to="/snapshots/$id"
                  params={{ id: item.snapshotId }}
                  className="group relative w-[260px] shrink-0 pt-7"
                >
                  <span className="absolute left-0 top-0 flex size-3.5 rounded-full border-2 border-blue-600 bg-card ring-4 ring-secondary transition-[transform,box-shadow] group-hover:scale-110 group-hover:ring-accent" />
                  <span className="block min-h-[140px] rounded-lg bg-secondary px-4 py-3 shadow-[var(--shadow-border)] transition-[background-color,box-shadow] group-hover:bg-accent">
                    <span className="text-[11px] font-semibold uppercase tracking-wide text-primary">
                      {statusLabel(item.phase)}
                    </span>
                    <span className="mt-2 block line-clamp-2 text-sm font-semibold leading-5 text-foreground">
                      {item.title}
                    </span>
                    <span className="mt-2 block line-clamp-2 text-xs leading-5 text-muted-foreground">
                      {item.snapshotTitle} · {compactDate(item.occurredAt)}
                    </span>
                  </span>
                </Link>
              ))}
            </div>
          </div>
        </div>
      )}
    </section>
  );
}
