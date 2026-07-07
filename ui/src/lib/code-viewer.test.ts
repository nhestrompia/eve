import { describe, expect, it } from 'vitest';
import { formatBytes, languageFromPath, shikiLanguage } from './code-viewer';

describe('code viewer helpers', () => {
  it('detects common languages from paths', () => {
    expect(languageFromPath('src/auth/github.ts')).toBe('tsx');
    expect(languageFromPath('cmd/eve/main.go')).toBe('go');
    expect(languageFromPath('Dockerfile')).toBe('dockerfile');
    expect(languageFromPath('README.md')).toBe('markdown');
    expect(languageFromPath('unknown.file')).toBe('text');
  });

  it('uses diff highlighting for diff mode', () => {
    expect(shikiLanguage('go', 'main.go', 'diff')).toBe('diff');
    expect(shikiLanguage('text', 'main.go', 'full')).toBe('go');
  });

  it('formats byte counts compactly', () => {
    expect(formatBytes(0)).toBe('0 B');
    expect(formatBytes(900)).toBe('900 B');
    expect(formatBytes(2048)).toBe('2.0 KiB');
    expect(formatBytes(240 * 1024)).toBe('240 KiB');
  });
});
