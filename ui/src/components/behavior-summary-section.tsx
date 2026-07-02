import { ChevronDown, CheckCircle2, Target } from 'lucide-react';
import { useState } from 'react';
import type { Behavior } from '../types';
import { behaviorClaims } from '../lib/evolution-display';
import { Button } from './ui/button';

export function BehaviorSummarySection({ behavior }: { behavior: Behavior }) {
  const [expanded, setExpanded] = useState(false);
  const claims = behaviorClaims(behavior);
  const visible = expanded ? claims : claims.slice(0, 3);

  return (
    <section className="grid grid-cols-[44px_minmax(0,1fr)] gap-5 py-8">
      <div className="flex size-10 items-center justify-center rounded-full bg-blue-50 text-blue-700 shadow-[0_0_0_1px_rgba(37,99,235,0.08)]">
        <Target className="size-5" />
      </div>
      <div className="min-w-0">
        <h2 className="text-lg font-semibold text-balance">What was implemented</h2>
        <div className="mt-6 space-y-4">
          {visible.length === 0 ? <p className="text-muted-foreground">No behavior changes were recorded.</p> : null}
          {visible.map((claim, index) => (
            <div key={`${claim.description}-${index}`} className="grid grid-cols-[20px_minmax(0,1fr)] gap-4">
              <CheckCircle2 className="mt-0.5 size-4 text-emerald-600" />
              <p className="text-[15px] leading-6 text-pretty">{claim.description}</p>
            </div>
          ))}
        </div>
        {claims.length > 3 ? (
          <Button
            type="button"
            variant="secondary"
            className="mt-6 h-10 gap-2 rounded-lg"
            aria-expanded={expanded}
            onClick={() => setExpanded((value) => !value)}
          >
            {expanded ? 'Show less behavior' : 'Show full behavior'}
            <ChevronDown className={`size-4 transition-transform duration-150 ${expanded ? 'rotate-180' : ''}`} />
          </Button>
        ) : null}
      </div>
    </section>
  );
}
