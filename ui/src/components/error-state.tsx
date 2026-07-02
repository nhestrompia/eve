export function ErrorState({ error }: { error: unknown }) {
  const message = error instanceof Error ? error.message : String(error);
  return (
    <div className="rounded-lg border bg-white p-8">
      <p className="mb-2 font-mono text-xs text-red-600">Unable to load</p>
      <h1 className="text-xl font-semibold text-balance">{message}</h1>
    </div>
  );
}
