import type {
  CheckoutResponse,
  ConfigResponse,
  DetailResponse,
  Evolution,
  EvolutionSummary,
  RepositorySummary,
  SearchResponse,
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
};

type SnapshotDetailAPI = {
  snapshot: Snapshot;
  summary: SnapshotSummary;
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
    evolutionCount: repo.snapshotCount,
    snapshotCount: repo.snapshotCount,
    commitCount: 0,
    latestAt: repo.latestAt,
    latestEvolution: repo.latestSnapshot,
    latestSnapshot: repo.latestSnapshot,
    latestTitle: repo.latestTitle,
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

function snapshotToEvolution(snapshot: Snapshot): Evolution {
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
    sessions: [],
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

function snapshotToSummary(summary: SnapshotSummary): EvolutionSummary {
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
    sessionProviders: [],
    createdAt: summary.createdAt,
    updatedAt: summary.createdAt
  };
}

async function snapshots(repo?: unknown): Promise<EvolutionSummary[]> {
  const repoId = typeof repo === 'string' && repo ? repo : await defaultRepoId();
  if (!repoId) return [];
  const rows = await request<SnapshotSummary[]>(`/api/repos/${encodeURIComponent(repoId)}/snapshots`);
  return rows.map(snapshotToSummary);
}

async function snapshotDetail(id: string): Promise<DetailResponse> {
  const repoId = await defaultRepoId();
  const detail = await request<SnapshotDetailAPI>(`/api/repos/${encodeURIComponent(repoId)}/snapshots/${encodeURIComponent(id)}`);
  const summary = snapshotToSummary(detail.summary);
  return {
    snapshot: detail.snapshot,
    evolution: snapshotToEvolution(detail.snapshot),
    summary,
    sessions: [],
    providers: [],
    commits: detail.snapshot.implementation.commits.map((hash) => ({
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
  const imageArtifacts = detail.snapshot.artifacts.filter((artifact) => artifact.type === 'screenshot' && (artifact.url || artifact.path));
  return {
    id: detail.snapshot.id,
    title: detail.snapshot.title,
    outcome: detail.snapshot.summary,
    behavior: detail.evolution.behavior,
    verification: detail.evolution.verification,
    repository: '',
    commit: detail.snapshot.implementation.gitState,
    checkoutCommand: `eve checkout ${detail.snapshot.id}`,
    snapshotImages: imageArtifacts.map((artifact, index) => ({
      id: artifact.path || artifact.url || `artifact-${index}`,
      title: artifact.description || artifact.path || artifact.url || `Artifact ${index + 1}`,
      url: artifact.url || `/${artifact.path}`,
      mimeType: artifact.mimeType || 'image/png',
      source: artifact.path || artifact.url
    }))
  };
}

export const api = {
  config: () => request<ConfigResponse>('/api/config'),
  repositories: async () => (await repositoriesRaw()).map(adaptRepo),
  snapshots,
  snapshotDetail,
  snapshot,
  sessions: async (id: string): Promise<SessionListResponse> => ({ evolutionId: id, sessions: [], providers: [] }),
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
  transcript: async (id: string, key: string): Promise<SessionTranscriptResponse> => ({
    evolutionId: id,
    provider: '',
    id: key,
    key,
    title: 'No transcript',
    markdown: 'Snapshots store conversations as artifacts, not first-class sessions.',
    sanitized: true
  }),
  checkout: async (id: string) => {
    const repoId = await defaultRepoId();
    return request<CheckoutResponse>(`/api/repos/${encodeURIComponent(repoId)}/snapshots/${encodeURIComponent(id)}/checkout`, { method: 'POST' });
  }
};
