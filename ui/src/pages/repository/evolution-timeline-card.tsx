import { Link } from "@tanstack/react-router";
import { StatusBadge } from "../../components/status-badge";
import { compactDate } from "../../format";
import type { EvolutionSummary } from "../../types";

export function EvolutionTimelineCard({
  evolutions,
  spacious = false,
}: {
  evolutions: EvolutionSummary[];
  spacious?: boolean;
}): React.JSX.Element {
  return (
    <section
      className={`rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)] ${
        spacious ? "" : "xl:flex xl:h-[576px] xl:flex-col"
      }`}
    >
      <div className="mb-5 flex items-center gap-2">
        <h2 className="text-base font-semibold">Snapshot timeline</h2>
      </div>
      {evolutions.length === 0 ? (
        <p className="text-sm text-muted-foreground">No timeline entries yet.</p>
      ) : (
        <div
          className={`relative space-y-0 overflow-y-auto overscroll-contain pl-8 pr-1 ${
            spacious ? "max-h-[680px]" : "max-h-[520px] xl:min-h-0 xl:flex-1"
          }`}
        >
          {evolutions.map((evolution, index) => (
            <Link
              key={evolution.id}
              to="/snapshots/$id"
              params={{ id: evolution.id }}
              className="group relative block pb-7 last:pb-0"
            >
              {index < evolutions.length - 1 ? (
                <span className="absolute -left-[20px] top-5 h-full w-px bg-blue-600" />
              ) : null}
              <span className="absolute -left-[26px] top-1 flex size-3.5 rounded-full border-2 border-blue-600 bg-white ring-4 ring-blue-50" />
              <span className="flex min-w-0 flex-wrap items-start justify-between gap-3">
                <strong className="max-w-[26ch] text-sm font-semibold leading-5 text-balance group-hover:text-blue-700">
                  {evolution.title}
                </strong>
                <StatusBadge status={evolution.status} />
              </span>
              <span className="mt-2 block text-xs leading-5 text-muted-foreground">
                {evolution.type} · {compactDate(evolution.updatedAt || evolution.createdAt)}
              </span>
            </Link>
          ))}
        </div>
      )}
    </section>
  );
}
