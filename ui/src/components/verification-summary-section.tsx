import { Link } from '@tanstack/react-router';
import { ArrowRight, CheckCircle2, Circle, ShieldCheck, XCircle } from 'lucide-react';
import type { Verification } from '../types';
import { titleCase } from '../lib/evolution-display';

export function VerificationSummarySection({ values, evolutionId }: { values: Verification[]; evolutionId: string }) {
  const visible = values.slice(0, 3);

  return (
    <section className="grid grid-cols-[44px_minmax(0,1fr)] gap-5 border-t py-8">
      <div className="flex size-10 items-center justify-center rounded-full bg-violet-50 text-violet-700 shadow-[0_0_0_1px_rgba(124,58,237,0.08)]">
        <ShieldCheck className="size-5" />
      </div>
      <div className="min-w-0">
        <h2 className="text-lg font-semibold text-balance">Verification</h2>
        <div className="mt-6 max-w-[640px] rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.08),0_6px_16px_-12px_rgba(15,23,42,0.34)]">
          {visible.length === 0 ? <p className="p-2 text-muted-foreground">No verification is recorded for this state.</p> : null}
          <div className="space-y-1">
            {visible.map((item, index) => {
              const Icon = item.status === 'failed' ? XCircle : item.status === 'pending' ? Circle : CheckCircle2;
              const tone = item.status === 'failed' ? 'text-red-600' : item.status === 'pending' ? 'text-orange-500' : 'text-emerald-600';
              return (
                <div key={`${item.status}-${item.reference}-${index}`} className="grid grid-cols-[20px_minmax(0,1fr)_96px] items-center gap-4 rounded-md px-2 py-3">
                  <Icon className={`size-4 ${tone}`} />
                  <span className="truncate">{item.reference || item.type || 'Verification'}</span>
                  <span className={`text-right text-sm ${item.status === 'failed' ? 'text-red-600' : 'text-muted-foreground'}`}>
                    {titleCase(item.status || 'unknown')}
                  </span>
                </div>
              );
            })}
          </div>
          <Link
            className="mt-4 inline-flex min-h-10 items-center gap-2 rounded-md px-2 text-sm font-medium text-blue-700 hover:bg-blue-50"
            to="/evolutions/$id/verification"
            params={{ id: evolutionId }}
          >
            View all results <ArrowRight className="size-4" />
          </Link>
        </div>
      </div>
    </section>
  );
}
