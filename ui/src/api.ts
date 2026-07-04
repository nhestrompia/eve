import type {
  CheckoutResponse,
  ConfigResponse,
  DetailResponse,
  Evolution,
  EvolutionSummary,
  OpenEditorResponse,
  RepositorySummary,
  SearchResponse,
  SessionRecord,
  SessionListResponse,
  SessionTranscriptResponse,
  Snapshot,
  SnapshotResponse,
  SnapshotSummary,
  Verification
} from './types';

type RepoAPI = {
  id: string;
  root: string;
  snapshotCount: number;
  latestAt: string;
  latestSnapshot: string;
  latestTitle: string;
  branch?: string;
  head?: string;
  dirty?: boolean;
  remoteUrl?: string;
  readme?: string;
  primaryLanguage?: string;
  sizeBytes?: number;
  createdAt?: string;
  latestGitState?: string;
};

type SnapshotDetailAPI = {
  snapshot: Snapshot;
  summary: SnapshotSummary;
  sessions?: SessionRecord[];
  providers?: DetailResponse['providers'];
  commits?: DetailResponse['commits'];
  rawJson: unknown;
};

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    headers: { Accept: 'application/json', ...(init?.headers ?? {}) },
    ...init
  });
  const data = await response.json().catch(() => undefined);
  if (!response.ok) {
    const message = data && typeof data.error === 'string' ? data.error : `Request failed: ${response.status}`;
    throw new Error(message);
  }
  return data as T;
}

async function defaultRepoId() {
  const repos = await repositoriesRaw();
  return repos[0]?.id ?? '';
}

async function repositoriesRaw() {
  return request<RepoAPI[]>('/api/repos');
}

function adaptRepo(repo: RepoAPI): RepositorySummary {
  return {
    id: repo.id,
    root: repo.root,
    name: repo.id,
    remoteUrl: repo.remoteUrl,
    branch: repo.branch,
    head: repo.head,
    dirty: repo.dirty,
    readme: repo.readme,
    primaryLanguage: repo.primaryLanguage,
    sizeBytes: repo.sizeBytes,
    createdAt: repo.createdAt,
    evolutionCount: repo.snapshotCount,
    snapshotCount: repo.snapshotCount,
    commitCount: 0,
    latestAt: repo.latestAt,
    latestEvolution: repo.latestSnapshot,
    latestSnapshot: repo.latestSnapshot,
    latestTitle: repo.latestTitle,
    latestGitState: repo.latestGitState,
    sessionProviders: []
  };
}

function validationToVerification(values: Snapshot['validation']): Verification[] {
  return values.map((value) => ({
    type: 'command',
    status: value.status,
    reference: value.command
  }));
}

function snapshotToEvolution(snapshot: Snapshot, sessions: SessionRecord[] = []): Evolution {
  return {
    eve: { version: 0 },
    metadata: {
      id: snapshot.id,
      title: snapshot.title,
      type: snapshot.type,
      status: 'completed',
      created_at: snapshot.createdAt,
      updated_at: snapshot.createdAt
    },
    intent: snapshot.title,
    outcome: snapshot.summary,
    behavior: { added: [{ description: snapshot.userVisibleChange || snapshot.summary }] },
    decisions: snapshot.decisions,
    risks: snapshot.risks,
    verification: validationToVerification(snapshot.validation),
    sessions: sessions.map((session) => ({
      provider: session.provider,
      id: session.id,
      uri: session.uri
    })),
    timeline: snapshot.timeline.map((entry) => ({
      timestamp: entry.occurredAt,
      event: entry.phase,
      description: entry.summary || entry.title
    })),
    relationships: snapshot.relationships,
    implementation: {
      repositories: {},
      snapshot: snapshot.implementation.gitState,
      commits: snapshot.implementation.commits
    },
    extensions: {}
  };
}

function snapshotToSummary(summary: SnapshotSummary, sessionProviders: string[] = []): EvolutionSummary {
  return {
    id: summary.id,
    title: summary.title,
    type: summary.type,
    status: 'completed',
    outcome: summary.summary,
    snapshot: summary.gitState,
    commitCount: summary.gitState ? 1 : 0,
    verificationState: summary.validationState,
    verificationSummary: summary.validationState,
    sessionProviders,
    createdAt: summary.createdAt,
    updatedAt: summary.createdAt
  };
}

async function snapshots(repo?: unknown): Promise<EvolutionSummary[]> {
  const repoId = typeof repo === 'string' && repo ? repo : await defaultRepoId();
  if (!repoId) return [];
  const rows = await request<SnapshotSummary[]>(`/api/repos/${encodeURIComponent(repoId)}/snapshots`);
  return rows.map((row) => snapshotToSummary(row));
}

async function repository(repo: string): Promise<RepositorySummary> {
  return adaptRepo(await request<RepoAPI>(`/api/repos/${encodeURIComponent(repo)}`));
}

