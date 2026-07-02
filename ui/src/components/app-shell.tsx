import { Outlet } from '@tanstack/react-router';
import { useEffect, useState } from 'react';
import { SearchCommand } from './search-command';
import { Sidebar } from './sidebar';
import { TopBar } from './top-bar';

export function AppShell() {
  const [searchOpen, setSearchOpen] = useState(false);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        setSearchOpen(true);
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  return (
    <div className="app-backdrop grid min-h-dvh grid-cols-[240px_minmax(0,1fr)] text-[13px] text-foreground">
      <Sidebar />
      <div className="min-w-0">
        <TopBar onSearch={() => setSearchOpen(true)} />
        <Outlet />
      </div>
      <SearchCommand open={searchOpen} onOpenChange={setSearchOpen} />
    </div>
  );
}
