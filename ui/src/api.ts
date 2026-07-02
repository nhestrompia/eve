import type {
  CheckoutResponse,
  ConfigResponse,
  DetailResponse,
  EvolutionSummary,
  RepositorySummary,
  SearchResponse,
  SessionListResponse,
  SessionTranscriptResponse,
  SnapshotResponse
} from './types';

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

export const api = {
  config: () => request<ConfigResponse>('/api/config'),
  repositories: () => request<RepositorySummary[]>('/api/repositories'),
  evolutions: (repo?: unknown) => {
    const repoName = typeof repo === 'string' ? repo : '';
    return request<EvolutionSummary[]>(`/api/evolutions${repoName ? `?repo=${encodeURIComponent(repoName)}` : ''}`);
  },
  evolution: (id: string) => request<DetailResponse>(`/api/evolutions/${encodeURIComponent(id)}`),
  snapshot: (id: string) => request<SnapshotResponse>(`/api/evolutions/${encodeURIComponent(id)}/snapshot`),
  sessions: (id: string) => request<SessionListResponse>(`/api/evolutions/${encodeURIComponent(id)}/sessions`),
  search: (query: string) => request<SearchResponse>(`/api/search?q=${encodeURIComponent(query)}`),
  transcript: (id: string, key: string) =>
    request<SessionTranscriptResponse>(`/api/evolutions/${encodeURIComponent(id)}/sessions/${encodeURIComponent(key)}`),
  checkout: (id: string) =>
    request<CheckoutResponse>(`/api/evolutions/${encodeURIComponent(id)}/checkout`, { method: 'POST' })
};
