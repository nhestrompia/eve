import { Clock } from 'lucide-react';
import type { Evolution } from '../types';
import { Card, CardContent, CardHeader } from './ui/card';
import { SectionHeading } from './section-heading';

export function ActivityCard({ evolution }: { evolution: Evolution }) {
  const timeline = evolution.timeline.length > 0 ? evolution.timeline : [{ event: 'created', description: 'Evolution created.' }];

  return (
    <Card>
      <CardHeader>
        <SectionHeading icon={Clock} title="Snapshot Activity" />
      </CardHeader>
      <CardContent>
        <ol className="relative space-y-5">
          {timeline.map((entry, index) => (
            <li key={`${entry.event}-${index}`} className="grid grid-cols-[18px_minmax(0,1fr)] gap-4 text-xs">
              <span className="relative flex justify-center">
                <span className="size-3 rounded-full border-2 border-emerald-600 bg-white" />
                {index < timeline.length - 1 ? <span className="absolute top-3 h-7 w-px bg-emerald-600" /> : null}
              </span>
              <span className="grid grid-cols-1 gap-1 sm:grid-cols-[110px_minmax(0,1fr)] sm:gap-3">
                <strong className="capitalize">{(entry.event || 'event').replaceAll('_', ' ')}</strong>
                <span className="truncate text-muted-foreground">{entry.timestamp ? new Date(entry.timestamp).toLocaleString() : entry.description}</span>
              </span>
            </li>
          ))}
        </ol>
      </CardContent>
    </Card>
  );
}
