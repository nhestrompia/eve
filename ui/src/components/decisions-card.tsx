import { Link } from '@tanstack/react-router';
import { ArrowRight, Scale } from 'lucide-react';
import { Card, CardContent, CardHeader } from './ui/card';
import { SectionHeading } from './section-heading';

export function DecisionsCard({ decisions, evolutionId }: { decisions: unknown[]; evolutionId: string }) {
  const first = decisions[0] as Record<string, unknown> | undefined;

  return (
    <Card>
      <CardHeader>
        <SectionHeading icon={Scale} title="Decisions" />
      </CardHeader>
      <CardContent className="space-y-4">
        <div>
          <p className="font-semibold">{String(first?.title ?? first?.decision ?? 'Use product snapshots as the primary UI object.')}</p>
        </div>
        <div>
          <p className="text-xs font-medium text-muted-foreground">Reason</p>
          <p className="mt-1 text-sm text-muted-foreground text-pretty">
            {String(first?.reason ?? 'Product state is faster to understand than implementation history alone.')}
          </p>
        </div>
        <div>
          <p className="text-xs font-medium text-muted-foreground">Tradeoff</p>
          <p className="mt-1 text-sm text-muted-foreground text-pretty">
            {String(first?.tradeoff ?? 'Implementation details remain available, but they are secondary to behavior.')}
          </p>
        </div>
        <Link
          className="inline-flex items-center gap-2 text-sm font-medium text-blue-700"
          to="/json/$id"
          params={{ id: evolutionId }}
          hash="decisions"
        >
          View all decisions <ArrowRight className="size-4" />
        </Link>
      </CardContent>
    </Card>
  );
}
