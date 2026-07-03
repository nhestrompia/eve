import { Link } from '@tanstack/react-router';
import { ArrowRight, CheckCircle2, ShieldCheck, XCircle, Circle } from 'lucide-react';
import type { Verification } from '../types';
import { SectionHeading } from './section-heading';
import { Card, CardContent, CardHeader } from './ui/card';

export function VerificationCard({
  values,
  evolutionId,
  showLink = true
}: {
  values: Verification[];
  evolutionId: string;
  showLink?: boolean;
}) {
  return (
    <Card>
      <CardHeader>
        <SectionHeading icon={ShieldCheck} title="Verification" />
      </CardHeader>
      <CardContent>
        <ul className="space-y-4">
          {values.length === 0 ? <li className="text-muted-foreground">No verification recorded.</li> : null}
          {values.map((value, index) => {
            const Icon = value.status === 'failed' ? XCircle : value.status === 'pending' ? Circle : CheckCircle2;
            const color = value.status === 'failed' ? 'text-red-600' : value.status === 'pending' ? 'text-orange-500' : 'text-emerald-600';
            return (
              <li key={`${value.status}-${index}`} className="flex items-center justify-between gap-5">
                <span className="flex min-w-0 items-center gap-3">
                  <Icon className={`size-4 shrink-0 ${color}`} />
                  <span className="truncate">{value.reference || value.type || 'Verification'}</span>
                </span>
                <span className={value.status === 'failed' ? 'text-red-600' : 'text-muted-foreground'}>{title(value.status)}</span>
              </li>
            );
          })}
        </ul>
        {showLink ? (
          <Link
            className="mt-6 inline-flex items-center gap-2 text-sm font-medium text-blue-700"
            to="/snapshots/$id/verification"
            params={{ id: evolutionId }}
          >
            View all results <ArrowRight className="size-4" />
          </Link>
        ) : null}
      </CardContent>
    </Card>
  );
}

function title(value: string) {
  return value.charAt(0).toUpperCase() + value.slice(1);
}
