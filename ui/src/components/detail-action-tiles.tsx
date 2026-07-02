import { Link } from '@tanstack/react-router';
import { Clock3, Code2, GitFork, Scale } from 'lucide-react';
import type { DetailResponse } from '../types';
import { activityEntries } from '../lib/evolution-display';

export function DetailActionTiles({ detail }: { detail: DetailResponse }) {
  const id = detail.summary.id;
  const tiles = [
    {
      title: 'Implementation',
      subtitle: `${detail.commits.length} commits · ${detail.sessions.length} sessions`,
      icon: Code2,
      to: '/evolutions/$id/implementation' as const
    },
    {
      title: 'Decisions & Risks',
      subtitle: `${detail.evolution.decisions.length} decisions · ${detail.evolution.risks.length} risks`,
      icon: Scale,
      to: '/evolutions/$id/decisions' as const
    },
    {
      title: 'Related Evolutions',
      subtitle: relationshipSummary(detail),
      icon: GitFork,
      to: '/evolutions/$id/relationships' as const
    },
    {
      title: 'Evolution Activity',
      subtitle: `${activityEntries(detail.evolution).length} events`,
      icon: Clock3,
      to: '/evolutions/$id/activity' as const
    }
  ];

  return (
    <nav className="grid grid-cols-4 gap-3 border-t py-8" aria-label="Evolution detail sections">
      {tiles.map((tile) => {
        const Icon = tile.icon;
        return (
          <Link
            key={tile.title}
            to={tile.to}
            params={{ id }}
            className="group grid min-h-16 grid-cols-[28px_minmax(0,1fr)] items-center gap-3 rounded-lg bg-white px-4 py-3 shadow-[0_0_0_1px_rgba(15,23,42,0.08)] transition-[box-shadow,background-color,scale] duration-150 hover:bg-slate-50 hover:shadow-[0_0_0_1px_rgba(15,23,42,0.12),0_8px_20px_-16px_rgba(15,23,42,0.42)] active:scale-[0.96]"
          >
            <Icon className="size-5 text-slate-600" />
            <span className="min-w-0">
              <span className="block truncate font-semibold">{tile.title}</span>
              <span className="block truncate text-xs text-muted-foreground">{tile.subtitle}</span>
            </span>
          </Link>
        );
      })}
    </nav>
  );
}

function relationshipSummary(detail: DetailResponse): string {
  const entries = Object.entries(detail.evolution.relationships)
    .flatMap(([kind, values]) => (values ?? []).map((value) => `${kind.replaceAll('_', ' ')} ${value}`));
  return entries.length > 0 ? entries.slice(0, 2).join(' · ') : 'No relationships';
}
