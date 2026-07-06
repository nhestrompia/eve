import { Check, Clipboard, FileText, History, ListChecks, Save } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import {
  buildChangelogGroups,
  changelogCandidatesForSnapshot,
  formatChangelogMarkdown,
  snapshotChangelogText,
} from '../lib/changelog';
import type { DetailResponse, EvolutionSummary } from '../types';
import { compactDate, humanDate, statusLabel } from '../format';
import { Button } from './ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './ui/dialog';

type SavedChangelog = {
  id: string;
  repository: string;
  baseSnapshotId: string;
  title: string;
  createdAt: string;
  selectedSnapshotIds: string[];
  markdown: string;
};

const storageKey = 'eve.generatedChangelogs.v1';

export function SnapshotChangelogDialog({
  detail,
  evolutions,
}: {
  detail: DetailResponse;
  evolutions: EvolutionSummary[];
}) {
  const current = useMemo(
    () => ({
      ...detail.summary,
      userVisibleChange: detail.snapshot.userVisibleChange || detail.summary.userVisibleChange,
    }),
    [detail],
  );
  const candidates = useMemo(() => {
    const rows = changelogCandidatesForSnapshot(evolutions, current);
    return rows.length > 0 ? rows : [current];
  }, [current, evolutions]);
  const visibleCandidates = useMemo(() => [...candidates].reverse(), [candidates]);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set([current.id]));
  const [saved, setSaved] = useState<SavedChangelog[]>([]);
  const [activeSavedId, setActiveSavedId] = useState<string | null>(null);
  const [copyState, setCopyState] = useState<'idle' | 'copied'>('idle');

  useEffect(() => {
    setSelectedIds(new Set([current.id]));
    setActiveSavedId(null);
  }, [current.id]);

  useEffect(() => {
    setSaved(readSavedChangelogs().filter((item) => item.repository === detail.repository));
  }, [detail.repository]);

  const selectedSnapshots = useMemo(
    () => candidates.filter((candidate) => selectedIds.has(candidate.id)),
    [candidates, selectedIds],
  );
  const generatedMarkdown = useMemo(
    () => formatChangelogMarkdown(buildChangelogGroups(selectedSnapshots)),
    [selectedSnapshots],
  );
  const activeSaved = saved.find((item) => item.id === activeSavedId);
  const previewMarkdown = activeSaved?.markdown ?? generatedMarkdown;

  const setPreset = (count: number) => {
    const next = candidates.slice(Math.max(0, candidates.length - count));
    setSelectedIds(new Set(next.map((candidate) => candidate.id)));
    setActiveSavedId(null);
  };

  const toggleCandidate = (id: string) => {
    setSelectedIds((previous) => {
      const next = new Set(previous);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
    setActiveSavedId(null);
  };

  const saveGenerated = () => {
    const createdAt = new Date().toISOString();
    const next: SavedChangelog = {
      id: `changelog_${createdAt.replaceAll(/[-:.TZ]/g, '')}`,
      repository: detail.repository,
      baseSnapshotId: current.id,
      title: `${selectedSnapshots.length} changes through ${current.title}`,
      createdAt,
      selectedSnapshotIds: selectedSnapshots.map((snapshot) => snapshot.id),
      markdown: generatedMarkdown,
    };
    const all = [next, ...readSavedChangelogs().filter((item) => item.id !== next.id)].slice(0, 30);
    writeSavedChangelogs(all);
    setSaved(all.filter((item) => item.repository === detail.repository));
    setActiveSavedId(next.id);
  };

  const loadSaved = (item: SavedChangelog) => {
    setSelectedIds(new Set(item.selectedSnapshotIds));
    setActiveSavedId(item.id);
  };

  const copyPreview = async () => {
    await navigator.clipboard.writeText(previewMarkdown);
    setCopyState('copied');
    window.setTimeout(() => setCopyState('idle'), 1600);
  };

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="outline" className="h-12 w-full justify-center gap-3 rounded-lg px-5 sm:w-auto sm:min-w-[220px]">
          <FileText className="size-4" />
          Generate changelog
        </Button>
      </DialogTrigger>
      <DialogContent className="max-w-[min(1120px,calc(100vw-32px))]">
        <DialogHeader>
          <DialogTitle>Generate changelog</DialogTitle>
          <DialogDescription>
            Choose snapshot changes through {current.title}. EVE groups the result with the same rules as the CLI changelog.
          </DialogDescription>
        </DialogHeader>

        <div className="grid min-h-0 gap-4 overflow-hidden lg:grid-cols-[minmax(0,1fr)_340px]">
          <div className="min-h-0 space-y-4 overflow-auto pr-1">
            <section className="rounded-lg bg-secondary p-3 shadow-[var(--shadow-border)]">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h3 className="text-sm font-semibold">Changes to include</h3>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {selectedSnapshots.length} of {candidates.length} selected
                  </p>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button type="button" variant="secondary" size="sm" onClick={() => setPreset(1)}>
                    Current
                  </Button>
                  <Button type="button" variant="secondary" size="sm" onClick={() => setPreset(5)}>
                    Last 5
                  </Button>
                  <Button type="button" variant="secondary" size="sm" onClick={() => setPreset(candidates.length)}>
                    All
                  </Button>
                </div>
              </div>

              <div className="mt-3 grid max-h-[300px] gap-2 overflow-y-auto pr-1">
                {visibleCandidates.map((candidate) => {
                  const selected = selectedIds.has(candidate.id);
                  return (
                    <button
                      type="button"
                      key={candidate.id}
                      onClick={() => toggleCandidate(candidate.id)}
                      className={`grid min-h-16 grid-cols-[20px_minmax(0,1fr)] gap-3 rounded-md bg-card px-3 py-2.5 text-left shadow-[var(--shadow-border)] transition-[background-color,scale] active:scale-[0.99] ${
                        selected ? 'bg-accent' : 'hover:bg-background'
                      }`}
                    >
                      <span
                        className={`mt-1 flex size-5 items-center justify-center rounded-md border ${
                          selected ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-background'
                        }`}
                        aria-hidden="true"
                      >
                        {selected ? <Check className="size-3.5" /> : null}
                      </span>
                      <span className="min-w-0">
                        <span className="block truncate text-sm font-semibold">{snapshotChangelogText(candidate)}</span>
                        <span className="mt-1 block truncate text-xs text-muted-foreground">
                          {statusLabel(candidate.type)} · {compactDate(candidate.createdAt)} · {candidate.id}
                        </span>
                      </span>
                    </button>
                  );
                })}
              </div>
            </section>

            <section className="rounded-lg bg-card shadow-[var(--shadow-border)]">
              <div className="flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
                <div>
                  <h3 className="text-sm font-semibold">Preview</h3>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {activeSaved ? `Saved ${humanDate(activeSaved.createdAt)}` : 'Unsaved draft'}
                  </p>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button type="button" variant="outline" size="sm" onClick={copyPreview}>
                    <Clipboard className="size-3.5" />
                    {copyState === 'copied' ? 'Copied' : 'Copy'}
                  </Button>
                  <Button type="button" size="sm" onClick={saveGenerated} disabled={selectedSnapshots.length === 0}>
                    <Save className="size-3.5" />
                    Save
                  </Button>
                </div>
              </div>
              <pre className="max-h-[300px] overflow-auto whitespace-pre-wrap p-4 text-sm leading-6 text-foreground">
                {previewMarkdown}
              </pre>
            </section>
          </div>

          <aside className="min-h-0 rounded-lg bg-secondary p-3 shadow-[var(--shadow-border)]">
            <div className="flex items-center gap-2">
              <History className="size-4 text-primary" />
              <h3 className="text-sm font-semibold">Past changelogs</h3>
            </div>
            <p className="mt-1 text-xs text-muted-foreground">Saved locally for this repository.</p>
            {saved.length === 0 ? (
              <div className="mt-3 rounded-md bg-card p-3 text-sm text-muted-foreground shadow-[var(--shadow-border)]">
                No changelogs have been saved from this browser yet.
              </div>
            ) : (
              <div className="mt-3 grid max-h-[560px] gap-2 overflow-y-auto pr-1">
                {saved.map((item) => (
                  <button
                    type="button"
                    key={item.id}
                    onClick={() => loadSaved(item)}
                    className={`rounded-md bg-card p-3 text-left shadow-[var(--shadow-border)] transition-[background-color] ${
                      activeSavedId === item.id ? 'bg-accent' : 'hover:bg-background'
                    }`}
                  >
                    <span className="flex items-center justify-between gap-3">
                      <span className="line-clamp-2 text-sm font-semibold">{item.title}</span>
                      <ListChecks className="size-4 shrink-0 text-muted-foreground" />
                    </span>
                    <span className="mt-2 block text-xs text-muted-foreground">
                      {item.selectedSnapshotIds.length} changes · {compactDate(item.createdAt)}
                    </span>
                  </button>
                ))}
              </div>
            )}
          </aside>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function readSavedChangelogs(): SavedChangelog[] {
  if (typeof window === 'undefined') return [];
  try {
    const parsed = JSON.parse(window.localStorage.getItem(storageKey) ?? '[]');
    return Array.isArray(parsed) ? parsed.filter(isSavedChangelog) : [];
  } catch {
    return [];
  }
}

function writeSavedChangelogs(items: SavedChangelog[]) {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(storageKey, JSON.stringify(items));
}

function isSavedChangelog(value: unknown): value is SavedChangelog {
  if (!value || typeof value !== 'object') return false;
  const item = value as Partial<SavedChangelog>;
  return (
    typeof item.id === 'string' &&
    typeof item.repository === 'string' &&
    typeof item.baseSnapshotId === 'string' &&
    typeof item.title === 'string' &&
    typeof item.createdAt === 'string' &&
    typeof item.markdown === 'string' &&
    Array.isArray(item.selectedSnapshotIds)
  );
}
