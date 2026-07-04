import { Check, Copy } from 'lucide-react';
import { useState } from 'react';
import { cn } from '../lib/utils';

export function MarkdownViewer({
  content,
  className,
  surface = 'card'
}: {
  content: string;
  className?: string;
  surface?: 'card' | 'bare';
}) {
  const lines = content.split(/\r?\n/);
  const blocks: React.ReactNode[] = [];
  let code: string[] = [];
  let inCode = false;

  lines.forEach((line, index) => {
    if (line.startsWith('```')) {
      if (inCode) {
        blocks.push(
          <CopyableCodeBlock key={`code-${index}`} value={code.join('\n')} />
        );
        code = [];
      }
      inCode = !inCode;
      return;
    }
    if (inCode) {
      code.push(line);
      return;
    }
    if (line.startsWith('# ')) blocks.push(<h1 key={index} className="text-2xl font-semibold text-balance">{line.slice(2)}</h1>);
    else if (line.startsWith('## ')) blocks.push(<h2 key={index} className="mt-8 text-lg font-semibold text-balance">{line.slice(3)}</h2>);
    else if (line.startsWith('### ')) blocks.push(<h3 key={index} className="mt-6 text-base font-semibold text-balance">{line.slice(4)}</h3>);
    else if (line.startsWith('- ')) blocks.push(<p key={index} className="font-mono text-xs text-muted-foreground">{line}</p>);
    else if (line.trim() !== '') blocks.push(<p key={index} className="max-w-[76ch] text-pretty">{line}</p>);
  });

  return (
    <div className={cn('space-y-4', surface === 'card' ? 'rounded-lg border bg-white p-8' : '', className)}>
      {blocks}
    </div>
  );
}

function CopyableCodeBlock({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);

  const copyCode = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1200);
  };

  return (
    <div className="group relative rounded-lg bg-slate-950">
      <button
        type="button"
        className="absolute right-2 top-2 inline-flex h-8 items-center gap-1.5 rounded-md bg-white/10 px-2 text-xs font-medium text-white/80 opacity-100 transition hover:bg-white/15 hover:text-white sm:opacity-0 sm:group-hover:opacity-100"
        onClick={copyCode}
        aria-label={copied ? 'Code copied' : 'Copy code'}
      >
        {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
        {copied ? 'Copied' : 'Copy'}
      </button>
      <pre className="overflow-auto rounded-lg p-4 pr-20 font-mono text-xs text-white">
        {value}
      </pre>
    </div>
  );
}
