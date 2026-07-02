import { Link, useNavigate, useSearch } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { FormEvent, useState } from 'react';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';

export function SearchPage() {
  const search = useSearch({ from: '/search' });
  const navigate = useNavigate();
  const [query, setQuery] = useState(search.q ?? '');
  const evolutions = useQuery({ queryKey: ['evolutions'], queryFn: api.evolutions });
  const results = useQuery({
    queryKey: ['search', search.q],
    queryFn: () => api.search(search.q ?? ''),
    enabled: Boolean(search.q)
  });

  const submit = (event: FormEvent) => {
    event.preventDefault();
    navigate({ to: '/search', search: { q: query } });
  };

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={undefined}>
      <div className="space-y-6">
        <div>
          <p className="font-mono text-sm text-muted-foreground">Find State</p>
          <h1 className="mt-2 text-3xl font-semibold text-balance">Search</h1>
        </div>
        <form className="flex gap-3" onSubmit={submit}>
          <Input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search product history" />
          <Button type="submit">Search</Button>
        </form>
        {results.error ? <ErrorState error={results.error} /> : null}
        <div className="space-y-3">
          {results.data?.results.map((result) => (
            <Link
              key={result.evolution.id}
              to="/evolutions/$id"
              params={{ id: result.evolution.id }}
              className="grid grid-cols-[90px_minmax(0,1fr)] gap-4 rounded-lg border bg-white p-4 hover:bg-slate-50"
            >
              <span className="font-mono font-semibold text-blue-700">{result.evolution.id}</span>
              <span className="min-w-0">
                <strong className="block truncate">{result.evolution.title}</strong>
                <span className="block truncate text-muted-foreground">{result.matches.join(' · ')}</span>
              </span>
            </Link>
          ))}
        </div>
      </div>
    </EvolutionShell>
  );
}
