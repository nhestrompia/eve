export type Verification = {
  type?: string;
  status: string;
  reference?: string;
};

export type BehaviorClaim = {
  description: string;
  evidence?: {
    commits?: string[];
    files?: string[];
    tests?: string[];
  };
};

export type Behavior = {
  added?: BehaviorClaim[];
  changed?: BehaviorClaim[];
  removed?: BehaviorClaim[];
  fixed?: BehaviorClaim[];
};

export type Evolution = {
  eve: { version: number };
  metadata: {
    id?: string;
    title?: string;
    type?: string;
    status: string;
    created_by?: string;
    created_at?: string;
    updated_at?: string;
  };
  intent: string;
  outcome: string;
  behavior: Behavior;
  decisions: unknown[];
  risks: unknown[];
  verification: Verification[];
  sessions: Array<{ provider?: string; id?: string; uri?: string }>;
  timeline: Array<{
    timestamp?: string;
    event?: string;
    description?: string;
    actor?: { type?: string; provider?: string; id?: string };
  }>;
  relationships: Record<string, string[] | undefined>;
  implementation: {
    repositories?: Record<string, { status?: string }>;
    snapshot?: string;
    commits?: string[];
    pull_requests?: string[];
    files_changed?: number;
    insertions?: number;
    deletions?: number;
  };
  extensions: Record<string, unknown>;
};

export type EvolutionSummary = {
  id: string;
  title: string;
  type: string;
  status: string;
  outcome: string;
  snapshot: string;
  commitCount: number;
  verificationState: string;
  verificationSummary: string;
  sessionProviders: string[];
  createdAt: string;
  updatedAt: string;
};

export type SessionRecord = {
  provider: string;
  providerName: string;
  id: string;
  key: string;
  uri?: string;
  title?: string;
  transcript?: string;
  raw?: string;
  sanitized: boolean;
  format?: string;
  attachedAt?: string;
  source?: string;
  hasTranscript: boolean;
  status: string;
  captureHint: string;
  localSources: SessionSource[];
  rootsChecked: string[];
  preview: SessionPreview;
};

export type SessionSource = {
  path: string;
  format: string;
  size: number;
  modifiedAt: string;
  title?: string;
  match?: string;
};

export type SessionPreview = {
  eventCount: number;
  messageCount: number;
  userMessages: number;
  agentMessages: number;
  toolCalls: number;
  firstTimestamp?: string;
  lastTimestamp?: string;
  headings?: string[];
};

export type ProviderInfo = {
  provider: string;
  name: string;
  roots: string[];
  available: boolean;
  importCommand: string;
  displays: string[];
};

export type GitCommit = {
  hash: string;
  shortHash: string;
  subject: string;
  authorName: string;
  authoredAt: string;
  committedAt: string;
};

export type DetailResponse = {
  evolution: Evolution;
  summary: EvolutionSummary;
  sessions: SessionRecord[];
  providers: ProviderInfo[];
  commits: GitCommit[];
  rawJson: unknown;
};

export type SessionListResponse = {
  evolutionId: string;
  sessions: SessionRecord[];
  providers: ProviderInfo[];
};

export type SnapshotResponse = {
  id: string;
  title: string;
  outcome: string;
  behavior: Behavior;
  verification: Verification[];
  repository: string;
  commit: string;
  checkoutCommand: string;
  snapshotImages: SnapshotImage[];
};

export type SnapshotImage = {
  id: string;
  title: string;
  url: string;
  mimeType: string;
  source?: string;
  attachedAt?: string;
};

export type ConfigResponse = {
  protocolVersion: number;
  cliVersion: string;
  repository: string;
  addr: string;
  eveDir: string;
  initialized: boolean;
};

export type RepositorySummary = {
  name: string;
  remoteUrl?: string;
  evolutionCount: number;
  snapshotCount: number;
  commitCount: number;
  latestAt: string;
  latestEvolution: string;
  latestTitle: string;
  sessionProviders: string[];
};

export type SearchResponse = {
  query: string;
  results: Array<{
    evolution: EvolutionSummary;
    matches: string[];
  }>;
};

export type SessionTranscriptResponse = {
  evolutionId: string;
  provider: string;
  id: string;
  key: string;
  title: string;
  markdown: string;
  sanitized: boolean;
};

export type CheckoutResponse = {
  id: string;
  title: string;
  repository: string;
  commit: string;
  command: string;
  exitCode: number;
  stdout: string;
  stderr: string;
};
