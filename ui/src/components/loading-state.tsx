export function LoadingState({ label = 'Loading' }: { label?: string }) {
  return <div className="rounded-lg border bg-white p-8 font-mono text-sm text-muted-foreground">{label}...</div>;
}
