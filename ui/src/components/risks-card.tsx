import { ArrowRight, TriangleAlert } from 'lucide-react';
import { Card, CardContent, CardHeader } from './ui/card';
import { SectionHeading } from './section-heading';

export function RisksCard({ risks }: { risks: unknown[] }) {
  const rendered = risks.length > 0 ? risks.map((risk) => String(risk)) : ['Session transcript may be missing.', 'Snapshot commit may not be present locally.', 'Checkout requires a clean working tree.'];

  return (
    <Card>
      <CardHeader>
        <SectionHeading icon={TriangleAlert} title="Risks" />
      </CardHeader>
      <CardContent>
        <ul className="list-disc space-y-3 pl-5 text-sm text-muted-foreground">
          {rendered.map((risk, index) => (
            <li key={`${risk}-${index}`}>{risk}</li>
          ))}
        </ul>
        <a className="mt-6 inline-flex items-center gap-2 text-sm font-medium text-blue-700" href="#risks">
          View all risks <ArrowRight className="size-4" />
        </a>
      </CardContent>
    </Card>
  );
}
