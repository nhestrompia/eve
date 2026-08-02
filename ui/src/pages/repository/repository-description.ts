import { useEffect, useMemo, useState } from "react";

import type { RepositorySummary } from "../../types";

export const DEFAULT_REPOSITORY_DESCRIPTION =
  "Track product states, snapshots, sessions, and verification recorded for this repository.";

export function useRepositoryDescription(
  repository: RepositorySummary,
): readonly [string, (value: string) => void] {
  const storageKey = useMemo(
    () => `eve:repository-description:${repository.name}`,
    [repository.name],
  );
  const [description, setDescription] = useState(
    DEFAULT_REPOSITORY_DESCRIPTION,
  );

  useEffect(() => {
    const saved = window.localStorage.getItem(storageKey);
    setDescription(saved?.trim() || DEFAULT_REPOSITORY_DESCRIPTION);
  }, [storageKey]);

  const saveDescription = (value: string) => {
    const next = value.trim() || DEFAULT_REPOSITORY_DESCRIPTION;
    window.localStorage.setItem(storageKey, next);
    setDescription(next);
  };

  return [description, saveDescription] as const;
}
