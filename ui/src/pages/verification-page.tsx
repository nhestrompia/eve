import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { CheckCircle2, Circle, XCircle } from 'lucide-react';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';

export function VerificationPage() {
  const { id } = useParams({ from: '/evolutions/$id/verification' });
  const evolutions = useQuery({ queryKey: ['evolutions'], queryFn: api.evolutions });
  const detail = useQuery({ queryKey: ['evolution', id], queryFn: () => api.evolution(id) });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {detail.isLoading ? <LoadingState label="Loading verification" /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {detail.data ? (
        <section className="space-y-6">
          <Header eyebrow={id} title="Verification" subtitle="Checks and evidence recorded for this product state." />
          <div className="grid gap-4">
            {detail.data.evolution.verification.length === 0 ? (
              <EmptyPanel text="No verification is recorded in this Evolution." />
            ) : (
              detail.data.evolution.verification.map((item, index) => {
                const Icon = item.status === 'failed' ? XCircle : item.status === 'pending' ? Circle : CheckCircle2;
                const tone = item.status === 'failed' ? 'text-red-600' : item.status === 'pending' ? 'text-orange-500' : 'text-emerald-600';
                return (
                  <article key={`${item.status}-${index}`} className="rounded-lg border bg-white p-5">
                    <div className="flex items-start justify-between gap-6">
                      <div className="flex gap-4">
                        <Icon className={`mt-1 size-5 ${tone}`} />
                        <div>
                          <h2 className="font-semibold capitalize">{item.type || 'Verification'}</h2>
                          <p className="mt-2 font-mono text-sm text-muted-foreground">{item.reference || 'No command/reference recorded.'}</p>
                        </div>
                      </div>
                      <span className="rounded-md border px-2 py-1 text-sm capitalize">{item.status}</span>
                    </div>
                  </article>
                );
              })
            )}
          </div>
        </section>
      ) : null}
    </EvolutionShell>
  );
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
