import { api } from '../api';
import type { PhoneDeviceStatus, PhoneStatus } from '../types';

declare global {
  interface Navigator {
    standalone?: boolean;
    setAppBadge?: (contents?: number) => Promise<void>;
    clearAppBadge?: () => Promise<void>;
  }
}

const subscriptionStorageKey = 'eve-phone-subscription-id';

export function isPhoneStandalone() {
  return Boolean(
    navigator.standalone ||
      (typeof window.matchMedia === 'function' && window.matchMedia('(display-mode: standalone)').matches)
  );
}

export function phonePushSupported() {
  return 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;
}

export function currentNotificationPermission(): NotificationPermission | 'unsupported' {
  return 'Notification' in window ? Notification.permission : 'unsupported';
}

export function savedPhoneSubscriptionID() {
  return window.localStorage.getItem(subscriptionStorageKey) ?? '';
}

export async function enablePhoneNotifications(status: PhoneStatus): Promise<PhoneDeviceStatus> {
  if (!isPhoneStandalone()) {
    throw new Error('Add EVE to the Home Screen before enabling notifications.');
  }
  if (!phonePushSupported() || !status.vapidPublicKey) {
    throw new Error('Web Push is unavailable on this device.');
  }
  const permission = await Notification.requestPermission();
  if (permission !== 'granted') {
    throw new Error('Notification permission was not granted.');
  }
  const registration = await navigator.serviceWorker.register('/sw.js', { scope: '/' });
  await navigator.serviceWorker.ready;
  let subscription = await registration.pushManager.getSubscription();
  if (!subscription) {
    subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(status.vapidPublicKey)
    });
  }
  const json = subscription.toJSON();
  if (!json.endpoint || !json.keys?.p256dh || !json.keys?.auth) {
    throw new Error('Safari returned an incomplete push subscription.');
  }
  const device = await api.registerPhoneSubscription({
    endpoint: json.endpoint,
    expirationTime: subscription.expirationTime,
    keys: { p256dh: json.keys.p256dh, auth: json.keys.auth },
    device: {
      label: /iPad/i.test(navigator.userAgent) ? 'iPad' : 'iPhone',
      userAgent: navigator.userAgent
    }
  });
  window.localStorage.setItem(subscriptionStorageKey, device.id);
  return device;
}

export async function removePhoneNotifications(id: string) {
  const registration = await navigator.serviceWorker.getRegistration('/');
  const subscription = await registration?.pushManager.getSubscription();
  await subscription?.unsubscribe();
  await api.removePhoneSubscription(id);
  window.localStorage.removeItem(subscriptionStorageKey);
}

export async function updatePhoneBadge(count: number) {
  if (count > 0 && navigator.setAppBadge) {
    await navigator.setAppBadge(count).catch(() => undefined);
  } else if (count === 0 && navigator.clearAppBadge) {
    await navigator.clearAppBadge().catch(() => undefined);
  }
}

export function urlBase64ToUint8Array(value: string) {
  const padding = '='.repeat((4 - (value.length % 4)) % 4);
  const base64 = (value + padding).replace(/-/g, '+').replace(/_/g, '/');
  const raw = window.atob(base64);
  return Uint8Array.from(raw, (character) => character.charCodeAt(0));
}
