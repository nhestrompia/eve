import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import {
  Bell,
  BellOff,
  CheckCircle2,
  ChevronRight,
  Clock3,
  GitBranch,
  RefreshCw,
  Send,
  ShieldCheck,
  Smartphone,
  WifiOff
} from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { toast } from 'sonner';
import { api } from '../api';
import { relativeTime } from '../components/dashboard-chrome';
import { Button } from '../components/ui/button';
import { currentRevision } from '../components/plan-review/plan-review-validation';
import {
  currentNotificationPermission,
  enablePhoneNotifications,
  isPhoneStandalone,
  phonePushSupported,
  removePhoneNotifications,
  savedPhoneSubscriptionID,
  updatePhoneBadge
} from '../lib/phone-push';

export function PhonePage() {
  const queryClient = useQueryClient();
  const status = useQuery({ queryKey: ['phone-status'], queryFn: api.phoneStatus, retry: false });
  const plans = useQuery({
    queryKey: ['plan-requests', 'phone-review'],
    queryFn: () => api.planRequests('review'),
    retry: false,
    enabled: status.data?.enabled === true
  });
  const [standalone] = useState(() => isPhoneStandalone());
  const [permission, setPermission] = useState(() => currentNotificationPermission());
  const [subscriptionID, setSubscriptionID] = useState(() => savedPhoneSubscriptionID());
  const connectedDevice = status.data?.devices.find((device) => device.id === subscriptionID);
  const queue = useMemo(
    () => [...(plans.data ?? [])].sort((left, right) => (left.createdAt ?? '').localeCompare(right.createdAt ?? '')),
    [plans.data]
  );

  useEffect(() => {
    if (status.data) void updatePhoneBadge(status.data.pendingPlanCount);
  }, [status.data]);

  const enable = useMutation({
    mutationFn: () => {
      if (!status.data) throw new Error('Phone status is unavailable.');
      return enablePhoneNotifications(status.data);
    },
    onSuccess: async (device) => {
      setPermission('granted');
      setSubscriptionID(device.id);
      await queryClient.invalidateQueries({ queryKey: ['phone-status'] });
      toast.success('Notifications enabled');
    },
    onError: (error) => {
      setPermission(currentNotificationPermission());
      toast.error('Could not enable notifications', { description: errorMessage(error) });
    }
  });
  const testNotification = useMutation({
    mutationFn: () => api.testPhoneSubscription(subscriptionID),
    onSuccess: () => toast.success('Test notification sent'),
    onError: (error) => toast.error('Test notification failed', { description: errorMessage(error) })
  });
  const disconnect = useMutation({
    mutationFn: () => removePhoneNotifications(subscriptionID),
    onSuccess: async () => {
      setSubscriptionID('');
      await queryClient.invalidateQueries({ queryKey: ['phone-status'] });
      toast.success('This device was disconnected');
    },
    onError: (error) => toast.error('Could not disconnect', { description: errorMessage(error) })
  });

  if (status.isLoading) {
    return <PhoneMessage icon={<RefreshCw className="animate-spin" />} title="Connecting to your Mac" detail="Checking the private EVE runtime…" />;
  }
  if (status.error) {
    return (
      <PhoneMessage
        icon={<WifiOff />}
        title="Your Mac is out of reach"
		detail="Reconnect Tailscale and make sure the Mac is awake, then try again. VPN On Demand is recommended on iPhone."
        action={<Button onClick={() => void status.refetch()}>Try again</Button>}
      />
    );
  }
  if (!status.data?.enabled) {
    return (
      <PhoneMessage
        icon={<Smartphone />}
        title="Phone approvals are not enabled"
        detail="On the Mac, run this command once to create the private connection."
        action={<code className="phone-command">eve phone setup</code>}
      />
    );
  }

  return (
    <main className="phone-page">
      <header className="phone-home-header">
        <div>
          <img src="/eve.svg" alt="EVE" className="eve-logo h-7 w-auto" />
          <h1>Plans waiting for you</h1>
          <p>{queue.length === 0 ? 'Agents can keep moving when you decide.' : `${queue.length} ${queue.length === 1 ? 'plan needs' : 'plans need'} your judgment.`}</p>
        </div>
        <div className="phone-private-mark" title="Private through Tailscale">
          <ShieldCheck aria-hidden="true" />
          <span>Private</span>
        </div>
      </header>

      <section className="phone-connection" aria-labelledby="phone-connection-title">
        <div className="phone-connection-copy">
          {connectedDevice ? <CheckCircle2 className="text-emerald-600" aria-hidden="true" /> : permission === 'denied' ? <BellOff className="text-red-600" aria-hidden="true" /> : <Bell className="text-indigo-600" aria-hidden="true" />}
          <div>
            <h2 id="phone-connection-title">{connectedDevice ? 'This iPhone is connected' : standalone ? 'Never miss a waiting plan' : 'Install EVE Approvals'}</h2>
            <p>{connectionCopy({ standalone, permission, connected: Boolean(connectedDevice) })}</p>
          </div>
        </div>
		{connectedDevice ? (
		  <p className="phone-device-detail">
			{connectedDevice.label} · {connectedDevice.lastSuccessAt ? `Last notification ${relativeTime(connectedDevice.lastSuccessAt)}` : 'No notification delivered yet'}
		  </p>
		) : null}
        {connectedDevice ? (
          <div className="phone-connection-actions">
            <Button variant="outline" onClick={() => testNotification.mutate()} disabled={testNotification.isPending}>
              <Send aria-hidden="true" /> Send test
            </Button>
            <Button variant="ghost" onClick={() => disconnect.mutate()} disabled={disconnect.isPending}>Disconnect</Button>
          </div>
        ) : standalone && permission !== 'denied' && phonePushSupported() ? (
          <Button size="lg" onClick={() => enable.mutate()} disabled={enable.isPending}>
            <Bell aria-hidden="true" /> {enable.isPending ? 'Connecting…' : 'Enable notifications'}
          </Button>
        ) : !standalone ? (
          <ol className="phone-install-steps">
            <li>Open this page in Safari.</li>
            <li>Tap Share, then Add to Home Screen.</li>
            <li>Open EVE from its new Home Screen icon.</li>
          </ol>
        ) : permission === 'denied' ? (
          <p className="phone-settings-note">Open iPhone Settings → Notifications → EVE, then allow notifications.</p>
        ) : (
          <p className="phone-settings-note">Web Push needs iOS or iPadOS 16.4 or newer.</p>
        )}
      </section>

      <section className="phone-queue" aria-labelledby="phone-queue-title">
        <div className="phone-section-heading">
          <h2 id="phone-queue-title">Approval queue</h2>
          <button type="button" onClick={() => void plans.refetch()} aria-label="Refresh approval queue"><RefreshCw aria-hidden="true" /></button>
        </div>
        {plans.isLoading ? <p className="phone-muted">Reading pending plans…</p> : null}
        {plans.error ? <p className="phone-error">The approval queue could not be loaded. Check Tailscale and try again.</p> : null}
        {!plans.isLoading && !plans.error && queue.length === 0 ? (
          <div className="phone-empty">
            <CheckCircle2 aria-hidden="true" />
            <h3>Nothing is waiting</h3>
            <p>New plans will appear here and notify this device.</p>
          </div>
        ) : null}
        <div className="phone-plan-list">
          {queue.map((plan) => {
            const revision = currentRevision(plan);
            return (
              <Link
                key={plan.planRequestId}
                to="/phone/plans/$planRequestId"
                params={{ planRequestId: plan.planRequestId }}
                className="phone-plan-row"
              >
                <div className="phone-plan-state" data-stale={plan.state === 'stale'} aria-hidden="true" />
                <div className="min-w-0 flex-1">
                  <div className="phone-plan-meta">
                    <strong>{plan.repository}</strong>
                    <span><GitBranch aria-hidden="true" /> {plan.branch}</span>
                  </div>
                  <h3>{revision?.goal || 'Plan awaiting review'}</h3>
                  <p><Clock3 aria-hidden="true" /> {relativeTime(plan.createdAt)}</p>
                </div>
                <ChevronRight aria-hidden="true" />
              </Link>
            );
          })}
        </div>
      </section>
      <footer className="phone-footer">Connected as {status.data.tailscaleLogin}</footer>
    </main>
  );
}

function PhoneMessage({ icon, title, detail, action }: { icon: ReactNode; title: string; detail: string; action?: ReactNode }) {
  return (
    <main className="phone-message">
      <img src="/eve.svg" alt="EVE" className="eve-logo h-8 w-auto" />
      <div className="phone-message-icon">{icon}</div>
      <h1>{title}</h1>
      <p>{detail}</p>
      {action}
    </main>
  );
}

function connectionCopy({ standalone, permission, connected }: { standalone: boolean; permission: string; connected: boolean }) {
  if (connected) return 'Future plans will arrive on the Lock Screen. Your Mac must be awake and online.';
  if (!standalone) return 'Apple enables notifications only after the private page is added to your Home Screen.';
  if (permission === 'denied') return 'Notifications are blocked for this Home Screen app.';
  return 'A notification opens the exact revision that needs your decision.';
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}
