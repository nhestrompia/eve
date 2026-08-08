# EVE Approvals technical beta

EVE Approvals lets an iPhone or iPad review and decide pending EVE plans while the EVE runtime stays bound to the Mac's localhost interface. Tailscale Serve supplies private HTTPS and the signed-in Tailscale identity. Web Push supplies visible notifications when the Home Screen app is suspended.

This beta requires:

- A macOS host with Tailscale 1.52 or newer installed and authenticated.
- An iPhone or iPad running iOS/iPadOS 16.4 or newer and signed into the same Tailscale account.
- MagicDNS and HTTPS certificates enabled for the tailnet. The setup command can open Tailscale's HTTPS consent flow when needed.

EVE does not install Tailscale, change tailnet ACLs or Grants, expose the runtime on the LAN, or use Tailscale Funnel.

## Set up the Mac

Build or install EVE, then run:

```sh
eve phone setup
```

The command verifies Tailscale, reserves private HTTPS port `8443`, installs the `com.nhestrompia.eve.phone` LaunchAgent, waits for the runtime, and prints a private URL and QR code. If another service already owns that Serve port, EVE stops without replacing it.

Scan the QR code with the iPhone, open the private URL, and use Share → Add to Home Screen. Launch EVE from the new Home Screen icon and tap **Enable notifications**. Notification permission is intentionally unavailable in an ordinary Safari tab because iOS Web Push requires an installed Home Screen app and a direct user gesture.

Run `eve phone status` to inspect the configured origin, Tailscale identity, LaunchAgent, Serve mapping, registered device labels, push health, and pending notification work. `eve doctor` includes the same phone-access diagnostics alongside the existing repository checks.

## Use the phone surface

The queue shows current pending plans oldest first. A notification opens the exact plan revision. Review repository, branch, revision, goal, acceptance criteria, allowed paths, milestones, suite/checks, and base context before deciding.

You may edit the goal, acceptance criteria, allowed paths, and required suite. Milestones and resolved checks remain read-only. An edited approval creates and locks a new human-authored revision. Every decision carries the revision that was reviewed; if the plan changed on the Mac, EVE reloads it and will not submit the stale decision.

Approvals are never available as lock-screen actions and plans are not cached for offline use.

## Keep Tailscale reachable on iPhone

Enable Tailscale's VPN On Demand for the tailnet if you want notification links to open reliably away from Wi-Fi. Web Push can wake the PWA without an active VPN, but opening or deciding the plan still requires a working Tailscale connection to the Mac.

If the page says Tailscale is unavailable, open Tailscale, reconnect the VPN, return to EVE Approvals, and retry. An offline page never means an approval was queued.

## Recover notification permission

If permission was denied, open iOS Settings, locate **Notifications**, select **EVE**, and enable **Allow Notifications**. If EVE is not listed, remove the Home Screen app, open its Tailscale URL in Safari, add it to the Home Screen again, and enable notifications from inside the installed app.

Use **Send test notification** on the queue screen to check the selected device. Use **Disconnect this device** to delete its push registration while leaving phone access enabled for other devices.

## Disable or reset

To stop the LaunchAgent and remove only EVE's `8443` Serve listener while preserving keys and device registrations:

```sh
eve phone disable
```

Re-running `eve phone setup` re-enables the same private origin and subscriptions. If the Tailscale host or login changed, setup refuses to rotate the origin unless you explicitly run `eve phone setup --replace`; replacement clears registrations so installed devices must subscribe again.

To delete VAPID keys, subscriptions, and delivery history:

```sh
eve phone reset
```

The reset command asks for confirmation. Use `--yes` only for non-interactive cleanup. All installed devices must enable notifications again after a reset.

## Mac sleep and troubleshooting

The Mac must be awake, online, and connected to Tailscale to originate a notification or serve a plan. EVE reconciles pending work when the runtime starts and once per minute, so undelivered work is retried after wake. A newly registered device is not sent notifications for plans that were already pending, though those plans are immediately visible in its queue.

Useful checks:

```sh
eve phone status
eve doctor
tail -f "$HOME/Library/Logs/eve/phone-daemon.log"
tailscale serve status --json
```

Common recovery actions are to reconnect Tailscale, wake the Mac, confirm the phone and Mac use the same Tailscale login, and send a test notification. Push endpoints, encryption keys, and the VAPID private key are never printed by EVE. Logs remain local and contain only sanitized device and delivery identifiers.

For defense in depth, a tailnet administrator can add a same-user Grant restricting access to the Mac. EVE still enforces the exact Serve host, HTTPS forwarding, Tailscale login, same-origin browser mutations, and revision checks at the application boundary.
