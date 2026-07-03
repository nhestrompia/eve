import { Link, useRouterState } from '@tanstack/react-router';
import { Link as LinkIcon, MoreHorizontal, Search } from 'lucide-react';
import { useState } from 'react';
import { Button } from './ui/button';

export function TopBar({ onSearch }: { onSearch: () => void }) {
  const state = useRouterState();
  const [copied, setCopied] = useState(false);
  const match = state.location.pathname.match(/EV-\d+/);
  const id = match?.[0];
  const hasDetailRail = /^\/evolutions\/EV-\d+$/.test(state.location.pathname);

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
        <span>Activity</span>
        {id ? (
          <>
            <span>›</span>
            <span className="font-semibold text-foreground">#{id.replace('EV-', '')}</span>
          </>
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
