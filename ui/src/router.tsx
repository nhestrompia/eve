import { createRootRoute, createRoute, createRouter } from '@tanstack/react-router';
import { AppShell } from './components/app-shell';
import { ActivityPage } from './pages/activity-page';
import { ConfigPage } from './pages/config-page';
import { DecisionsPage } from './pages/decisions-page';
import { EvolutionDetailPage } from './pages/evolution-detail-page';
import { ImplementationPage } from './pages/implementation-page';
import { RawJsonPage } from './pages/raw-json-page';
import { RepositoryPage } from './pages/repository-page';
import { RelationshipsPage } from './pages/relationships-page';
import { RisksPage } from './pages/risks-page';
import { SearchPage } from './pages/search-page';
import { SessionsOverviewPage } from './pages/sessions-overview-page';
import { SessionPage } from './pages/session-page';
import { SnapshotPage } from './pages/snapshot-page';
import { TimelinePage } from './pages/timeline-page';
import { VerificationPage } from './pages/verification-page';

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

const repositoryRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/repositories/$repo',
  component: RepositoryPage
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

const sessionsOverviewRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/evolutions/$id/sessions',
  component: SessionsOverviewPage
});

const verificationRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/evolutions/$id/verification',
  component: VerificationPage
});

const decisionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/evolutions/$id/decisions',
  component: DecisionsPage
});

const risksRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/evolutions/$id/risks',
  component: RisksPage
});

const implementationRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/evolutions/$id/implementation',
  component: ImplementationPage
});

const relationshipsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/evolutions/$id/relationships',
  component: RelationshipsPage
});

const activityRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/evolutions/$id/activity',
  component: ActivityPage
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

const routeTree = rootRoute.addChildren([
  indexRoute,
  repositoryRoute,
  evolutionRoute,
  snapshotRoute,
  sessionRoute,
  sessionsOverviewRoute,
  verificationRoute,
  decisionsRoute,
  risksRoute,
  implementationRoute,
  relationshipsRoute,
  activityRoute,
  searchRoute,
  jsonRoute,
  configRoute
]);

export const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
