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
  verificationState: string;
  verificationSummary: string;
  sessionProviders: string[];
  createdAt: string;
  updatedAt: string;
};

export type SessionRecord = {
  provider: string;
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
};

export type DetailResponse = {
  evolution: Evolution;
  summary: EvolutionSummary;
  sessions: SessionRecord[];
  rawJson: unknown;
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
};

export type ConfigResponse = {
  protocolVersion: number;
  cliVersion: string;
  repository: string;
  addr: string;
  eveDir: string;
  initialized: boolean;
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
