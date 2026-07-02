import { createRootRoute, createRoute, createRouter } from '@tanstack/react-router';
import { AppShell } from './components/app-shell';
import { ConfigPage } from './pages/config-page';
import { EvolutionDetailPage } from './pages/evolution-detail-page';
import { RawJsonPage } from './pages/raw-json-page';
import { SearchPage } from './pages/search-page';
import { SessionPage } from './pages/session-page';
import { SnapshotPage } from './pages/snapshot-page';
import { TimelinePage } from './pages/timeline-page';

const rootRoute = createRootRoute({ component: AppShell });

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: TimelinePage
});

const evolutionRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/evolutions/$id',
  component: EvolutionDetailPage
});

const snapshotRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/evolutions/$id/snapshot',
  component: SnapshotPage
});

const sessionRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/evolutions/$id/session/$sessionId',
  component: SessionPage
});

const searchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/search',
  validateSearch: (search: Record<string, unknown>) => ({ q: typeof search.q === 'string' ? search.q : '' }),
  component: SearchPage
});

const jsonRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/json/$id',
  component: RawJsonPage
});

const configRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/config',
  component: ConfigPage
});

const routeTree = rootRoute.addChildren([indexRoute, evolutionRoute, snapshotRoute, sessionRoute, searchRoute, jsonRoute, configRoute]);

export const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
