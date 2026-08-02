import {
  Box,
  Calendar,
  Code2,
  Edit3,
  HardDrive,
  Save,
  X,
} from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "../../components/ui/button";
import { compactDate } from "../../format";
import type { RepositorySummary } from "../../types";
import { DEFAULT_REPOSITORY_DESCRIPTION } from "./repository-description";
import { RailCard } from "./rail-card";
import { formatBytes } from "./types";

export function RepositoryFactsCard({
  repository,
  description,
  onDescriptionChange,
}: {
  repository: RepositorySummary;
  description: string;
  onDescriptionChange: (value: string) => void;
}): React.JSX.Element {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(description);
  useEffect(() => setDraft(description), [description]);
  const rows = [
    ["Language", repository.primaryLanguage || "Unknown", Code2],
    ["Size", formatBytes(repository.sizeBytes), HardDrive],
    ["Created", compactDate(repository.createdAt), Calendar],
  ] as const;

  return (
    <RailCard title="Repository overview">
      <div className="space-y-5">
        <div className="grid grid-cols-[18px_minmax(0,1fr)] gap-3">
          <Box className="mt-0.5 size-4 text-slate-500" />
          <div className="min-w-0">
            <div className="flex items-center justify-between gap-3">
              <p className="text-xs font-medium text-muted-foreground">
                Description
              </p>
              <button
                type="button"
                className="inline-flex size-7 items-center justify-center rounded-md text-slate-500 hover:bg-slate-50 hover:text-slate-950"
                onClick={() => setEditing((value) => !value)}
                aria-label={
                  editing ? "Cancel description edit" : "Edit description"
                }
              >
                {editing ? (
                  <X className="size-3.5" />
                ) : (
                  <Edit3 className="size-3.5" />
                )}
              </button>
            </div>
            {editing ? (
              <form
                className="mt-2 space-y-2"
                onSubmit={(event) => {
                  event.preventDefault();
                  const value = draft.trim() || DEFAULT_REPOSITORY_DESCRIPTION;
                  onDescriptionChange(value);
                  setEditing(false);
                }}
              >
                <textarea
                  value={draft}
                  onChange={(event) => setDraft(event.target.value)}
                  className="min-h-24 w-full resize-y rounded-md border bg-white px-3 py-2 text-sm leading-5 text-slate-700 outline-none focus:ring-2 focus:ring-blue-500"
                />
                <div className="flex justify-end gap-2">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setDraft(description);
                      setEditing(false);
                    }}
                  >
                    Cancel
                  </Button>
                  <Button type="submit" size="sm" className="gap-2">
                    <Save className="size-3.5" />
                    Save
                  </Button>
                </div>
              </form>
            ) : (
              <p className="mt-1 text-sm leading-5 text-slate-700 text-pretty">
                {description}
              </p>
            )}
          </div>
        </div>
        {rows.map(([label, value, Icon]) => (
          <div
            key={label}
            className="grid grid-cols-[18px_minmax(0,1fr)] gap-3"
          >
            <Icon className="mt-0.5 size-4 text-slate-500" />
            <div className="min-w-0">
              <p className="text-xs font-medium text-muted-foreground">
                {label}
              </p>
              <p className="mt-1 text-sm leading-5 text-slate-700 text-pretty">
                {value}
              </p>
            </div>
          </div>
        ))}
      </div>
    </RailCard>
  );
}
