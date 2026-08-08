import { Outlet } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { useRouterState } from '@tanstack/react-router';
import { Toaster } from 'sonner';
import { useEffect, useState } from 'react';
import { api } from '../api';
import { SearchCommand } from './search-command';
import { Sidebar } from './sidebar';
import { PendingSnapshotBanner } from './pending-snapshot-banner';
import { PendingPlanBanner } from './pending-plan-banner';
import { TopBar } from './top-bar';

export function AppShell() {
  const state = useRouterState();
  if (state.location.pathname === "/phone" || state.location.pathname.startsWith("/phone/")) {
    return (
      <div className="phone-app-shell min-h-dvh bg-background text-foreground">
        <Outlet />
        <Toaster position="top-center" richColors closeButton />
      </div>
    );
  }
  return <DashboardAppShell />;
}

function DashboardAppShell() {
  const state = useRouterState();
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const pendingPlans = useQuery({
    queryKey: ['pending-plan-requests'],
    queryFn: () => api.planRequests('review'),
    retry: false
  });

  const openSearch = (query = '') => {
    setSearchQuery(query);
    setSearchOpen(true);
  };

  const dashboardRoute = ['/', '/plans', '/repositories', '/snapshots'].includes(state.location.pathname);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        openSearch();
      }
    };
    const onOpenSearch = (event: Event) => {
      const query = event instanceof CustomEvent && typeof event.detail?.query === 'string' ? event.detail.query : '';
      openSearch(query);
    };
    window.addEventListener('keydown', onKeyDown);
    window.addEventListener('eve:open-search', onOpenSearch);
    return () => {
      window.removeEventListener('keydown', onKeyDown);
      window.removeEventListener('eve:open-search', onOpenSearch);
    };
  }, []);

  return (
    <div className="app-backdrop min-h-dvh text-[13px] text-foreground md:pl-[216px]">
      <Sidebar onSearch={openSearch} />
      <div className="min-w-0">
        {dashboardRoute ? null : <TopBar onSearch={() => openSearch()} />}
        {!dashboardRoute && (pendingPlans.data?.length ?? 0) > 0 ? (
          <div className="px-4 pt-4 md:px-8">
            <PendingPlanBanner plans={pendingPlans.data ?? []} />
          </div>
        ) : null}
        {dashboardRoute ? null : <PendingSnapshotBanner />}
        <Outlet />
      </div>
      <SearchCommand open={searchOpen} initialQuery={searchQuery} onOpenChange={setSearchOpen} />
      <Toaster position="bottom-right" richColors closeButton />
    </div>
  );
}
