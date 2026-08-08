const FALLBACK_URL = '/phone';
const PLAN_URL = /^\/phone\/plans\/planreq_[A-Za-z0-9_-]{8,120}$/;

function safePhoneURL(value) {
  return value === FALLBACK_URL || (typeof value === 'string' && PLAN_URL.test(value))
    ? value
    : FALLBACK_URL;
}

self.addEventListener('push', (event) => {
  let payload = {};
  try {
    payload = event.data ? event.data.json() : {};
  } catch {
    payload = {};
  }
  const valid = payload && payload.version === 1;
	const url = valid ? safePhoneURL(payload.url) : FALLBACK_URL;
  const connected = valid && payload.type === 'connected';
  const repository = valid && typeof payload.repository === 'string' ? payload.repository : 'EVE';
  const goal = valid && typeof payload.goal === 'string' ? payload.goal : 'Open EVE to review the waiting plan.';
  const title = connected ? 'EVE notifications are connected' : 'Plan needs approval';
  const body = connected ? goal : `${repository} · ${goal}`;
  const pendingCount = valid && Number.isFinite(payload.pendingCount) ? payload.pendingCount : 0;
  const tag = connected ? 'eve:connected' : `plan:${payload.planRequestId || 'pending'}`;

  event.waitUntil(Promise.all([
    self.registration.showNotification(title, {
      body,
      tag,
      renotify: !connected,
      icon: '/icons/eve-192.png',
      badge: '/icons/eve-192.png',
      data: { url }
    }),
    pendingCount > 0 && self.registration.setAppBadge
      ? self.registration.setAppBadge(pendingCount)
      : Promise.resolve()
  ]));
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const rawURL = event.notification.data && event.notification.data.url;
	const relativeURL = safePhoneURL(rawURL);
  const target = new URL(relativeURL, self.location.origin).href;
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then(async (clients) => {
      for (const client of clients) {
        if (new URL(client.url).origin === self.location.origin) {
          await client.navigate(target);
          return client.focus();
        }
      }
      return self.clients.openWindow(target);
    })
  );
});
