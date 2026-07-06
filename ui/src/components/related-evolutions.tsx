import { Link } from '@tanstack/react-router';
import { ArrowRight, GitBranch } from 'lucide-react';
import type { Evolution } from '../types';
import { relationshipSummary } from '../lib/evolution-display';
import { SnapshotRelationshipList } from './snapshot-relationship-list';
import { Card, CardContent, CardHeader } from './ui/card';

export function RelatedEvolutions({ evolution }: { evolution: Evolution }) {
  return (
    <Card>
      <CardHeader className="flex-col gap-3 space-y-0 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <GitBranch className="size-4 text-slate-600" />
          <div>
            <h2 className="text-sm font-semibold text-balance">Related Snapshots</h2>
            <p className="mt-1 text-xs text-muted-foreground">{relationshipSummary(evolution.relationships)}</p>
          </div>
        </div>
        <Link
          className="inline-flex items-center gap-2 text-sm font-medium text-blue-700"
          to="/snapshots/$id/relationships"
          params={{ id: evolution.metadata.id ?? '' }}
        >
          View all relationships <ArrowRight className="size-4" />
        </Link>
      </CardHeader>
      <CardContent>
        <SnapshotRelationshipList relationships={evolution.relationships} />
      </CardContent>
    </Card>
  );
}
