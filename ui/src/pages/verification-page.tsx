import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { CheckCircle2, Circle, ShieldAlert, XCircle } from 'lucide-react';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import type { Snapshot } from '../types';
import type { Verification } from '../types';

export function VerificationPage() {
  const { id } = useParams({ from: '/snapshots/$id/verification' });
  const evolutions = useQuery({ queryKey: ['snapshots'], queryFn: api.snapshots });
  const detail = useQuery({ queryKey: ['snapshot-detail', id], queryFn: () => api.snapshotDetail(id) });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {detail.isLoading ? <LoadingState label="Loading verification" /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {detail.data ? (
        <section className="space-y-6">
          <Header eyebrow={id} title="Verification" subtitle="Checks and evidence recorded for this product state." />
          <TrustBoundary />
          {detail.data.snapshot.verification ? (
            <AggregatePanel snapshot={detail.data.snapshot} />
          ) : <EmptyPanel text="No EVE-executed verification is recorded in this Snapshot." />}
          <ClaimSection title="Agent-reported validation claims" empty="No agent-reported validation claims are recorded." items={detail.data.evolution.verification.filter((item) => item.type === 'command' && item.provenance !== 'legacy_unattributed')} />
          <ClaimSection title="Legacy validation evidence" empty="No legacy validation evidence is recorded." items={detail.data.evolution.verification.filter((item) => item.type === 'command' && item.provenance === 'legacy_unattributed')} />
        </section>
      ) : null}
    </EvolutionShell>
  );
}

function ClaimSection({ title, empty, items }: { title: string; empty: string; items: Verification[] }) {
  return <section className="space-y-3">
    <h2 className="text-lg font-semibold">{title}</h2>
    <div className="grid gap-4">
      {items.length === 0 ? <EmptyPanel text={empty} /> : items.map((item, index) => {
                const failed = item.status === 'failed';
                const passed = item.status === 'passed';
                const Icon = failed ? XCircle : passed ? CheckCircle2 : Circle;
                const tone = failed ? 'text-red-600' : passed ? 'text-emerald-600' : 'text-orange-500';
                return (
                  <article key={`${item.status}-${index}`} className="rounded-lg border bg-white p-5">
                    <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between sm:gap-6">
                      <div className="flex min-w-0 gap-4">
                        <Icon className={`mt-1 size-5 shrink-0 ${tone}`} />
                        <div className="min-w-0">
                          <h2 className="font-semibold capitalize">{item.type || 'Verification'}</h2>
                          <p className="mt-2 break-all font-mono text-sm text-muted-foreground">{item.reference || 'No command/reference recorded.'}</p>
                          {item.provenance ? <p className="mt-2 text-xs text-muted-foreground">{provenanceTitle(item.provenance)}</p> : null}
                        </div>
                      </div>
                      <span className="w-fit rounded-md border px-2 py-1 text-sm capitalize">{item.status}</span>
                    </div>
                  </article>
                );
              })}
    </div>
  </section>;
}

export function TrustBoundary() {
  return (
    <aside className="flex gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-950">
      <ShieldAlert className="mt-0.5 size-5 shrink-0" />
      <p><strong>Tamper-evident local evidence.</strong> This reflects checks configured and executed within this repository. It does not protect against an adversarial actor with repository and shell write access.</p>
    </aside>
  );
}

