import { Link } from '@tanstack/react-router';
import { ArrowRight, Code2 } from 'lucide-react';
import type { Evolution, GitCommit } from '../types';
import { compactDate, shortCommit } from '../format';
import { SectionHeading } from './section-heading';
import { Card, CardContent, CardHeader } from './ui/card';

export function ImplementationCard({ evolution, commits }: { evolution: Evolution; commits: GitCommit[] }) {
  const commitRows =
    commits.length > 0
      ? commits
      : (evolution.implementation.commits ?? []).map((hash) => ({
          hash,
          shortHash: shortCommit(hash),
          subject: hash === evolution.implementation.snapshot ? 'Snapshot commit' : 'Implementation commit',
          authorName: '',
          authoredAt: '',
          committedAt: ''
        }));

  return (
    <Card>
      <CardHeader>
        <SectionHeading icon={Code2} title="Implementation" />
      </CardHeader>
      <CardContent>
        <p className="mb-4 text-sm text-muted-foreground">Contributed Commits</p>
        <div className="space-y-4">
          {commitRows.slice(0, 3).map((commit) => (
            <div key={commit.hash} className="grid grid-cols-[82px_minmax(0,1fr)] gap-3 sm:grid-cols-[82px_minmax(0,1fr)_108px]">
              <code className="truncate rounded-md bg-slate-100 px-2 py-1 font-mono text-xs">{commit.shortHash || shortCommit(commit.hash)}</code>
              <span className="min-w-0 truncate text-muted-foreground">{commit.subject}</span>
              <span className="col-start-2 truncate text-muted-foreground sm:col-start-auto sm:text-right">{compactDate(commit.committedAt || commit.authoredAt)}</span>
            </div>
          ))}
          {commitRows.length === 0 ? <p className="text-muted-foreground">No contributed commits recorded.</p> : null}
        </div>
        <Link
          className="mt-6 inline-flex items-center gap-2 text-sm font-medium text-blue-700"
          to="/snapshots/$id/implementation"
          params={{ id: evolution.metadata.id ?? '' }}
        >
          View implementation <ArrowRight className="size-4" />
        </Link>
      </CardContent>
    </Card>
  );
}
