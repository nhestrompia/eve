import { humanDate, statusLabel } from "../../format";
import type { ComparisonResponse } from "../../types";
import { ComparisonListPanel } from "./comparison-list-panel";
import { ComparisonStat } from "./comparison-stat";
import { ComparisonTimelineCompact } from "./comparison-timeline-compact";

export function ComparisonBoard({
  comparison,
}: {
  comparison: ComparisonResponse;
}): React.JSX.Element {
  const totalChanges =
    comparison.added.length +
    comparison.changed.length +
    comparison.fixed.length;
  const selectedSnapshotCount = comparison.range.length + 1;

  return (
    <section className="space-y-4">
      <div className="space-y-4">
        <div className="rounded-lg bg-card p-5 shadow-[var(--shadow-border)]">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="max-w-[68ch]">
              <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                Comparison summary
              </p>
              <h3 className="mt-1 text-xl font-semibold text-balance">
                {comparison.from.title} to {comparison.to.title}
              </h3>
              <p className="mt-2 text-sm leading-6 text-muted-foreground text-pretty">
                Product changes, decisions, risks, validation, and timeline
                entries are grouped so the delta is visible without reading the
                full record stream.
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
              <ComparisonStat label="Selected" value={selectedSnapshotCount} />
              <ComparisonStat label="Changes" value={totalChanges} />
              <ComparisonStat label="Decisions" value={comparison.decisions.length} />
              <ComparisonStat label="Risks" value={comparison.risks.length} />
            </div>
          </div>
        </div>

        <div className="grid gap-4 lg:grid-cols-3">
          <ComparisonListPanel
            title="Added"
            count={comparison.added.length}
            empty="No added product changes."
            items={comparison.added.map((item) => ({
              key: `${item.snapshotId}-${item.text}`,
              snapshotId: item.snapshotId,
              title: item.text,
              meta: `${item.snapshotTitle} · ${humanDate(item.createdAt)}`,
            }))}
          />
          <ComparisonListPanel
            title="Changed"
            count={comparison.changed.length}
            empty="No changed product behavior."
            items={comparison.changed.map((item) => ({
              key: `${item.snapshotId}-${item.text}`,
              snapshotId: item.snapshotId,
              title: item.text,
              meta: `${item.snapshotTitle} · ${humanDate(item.createdAt)}`,
            }))}
          />
          <ComparisonListPanel
            title="Fixed"
            count={comparison.fixed.length}
            empty="No fixes in this span."
            items={comparison.fixed.map((item) => ({
              key: `${item.snapshotId}-${item.text}`,
              snapshotId: item.snapshotId,
              title: item.text,
              meta: `${item.snapshotTitle} · ${humanDate(item.createdAt)}`,
            }))}
          />
        </div>

        <ComparisonTimelineCompact items={comparison.timeline} />
      </div>

      <aside className="grid gap-4 lg:grid-cols-3">
        <ComparisonListPanel
          title="Decisions"
          count={comparison.decisions.length}
          empty="No decisions recorded."
          compact
          items={comparison.decisions.map((item) => ({
            key: `${item.snapshotId}-${item.title}`,
            snapshotId: item.snapshotId,
            title: item.title,
            meta: item.rationale || item.snapshotTitle,
          }))}
        />
        <ComparisonListPanel
          title="Risks"
          count={comparison.risks.length}
          empty="No risks recorded."
          compact
          items={comparison.risks.map((item) => ({
            key: `${item.snapshotId}-${item.title}`,
            snapshotId: item.snapshotId,
            title: item.title,
            meta: `${statusLabel(item.severity)}${item.mitigation ? ` · ${item.mitigation}` : ""}`,
          }))}
        />
        <ComparisonListPanel
          title="Validation"
          count={comparison.validation.length}
          empty="No validation recorded."
          compact
          items={comparison.validation.map((item) => ({
            key: `${item.snapshotId}-${item.command}`,
            snapshotId: item.snapshotId,
            title: item.command,
            meta: `${item.snapshotTitle} · ${statusLabel(item.status)}`,
          }))}
        />
      </aside>
    </section>
  );
}
