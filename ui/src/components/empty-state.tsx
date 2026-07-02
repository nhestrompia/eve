export function EmptyState({ title, detail }: { title: string; detail: string }) {
  return (
    <div className="rounded-lg border bg-white p-8">
      <p className="mb-2 font-mono text-xs text-muted-foreground">EVE UI</p>
      <h1 className="text-2xl font-semibold text-balance">{title}</h1>
      <p className="mt-2 max-w-xl text-muted-foreground text-pretty">{detail}</p>
    </div>
  );
}
