import { Link } from '@tanstack/react-router';
import { BookOpen, ChevronDown, FileClock, GitBranch, History, Search, Settings, Sun, UserRound } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { Button } from './ui/button';

export function Sidebar() {
  const config = useQuery({ queryKey: ['config'], queryFn: api.config });

  return (
    <aside className="sticky top-0 flex h-dvh flex-col border-r bg-white/78">
      <div className="flex h-[76px] items-center gap-3 px-7">
        <div className="flex size-9 items-center justify-center rounded-full bg-slate-950 text-white">
          <GitBranch className="size-5" />
        </div>
        <span className="text-[26px] font-semibold text-balance">EVE</span>
      </div>

      <nav className="space-y-1 px-4">
        <Link
          to="/"
          className="flex h-12 items-center gap-4 rounded-lg px-4 font-medium text-slate-950"
          activeProps={{ className: 'bg-slate-100 shadow-sm' }}
        >
          <History className="size-4 text-blue-600" />
          History
        </Link>
        <Link
          to="/"
          className="flex h-12 items-center gap-4 rounded-lg px-4 text-muted-foreground hover:bg-slate-50 hover:text-foreground"
        >
          <FileClock className="size-4" />
          Snapshots
        </Link>
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
        <div className="flex h-12 items-center justify-between rounded-lg bg-slate-50 px-3">
          <div className="flex items-center gap-3">
            <BookOpen className="size-4 text-slate-500" />
            <span className="font-semibold">{config.data?.repository ?? 'eve'}</span>
          </div>
          <span className="size-2 rounded-full bg-emerald-500" />
        </div>
        <Button variant="ghost" className="mt-2 h-10 w-full justify-start gap-3 px-2 text-muted-foreground">
          <span className="text-xl leading-none">+</span>
          Add Repository
        </Button>
      </div>

      <div className="mt-auto flex items-center justify-between border-t p-5">
        <div className="flex items-center gap-3">
          <div className="flex size-10 items-center justify-center rounded-full bg-slate-200">
            <UserRound className="size-5 text-slate-600" />
          </div>
          <div>
            <p className="font-semibold leading-4">Umut</p>
            <p className="text-xs text-muted-foreground">Local · ~/.eve</p>
          </div>
        </div>
        <div className="flex gap-1">
          <Button variant="ghost" size="icon" aria-label="Settings">
            <Settings className="size-4" />
          </Button>
          <Button variant="ghost" size="icon" aria-label="Theme">
            <Sun className="size-4" />
          </Button>
          <Button variant="ghost" size="icon" aria-label="Account menu">
            <ChevronDown className="size-4" />
          </Button>
        </div>
      </div>
    </aside>
  );
}
