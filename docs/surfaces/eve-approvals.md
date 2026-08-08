# EVE Approvals surface brief

## Scope and mode

Operate mode for `/phone` and `/phone/plans/:planRequestId`. This is a focused extension of the existing EVE plan-review experience, not a separate brand or a mobile version of the complete dashboard.

## Audience and job

A developer has stepped away from the Mac and receives notice that an agent is waiting. They need to establish repository, branch, revision, intended outcome, allowed scope, and verification before approving or returning useful feedback.

## Task and information order

The queue answers what is waiting and where. The review route presents goal, acceptance criteria, scope, milestones, suite/checks, and base context before a sticky decision bar. Editing is deliberate and limited to goal, acceptance criteria, allowed paths, and suite. Revision conflicts stop submission and explain what changed.

## Direction

Inherit EVE's restrained light surfaces, slate hierarchy, indigo primary action, compact metadata, and honest status colors. On phone, remove dashboard chrome and let the plan read as one continuous review document rather than a stack of equal cards. The memorable moment is the transition from review to an explicit revision-aware confirmation at the bottom edge of the device.

## Implemented composition

The phone shell is a full-height white surface, centered at a maximum width of 680px against a quiet slate backdrop on larger viewports. The queue uses divider-separated rows rather than cards; each row shows repository, branch, a two-line goal, and waiting age, with the oldest request first. The review route keeps repository, branch, and current revision in a translucent sticky header, separates document sections with rules, and reserves bottom padding for a fixed two-action decision bar.

## Interaction and state handling

Notification setup is progressive: the disabled runtime shows the exact Mac command, Safari shows Home Screen installation steps, the installed app exposes notification permission from a direct tap, and a connected device offers test and disconnect actions. Denied permission points to iPhone Settings instead of presenting a dead control.

Editing is an explicit mode and is limited to goal, acceptance criteria, allowed paths, and required suite. Approval always opens a confirmation that names the revision consequence; approving edits states that EVE will create and lock the next human revision. Requesting changes replaces the decision actions with a focused, required feedback field. A stale plan uses an amber explanation block, removes edit and decision controls, and lists the reasons approval is disabled. Revision conflicts reload the latest plan and require a fresh review; completed decisions resolve to a short outcome screen with a route back to the queue.

## Constraints

- Safe-area-aware at 320px and wider.
- Minimum 44px touch targets.
- No approval from a notification action.
- No offline plan cache or implication that an offline decision succeeded.
- Repository and shortened goal may appear on the lock screen.
- Permission is requested only from a direct tap after Home Screen installation.
