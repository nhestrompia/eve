import { useParams } from '@tanstack/react-router';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { MarkdownViewer } from '../components/markdown-viewer';
import { StatusBadge } from '../components/status-badge';

export function SessionPage() {
  const { id, sessionId } = useParams({ from: '/snapshots/$id/session/$sessionId' });
  const evolutions = useQuery({ queryKey: ['snapshots'], queryFn: api.snapshots });
  const transcript = useQuery({
    queryKey: ['session', id, sessionId],
    queryFn: () => api.transcript(id, sessionId),
    retry: false
  });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {transcript.isLoading ? <LoadingState label="Loading session" /> : null}
      {transcript.error ? <ErrorState error={transcript.error} /> : null}
      {transcript.data ? (
        <article className="space-y-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-mono text-sm font-semibold text-blue-700">{id}</p>
              <h1 className="mt-2 text-3xl font-semibold text-balance">{transcript.data.title}</h1>
            </div>
            <StatusBadge status={transcript.data.sanitized ? 'sanitized' : 'raw'} />
          </div>
          <MarkdownViewer content={transcript.data.markdown} />
        </article>
      ) : null}
    </EvolutionShell>
  );
}
