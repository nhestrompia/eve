import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

describe('EVE Approvals PWA assets', () => {
  it('has a standalone manifest rooted at the focused phone route', () => {
    const manifest = JSON.parse(readFileSync(new URL('../public/manifest.webmanifest', import.meta.url), 'utf8'));
    expect(manifest).toMatchObject({ id: '/phone', start_url: '/phone', scope: '/', display: 'standalone' });
    expect(manifest.icons).toHaveLength(3);
  });

  it('always displays a notification and restricts click URLs to the phone surface', () => {
    const worker = readFileSync(new URL('../public/sw.js', import.meta.url), 'utf8');
    expect(worker).toContain("self.addEventListener('push'");
    expect(worker).toContain('showNotification');
	expect(worker).toContain('safePhoneURL');
	expect(worker).toContain('PLAN_URL.test(value)');
    expect(worker).not.toContain('cache.add');
  });
});
