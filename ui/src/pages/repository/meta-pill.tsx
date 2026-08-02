import type { LucideIcon } from "lucide-react";

export function MetaPill({
  icon: Icon,
  label,
  tone,
}: {
  icon: LucideIcon;
  label: string;
  tone?: "success" | "warning";
}): React.JSX.Element {
  return (
    <span
      className={`inline-flex h-8 max-w-full items-center gap-2 rounded-md bg-white px-3 text-xs font-medium shadow-[0_0_0_1px_rgba(15,23,42,0.12)] ${
        tone === "success"
          ? "text-emerald-700"
          : tone === "warning"
            ? "text-orange-700"
            : "text-slate-600"
      }`}
    >
      <Icon className="size-3.5 shrink-0" />
      <span className="truncate">{label}</span>
    </span>
  );
}
