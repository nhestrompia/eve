import { Link } from '@tanstack/react-router';
import { ArrowRight, Code2 } from 'lucide-react';
import type { Evolution } from '../types';
import { shortCommit } from '../format';
import { SectionHeading } from './section-heading';
import { Card, CardContent, CardHeader } from './ui/card';

export function ImplementationCard({ evolution }: { evolution: Evolution }) {
  const commits = evolution.implementation.commits ?? [];

  return (
    <Card>
      <CardHeader>
        <SectionHeading icon={Code2} title="Implementation" />
      </CardHeader>
      <CardContent>
        <p className="mb-4 text-sm text-muted-foreground">Contributed Commits</p>
        <div className="space-y-4">
          {commits.slice(0, 3).map((commit, index) => (
            <div key={commit} className="grid grid-cols-[82px_minmax(0,1fr)_auto] gap-3">
              <code className="truncate rounded-md bg-slate-100 px-2 py-1 font-mono text-xs">{shortCommit(commit)}</code>
              <span className="min-w-0 truncate text-muted-foreground">{commit === evolution.implementation.snapshot ? 'Snapshot' : 'Implementation'}</span>
              <span className="text-muted-foreground">{index + 3} days ago</span>
            </div>
          ))}
          {commits.length === 0 ? <p className="text-muted-foreground">No contributed commits recorded.</p> : null}
        </div>
        <Link
          className="mt-6 inline-flex items-center gap-2 text-sm font-medium text-blue-700"
          to="/json/$id"
          params={{ id: evolution.metadata.id ?? '' }}
          hash="implementation"
        >
          View {Math.max(commits.length - 3, 0)} more commits <ArrowRight className="size-4" />
        </Link>
      </CardContent>
    </Card>
  );
}
