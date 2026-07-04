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

  let codeLanguage = '';

  lines.forEach((line, index) => {
    if (line.startsWith('```')) {
      if (inCode) {
        const value = code.join('\n');
        blocks.push(codeLanguage === 'mermaid' ? <MermaidBlock key={`code-${index}`} value={value} /> : <CopyableCodeBlock key={`code-${index}`} value={value} />);
        code = [];
        codeLanguage = '';
      } else {
        codeLanguage = line.slice(3).trim().toLowerCase();
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

function MermaidBlock({ value }: { value: string }) {
  const diagram = parseFlowchart(value);

  if (!diagram) {
    return <CopyableCodeBlock value={value} />;
  }

  const width = Math.max(560, diagram.nodes.length * 220);
  const height = Math.max(220, Math.ceil(diagram.nodes.length / 2) * 140 + 80);
  const positions = new Map(
    diagram.nodes.map((node, index) => {
      const columns = diagram.nodes.length > 4 ? 3 : 2;
      const column = index % columns;
      const row = Math.floor(index / columns);
      return [node.id, { x: 120 + column * 220, y: 70 + row * 140 }];
    })
  );

  return (
    <div className="overflow-auto rounded-lg border bg-slate-50 p-4">
      <svg width={width} height={height} role="img" aria-label="Mermaid diagram" className="max-w-none">
        <defs>
          <marker id="mermaid-arrow" markerWidth="10" markerHeight="10" refX="8" refY="3" orient="auto" markerUnits="strokeWidth">
            <path d="M0,0 L0,6 L9,3 z" className="fill-slate-500" />
          </marker>
        </defs>
        {diagram.edges.map((edge, index) => {
          const from = positions.get(edge.from);
          const to = positions.get(edge.to);
          if (!from || !to) return null;
          return (
            <line
              key={`${edge.from}-${edge.to}-${index}`}
              x1={from.x + 80}
              y1={from.y}
              x2={to.x - 90}
              y2={to.y}
              className="stroke-slate-500"
              strokeWidth="2"
              markerEnd="url(#mermaid-arrow)"
            />
          );
        })}
        {diagram.nodes.map((node) => {
          const position = positions.get(node.id);
          if (!position) return null;
          return (
            <g key={node.id} transform={`translate(${position.x - 86} ${position.y - 34})`}>
              <rect width="172" height="68" rx="8" className="fill-white stroke-slate-300" />
              <foreignObject width="172" height="68">
                <div className="flex h-[68px] items-center justify-center px-3 text-center text-sm font-semibold leading-5 text-slate-950">
                  {node.label}
                </div>
              </foreignObject>
            </g>
          );
        })}
      </svg>
    </div>
  );
}

function parseFlowchart(value: string) {
  const lines = value
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('%%'));
  if (!/^(flowchart|graph)\s+/i.test(lines[0] ?? '')) return null;

  const nodes = new Map<string, string>();
  const edges: Array<{ from: string; to: string }> = [];
  const nodePattern = /([A-Za-z0-9_-]+)(?:\[(.*?)\]|\((.*?)\)|\{(.*?)\})?/g;

  for (const line of lines.slice(1)) {
    if (!line.includes('-->')) continue;
    const [left, right] = line.split('-->').map((part) => part.trim());
    const parsedLeft = parseMermaidNode(left, nodePattern);
    const parsedRight = parseMermaidNode(right, nodePattern);
    if (!parsedLeft || !parsedRight) continue;
    nodes.set(parsedLeft.id, parsedLeft.label);
    nodes.set(parsedRight.id, parsedRight.label);
    edges.push({ from: parsedLeft.id, to: parsedRight.id });
  }

  if (nodes.size === 0) return null;
  return {
    nodes: Array.from(nodes, ([id, label]) => ({ id, label })),
    edges
  };
}

function parseMermaidNode(value: string, pattern: RegExp) {
  pattern.lastIndex = 0;
  const match = pattern.exec(value);
  if (!match) return null;
  const id = match[1];
  const label = match[2] || match[3] || match[4] || id;
  return { id, label };
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
