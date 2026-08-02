import { useQuery } from "@tanstack/react-query";
import {
  GitCompareArrows,
  RotateCcw,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { api } from "../../api";
import { LoadingState } from "../../components/loading-state";
import { Button } from "../../components/ui/button";
import {
  compareEvolutionOrder,
  defaultComparisonPair,
  orderedComparisonPair,
} from "../../lib/comparison";
import type { EvolutionSummary } from "../../types";
import { ComparisonBoard } from "./comparison-board";
import { ComparisonInlineError } from "./comparison-inline-error";
import { ComparisonRangeControls } from "./comparison-range-controls";
import { comparisonPairByCount } from "./comparison-utils";

export function RepositoryComparePanel({
  evolutions,
}: {
  evolutions: EvolutionSummary[];
}): React.JSX.Element {
  const [fromId, setFromId] = useState("");
  const [toId, setToId] = useState("");
  const [activeEndpoint, setActiveEndpoint] = useState<"from" | "to">("from");
  const [snapshotQuery, setSnapshotQuery] = useState("");
  const chronological = useMemo(
    () => [...evolutions].sort(compareEvolutionOrder),
    [evolutions],
  );

  useEffect(() => {
    if (chronological.length < 2) {
      setFromId("");
      setToId("");
      return;
    }
    const ids = new Set(chronological.map((evolution) => evolution.id));
    if (ids.has(fromId) && ids.has(toId)) return;
    const pair = defaultComparisonPair(chronological);
    setFromId(pair?.from ?? "");
    setToId(pair?.to ?? "");
  }, [chronological, fromId, toId]);

  const orderedPair = orderedComparisonPair(chronological, fromId, toId);
  const comparisonFromId = orderedPair?.from ?? "";
  const comparisonToId = orderedPair?.to ?? "";
  const comparison = useQuery({
    queryKey: ["compare", comparisonFromId, comparisonToId],
    queryFn: () => api.compare(comparisonFromId, comparisonToId),
    enabled: Boolean(comparisonFromId && comparisonToId),
  });
  const fromSnapshot = chronological.find(
    (evolution) => evolution.id === comparisonFromId,
  );
  const toSnapshot = chronological.find(
    (evolution) => evolution.id === comparisonToId,
  );
  const fromIndex = chronological.findIndex(
    (evolution) => evolution.id === comparisonFromId,
  );
  const toIndex = chronological.findIndex(
    (evolution) => evolution.id === comparisonToId,
  );
  const includedSnapshots =
    fromIndex >= 0 && toIndex >= 0
      ? chronological.slice(fromIndex, toIndex + 1)
      : [];

  const setComparisonPair = (firstId: string, secondId: string) => {
    const next = orderedComparisonPair(chronological, firstId, secondId);
    if (!next) return;
    setFromId(next.from);
    setToId(next.to);
  };

  const pickSnapshot = (id: string) => {
    if (activeEndpoint === "from") {
      setComparisonPair(id, comparisonToId || toId);
      setActiveEndpoint("to");
      return;
    }
    setComparisonPair(comparisonFromId || fromId, id);
    setActiveEndpoint("from");
  };

  const resetToLatestRange = () => {
    const pair = defaultComparisonPair(chronological);
    if (pair) {
      setFromId(pair.from);
      setToId(pair.to);
    }
  };

  if (chronological.length < 2) {
    return (
      <section className="rounded-lg bg-card p-5 shadow-[var(--shadow-border)]">
        <h2 className="flex items-center gap-2 text-base font-semibold">
          <GitCompareArrows className="size-4 text-blue-600" />
          Compare snapshots
        </h2>
        <p className="mt-3 text-sm text-muted-foreground">
          Record at least two snapshots in this repository to compare product states.
        </p>
      </section>
    );
  }

  return (
    <section className="space-y-4">
      <section className="overflow-hidden rounded-lg bg-card text-card-foreground shadow-[var(--shadow-border)]">
        <div className="p-5 sm:p-6">
          <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div className="max-w-[64ch]">
              <p className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                Product range
              </p>
              <h2 className="mt-2 flex items-center gap-2 text-xl font-semibold text-balance">
                <GitCompareArrows className="size-5 text-blue-600" />
                Compare snapshots
              </h2>
              <p className="mt-3 text-sm leading-6 text-muted-foreground text-pretty">
                Choose an endpoint, then search snapshots by title, type, date,
                or id. EVE keeps the final range in chronological order.
              </p>
            </div>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="min-h-10 shrink-0 gap-2 transition-[scale,background-color,box-shadow] active:scale-[0.96]"
              onClick={resetToLatestRange}
            >
              <RotateCcw className="size-3.5" />
              Latest range
            </Button>
          </div>

          <ComparisonRangeControls
            evolutions={chronological}
            fromId={comparisonFromId}
            toId={comparisonToId}
            fromSnapshot={fromSnapshot}
            toSnapshot={toSnapshot}
            includedSnapshots={includedSnapshots}
            activeEndpoint={activeEndpoint}
            query={snapshotQuery}
            onQueryChange={setSnapshotQuery}
            onEndpointChange={setActiveEndpoint}
            onPickSnapshot={pickSnapshot}
            onPreset={(count) => {
              const pair = comparisonPairByCount(chronological, count);
              if (pair) {
                setFromId(pair.from);
                setToId(pair.to);
              }
            }}
          />
        </div>
      </section>

      <div className="min-w-0">
        {!orderedPair ? (
          <div className="rounded-lg bg-card p-5 text-sm text-muted-foreground shadow-[var(--shadow-border)]">
            Choose two different snapshots to compare.
          </div>
        ) : null}
        {comparison.isLoading ? <LoadingState label="Comparing Snapshots" /> : null}
        {comparison.error ? <ComparisonInlineError error={comparison.error} /> : null}
        {comparison.data ? <ComparisonBoard comparison={comparison.data} /> : null}
      </div>
    </section>
  );
}
