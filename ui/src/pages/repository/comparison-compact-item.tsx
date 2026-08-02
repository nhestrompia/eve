import { Link } from "@tanstack/react-router";

import type { ComparisonPanelItem } from "./comparison-types";

export function ComparisonCompactItem({
  item,
}: {
  item: ComparisonPanelItem;
}): React.JSX.Element {
  return (
    <Link
      to="/snapshots/$id"
      params={{ id: item.snapshotId }}
      className="block px-4 py-3 transition-[background-color] hover:bg-secondary"
    >
      <p className="line-clamp-3 text-sm font-medium leading-5 text-foreground">
        {item.title}
      </p>
      {item.meta ? (
        <p className="mt-1 line-clamp-2 text-xs leading-5 text-muted-foreground">
          {item.meta}
        </p>
      ) : null}
    </Link>
  );
}
