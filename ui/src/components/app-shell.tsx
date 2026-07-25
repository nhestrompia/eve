import { Outlet } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { Toaster } from 'sonner';
import { useEffect, useState } from 'react';
import { api } from '../api';
import { SearchCommand } from './search-command';
import { Sidebar } from './sidebar';
import { PendingSnapshotBanner } from './pending-snapshot-banner';
import { PendingPlanBanner } from './pending-plan-banner';
import { TopBar } from './top-bar';

export function AppShell() {
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const pendingPlans = useQuery({
    queryKey: ['pending-plan-requests'],
    queryFn: () => api.planRequests('pending_approval'),
    refetchInterval: 2_000,
    retry: false
  });

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
        {(pendingPlans.data?.length ?? 0) > 0 ? (
          <div className="px-4 pt-4 md:px-8">
            <PendingPlanBanner plans={pendingPlans.data ?? []} />
          </div>
        ) : null}
        <PendingSnapshotBanner />
        <Outlet />
      </div>
      <SearchCommand open={searchOpen} initialQuery={searchQuery} onOpenChange={setSearchOpen} />
      <Toaster position="bottom-right" richColors closeButton />
    </div>
  );
}
