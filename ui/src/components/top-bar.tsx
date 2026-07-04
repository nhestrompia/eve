import { useMutation, useQuery } from '@tanstack/react-query';
import { Link, useRouterState } from '@tanstack/react-router';
import { Code2, ExternalLink, Link as LinkIcon, MoreHorizontal, Search } from 'lucide-react';
import { useState } from 'react';
import { api } from '../api';
import { Button } from './ui/button';

export function TopBar({ onSearch }: { onSearch: () => void }) {
  const state = useRouterState();
  const config = useQuery({ queryKey: ['config'], queryFn: api.config, refetchInterval: 10_000 });
  const [copied, setCopied] = useState(false);
  const match = state.location.pathname.match(/snap_[^/]+|EV-\d+/);
  const id = match?.[0];
  const repoMatch = state.location.pathname.match(/^\/repositories\/([^/]+)/);
  const repo = repoMatch ? decodeURIComponent(repoMatch[1]) : undefined;
  const hasDetailRail = /^\/snapshots\/[^/]+$/.test(state.location.pathname);
  const isSnapshotRoute = /^\/snapshots\/[^/]+/.test(state.location.pathname);
  const currentGitState = config.data?.currentGitState;
  const latestGitState = config.data?.latestGitState;
  const isBehindLatest = Boolean(isSnapshotRoute && currentGitState && latestGitState && currentGitState !== latestGitState);
  const repository = useQuery({
    queryKey: ['repository', repo],
    queryFn: () => api.repository(repo ?? ''),
    enabled: !!repo
  });
  const openEditor = useMutation({
    mutationFn: () => api.openRepositoryInEditor(repo ?? '')
  });
  const remoteUrl = repository.data?.remoteUrl;

  const copyURL = async () => {
    await navigator.clipboard.writeText(window.location.href);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  };

  return (
    <header
      className={`sticky top-0 z-20 flex min-h-16 items-center justify-between gap-3 border-b bg-white/88 px-4 backdrop-blur md:h-[76px] md:px-8 ${hasDetailRail ? 'xl:pr-[488px]' : ''}`}
    >
      <div className="flex min-w-0 items-center gap-3 text-sm text-muted-foreground">
        {repo ? (
          <>
            <span>Repositories</span>
            <span>›</span>
            <span className="truncate font-semibold text-foreground">{repo}</span>
          </>
        ) : (
          <span>Activity</span>
        )}
        {!repo && id ? (
          <>
            <span>›</span>
            <span className="truncate font-semibold text-foreground">{id}</span>
          </>
        ) : null}
        {isBehindLatest ? (
          <span className="hidden max-w-[260px] truncate rounded-md bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700 shadow-[0_0_0_1px_rgba(245,158,11,0.16)] lg:inline">
            Viewing an earlier product version
          </span>
        ) : null}
      </div>
      <div className="flex shrink-0 items-center justify-end gap-2 md:gap-3">
        <Button
          variant="outline"
          size="icon"
          aria-label="Open search"
          title="Open search"
          onClick={onSearch}
          className="md:hidden"
        >
          <Search className="size-4" />
        </Button>
        {repo ? (
          <>
            <Button
              variant="outline"
              className="h-10 gap-2 rounded-lg px-3 md:px-4"
              aria-label="Open repository in editor"
              title={openEditor.data?.stderr || 'Open repository in editor'}
              disabled={openEditor.isPending}
              onClick={() => openEditor.mutate()}
            >
              <Code2 className="size-4" />
              <span className="hidden sm:inline">{openEditor.isPending ? 'Opening...' : 'Open in editor'}</span>
            </Button>
            {remoteUrl ? (
              <Button asChild variant="outline" className="h-10 gap-2 rounded-lg px-3 md:px-4">
                <a href={remoteUrl} target="_blank" rel="noreferrer" aria-label="View repository on GitHub" title="View repository on GitHub">
                  <ExternalLink className="size-4" />
                  <span className="hidden sm:inline">View on GitHub</span>
                </a>
              </Button>
            ) : (
              <Button
                variant="outline"
                className="h-10 gap-2 rounded-lg px-3 md:px-4"
                aria-label="No GitHub remote configured"
                title="No GitHub remote configured"
                disabled
              >
                <ExternalLink className="size-4" />
                <span className="hidden sm:inline">View on GitHub</span>
              </Button>
            )}
          </>
        ) : null}
        <Button
          variant="outline"
          className="h-10 gap-2 rounded-lg px-3 md:px-4"
          aria-label={copied ? 'Page link copied' : 'Copy page link'}
          title={copied ? 'Copied' : 'Copy page link'}
          onClick={copyURL}
        >
          <LinkIcon className="size-4" />
          <span className="hidden sm:inline">{copied ? 'Copied' : 'Copy link'}</span>
        </Button>
        <Button asChild variant="outline" size="icon" aria-label="Open EVE config" title="Open EVE config">
          <Link to="/config">
            <MoreHorizontal className="size-5" />
          </Link>
        </Button>
      </div>
    </header>
  );
}
