import { Link } from '@tanstack/react-router';
import { ArrowRight, FileText, Sparkles } from 'lucide-react';
import type { DetailResponse } from '../types';
import { Card, CardContent, CardHeader } from './ui/card';

export function JourneyCard({ detail }: { detail: DetailResponse }) {
  const sessions = detail.sessions;
  const timeline = detail.evolution.timeline;

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center gap-3">
          <Sparkles className="size-4 text-slate-600" />
          <h2 className="text-sm font-semibold text-balance">Implementation Journey</h2>
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-5">
          {sessions.length > 0 ? sessions.map((session, index) => (
            <div key={session.key} className="grid grid-cols-[24px_36px_minmax(0,1fr)] items-center gap-3 text-xs sm:grid-cols-[24px_36px_96px_minmax(0,1fr)] sm:gap-4">
              <span className="flex size-6 items-center justify-center rounded-full border bg-white font-mono text-muted-foreground">{index + 1}</span>
              <span className="flex size-8 items-center justify-center rounded-lg bg-violet-50 text-violet-700">
                <FileText className="size-4" />
              </span>
              <span className="hidden font-semibold capitalize sm:block">{session.provider}</span>
              <span className="min-w-0">
                <span className="block truncate text-muted-foreground">
                  {session.hasTranscript ? `Transcript: ${session.title || session.id}` : `Reference only: ${session.id}`}
                </span>
                {!session.hasTranscript ? (
                  <code className="mt-1 block truncate font-mono text-[11px] text-muted-foreground">{session.captureHint}</code>
                ) : null}
              </span>
            </div>
          )) : (
            <p className="text-sm text-muted-foreground">No AI sessions are recorded for this Evolution.</p>
          )}
          {sessions.length === 0 && timeline.length > 0 ? (
            <div className="rounded-md border bg-slate-50 p-3 text-xs text-muted-foreground">
              Timeline entries are available, but no session provider/id was attached.
            </div>
          ) : null}
        </div>
        {detail.sessions.find((session) => session.hasTranscript) ? (
          <Link
            className="mt-6 inline-flex items-center gap-2 text-sm font-medium text-blue-700"
            to="/snapshots/$id/session/$sessionId"
            params={{
              id: detail.summary.id,
              sessionId: detail.sessions.find((session) => session.hasTranscript)?.key ?? ''
            }}
          >
            View full conversation threads <ArrowRight className="size-4" />
          </Link>
        ) : (
          <Link
            className="mt-6 inline-flex items-center gap-2 text-sm font-medium text-blue-700"
            to="/snapshots/$id/sessions"
            params={{ id: detail.summary.id }}
          >
            View session references <ArrowRight className="size-4" />
          </Link>
        )}
      </CardContent>
    </Card>
  );
}