export function AggregatePanel({ snapshot }: { snapshot: Snapshot }) {
  const verification = snapshot.verification!;
  const failed = verification.status === 'failed' || verification.integrity?.startsWith('evidence_');
  const passed = verification.status === 'required_checks_passed' && verification.integrity === 'matched';
  const Icon = failed ? XCircle : passed ? CheckCircle2 : Circle;
  const tone = failed ? 'text-red-600' : passed ? 'text-emerald-600' : 'text-orange-500';
  const policy = verification.policyChange;
  return (
    <article className="space-y-5 rounded-lg border bg-white p-5">
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <Icon className={`mt-0.5 size-5 shrink-0 ${tone}`} />
          <div><h2 className="font-semibold">{statusTitle(verification.status)}</h2><p className="mt-1 text-sm text-muted-foreground">Profile: {verification.profile || 'not resolved'}</p></div>
        </div>
        <span className="rounded-md border px-2 py-1 text-sm">Integrity: {verification.integrity || 'unknown'}</span>
      </div>
      {policy?.changed ? (
        <div className={`rounded-md border p-3 text-sm ${policy.requirementsReduced ? 'border-red-200 bg-red-50 text-red-900' : 'border-amber-200 bg-amber-50 text-amber-900'}`}>
          {policy.requirementsReduced ? 'Verification requirements were reduced in this change.' : policy.policyIntroduced ? 'Verification policy was introduced in this change.' : 'Verification policy changed in this change.'}
          {policy.addedChecks?.length ? ` Added: ${policy.addedChecks.join(', ')}.` : ''}
          {policy.removedChecks?.length ? ` Removed: ${policy.removedChecks.join(', ')}.` : ''}
        </div>
      ) : null}
      <dl className="grid gap-3 text-sm sm:grid-cols-2">
        <Detail label="Commit" value={snapshot.implementation.gitState} />
        <Detail label="Selected run" value={verification.selectedRunId} />
        <Detail label="Resolved suite" value={verification.suite} />
        <Detail label="Run started" value={verification.runStartedAt} />
        <Detail label="Run completed" value={verification.runCompletedAt} />
        <Detail label="Configuration hash" value={verification.configBlobHash} />
        <Detail label="Run record digest" value={verification.runRecordDigest} />
        <Detail label="Required checks" value={verification.requiredChecks?.join(', ')} />
        <Detail label="Executed checks" value={verification.ranChecks?.join(', ')} />
      </dl>
      {verification.checkResults?.length ? <div><h3 className="text-sm font-semibold">Check results</h3><div className="mt-2 space-y-2">{verification.checkResults.map((check) => <div key={check.checkId} className="rounded-md border p-3 text-sm"><div className="flex items-center justify-between gap-3"><strong>{check.checkId}</strong><span>{statusTitle(check.status)}{check.exitCode !== undefined ? ` · exit ${check.exitCode}` : ''}</span></div><p className="mt-1 text-xs text-muted-foreground">{check.startedAt || 'Unknown start'} → {check.completedAt || 'Unknown completion'} · {check.outputBytes ?? 0} redacted bytes · {check.outputDigest || 'no digest'}</p>{check.output ? <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap text-xs text-muted-foreground">{check.output}{check.truncated ? '\n… output truncated' : ''}</pre> : null}</div>)}</div></div> : null}
      {verification.executorFingerprint ? <DetailGroup title="Executor" values={verification.executorFingerprint} /> : null}
      {verification.refContext ? <DetailGroup title="Reference context" values={verification.refContext} /> : null}
    </article>
  );
}

function Detail({ label, value }: { label: string; value?: string }) {
  return <div><dt className="font-medium text-slate-700">{label}</dt><dd className="mt-1 break-all font-mono text-xs text-muted-foreground">{value || 'Not recorded'}</dd></div>;
}

function DetailGroup({ title, values }: { title: string; values: Record<string, unknown> }) {
  return <div><h3 className="text-sm font-semibold">{title}</h3><div className="mt-2 grid gap-2 sm:grid-cols-2">{Object.entries(values).map(([key, value]) => <Detail key={key} label={key} value={Array.isArray(value) ? value.join(', ') : String(value)} />)}</div></div>;
}

function statusTitle(value: string) {
  if (value === 'required_checks_passed') return 'Required checks passed';
  if (value === 'not_configured') return 'Not configured';
  if (value === 'not_run') return 'Not run';
  if (value === 'incomplete') return 'Incomplete verification';
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function provenanceTitle(value: string) {
  if (value === 'executed_by_eve') return 'Executed by EVE';
  if (value === 'reported_by_agent') return 'Reported by agent';
  if (value === 'legacy_unattributed') return 'Legacy / unattributed';
  return value;
}

export function Header({ eyebrow, title, subtitle }: { eyebrow: string; title: string; subtitle: string }) {
  return (
    <div>
      <p className="font-mono text-sm font-semibold text-blue-700">{eyebrow}</p>
      <h1 className="mt-2 text-3xl font-semibold text-balance">{title}</h1>
      <p className="mt-2 max-w-3xl text-muted-foreground text-pretty">{subtitle}</p>
    </div>
  );
}

export function EmptyPanel({ text }: { text: string }) {
  return <div className="rounded-lg border bg-white p-6 text-muted-foreground">{text}</div>;
}
