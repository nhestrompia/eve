import type { Behavior, BehaviorClaim, Evolution, Verification } from '../types';

export type DisplayRecord = {
  title: string;
  body?: string;
  meta?: Array<{ label: string; value: string }>;
};

export function behaviorClaims(behavior: Behavior): BehaviorClaim[] {
  return [
    ...(behavior.added ?? []),
    ...(behavior.changed ?? []),
    ...(behavior.fixed ?? []),
    ...(behavior.removed ?? [])
  ];
}

export function displayVerification(item: Verification): DisplayRecord {
  return {
    title: item.reference || item.type || 'Verification',
    meta: [
      { label: 'Status', value: titleCase(item.status || 'unknown') },
      ...(item.type && item.reference ? [{ label: 'Type', value: titleCase(item.type) }] : [])
    ]
  };
}

export function displayDecision(value: unknown): DisplayRecord {
  if (isRecord(value)) {
    const title = stringValue(value.title) || stringValue(value.decision) || stringValue(value.name) || 'Decision';
    return {
      title,
      body: stringValue(value.reason) || stringValue(value.summary) || stringValue(value.description),
      meta: compactMeta([
        ['Tradeoff', stringValue(value.tradeoff)],
        ['Owner', stringValue(value.owner)],
        ['Status', stringValue(value.status)]
      ])
    };
  }
  return { title: String(value || 'Decision') };
}

export function displayRisk(value: unknown): DisplayRecord {
  if (isRecord(value)) {
    const title = stringValue(value.title) || stringValue(value.risk) || stringValue(value.description) || 'Risk';
    return {
      title,
      body: stringValue(value.mitigation) || stringValue(value.impact) || stringValue(value.reason),
      meta: compactMeta([
        ['Severity', stringValue(value.severity)],
        ['Likelihood', stringValue(value.likelihood)],
        ['Status', stringValue(value.status)]
      ])
    };
  }
  return { title: String(value || 'Risk') };
}

export function activityEntries(evolution: Evolution) {
  if (evolution.timeline.length > 0) return evolution.timeline;
  return [
    {
      event: 'created',
      description: 'Evolution created.',
      timestamp: evolution.metadata.created_at,
      actor: { type: 'tool', provider: evolution.metadata.created_by }
    }
  ];
}

export function titleCase(value: string): string {
  return value
    .replaceAll('_', ' ')
    .split(' ')
    .filter(Boolean)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

function compactMeta(values: Array<[string, string | undefined]>): DisplayRecord['meta'] {
  return values.flatMap(([label, value]) => (value ? [{ label, value }] : []));
}

function stringValue(value: unknown): string | undefined {
  if (typeof value === 'string' && value.trim()) return value;
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  return undefined;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value);
}
