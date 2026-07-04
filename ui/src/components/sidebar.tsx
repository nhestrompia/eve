import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import {
  BookOpen,
  History,
  Plus,
  Search,
  Settings,
  Sun,
} from "lucide-react";
import { ChangeEvent, FormEvent, useEffect, useMemo, useState } from "react";
import { api } from "../api";
import { Button } from "./ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "./ui/dialog";
import { Input } from "./ui/input";

export function Sidebar({ onSearch }: { onSearch: (query?: string) => void }) {
  const config = useQuery({ queryKey: ["config"], queryFn: api.config });
  const evolutions = useQuery({
    queryKey: ["evolutions"],
    queryFn: api.snapshots,
  });
  const repositories = useQuery({
    queryKey: ["repositories"],
    queryFn: api.repositories,
  });
  const navigate = useNavigate();
  const [repositoryName, setRepositoryName] = useState("");
  const [repositoryDialogOpen, setRepositoryDialogOpen] = useState(false);
  const [localRepositories, setLocalRepositories] = useState<string[]>([]);

  useEffect(() => {
    const raw = window.localStorage.getItem(LOCAL_REPOSITORIES_KEY);
    if (!raw) return;
    try {
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed)) {
        setLocalRepositories(parsed.filter((value): value is string => typeof value === "string"));
      }
    } catch {
      setLocalRepositories([]);
    }
  }, []);

  const repositoryRows = useMemo(
    () =>
      mergeSidebarRepositories(
        repositories.data ?? [],
        localRepositories,
        config.data?.repository ?? "eve"
      ),
    [repositories.data, localRepositories, config.data?.repository]
  );

  const submitSearch = (event: FormEvent) => {
    event.preventDefault();
    onSearch();
  };

  const openSearchFromInput = (event: ChangeEvent<HTMLInputElement>) => {
    onSearch(event.target.value);
  };

  const addRepository = (event: FormEvent) => {
    event.preventDefault();
    const name = repositoryName.trim();
    if (!name) return;
    const next = Array.from(new Set([...localRepositories, name])).sort((left, right) =>
      left.localeCompare(right)
    );
    setLocalRepositories(next);
    window.localStorage.setItem(LOCAL_REPOSITORIES_KEY, JSON.stringify(next));
    setRepositoryName("");
    setRepositoryDialogOpen(false);
    void navigate({ to: "/repositories/$repo", params: { repo: name } });
  };

  return (
    <aside className="flex flex-col border-b bg-white/78 md:fixed md:inset-y-0 md:left-0 md:z-30 md:w-[240px] md:overflow-y-auto md:border-b-0 md:border-r">
      <Link to="/" aria-label="Go to activity" className="flex h-16 items-center px-4 transition-opacity hover:opacity-80 md:h-[76px] md:px-7">
        <img src="/eve.svg" alt="eve" className="h-10 w-[108px] object-contain object-left" />
      </Link>

      <form onSubmit={submitSearch} className="hidden px-5 pb-5 md:block">
        <label className="sr-only" htmlFor="sidebar-search">
          Search Snapshots
        </label>
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            id="sidebar-search"
            value=""
            onFocus={() => onSearch()}
            onChange={openSearchFromInput}
            placeholder="Search..."
            className="h-11 rounded-lg bg-white pl-10 pr-12 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]"
          />
          <kbd className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 font-mono text-xs text-muted-foreground">
            ⌘K
          </kbd>
        </div>
      </form>

      <nav className="flex gap-1 overflow-x-auto px-3 pb-3 md:block md:space-y-1 md:overflow-visible md:px-4 md:pb-0">
        <Link
          to="/"
          className="flex h-11 shrink-0 items-center gap-3 rounded-lg px-4 font-medium text-slate-950 md:h-12 md:gap-4"
          activeProps={{ className: "bg-slate-100 shadow-sm" }}
        >
          <History className="size-4 text-blue-600" />
          Activity
        </Link>
      </nav>

      <div className="mx-4 border-t md:mx-7 md:my-6" />

      <div className="px-4 py-3 md:px-5 md:py-0">
        <p className="mb-3 px-2 text-xs font-medium text-muted-foreground">
          Repositories
        </p>
        <div className="flex gap-2 overflow-x-auto md:block md:space-y-1 md:overflow-visible">
          {repositoryRows.map((repo) => (
            <Link
              key={repo.name}
              to="/repositories/$repo"
              params={{ repo: repo.name }}
              className="flex min-h-11 shrink-0 items-center justify-between gap-3 rounded-lg px-3 hover:bg-slate-50 md:min-h-12 md:w-full"
              activeProps={{ className: "bg-slate-100 shadow-sm" }}
            >
              <span className="flex min-w-0 items-center gap-3">
                <BookOpen className="size-4 shrink-0 text-slate-500" />
                <span className="max-w-36 truncate font-semibold md:max-w-none">
                  {repo.name}
                </span>
              </span>
              <span className="ml-3 rounded-full bg-slate-100 px-2 py-0.5 text-xs text-muted-foreground">
                {repo.evolutionCount}
              </span>
            </Link>
          ))}
        </div>
        <Dialog open={repositoryDialogOpen} onOpenChange={setRepositoryDialogOpen}>
          <DialogTrigger asChild>
            <button className="mt-3 flex min-h-10 w-fit items-center gap-3 rounded-lg px-2 text-muted-foreground hover:bg-slate-50 hover:text-foreground md:w-auto">
              <Plus className="size-4" />
              Add repository
            </button>
          </DialogTrigger>
          <DialogContent className="max-w-[560px]">
            <DialogHeader>
              <DialogTitle>Add repository</DialogTitle>
              <DialogDescription>
                Add a repository shortcut to this browser. EVE counts it once an Evolution records it in implementation metadata.
              </DialogDescription>
            </DialogHeader>
            <form className="space-y-4" onSubmit={addRepository}>
              <div>
                <label htmlFor="repository-name" className="text-sm font-medium">
                  Repository name
                </label>
                <Input
                  id="repository-name"
                  value={repositoryName}
                  onChange={(event) => setRepositoryName(event.target.value)}
                  placeholder="docs"
                  className="mt-2"
                />
              </div>
              <div className="rounded-lg bg-slate-50 p-4">
                <p className="text-sm font-medium">Record future work with:</p>
                <code className="mt-2 block break-all font-mono text-xs text-muted-foreground">
                  eve add implementation --repository {repositoryName.trim() || "<name>"} --status merged
                </code>
              </div>
              <div className="flex justify-end gap-3">
                <Button type="button" variant="outline" onClick={() => setRepositoryDialogOpen(false)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={!repositoryName.trim()}>
                  Add repository
                </Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <div className="mt-auto flex items-center justify-between border-t p-4 md:p-5">
        <div>
          <p className="font-semibold leading-4">Local</p>
          <p className="text-xs text-muted-foreground">
            {config.data?.repository ?? "repository"}
          </p>
        </div>
        <div className="flex gap-1">
          <Button
            asChild
            variant="ghost"
            size="icon"
            aria-label="Open EVE config"
          >
            <Link to="/config">
              <Settings className="size-4" />
            </Link>
          </Button>
          <Button
            variant="ghost"
            size="icon"
            aria-label="Toggle theme"
            onClick={() =>
              document.documentElement.classList.toggle("dark-preview")
            }
          >
            <Sun className="size-4" />
          </Button>
        </div>
      </div>
    </aside>
  );
}

const LOCAL_REPOSITORIES_KEY = "eve:sidebar-repositories";

type SidebarRepository = {
  name: string;
  evolutionCount: number;
};

function mergeSidebarRepositories(
  repositories: SidebarRepository[],
  localRepositories: string[],
  fallbackRepository: string
) {
  const rows = new Map<string, SidebarRepository>();
  const source = repositories.length > 0 ? repositories : [{ name: fallbackRepository, evolutionCount: 0 }];
  for (const repo of source) {
    rows.set(repo.name, repo);
  }
  for (const name of localRepositories) {
    const trimmed = name.trim();
    if (!trimmed || rows.has(trimmed)) continue;
    rows.set(trimmed, { name: trimmed, evolutionCount: 0 });
  }
  return Array.from(rows.values()).sort((left, right) => {
    if (left.evolutionCount !== right.evolutionCount) {
      return right.evolutionCount - left.evolutionCount;
    }
    return left.name.localeCompare(right.name);
  });
}
