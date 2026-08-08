// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest';
import { isPhoneStandalone, phonePushSupported, urlBase64ToUint8Array } from './phone-push';

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('phone push capabilities', () => {
  it('decodes URL-safe VAPID public keys', () => {
    expect([...urlBase64ToUint8Array('AQID-_8')]).toEqual([1, 2, 3, 251, 255]);
  });

  it('requires standalone display mode for the iOS permission flow', () => {
    Object.defineProperty(navigator, 'standalone', { configurable: true, value: false });
    Object.defineProperty(window, 'matchMedia', {
      configurable: true,
      value: vi.fn(() => ({ matches: true }))
    });
    expect(isPhoneStandalone()).toBe(true);
  });

  it('feature-detects every Web Push primitive', () => {
    Object.defineProperty(navigator, 'serviceWorker', { configurable: true, value: {} });
    vi.stubGlobal('PushManager', class PushManager {});
    vi.stubGlobal('Notification', { permission: 'default' });
    expect(phonePushSupported()).toBe(true);
  });
});
