import { Link, useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { compactDate } from '../format';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { Button } from '../components/ui/button';
import { EmptyPanel, Header } from './verification-page';

export function SessionsOverviewPage() {
  const { id } = useParams({ from: '/evolutions/$id/sessions' });
  const evolutions = useQuery({ queryKey: ['evolutions'], queryFn: api.evolutions });
  const detail = useQuery({ queryKey: ['evolution', id], queryFn: () => api.evolution(id) });
  const sessions = useQuery({ queryKey: ['sessions', id], queryFn: () => api.sessions(id) });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {detail.isLoading || sessions.isLoading ? <LoadingState label="Loading AI sessions" /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {sessions.error ? <ErrorState error={sessions.error} /> : null}
      {detail.data && sessions.data ? (
        <section className="space-y-6">
          <Header eyebrow={id} title="AI sessions" subtitle="Conversation references and transcript artifacts connected to this Evolution." />
          {sessions.data.sessions.length === 0 ? <EmptyPanel text="No AI sessions are recorded in this Evolution." /> : null}
          <div className="grid gap-4">
            {sessions.data.sessions.map((session) => (
              <article key={session.key} className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.08)]">
                <div className="flex items-start justify-between gap-6">
                  <div>
                    <p className="font-semibold">{session.providerName}</p>
                    <p className="mt-1 font-mono text-sm text-muted-foreground">{session.id}</p>
                  </div>
                  <span className="rounded-md bg-secondary px-2 py-1 text-sm">
                    {session.hasTranscript ? 'Transcript attached' : session.localSources.length > 0 ? 'Local transcript found' : 'Reference only'}
                  </span>
                </div>
                <div className="mt-5 grid grid-cols-4 gap-3">
                  <Metric label="Events" value={session.preview.eventCount} />
                  <Metric label="Messages" value={session.preview.messageCount} />
                  <Metric label="User" value={session.preview.userMessages} />
                  <Metric label="Agent" value={session.preview.agentMessages} />
                </div>
                {session.hasTranscript || session.localSources.length > 0 ? (
                  <Button asChild className="mt-5">
                    <Link to="/evolutions/$id/session/$sessionId" params={{ id, sessionId: session.key }}>
                      {session.hasTranscript ? 'Read transcript' : 'Read local candidate'}
                    </Link>
                  </Button>
                ) : (
                  <div className="mt-5 rounded-md bg-slate-50 p-4">
                    <p className="text-sm text-muted-foreground">EVE has the provider/id but no attached transcript artifact yet.</p>
                    <code className="mt-2 block font-mono text-xs">{session.captureHint}</code>
                  </div>
                )}
                {(session.localSources ?? []).length > 0 ? (
                  <div className="mt-5">
                    <p className="text-sm font-semibold">Matching local source files</p>
                    <div className="mt-2 space-y-2">
                      {(session.localSources ?? []).map((source) => (
                        <div key={source.path} className="grid grid-cols-[minmax(0,1fr)_110px_120px] gap-3 rounded-md bg-slate-50 px-3 py-2 text-sm">
                          <span className="min-w-0">
                            <span className="block truncate font-medium">{source.title || source.path}</span>
                            <span className="block truncate font-mono text-xs text-muted-foreground">{source.path}</span>
                            {source.match ? <span className="block text-xs text-muted-foreground">Matched by {source.match}</span> : null}
                          </span>
                          <span>{source.format}</span>
                          <span className="text-right text-muted-foreground">{compactDate(source.modifiedAt)}</span>
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}
              </article>
            ))}
          </div>
          <section className="rounded-lg border bg-white p-5">
            <h2 className="text-lg font-semibold">Supported conversation sources</h2>
            <div className="mt-4 grid grid-cols-2 gap-4">
              {sessions.data.providers.map((provider) => (
                <article key={provider.provider} className="rounded-lg border p-4">
                  <div className="flex items-center justify-between gap-4">
                    <h3 className="font-semibold">{provider.name}</h3>
                    <span className="rounded-md border px-2 py-1 text-xs">{provider.available ? 'Root found' : 'No root found'}</span>
                  </div>
                  <code className="mt-3 block font-mono text-xs">{provider.importCommand}</code>
                  <ul className="mt-3 list-disc space-y-1 pl-5 text-sm text-muted-foreground">
                    {provider.displays.map((item) => (
                      <li key={item}>{item}</li>
                    ))}
                  </ul>
                </article>
              ))}
            </div>
          </section>
        </section>
      ) : null}
    </EvolutionShell>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-md border bg-slate-50 p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 font-mono text-lg font-semibold">{value}</p>
    </div>
  );
}
