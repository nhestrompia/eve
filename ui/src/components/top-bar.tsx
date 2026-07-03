import { Link, useRouterState } from '@tanstack/react-router';
import { Link as LinkIcon, MoreHorizontal } from 'lucide-react';
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
      className={`sticky top-0 z-20 flex h-[76px] items-center justify-between border-b bg-white/88 pl-8 backdrop-blur ${hasDetailRail ? 'pr-[488px]' : 'pr-8'}`}
    >
      <button type="button" onClick={onSearch} className="sr-only">
        Open search
      </button>
      <div className="flex items-center gap-3 text-sm text-muted-foreground">
        <span>Activity</span>
        {id ? (
          <>
            <span>›</span>
            <span className="font-semibold text-foreground">#{id.replace('EV-', '')}</span>
          </>
        ) : null}
      </div>
      <div className="flex items-center justify-end gap-3">
        <Button
          variant="outline"
          className="h-10 gap-2 rounded-lg"
          aria-label={copied ? 'Page link copied' : 'Copy page link'}
          title={copied ? 'Copied' : 'Copy page link'}
          onClick={copyURL}
        >
          <LinkIcon className="size-4" />
          {copied ? 'Copied' : 'Copy link'}
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
