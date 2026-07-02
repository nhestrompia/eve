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
        {decisions.length === 0 ? (
          <p className="text-sm text-muted-foreground">No decisions are recorded in this Evolution.</p>
        ) : (
          <>
            <div>
              <p className="font-semibold">{String(first?.title ?? first?.decision ?? 'Decision record')}</p>
            </div>
            {first?.reason ? (
              <div>
                <p className="text-xs font-medium text-muted-foreground">Reason</p>
                <p className="mt-1 text-sm text-muted-foreground text-pretty">{String(first.reason)}</p>
              </div>
            ) : null}
            {first?.tradeoff ? (
              <div>
                <p className="text-xs font-medium text-muted-foreground">Tradeoff</p>
                <p className="mt-1 text-sm text-muted-foreground text-pretty">{String(first.tradeoff)}</p>
              </div>
            ) : null}
          </>
        )}
        <Link
          className="inline-flex items-center gap-2 text-sm font-medium text-blue-700"
          to="/evolutions/$id/decisions"
          params={{ id: evolutionId }}
        >
          View decisions <ArrowRight className="size-4" />
        </Link>
      </CardContent>
    </Card>
  );
}
