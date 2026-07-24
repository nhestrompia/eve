import type {
  CheckoutResponse,
  ComparisonResponse,
  ConfigResponse,
  DetailResponse,
  Evolution,
  EvolutionSummary,
  OpenEditorResponse,
  PlanRecord,
  PendingSnapshotResponse,
  RepositorySummary,
  SearchResponse,
  SessionRecord,
  SessionListResponse,
  SessionTranscriptResponse,
  Snapshot,
  SnapshotCodeFileMode,
  SnapshotCodeFileResponse,
  SnapshotCodeFilesResponse,
  SnapshotResponse,
  SnapshotSummary,
  Verification
} from './types';

type RepoAPI = {
  id: string;
  root: string;
  snapshotCount: number;
  commitCount?: number;
  decisionCount?: number;
  riskCount?: number;
  artifactCount?: number;
  latestAt: string;
  latestSnapshot: string;
  latestTitle: string;
  branch?: string;
  head?: string;
  dirty?: boolean;
  pendingSnapshot?: RepositorySummary['pendingSnapshot'];
  remoteUrl?: string;
  readme?: string;
  primaryLanguage?: string;
  sizeBytes?: number;
  createdAt?: string;
  latestGitState?: string;
};

type SnapshotDetailAPI = {
  repository: string;
  snapshot: Snapshot;
  planRecord?: PlanRecord;
  summary: SnapshotSummary;
  sessions?: SessionRecord[];
  providers?: DetailResponse['providers'];
  commits?: DetailResponse['commits'];
  rawJson: unknown;
};

type SearchAPI = {
  query: string;
  results: Array<{
    evolution: SnapshotSummary;
    matches: string[];
  }>;
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
    commitCount: repo.commitCount ?? 0,
    decisionCount: repo.decisionCount ?? 0,
    riskCount: repo.riskCount ?? 0,
    artifactCount: repo.artifactCount ?? 0,
    latestAt: repo.latestAt,
    latestEvolution: repo.latestSnapshot,
    latestSnapshot: repo.latestSnapshot,
    latestTitle: repo.latestTitle,
    latestGitState: repo.latestGitState,
    pendingSnapshot: repo.pendingSnapshot,
    sessionProviders: []
  };
}

function validationToVerification(values: Snapshot['validation']): Verification[] {
  return values.map((value) => ({
    type: 'command',
    status: value.status,
    reference: value.command,
    provenance: value.provenance
  }));
}

function snapshotVerificationToVerification(value: Snapshot['verification']): Verification[] {
  if (!value) return [];
  const policyQualifier = value.policyChange?.requirementsReduced
    ? ' · requirements reduced'
    : value.policyChange?.changed ? ' · policy changed' : '';
  return [{
    type: 'required suite',
    status: value.status,
    reference: value.profile ? `Profile: ${value.profile}${policyQualifier}` : policyQualifier.slice(3) || undefined,
    provenance: value.selectedRunId ? 'executed_by_eve' : undefined
  }];
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
    verification: [...snapshotVerificationToVerification(snapshot.verification), ...validationToVerification(snapshot.validation)],
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
    repository: summary.repository,
    title: summary.title,
    type: summary.type,
    status: 'completed',
    outcome: summary.summary,
    userVisibleChange: summary.userVisibleChange,
    snapshot: summary.gitState,
    commitCount: summary.commitCount ?? 0,
    decisionCount: summary.decisionCount ?? 0,
    riskCount: summary.riskCount ?? 0,
    artifactCount: summary.artifactCount ?? 0,
    failedValidationCount: summary.failedValidationCount ?? 0,
    verificationState: summary.validationState,
    verificationSummary: summary.validationState,
    sessionProviders,
    createdAt: summary.createdAt,
    updatedAt: summary.createdAt
  };
}

async function snapshots(repo?: unknown): Promise<EvolutionSummary[]> {
  if (typeof repo === 'string' && repo) {
    const rows = await request<SnapshotSummary[]>(`/api/repos/${encodeURIComponent(repo)}/snapshots`);
    return rows.map((row) => snapshotToSummary(row));
  }
  const rows = await request<SnapshotSummary[]>('/api/snapshots');
  return rows
    .map((row) => snapshotToSummary(row))
    .sort((left, right) => right.createdAt.localeCompare(left.createdAt));
}

