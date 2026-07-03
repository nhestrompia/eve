import { CheckCircle2, Circle, ShieldCheck, XCircle } from 'lucide-react';
import type { Verification } from '../types';
import { titleCase } from '../lib/evolution-display';

export function VerificationSummarySection({ values }: { values: Verification[]; evolutionId: string }) {
  const visible = values.slice(0, 3);

  return (
    <section className="grid grid-cols-[44px_minmax(0,1fr)] gap-5 border-t py-8">
      <div className="flex size-10 items-center justify-center rounded-full bg-slate-100 text-slate-700 shadow-[0_0_0_1px_rgba(15,23,42,0.06)]">
        <ShieldCheck className="size-5" />
      </div>
      <div className="min-w-0">
        <h2 className="text-lg font-semibold text-balance">Verification (initial)</h2>
        <div className="mt-6 max-w-[640px] rounded-lg bg-white p-4 shadow-[0_0_0_1px_rgba(15,23,42,0.08),0_6px_16px_-12px_rgba(15,23,42,0.34)]">
          {visible.length === 0 ? <p className="p-2 text-muted-foreground">No verification is recorded for this state.</p> : null}
          <div className="space-y-1">
            {visible.map((item, index) => {
              const Icon = item.status === 'failed' ? XCircle : item.status === 'pending' ? Circle : CheckCircle2;
              const tone = item.status === 'failed' ? 'text-red-600' : item.status === 'pending' ? 'text-orange-500' : 'text-emerald-600';
              return (
                <div key={`${item.status}-${item.reference}-${index}`} className="grid grid-cols-[20px_minmax(0,1fr)] items-center gap-3 rounded-md px-2 py-3 sm:grid-cols-[20px_minmax(0,1fr)_96px] sm:gap-4">
                  <Icon className={`size-4 ${tone}`} />
                  <span className="truncate">{item.reference || item.type || 'Verification'}</span>
                  <span className={`col-start-2 text-sm sm:col-start-auto sm:text-right ${item.status === 'failed' ? 'text-red-600' : 'text-muted-foreground'}`}>
                    {titleCase(item.status || 'unknown')}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </section>
  );
}
