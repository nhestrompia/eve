import { Link } from "@tanstack/react-router";
import { ArrowRight, Bot, Search } from "lucide-react";
import type { ReactNode } from "react";

export function DashboardShell({
  title,
  subtitle,
  searchPlaceholder,
  children,
}: {
  title: string;
  subtitle: string;
  searchPlaceholder: string;
  children: ReactNode;
}) {
  return (
    <main className="min-h-dvh bg-[oklch(0.986_0.003_247)] px-5 pb-10 pt-7 sm:px-8 md:px-11 md:pb-16 md:pt-12">
      <div className="w-full max-w-[1360px]">
        <header className="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <h1 className="text-[38px] font-semibold leading-none tracking-[-0.03em] text-slate-950 sm:text-[44px]">
              {title}
            </h1>
            <p className="mt-3 max-w-[62ch] text-[15px] leading-6 text-slate-600">
              {subtitle}
            </p>
          </div>
          <DashboardSearch placeholder={searchPlaceholder} />
        </header>
        {children}
      </div>
    </main>
  );
}

export function DashboardSearch({ placeholder }: { placeholder: string }) {
  return (
    <button
      type="button"
      onClick={() => openDashboardSearch()}
      className="flex h-12 w-full max-w-[330px] items-center gap-3 rounded-lg border border-slate-200 bg-white/70 px-4 text-left text-sm text-slate-500 shadow-[0_1px_2px_rgba(15,23,42,0.03)] transition-colors hover:bg-white sm:w-[310px] lg:w-[330px]"
    >
      <Search className="size-5 shrink-0 text-slate-600" strokeWidth={1.8} />
      <span className="min-w-0 flex-1 truncate">{placeholder}</span>
      <kbd className="rounded-md bg-slate-100 px-2 py-1 font-mono text-[11px] font-medium text-slate-500">
        ⌘K
      </kbd>
    </button>
  );
}

export function openDashboardSearch(query = "") {
  window.dispatchEvent(new CustomEvent("eve:open-search", { detail: { query } }));
}

export function Pill({ tone, children }: { tone: PillTone; children: ReactNode }) {
  return (
    <span className={pillToneClasses[tone]}>
      {children}
    </span>
  );
}

export type PillTone = "verified" | "waiting" | "progress" | "pending" | "ready" | "rejected";

const pillToneClasses: Record<PillTone, string> = {
  verified: "inline-flex h-7 items-center rounded-md bg-emerald-50 px-3 text-xs font-medium text-emerald-700",
  waiting: "inline-flex h-7 items-center rounded-md bg-orange-50 px-3 text-xs font-medium text-orange-700",
  progress: "inline-flex h-7 items-center rounded-md bg-slate-100 px-3 text-xs font-medium text-slate-950",
  pending: "inline-flex h-7 items-center rounded-md bg-slate-50 px-3 text-xs font-medium text-slate-700",
  ready: "inline-flex h-7 items-center rounded-md bg-indigo-50 px-3 text-xs font-medium text-indigo-700",
  rejected: "inline-flex h-7 items-center rounded-md bg-red-50 px-3 text-xs font-medium text-red-700",
};

export function AgentAvatar({ agent }: { agent: string }) {
  const normalized = agent.toLowerCase();
  if (!normalized.includes("claude")) {
    return (
      <span className="inline-grid size-7 shrink-0 place-items-center rounded-full bg-black text-white">
        <Bot className="size-4" strokeWidth={1.8} />
      </span>
    );
  }
  return (
    <span className="inline-grid size-7 shrink-0 place-items-center overflow-hidden rounded-full bg-orange-500">
      <img src="/agents/claude.svg" alt="" className="size-7 object-cover" />
    </span>
  );
}

export function RowLink({
  to,
  params,
  children,
}: {
  to: string;
  params?: Record<string, string>;
  children: ReactNode;
}) {
  return (
    <Link
      to={to}
      params={params}
      className="grid min-h-[76px] grid-cols-1 gap-3 border-t border-slate-200 px-4 py-4 transition-colors hover:bg-slate-50/80 sm:px-6 lg:grid-cols-[minmax(0,1fr)_auto]"
    >
      {children}
    </Link>
  );
}

export function ViewAllLink({ to, label = "View all" }: { to: string; label?: string }) {
  return (
    <Link to={to} className="inline-flex items-center gap-3 text-sm font-medium text-slate-950 transition-opacity hover:opacity-70">
      {label}
      <ArrowRight className="size-4" strokeWidth={1.8} />
    </Link>
  );
}

export function relativeTime(value?: string) {
  if (!value) return "unknown";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  const seconds = Math.max(0, Math.round((Date.now() - date.getTime()) / 1000));
  if (seconds < 60) return "now";
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.round(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.round(days / 30);
  if (months < 12) return `${months}mo ago`;
  return `${Math.round(months / 12)}y ago`;
}

export function shortHash(value?: string) {
  if (!value) return "pending";
  return value.length > 7 ? value.slice(0, 7) : value;
}

export function verificationPercent(total: number, failed: number) {
  if (total <= 0) return 100;
  return Math.max(0, Math.round(((total - failed) / total) * 100));
}