async function repository(repo: string): Promise<RepositorySummary> {
  return adaptRepo(await request<RepoAPI>(`/api/repos/${encodeURIComponent(repo)}`));
}

async function snapshotDetail(id: string, repo?: string): Promise<DetailResponse> {
  const detail = await request<SnapshotDetailAPI>(
    repo
      ? `/api/repos/${encodeURIComponent(repo)}/snapshots/${encodeURIComponent(id)}`
      : `/api/snapshots/${encodeURIComponent(id)}`
  );
  const sessions = detail.sessions ?? [];
  const sessionProviders = Array.from(new Set(sessions.map((session) => session.provider).filter(Boolean)));
  const summary = snapshotToSummary(detail.summary, sessionProviders);
  return {
    repository: detail.repository || detail.summary.repository || repo || '',
    snapshot: detail.snapshot,
    planRecord: detail.planRecord,
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

async function snapshot(id: string, repo?: string): Promise<SnapshotResponse> {
  const detail = await snapshotDetail(id, repo);
  const repoId = detail.repository;
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

async function compare(from: string, to: string): Promise<ComparisonResponse> {
  const response = await request<ComparisonResponse>(`/api/compare?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`);
  return {
    ...response,
    range: response.range ?? [],
    added: response.added ?? [],
    changed: response.changed ?? [],
    fixed: response.fixed ?? [],
    decisions: response.decisions ?? [],
    risks: response.risks ?? [],
    validation: response.validation ?? [],
    timeline: response.timeline ?? []
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
  pendingSnapshot: (repo: string) => request<PendingSnapshotResponse>(`/api/repos/${encodeURIComponent(repo)}/pending`),
  repositories: async () => (await repositoriesRaw()).map(adaptRepo),
  repository,
  snapshots,
  snapshotDetail,
  snapshot,
  compare,
  sessions: async (id: string): Promise<SessionListResponse> => {
    const detail = await snapshotDetail(id);
    return { evolutionId: id, sessions: detail.sessions, providers: detail.providers };
  },
  search: async (query: string): Promise<SearchResponse> => {
    const response = await request<SearchAPI>(`/api/search?q=${encodeURIComponent(query)}`);
    return {
      query: response.query,
      results: response.results.map((result) => ({
        evolution: snapshotToSummary(result.evolution),
        matches: result.matches
      }))
    };
  },
  transcript: async (id: string, key: string): Promise<SessionTranscriptResponse> => {
    const repoId = (await snapshotDetail(id)).repository || (await defaultRepoId());
    return request<SessionTranscriptResponse>(
      `/api/repos/${encodeURIComponent(repoId)}/snapshots/${encodeURIComponent(id)}/sessions/${encodeURIComponent(key)}`
    );
  },
  checkout: async (id: string, repo?: string) => {
    const repoId = repo || (await snapshotDetail(id)).repository || (await defaultRepoId());
    return request<CheckoutResponse>(`/api/repos/${encodeURIComponent(repoId)}/snapshots/${encodeURIComponent(id)}/checkout`, { method: 'POST' });
  },
  snapshotCodeFiles: async (id: string, repo?: string) => {
    const repoId = repo || (await snapshotDetail(id)).repository || (await defaultRepoId());
    return request<SnapshotCodeFilesResponse>(`/api/repos/${encodeURIComponent(repoId)}/snapshots/${encodeURIComponent(id)}/code/files`);
  },
  snapshotCodeFile: async (id: string, path: string, mode: SnapshotCodeFileMode, repo?: string) => {
    const repoId = repo || (await snapshotDetail(id)).repository || (await defaultRepoId());
    return request<SnapshotCodeFileResponse>(
      `/api/repos/${encodeURIComponent(repoId)}/snapshots/${encodeURIComponent(id)}/code/file?path=${encodeURIComponent(path)}&mode=${encodeURIComponent(mode)}`
    );
  },
  openRepositoryInEditor: (repo: string) =>
    request<OpenEditorResponse>(`/api/repos/${encodeURIComponent(repo)}/open-editor`, { method: 'POST' })
};
