export function MarkdownViewer({ content }: { content: string }) {
  const lines = content.split(/\r?\n/);
  const blocks: React.ReactNode[] = [];
  let code: string[] = [];
  let inCode = false;

  lines.forEach((line, index) => {
    if (line.startsWith('```')) {
      if (inCode) {
        blocks.push(
          <pre key={`code-${index}`} className="overflow-auto rounded-lg bg-slate-950 p-4 font-mono text-xs text-white">
            {code.join('\n')}
          </pre>
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

  return <div className="space-y-4 rounded-lg border bg-white p-8">{blocks}</div>;
}
