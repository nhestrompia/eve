import { useRouterState } from '@tanstack/react-router';
import { MoreHorizontal, Search, Sun } from 'lucide-react';
import { useState } from 'react';
import { Button } from './ui/button';

export function TopBar({ onSearch }: { onSearch: () => void }) {
  const state = useRouterState();
  const [copied, setCopied] = useState(false);
  const match = state.location.pathname.match(/EV-\d+/);
  const id = match?.[0] ?? '#42';

  const copyURL = async () => {
    await navigator.clipboard.writeText(window.location.href);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  };

  return (
    <header className="sticky top-0 z-20 grid h-[76px] grid-cols-[260px_minmax(0,1fr)] border-b bg-white/82">
      <div className="flex items-center gap-3 border-r px-8 text-sm text-muted-foreground">
        <span>History</span>
        <span>›</span>
        <span className="font-semibold text-foreground">{id.startsWith('EV-') ? `#${id.replace('EV-', '')}` : id}</span>
      </div>
      <div className="flex items-center justify-end gap-5 px-10">
        <button
          type="button"
          onClick={onSearch}
          className="flex h-11 w-[440px] items-center justify-between rounded-lg border bg-card px-4 text-muted-foreground shadow-sm"
        >
          <span className="flex items-center gap-3">
            <Search className="size-4" />
            Search evolutions...
          </span>
          <kbd className="font-mono text-xs">⌘K</kbd>
        </button>
        <Button
          variant="ghost"
          size="icon"
          aria-label="Toggle theme preview"
          onClick={() => document.documentElement.classList.toggle('dark-preview')}
        >
          <Sun className="size-5" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          aria-label={copied ? 'Page link copied' : 'Copy page link'}
          title={copied ? 'Copied' : 'Copy page link'}
          onClick={copyURL}
        >
          <MoreHorizontal className="size-5" />
        </Button>
      </div>
    </header>
  );
}
