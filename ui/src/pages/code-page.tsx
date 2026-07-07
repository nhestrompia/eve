import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { ChevronDown, ChevronRight, Code2, FileCode2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { api } from '../api';
import { ErrorState } from '../components/error-state';
import { EvolutionShell } from '../components/evolution-shell';
import { LoadingState } from '../components/loading-state';
import { Button } from '../components/ui/button';
import { formatBytes, shikiLanguage, shikiTheme } from '../lib/code-viewer';
import { highlightCodeToHtml } from '../lib/shiki';
import type { SnapshotCodeFileMode, SnapshotCodeFileResponse, SnapshotCodeFileSummary } from '../types';
import { EmptyPanel, Header } from './verification-page';

export function CodePage() {
  const { id } = useParams({ from: '/snapshots/$id/code' });
  const evolutions = useQuery({ queryKey: ['snapshots'], queryFn: api.snapshots });
  const detail = useQuery({ queryKey: ['snapshot-detail', id], queryFn: () => api.snapshotDetail(id) });
  const files = useQuery({
    queryKey: ['snapshot-code-files', id, detail.data?.repository],
    queryFn: () => api.snapshotCodeFiles(id, detail.data?.repository),
    enabled: Boolean(detail.data?.repository)
  });
  const [showChanged, setShowChanged] = useState(false);
  const [selectedPath, setSelectedPath] = useState<string | undefined>();
  const [mode, setMode] = useState<SnapshotCodeFileMode>('diff');

  const allFiles = files.data?.files ?? [];
  const curatedFiles = allFiles.filter((file) => file.curated);
  const changedFiles = allFiles.filter((file) => !file.curated);
  const selectedFile = allFiles.find((file) => file.path === selectedPath);

  useEffect(() => {
    if (!files.data) return;
    setShowChanged(curatedFiles.length === 0);
    setSelectedPath((current) => {
      if (current && allFiles.some((file) => file.path === current)) return current;
      return allFiles.find((file) => file.previewable)?.path ?? allFiles[0]?.path;
    });
  }, [files.data, allFiles, curatedFiles.length]);

  useEffect(() => {
    setMode('diff');
  }, [selectedPath]);

  const fileContent = useQuery({
    queryKey: ['snapshot-code-file', id, detail.data?.repository, selectedPath, mode],
    queryFn: () => api.snapshotCodeFile(id, selectedPath ?? '', mode, detail.data?.repository),
    enabled: Boolean(detail.data?.repository && selectedPath && selectedFile?.previewable)
  });

  return (
    <EvolutionShell evolutions={evolutions.data ?? []} selectedId={id}>
      {detail.isLoading || files.isLoading ? <LoadingState label="Loading code" /> : null}
      {detail.error ? <ErrorState error={detail.error} /> : null}
      {files.error ? <ErrorState error={files.error} /> : null}
      {detail.data && files.data ? (
        <section className="space-y-6">
          <Header
            eyebrow={detail.data.summary.title || detail.data.snapshot.title || id}
            title="Code"
            subtitle="Relevant code behind this Snapshot, shown from the recorded Git state."
          />
          {allFiles.length === 0 ? (
            <EmptyPanel text="No changed files were found for this Snapshot." />
          ) : (
            <div className="code-tab-shell grid min-h-[620px] grid-cols-1 overflow-hidden rounded-lg border bg-card lg:grid-cols-[340px_minmax(0,1fr)]">
              <CodeFileList
                curatedFiles={curatedFiles}
                changedFiles={changedFiles}
                selectedPath={selectedPath}
                showChanged={showChanged}
                onToggleChanged={() => setShowChanged((value) => !value)}
                onSelect={setSelectedPath}
              />
              <CodeViewer
                file={selectedFile}
                response={fileContent.data}
                loading={fileContent.isLoading}
                error={fileContent.error}
                mode={mode}
                onModeChange={setMode}
              />
            </div>
          )}
        </section>
      ) : null}
    </EvolutionShell>
  );
}

function CodeFileList({
  curatedFiles,
  changedFiles,
  selectedPath,
  showChanged,
  onToggleChanged,
  onSelect
}: {
  curatedFiles: SnapshotCodeFileSummary[];
  changedFiles: SnapshotCodeFileSummary[];
  selectedPath?: string;
  showChanged: boolean;
  onToggleChanged: () => void;
  onSelect: (path: string) => void;
}) {
  return (
    <aside className="border-b bg-secondary lg:border-b-0 lg:border-r">
      <div className="border-b bg-card px-4 py-4">
        <div className="flex items-center gap-2">
          <Code2 className="size-4 text-blue-700" />
          <h2 className="font-semibold">Code</h2>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">Curated files</p>
      </div>

      <div className="max-h-[560px] overflow-auto p-3">
        {curatedFiles.length === 0 ? <p className="px-2 py-3 text-sm text-muted-foreground">No curated file evidence was detected.</p> : null}
        <div className="space-y-2">
          {curatedFiles.map((file) => (
            <CodeFileButton key={file.path} file={file} selected={selectedPath === file.path} onSelect={onSelect} />
          ))}
        </div>

        <div className="my-4 border-t" />

        <button
          type="button"
          className="flex w-full items-center justify-between rounded-md px-2 py-2 text-left text-sm font-semibold hover:bg-card"
          onClick={onToggleChanged}
          aria-expanded={showChanged}
        >
          <span>Show all changed files</span>
          <span className="flex items-center gap-2 text-xs text-muted-foreground">
            {changedFiles.length}
            {showChanged ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
          </span>
        </button>
        {showChanged ? (
          <div className="mt-2 space-y-2">
            {changedFiles.length === 0 ? <p className="px-2 py-3 text-sm text-muted-foreground">All changed files are already curated.</p> : null}
            {changedFiles.map((file) => (
              <CodeFileButton key={file.path} file={file} selected={selectedPath === file.path} onSelect={onSelect} />
            ))}
          </div>
        ) : null}
      </div>
    </aside>
  );
}

function CodeFileButton({
  file,
  selected,
  onSelect
}: {
  file: SnapshotCodeFileSummary;
  selected: boolean;
  onSelect: (path: string) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onSelect(file.path)}
      className={`grid w-full grid-cols-[18px_minmax(0,1fr)] gap-2 rounded-lg px-2.5 py-2.5 text-left transition-colors ${
        selected ? 'code-file-selected bg-card shadow-[0_0_0_1px_rgba(37,99,235,0.26)]' : 'hover:bg-card'
      }`}
    >
      <FileCode2 className="mt-0.5 size-4 text-slate-600" />
      <span className="min-w-0">
        <span className="block truncate font-mono text-xs font-semibold">{file.path}</span>
        <span className="mt-1 block truncate text-xs text-muted-foreground">{file.evidence || fileStatusLabel(file)}</span>
      </span>
    </button>
  );
}

function CodeViewer({
  file,
  response,
  loading,
  error,
  mode,
  onModeChange
}: {
  file?: SnapshotCodeFileSummary;
  response?: SnapshotCodeFileResponse;
  loading: boolean;
  error: Error | null;
  mode: SnapshotCodeFileMode;
  onModeChange: (mode: SnapshotCodeFileMode) => void;
}) {
  const unavailable = file && !file.previewable;

  return (
    <section className="min-w-0 bg-card">
      <div className="flex min-h-16 flex-col gap-3 border-b px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <h2 className="truncate font-mono text-sm font-semibold">{file?.path ?? 'Code viewer'}</h2>
          {file ? <p className="mt-1 text-xs text-muted-foreground">{fileStatusLabel(file)}</p> : null}
        </div>
        {file ? (
          <div className="flex w-fit rounded-md border bg-secondary p-1">
            <Button type="button" variant={mode === 'full' ? 'secondary' : 'ghost'} size="sm" onClick={() => onModeChange('full')}>
              Full file
            </Button>
            <Button type="button" variant={mode === 'diff' ? 'secondary' : 'ghost'} size="sm" onClick={() => onModeChange('diff')}>
              Diff
            </Button>
          </div>
        ) : null}
      </div>

      <div className="min-h-[540px] min-w-0">
        {!file ? <EmptyViewer text="Select a file to preview its code." /> : null}
        {unavailable ? <UnavailableViewer reason={file.reason || 'File preview is not available.'} /> : null}
        {file && !unavailable && loading ? <LoadingState label="Loading file" /> : null}
        {file && !unavailable && error ? <ErrorState error={error} /> : null}
        {file && !unavailable && response ? <HighlightedCode response={response} path={file.path} mode={mode} /> : null}
      </div>
    </section>
  );
}

function HighlightedCode({ response, path, mode }: { response: SnapshotCodeFileResponse; path: string; mode: SnapshotCodeFileMode }) {
  const [html, setHtml] = useState('');
  const language = useMemo(() => shikiLanguage(response.language, path, mode), [mode, path, response.language]);
  const theme = useCodeTheme();

  useEffect(() => {
    let cancelled = false;
    async function highlight() {
      if (response.previewable === false) {
        setHtml('');
        return;
      }
      try {
        const rendered = await highlightCodeToHtml(response.content || '', language, theme);
        if (!cancelled) setHtml(mode === 'diff' ? decorateDiffHtml(rendered, response.content || '') : rendered);
      } catch {
        if (!cancelled) {
          setHtml(`<pre class="code-fallback"><code>${escapeHtml(response.content || '')}</code></pre>`);
        }
      }
    }
    void highlight();
    return () => {
      cancelled = true;
    };
  }, [language, mode, response.content, response.previewable, theme]);

  if (response.previewable === false) {
    return <UnavailableViewer reason={response.reason || 'File preview is not available.'} />;
  }
  return (
    <div
      className={`code-viewer ${mode === 'diff' ? 'code-viewer-diff' : 'code-viewer-full'} max-h-[calc(100dvh-220px)] min-h-[540px] overflow-auto p-0`}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}

function EmptyViewer({ text }: { text: string }) {
  return <div className="flex min-h-[540px] items-center justify-center p-6 text-sm text-muted-foreground">{text}</div>;
}

function UnavailableViewer({ reason }: { reason: string }) {
  return (
    <div className="flex min-h-[540px] items-center justify-center p-6">
      <div className="max-w-sm rounded-lg border bg-slate-50 p-5 text-center">
        <p className="font-semibold">File preview unavailable</p>
        <p className="mt-2 text-sm text-muted-foreground">{reason}</p>
      </div>
    </div>
  );
}

function fileStatusLabel(file: SnapshotCodeFileSummary) {
  const status = file.status === 'A' ? 'Added' : file.status === 'M' ? 'Modified' : file.status === 'D' ? 'Deleted' : 'Changed';
  return `${status} · ${file.language || 'text'} · ${formatBytes(file.sizeBytes)}`;
}

function escapeHtml(value: string) {
  return value.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;');
}

function useCodeTheme() {
  const [theme, setTheme] = useState(shikiTheme);

  useEffect(() => {
    const update = () => setTheme(shikiTheme());
    update();
    const observer = new MutationObserver(update);
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] });
    return () => observer.disconnect();
  }, []);

  return theme;
}

function decorateDiffHtml(html: string, content: string) {
  const lines = content.split('\n');
  let index = 0;
  return html.replace(/<span class="line">/g, () => {
    const rawLine = lines[index] ?? '';
    index += 1;
    const kind = rawLine.startsWith('@@')
      ? 'diff-hunk'
      : rawLine.startsWith('+')
        ? 'diff-add'
        : rawLine.startsWith('-')
          ? 'diff-remove'
          : 'diff-context';
    return `<span class="line diff-line ${kind}" data-line="${index}">`;
  });
}
