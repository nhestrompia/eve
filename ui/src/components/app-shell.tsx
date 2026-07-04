import { Outlet } from '@tanstack/react-router';
import { Toaster } from 'sonner';
import { useEffect, useState } from 'react';
import { SearchCommand } from './search-command';
import { Sidebar } from './sidebar';
import { TopBar } from './top-bar';

export function AppShell() {
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  const openSearch = (query = '') => {
    setSearchQuery(query);
    setSearchOpen(true);
  };

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        openSearch();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  return (
    <div className="app-backdrop min-h-dvh text-[13px] text-foreground md:pl-[240px]">
      <Sidebar onSearch={openSearch} />
      <div className="min-w-0">
        <TopBar onSearch={() => openSearch()} />
        <Outlet />
      </div>
      <SearchCommand open={searchOpen} initialQuery={searchQuery} onOpenChange={setSearchOpen} />
      <Toaster position="bottom-right" richColors closeButton />
    </div>
  );
}
