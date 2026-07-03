import { Link } from '@tanstack/react-router';
import { ArrowRight, GitBranch } from 'lucide-react';
import type { Evolution } from '../types';
import { Card, CardContent, CardHeader } from './ui/card';

export function RelatedEvolutions({ evolution }: { evolution: Evolution }) {
  const relationships = evolution.relationships;

  const items = [
    ['Extends', relationships.extends?.[0] ?? '—'],
    ['Related to', relationships.related?.[0] ?? '—'],
    ['Corrected by', relationships.corrects?.[0] ?? '—'],
    ['Superseded by', relationships.supersedes?.[0] ?? '—']
  ];

  return (
    <Card>
      <CardHeader className="flex-col gap-3 space-y-0 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <GitBranch className="size-4 text-slate-600" />
          <h2 className="text-sm font-semibold text-balance">Related Evolutions</h2>
        </div>
        <Link
          className="inline-flex items-center gap-2 text-sm font-medium text-blue-700"
          to="/snapshots/$id/relationships"
          params={{ id: evolution.metadata.id ?? '' }}
        >
          View all relationships <ArrowRight className="size-4" />
        </Link>
      </CardHeader>
      <CardContent className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4 lg:gap-6">
        {items.map(([label, value]) => (
          <div key={label} className="space-y-2">
            <p className="text-xs text-muted-foreground">{label}</p>
            <p className="break-all font-mono text-xs">{value}</p>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
