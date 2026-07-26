import { Link, useRouterState } from "@tanstack/react-router";
import type { LucideIcon } from "lucide-react";
import {
  ClipboardList,
  Database,
  Home,
  Layers3,
  Search,
  Settings,
} from "lucide-react";

type NavigationItem = {
  label: string;
  to?: string;
  icon: LucideIcon;
  onClick?: () => void;
};

const NAV_ITEMS: NavigationItem[] = [
  { label: "Overview", to: "/", icon: Home },
  { label: "Plans", to: "/plans", icon: ClipboardList },
  { label: "Snapshots", to: "/snapshots", icon: Layers3 },
  { label: "Repositories", to: "/repositories", icon: Database },
  { label: "Search", icon: Search },
];

export function Sidebar({ onSearch }: { onSearch: (query?: string) => void }) {
  const state = useRouterState();
  const pathname = state.location.pathname;

  return (
    <aside className="z-30 border-b border-slate-200/80 bg-[oklch(0.986_0.003_247)] md:fixed md:inset-y-0 md:left-0 md:flex md:w-[216px] md:flex-col md:border-b-0 md:border-r">
      <div className="flex min-h-20 items-center justify-between gap-4 px-6 md:min-h-[118px] md:px-7">
        <Link
          to="/"
          aria-label="Go to overview"
          className="block transition-opacity hover:opacity-75"
        >
          <img
            src="/eve.svg"
            alt="eve"
            className="eve-logo h-8 w-[74px] object-contain object-left"
          />
        </Link>
        <button
          type="button"
          className="grid size-11 place-items-center rounded-lg text-slate-700 transition-colors hover:bg-slate-100 md:hidden"
          aria-label="Open search"
          onClick={() => onSearch()}
        >
          <Search className="size-5" />
        </button>
      </div>

      <nav className="flex gap-1 overflow-x-auto px-4 pb-4 md:block md:space-y-3 md:overflow-visible md:px-4 md:pb-0">
        {NAV_ITEMS.map((item) => {
          const active = item.to ? isActive(pathname, item.to) : false;
          return item.to ? (
            <Link
              key={item.label}
              to={item.to}
              className={navClassName(active)}
              activeProps={{ className: navClassName(true) }}
            >
              <item.icon className="size-5 shrink-0" strokeWidth={1.8} />
              <span className="truncate">{item.label}</span>
            </Link>
          ) : (
            <button
              key={item.label}
              type="button"
              className={navClassName(false)}
              onClick={() => onSearch()}
            >
              <item.icon className="size-5 shrink-0" strokeWidth={1.8} />
              <span className="truncate">{item.label}</span>
            </button>
          );
        })}
      </nav>

      <div className="hidden md:mt-auto md:block">
        <div className="px-4 pb-5">
          <Link
            to="/config"
            className="flex h-12 items-center gap-4 rounded-lg px-4 text-[15px] font-medium transition-colors hover:bg-slate-100"
          >
            <Settings className="size-5 text-slate-700" strokeWidth={1.8} />
            <span>Settings</span>
          </Link>
        </div>

        <div className="mx-7 border-t border-slate-200" />
      </div>
    </aside>
  );
}

function isActive(pathname: string, to: string) {
  if (to === "/") return pathname === "/";
  return pathname === to || pathname.startsWith(`${to}/`);
}

function navClassName(active: boolean) {
  return [
    "relative flex h-12 shrink-0 items-center gap-4 rounded-lg px-4 text-[15px] font-medium transition-[background-color,box-shadow,color] md:w-full",
    active
      ? "bg-white text-slate-950 shadow-[0_0_0_1px_rgba(15,23,42,0.08),0_10px_24px_rgba(15,23,42,0.05)]"
      : "text-slate-950 hover:bg-slate-100",
  ].join(" ");
}
