import { useRouterState } from '@tanstack/react-router';

export function TopBar({ onSearch }: { onSearch: () => void }) {
  const state = useRouterState();
  const match = state.location.pathname.match(/EV-\d+/);
  const id = match?.[0];

  return (
    <header className="sticky top-0 z-20 flex h-[76px] items-center border-b bg-white/88 px-8 backdrop-blur">
      <button type="button" onClick={onSearch} className="sr-only">
        Open search
      </button>
      <div className="flex items-center gap-3 text-sm text-muted-foreground">
        <span>Activity</span>
        {id ? (
          <>
            <span>›</span>
            <span className="font-semibold text-foreground">#{id.replace('EV-', '')}</span>
          </>
        ) : null}
      </div>
    </header>
  );
}
