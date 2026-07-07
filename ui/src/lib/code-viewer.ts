export function languageFromPath(path: string) {
  const file = path.split('/').pop()?.toLowerCase() ?? '';
  const ext = file.includes('.') ? file.slice(file.lastIndexOf('.')) : '';
  switch (file) {
    case 'dockerfile':
      return 'dockerfile';
    case 'makefile':
      return 'makefile';
    case 'go.mod':
    case 'go.sum':
      return 'go';
    case 'package.json':
    case 'tsconfig.json':
      return 'json';
    default:
      break;
  }
  switch (ext) {
    case '.go':
      return 'go';
    case '.ts':
    case '.tsx':
      return 'tsx';
    case '.js':
    case '.jsx':
    case '.mjs':
    case '.cjs':
      return 'javascript';
    case '.json':
      return 'json';
    case '.css':
      return 'css';
    case '.html':
      return 'html';
    case '.md':
    case '.mdx':
      return 'markdown';
    case '.yml':
    case '.yaml':
      return 'yaml';
    case '.sh':
    case '.bash':
    case '.zsh':
      return 'bash';
    case '.py':
      return 'python';
    case '.rb':
      return 'ruby';
    case '.rs':
      return 'rust';
    case '.java':
      return 'java';
    case '.c':
    case '.h':
      return 'c';
    case '.cpp':
    case '.cc':
    case '.cxx':
    case '.hpp':
      return 'cpp';
    case '.sql':
      return 'sql';
    case '.xml':
      return 'xml';
    default:
      return 'text';
  }
}

export function shikiLanguage(language: string | undefined, path: string, mode: 'diff' | 'full') {
  if (mode === 'diff') return 'diff';
  return language && language !== 'text' ? language : languageFromPath(path);
}

export function shikiTheme() {
  if (typeof document !== 'undefined' && document.documentElement.classList.contains('dark-preview')) {
    return 'github-dark';
  }
  return 'github-light';
}

export function formatBytes(value: number | undefined) {
  if (!value || value < 1) return '0 B';
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(value < 10 * 1024 ? 1 : 0)} KiB`;
  return `${(value / (1024 * 1024)).toFixed(1)} MiB`;
}
