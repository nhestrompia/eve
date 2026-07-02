import type { LucideIcon } from 'lucide-react';

export function SectionHeading({ icon: Icon, title }: { icon: LucideIcon; title: string }) {
  return (
    <div className="mb-5 flex items-center gap-3">
      <Icon className="size-4 text-slate-600" />
      <h2 className="text-sm font-semibold text-balance">{title}</h2>
    </div>
  );
}
