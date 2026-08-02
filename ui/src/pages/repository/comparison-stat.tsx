export function ComparisonStat({
  label,
  value,
}: {
  label: string;
  value: number;
}): React.JSX.Element {
  return (
    <div className="min-w-20 rounded-lg bg-secondary px-3 py-2 text-right shadow-[var(--shadow-border)]">
      <div className="text-lg font-semibold tabular-nums">{value}</div>
      <div className="text-[11px] font-medium text-muted-foreground">{label}</div>
    </div>
  );
}