async function snapshotDetail(id: string): Promise<DetailResponse> {
  const repoId = await defaultRepoId();
  const detail = await request<SnapshotDetailAPI>(`/api/repos/${encodeURIComponent(repoId)}/snapshots/${encodeURIComponent(id)}`);
  const sessions = detail.sessions ?? [];
  const sessionProviders = Array.from(new Set(sessions.map((session) => session.provider).filter(Boolean)));
  const summary = snapshotToSummary(detail.summary, sessionProviders);
  return {
    snapshot: detail.snapshot,
    evolution: snapshotToEvolution(detail.snapshot, sessions),
    summary,
    sessions,
    providers: detail.providers ?? [],
    commits:
      detail.commits ??
      detail.snapshot.implementation.commits.map((hash) => ({
        hash,
        shortHash: hash.slice(0, 8),
        subject: hash === detail.snapshot.implementation.gitState ? 'Snapshot commit' : 'Implementation commit',
        authorName: '',
        authoredAt: '',
        committedAt: ''
      })),
    rawJson: detail.rawJson
  };
}

async function snapshot(id: string): Promise<SnapshotResponse> {
  const detail = await snapshotDetail(id);
  const repoId = await defaultRepoId();
  const imageArtifacts = detail.snapshot.artifacts.filter(isImageArtifact);
  return {
    id: detail.snapshot.id,
    title: detail.snapshot.title,
    outcome: detail.snapshot.summary,
    behavior: detail.evolution.behavior,
    verification: detail.evolution.verification,
    repository: repoId,
    commit: detail.snapshot.implementation.gitState,
    checkoutCommand: `eve checkout ${detail.snapshot.id}`,
    snapshotImages: imageArtifacts
      .map((artifact, index) => ({
        id: artifact.path || artifact.url || artifact.uri || `artifact-${index}`,
        title: artifact.description || artifact.path || artifact.url || artifact.uri || `Artifact ${index + 1}`,
        url: artifact.url || artifact.uri || localArtifactHref(repoId, artifact.path) || '',
        mimeType: artifact.mimeType || 'image/png',
        source: artifact.path || artifact.url || artifact.uri
      }))
      .filter((image) => image.url)
  };
}

function isImageArtifact(artifact: { type: string; path?: string; url?: string; uri?: string; mimeType?: string }) {
  const source = artifact.path || artifact.url || artifact.uri || '';
  return (
    artifact.mimeType?.startsWith('image/') ||
    artifact.type.toLowerCase().includes('screenshot') ||
    artifact.type.toLowerCase().includes('image') ||
    /\.(png|jpe?g|gif|webp|avif|svg)$/i.test(source)
  );
}

function localArtifactHref(repo: string, artifactPath?: string) {
  if (!artifactPath) return undefined;
  if (/^https?:\/\//i.test(artifactPath)) return artifactPath;
  const normalized = artifactPath.replace(/^\/+/, '');
  const prefix = '.eve/artifacts/';
  if (!normalized.startsWith(prefix)) return undefined;
  const relative = normalized.slice(prefix.length);
  return `/api/repos/${encodeURIComponent(repo)}/artifacts/${relative
    .split('/')
    .map(encodeURIComponent)
    .join('/')}`;
}

export const api = {
  config: () => request<ConfigResponse>('/api/config'),
  repositories: async () => (await repositoriesRaw()).map(adaptRepo),
  repository,
  snapshots,
  snapshotDetail,
  snapshot,
  sessions: async (id: string): Promise<SessionListResponse> => {
    const detail = await snapshotDetail(id);
    return { evolutionId: id, sessions: detail.sessions, providers: detail.providers };
  },
  search: async (query: string): Promise<SearchResponse> => {
    const rows = await snapshots();
    const normalized = query.trim().toLowerCase();
    return {
      query,
      results: rows
        .filter((row) => !normalized || [row.id, row.title, row.type, row.outcome].some((value) => value.toLowerCase().includes(normalized)))
        .map((row) => ({ evolution: row, matches: [row.title, row.outcome].filter(Boolean) }))
    };
  },
  transcript: async (id: string, key: string): Promise<SessionTranscriptResponse> => {
    const repoId = await defaultRepoId();
    return request<SessionTranscriptResponse>(
      `/api/repos/${encodeURIComponent(repoId)}/snapshots/${encodeURIComponent(id)}/sessions/${encodeURIComponent(key)}`
    );
  },
  checkout: async (id: string) => {
    const repoId = await defaultRepoId();
    return request<CheckoutResponse>(`/api/repos/${encodeURIComponent(repoId)}/snapshots/${encodeURIComponent(id)}/checkout`, { method: 'POST' });
  },
  openRepositoryInEditor: (repo: string) =>
    request<OpenEditorResponse>(`/api/repos/${encodeURIComponent(repo)}/open-editor`, { method: 'POST' })
};
