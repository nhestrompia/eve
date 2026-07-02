import type { Behavior } from './types';

export function shortCommit(value?: string): string {
  if (!value) return 'none';
  return value.length > 12 ? value.slice(0, 12) : value;
}

export function countBehavior(behavior: Behavior): number {
  return (
    (behavior.added?.length ?? 0) +
    (behavior.changed?.length ?? 0) +
    (behavior.removed?.length ?? 0) +
    (behavior.fixed?.length ?? 0)
  );
}

export function humanDate(value?: string): string {
  if (!value) return 'unknown';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  });
}

export function monthYear(value?: string): string {
  if (!value) return 'Undated';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'Undated';
  return date.toLocaleString(undefined, { month: 'long', year: 'numeric' });
}

export function compactDate(value?: string): string {
  if (!value) return 'Unknown date';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}

export function statusLabel(value?: string): string {
  if (!value) return 'unknown';
  return value.replaceAll('_', ' ');
}
