import { ArrowRight, Sparkles } from 'lucide-react';
import type { DetailResponse } from '../types';
import { Card, CardContent, CardHeader } from './ui/card';

export function JourneyCard({ detail }: { detail: DetailResponse }) {
  const sessions = detail.sessions.length > 0 ? detail.sessions : detail.evolution.sessions.map((session) => ({
    provider: session.provider ?? 'agent',
    id: session.id ?? 'session',
    key: `${session.provider}:${session.id}`,
    sanitized: false,
    hasTranscript: false
  }));

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
          {sessions.slice(0, 4).map((session, index) => (
            <div key={session.key} className="grid grid-cols-[24px_36px_96px_minmax(0,1fr)_120px_84px] items-center gap-4 text-xs">
              <span className="flex size-6 items-center justify-center rounded-full border bg-white font-mono text-muted-foreground">{index + 1}</span>
              <span className="flex size-8 items-center justify-center rounded-lg bg-violet-50 text-violet-700">✦</span>
              <span className="font-semibold capitalize">{session.provider}</span>
              <span className="truncate text-muted-foreground">
                {index === 0 ? 'Defined architecture and product state.' : 'Implemented and verified snapshot behavior.'}
              </span>
              <span className="text-muted-foreground">May {12 + index}, 9:{14 + index} AM</span>
              <span className="text-muted-foreground">{17 + index * 14} messages</span>
            </div>
          ))}
        </div>
        <a className="mt-6 inline-flex items-center gap-2 text-sm font-medium text-blue-700" href="#sessions">
          View full conversation threads <ArrowRight className="size-4" />
        </a>
      </CardContent>
    </Card>
  );
}
