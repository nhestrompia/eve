import { Link } from '@tanstack/react-router';
import { Search, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { Button } from './ui/button';
import { Input } from './ui/input';

export function SearchCommand({
  open,
  initialQuery = '',
  onOpenChange
}: {
  open: boolean;
  initialQuery?: string;
  onOpenChange: (open: boolean) => void;
}) {
  const [query, setQuery] = useState('');
  const [debouncedQuery, setDebouncedQuery] = useState('');
  const results = useQuery({
    queryKey: ['command-search', debouncedQuery],
    queryFn: () => api.search(debouncedQuery),
    enabled: open,
    staleTime: 5_000
  });

  useEffect(() => {
    if (open) setQuery(initialQuery);
  }, [initialQuery, open]);

  useEffect(() => {
    if (!open) return;
    const timeout = window.setTimeout(() => setDebouncedQuery(query), 120);
    return () => window.clearTimeout(timeout);
  }, [open, query]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onOpenChange(false);
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [onOpenChange]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-40 flex items-start justify-center bg-slate-950/35 px-4 pt-20 backdrop-blur-[1px] sm:pt-28" role="dialog" aria-modal="true" aria-label="Search snapshots">
      <div className="w-full max-w-[760px] rounded-lg border bg-card p-4 text-card-foreground shadow-[0_24px_80px_-36px_rgba(15,23,42,0.65),0_0_0_1px_rgba(15,23,42,0.1)]">
        <div className="flex items-center gap-3">
          <Search className="size-4 text-muted-foreground" />
          <Input
            autoFocus
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search snapshots, summaries, validation, commits..."
            className="border-0 shadow-none focus-visible:ring-0"
          />
          <Button variant="ghost" size="icon" aria-label="Close search" onClick={() => onOpenChange(false)}>
            <X className="size-4" />
          </Button>
        </div>
        <div className="mt-4 max-h-[420px] overflow-auto">
          {results.isLoading || query !== debouncedQuery ? <p className="p-4 text-muted-foreground">Searching...</p> : null}
          {results.data?.results.length === 0 ? <p className="p-4 text-muted-foreground">No matching Snapshots.</p> : null}
          <div className="space-y-2">
            {results.data?.results.map((result) => (
              <SearchResultLink key={result.evolution.id} result={result} onSelect={() => onOpenChange(false)} />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function SearchResultLink({
  result,
  onSelect
}: {
  result: Awaited<ReturnType<typeof api.search>>['results'][number];
  onSelect: () => void;
}) {
  const subtitle = result.matches.find((match) => match !== result.evolution.title) || result.evolution.outcome;

  return (
    <Link
      to="/snapshots/$id"
      params={{ id: result.evolution.id }}
      onClick={onSelect}
      className="block rounded-lg border p-3 hover:bg-slate-50"
    >
      <span className="min-w-0">
        <span className="block text-sm font-semibold text-pretty">{result.evolution.title}</span>
        <span className="block truncate text-muted-foreground">{subtitle}</span>
      </span>
    </Link>
  );
}
