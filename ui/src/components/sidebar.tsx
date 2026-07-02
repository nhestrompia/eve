import { Link, useNavigate } from '@tanstack/react-router';
import { BookOpen, FileClock, GitBranch, History, Plus, Search, Settings, Sun } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { FormEvent, useState } from 'react';
import { api } from '../api';
import { Button } from './ui/button';
import { Input } from './ui/input';

export function Sidebar() {
  const config = useQuery({ queryKey: ['config'], queryFn: api.config });
  const evolutions = useQuery({ queryKey: ['evolutions'], queryFn: api.evolutions });
  const repositories = useQuery({ queryKey: ['repositories'], queryFn: api.repositories });
  const firstEvolution = evolutions.data?.[0]?.id;
  const navigate = useNavigate();
  const [query, setQuery] = useState('');

  const submitSearch = (event: FormEvent) => {
    event.preventDefault();
    void navigate({ to: '/search', search: { q: query } });
  };

  return (
    <aside className="sticky top-0 flex h-dvh flex-col border-r bg-white/78">
      <div className="flex h-[76px] items-center gap-3 px-7">
        <div className="flex size-9 items-center justify-center rounded-full bg-slate-950 text-white">
          <GitBranch className="size-5" />
        </div>
        <span className="text-[26px] font-semibold text-balance">EVE</span>
      </div>

      <form onSubmit={submitSearch} className="px-5 pb-5">
        <label className="sr-only" htmlFor="sidebar-search">
          Search Evolutions
        </label>
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            id="sidebar-search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search..."
            className="h-11 rounded-lg bg-white pl-10 pr-12 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]"
          />
          <kbd className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 font-mono text-xs text-muted-foreground">⌘K</kbd>
        </div>
      </form>

      <nav className="space-y-1 px-4">
        <Link
          to="/"
          className="flex h-12 items-center gap-4 rounded-lg px-4 font-medium text-slate-950"
          activeProps={{ className: 'bg-slate-100 shadow-sm' }}
        >
          <History className="size-4 text-blue-600" />
          Activity
        </Link>
        {firstEvolution ? (
          <Link
            to="/evolutions/$id/snapshot"
            params={{ id: firstEvolution }}
            className="flex h-12 items-center gap-4 rounded-lg px-4 text-muted-foreground hover:bg-slate-50 hover:text-foreground"
          >
            <FileClock className="size-4" />
            Snapshots
          </Link>
        ) : (
          <span
            aria-disabled="true"
            className="flex h-12 items-center gap-4 rounded-lg px-4 text-muted-foreground opacity-60"
          >
            <FileClock className="size-4" />
            Snapshots
          </span>
        )}
        <Link
          to="/search"
          search={{ q: '' }}
          className="flex h-12 items-center gap-4 rounded-lg px-4 text-muted-foreground hover:bg-slate-50 hover:text-foreground"
        >
          <Search className="size-4" />
          Search
        </Link>
      </nav>

      <div className="mx-7 my-6 border-t" />

      <div className="px-5">
        <p className="mb-3 px-2 text-xs font-medium text-muted-foreground">Repositories</p>
        <div className="space-y-1">
          {(repositories.data?.length ? repositories.data : [{ name: config.data?.repository ?? 'eve', evolutionCount: 0 }]).map((repo) => (
            <Link
              key={repo.name}
              to="/repositories/$repo"
              params={{ repo: repo.name }}
              className="flex min-h-12 items-center justify-between rounded-lg px-3 hover:bg-slate-50"
              activeProps={{ className: 'bg-slate-100 shadow-sm' }}
            >
              <span className="flex min-w-0 items-center gap-3">
                <BookOpen className="size-4 shrink-0 text-slate-500" />
                <span className="truncate font-semibold">{repo.name}</span>
              </span>
              <span className="ml-3 rounded-full bg-slate-100 px-2 py-0.5 text-xs text-muted-foreground">{repo.evolutionCount}</span>
            </Link>
          ))}
        </div>
        <Link
          to="/config"
          className="mt-3 flex min-h-10 items-center gap-3 rounded-lg px-2 text-muted-foreground hover:bg-slate-50 hover:text-foreground"
        >
          <Plus className="size-4" />
          Repository config
        </Link>
      </div>

      <div className="mt-auto flex items-center justify-between border-t p-5">
        <div>
          <p className="font-semibold leading-4">Local</p>
          <p className="text-xs text-muted-foreground">{config.data?.repository ?? 'repository'}</p>
        </div>
        <div className="flex gap-1">
          <Button asChild variant="ghost" size="icon" aria-label="Open EVE config">
            <Link to="/config">
              <Settings className="size-4" />
            </Link>
          </Button>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Toggle theme"
            onClick={() => document.documentElement.classList.toggle('dark-preview')}
          >
            <Sun className="size-4" />
          </Button>
        </div>
      </div>
    </aside>
  );
}
