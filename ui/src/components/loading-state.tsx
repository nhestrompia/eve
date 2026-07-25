import { Skeleton } from './ui/skeleton';

export function LoadingState({ label = 'Loading' }: { label?: string }) {
  return (
    <div className="rounded-lg border bg-white p-5" aria-label={`${label}...`} role="status">
      <span className="sr-only">{label}...</span>
      <div className="flex items-center gap-3">
        <Skeleton className="size-10 rounded-full" />
        <div className="min-w-0 flex-1 space-y-2">
          <Skeleton className="h-4 w-44 max-w-full" />
          <Skeleton className="h-3 w-72 max-w-full" />
        </div>
      </div>
      <div className="mt-6 grid gap-3 sm:grid-cols-2">
        <Skeleton className="h-24" />
        <Skeleton className="h-24" />
      </div>
      <div className="mt-4 space-y-3">
        <Skeleton className="h-12" />
        <Skeleton className="h-12" />
        <Skeleton className="h-12" />
      </div>
    </div>
  );
}
