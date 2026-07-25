import { AlertTriangle, CheckCircle2, CircleDashed, FileWarning, ShieldCheck } from 'lucide-react';
import type { PlanRecord, Snapshot } from '../types';

export function PlanAlignmentSection({ snapshot, plan }: { snapshot: Snapshot; plan?: PlanRecord }) {
  const conformance = snapshot.planConformance;
  const locked = plan?.revisions.find((revision) => revision.revision === plan.lockedRevision);
  const noPlan = !snapshot.plan || conformance?.noPlanOnFile;
  const failed = conformance?.status === 'failed';
  const matched = conformance?.status === 'matched';
  const Icon = noPlan ? FileWarning : failed ? AlertTriangle : matched ? CheckCircle2 : CircleDashed;
  const tone = noPlan || failed ? 'border-red-200 bg-red-50 text-red-950' : matched ? 'border-emerald-200 bg-emerald-50 text-emerald-950' : 'border-amber-200 bg-amber-50 text-amber-950';

  return (
    <section className="grid grid-cols-[44px_minmax(0,1fr)] gap-5 border-t py-8" aria-labelledby="plan-alignment-title">
      <div className="flex size-10 items-center justify-center rounded-full bg-slate-100 text-slate-700 shadow-[0_0_0_1px_rgba(15,23,42,0.06)]">
        <ShieldCheck className="size-5" aria-hidden="true" />
      </div>
      <div className="min-w-0">
        <h2 id="plan-alignment-title" className="text-lg font-semibold text-balance">Plan Alignment</h2>
        <div className={`mt-6 max-w-[760px] rounded-lg border p-5 ${tone}`}>
          <div className="flex items-start gap-3">
            <Icon className="mt-0.5 size-5 shrink-0" aria-hidden="true" />
            <div>
              <h3 className="font-semibold">
                {noPlan ? 'Unplanned work' : matched ? 'Implementation matches the locked plan' : failed ? 'Plan conformance failed' : 'Plan evidence is incomplete'}
              </h3>
              <p className="mt-1 text-sm opacity-80">
                {noPlan
                  ? 'This Snapshot was completed without a valid locked Plan reference.'
                  : `Locked revision ${snapshot.plan?.revision} · ${snapshot.plan?.id}`}
              </p>
            </div>
          </div>

          {locked ? (
            <div className="mt-5 grid gap-4 text-sm sm:grid-cols-2">
              <Detail label="Goal" value={locked.goal} />
              <Detail label="Approval source" value={`${locked.source} revision · ${plan?.approvedBy ?? 'local UI'}`} />
              <Detail label="Declared scope" values={locked.allowedPathGlobs} code />
              <Detail label="Required checks" values={locked.resolvedCheckIds.length ? locked.resolvedCheckIds : ['No configured checks']} code />
            </div>
          ) : null}

          {conformance && !noPlan ? (
            <div className="mt-5 grid gap-2 text-sm sm:grid-cols-3">
              <Signal label="Required checks" ok={conformance.requiredChecksStatus === 'passed' || conformance.requiredChecksStatus === 'not_configured'} value={conformance.requiredChecksStatus || 'incomplete'} />
              <Signal label="Policy" ok={conformance.policyMatched} value={conformance.policyMatched ? 'matched' : 'changed'} />
              <Signal label="Check definitions" ok={conformance.checkDefinitionsMatch} value={conformance.checkDefinitionsMatch ? 'matched' : 'changed'} />
            </div>
          ) : null}

          {conformance?.outOfScopePaths?.length ? (
            <div className="mt-5 rounded-md border border-red-200 bg-white/70 p-4">
              <h4 className="font-semibold text-red-900">Out-of-scope paths</h4>
              <ul className="mt-2 space-y-1 font-mono text-xs text-red-800">
                {conformance.outOfScopePaths.map((path) => <li key={path}>{path}</li>)}
              </ul>
            </div>
          ) : null}
        </div>
      </div>
    </section>
  );
}

function Detail({ label, value, values, code = false }: { label: string; value?: string; values?: string[]; code?: boolean }) {
  return (
    <div>
      <p className="font-medium">{label}</p>
      {value ? <p className={`mt-1 opacity-80 ${code ? 'font-mono text-xs' : ''}`}>{value}</p> : null}
      {values ? <ul className={`mt-1 space-y-1 opacity-80 ${code ? 'font-mono text-xs' : ''}`}>{values.map((item) => <li key={item}>{item}</li>)}</ul> : null}
    </div>
  );
}

function Signal({ label, ok, value }: { label: string; ok: boolean; value: string }) {
  return (
    <div className="rounded-md bg-white/70 p-3">
      <p className="text-xs font-medium uppercase tracking-wide opacity-60">{label}</p>
      <p className={`mt-1 font-semibold capitalize ${ok ? 'text-emerald-700' : 'text-red-700'}`}>{value.replaceAll('_', ' ')}</p>
    </div>
  );
}
