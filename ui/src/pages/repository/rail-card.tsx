import { Package } from "lucide-react";

import type { RailCardProps } from "./types";

export function RailCard({
  id,
  title,
  eyebrow,
  children,
}: RailCardProps): React.JSX.Element {
  return (
    <section
      id={id}
      className="rounded-lg bg-white p-5 shadow-[0_0_0_1px_rgba(15,23,42,0.1)]"
    >
      <div className="mb-5 flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold">{title}</h2>
        {eyebrow ? (
          <span className="flex items-center gap-1 text-xs font-medium text-muted-foreground">
            <Package className="size-3" />
            {eyebrow}
          </span>
        ) : null}
      </div>
      {children}
    </section>
  );
}
