import { Link } from '@tanstack/react-router';
import { ArrowRight, TriangleAlert } from 'lucide-react';
import { displayRisk } from '../lib/evolution-display';
import { Card, CardContent, CardHeader } from './ui/card';
import { SectionHeading } from './section-heading';

export function RisksCard({ risks, evolutionId }: { risks: unknown[]; evolutionId: string }) {
  const rendered = risks.map(displayRisk);

  return (
    <Card>
      <CardHeader>
        <SectionHeading icon={TriangleAlert} title="Risks" />
      </CardHeader>
      <CardContent>
        {rendered.length === 0 ? (
          <p className="text-sm text-muted-foreground">No risks are recorded in this Evolution.</p>
        ) : (
          <ul className="list-disc space-y-3 pl-5 text-sm text-muted-foreground">
            {rendered.map((risk, index) => (
              <li key={`${risk.title}-${index}`}>{risk.title}</li>
            ))}
          </ul>
        )}
        <Link
          className="mt-6 inline-flex items-center gap-2 text-sm font-medium text-blue-700"
          to="/evolutions/$id/risks"
          params={{ id: evolutionId }}
        >
          View risks <ArrowRight className="size-4" />
        </Link>
      </CardContent>
    </Card>
  );
}
