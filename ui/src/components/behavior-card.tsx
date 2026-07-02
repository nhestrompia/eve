import { CheckCircle2, ListChecks } from 'lucide-react';
import type { Behavior, BehaviorClaim } from '../types';
import { SectionHeading } from './section-heading';
import { Card, CardContent, CardHeader } from './ui/card';

export function BehaviorCard({ behavior }: { behavior: Behavior }) {
  const claims = flattenBehavior(behavior);

  return (
    <Card>
      <CardHeader>
        <SectionHeading icon={ListChecks} title="Behavior" />
      </CardHeader>
      <CardContent>
        {claims.length === 0 ? <p className="text-muted-foreground">No behavior claims recorded.</p> : null}
        <ul className="space-y-4">
          {claims.map((claim, index) => (
            <li key={`${claim.description}-${index}`} className="flex gap-3">
              <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-emerald-600" />
              <span className="text-pretty">{claim.description}</span>
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}

function flattenBehavior(behavior: Behavior): BehaviorClaim[] {
  return [...(behavior.added ?? []), ...(behavior.changed ?? []), ...(behavior.fixed ?? []), ...(behavior.removed ?? [])];
}
