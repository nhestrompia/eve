export function ComparisonInlineError({
  error,
}: {
  error: unknown;
}): React.JSX.Element {
  const message = error instanceof Error ? error.message : String(error);
  return (
    <div className="rounded-lg bg-card p-5 shadow-[0_0_0_1px_rgba(239,68,68,0.18),0_1px_2px_-1px_rgba(15,23,42,0.08)]">
      <p className="text-xs font-semibold uppercase tracking-wide text-red-600">
        Unable to compare
      </p>
      <p className="mt-2 text-sm font-medium text-foreground text-pretty">
        {message}
      </p>
    </div>
  );
}
