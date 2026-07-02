import { Link } from '@tanstack/react-router';
import { Search, X } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { Button } from './ui/button';
import { Input } from './ui/input';

export function SearchCommand({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const [query, setQuery] = useState('');
  const results = useQuery({
    queryKey: ['command-search', query],
    queryFn: () => api.search(query),
    enabled: open && query.trim().length > 0
  });

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        onOpenChange(true);
      }
      if (event.key === 'Escape') onOpenChange(false);
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [onOpenChange]);

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-40 flex items-start justify-center bg-slate-950/30 pt-28">
      <div className="w-[720px] rounded-lg border bg-white p-4 shadow-lg">
        <div className="flex items-center gap-3">
          <Search className="size-4 text-muted-foreground" />
          <Input
            autoFocus
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search evolutions, behavior, sessions, commits..."
            className="border-0 shadow-none focus-visible:ring-0"
          />
          <Button variant="ghost" size="icon" aria-label="Close search" onClick={() => onOpenChange(false)}>
            <X className="size-4" />
          </Button>
        </div>
        <div className="mt-4 max-h-[420px] overflow-auto">
          {query.trim() === '' ? <p className="p-4 text-muted-foreground">Type to search product history.</p> : null}
          {results.data?.results.length === 0 ? <p className="p-4 text-muted-foreground">No matching Evolutions.</p> : null}
          <div className="space-y-2">
            {results.data?.results.map((result) => (
              <Link
                key={result.evolution.id}
                to="/evolutions/$id"
                params={{ id: result.evolution.id }}
                onClick={() => onOpenChange(false)}
                className="grid grid-cols-[80px_minmax(0,1fr)] gap-4 rounded-lg border p-3 hover:bg-slate-50"
              >
                <span className="font-mono font-semibold text-blue-700">{result.evolution.id}</span>
                <span className="min-w-0">
                  <span className="block truncate font-semibold">{result.evolution.title}</span>
                  <span className="block truncate text-muted-foreground">{result.matches[0]}</span>
                </span>
              </Link>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
