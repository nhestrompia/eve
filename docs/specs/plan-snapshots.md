# Plan Snapshots v0.1

Plan Snapshots add a local human gate before Snapshot-worthy implementation:

```text
declare_plan
  → persist pending request
  → return when approved if the connection survives
  → otherwise resume using planRequestId
  → lock an immutable revision
  → implement and verify
  → complete_snapshot(planId, planRevision)
```

EVE does not run the agent or its implementation work.

## Durable request protocol

`planRequestId` is the durable primitive. It is chosen by the caller and must
remain stable across tool timeout, cancellation, MCP restart, daemon restart,
and agent restart.

- A first `declare_plan` call supplies the ID, goal, acceptance-criteria
  Markdown, allowed repository-relative path globs, optional ordered
  milestones, and optional configured verification suite.
- A repeated call with the same ID and identical proposal is idempotent. A
  repeated call may omit the proposal to resume waiting.
- `get_plan_request` reads state without waiting.
- Cancellation ends only the current waiter. It does not cancel or delete the
  request.
- A conflicting reuse of an ID fails.

Pending requests are stored under Git's private EVE directory and therefore do
not dirty the implementation tree. Rejected, stale, and superseded requests
remain there as local audit history.

## Scope and revisions

`allowedPathGlobs` is required. Matching is case-sensitive and
repository-relative. `*` matches within one path segment and `**` crosses
directory boundaries. Absolute paths, traversal, negation, backslashes, and
other glob syntax are rejected.

Agent and human revisions are immutable. Approving unchanged content locks the
agent revision. Approving edits appends and locks a human revision, preserving
the original. Rejecting always requires feedback.

One request may be active per repository branch. A new declaration supersedes
a pending request and wakes its waiters. A locked, unfulfilled Plan blocks a
replacement until a Snapshot fulfills it.

## Staleness

Approval is terminally stale when the reviewed repository context no longer
matches:

- repository HEAD;
- branch;
- tracked or nonignored untracked working-tree fingerprint;
- verification policy/configuration hash; or
- configured suite, resolved check definitions, or check-definition digest.

Approval then returns the exact reasons and a fresh request is required.
Policy changes after lock are completion-time conformance failures.

## Completion and conformance

`complete_snapshot` accepts `planId` and `planRevision` as a pair. A reference
must identify the locked revision for the same repository and branch, and the
locked base must be an ancestor of implementation HEAD. Invalid references
fail. Omitting both remains compatible with adopted repositories but records a
prominent `no_plan` conformance result.

EVE compares rename-aware changes from the locked base to implementation HEAD.
Both old and new rename paths are evaluated. It also compares the selected
verification run with the locked policy hash, suite, ordered check IDs, and
fully resolved check definitions.

Conformance is:

- `matched` when scope, policy, definitions, and one eligible check run match;
- `failed` for check failure, policy/definition drift, or out-of-scope paths;
- `incomplete` when evidence is missing or indeterminate; or
- `no_plan` when no Plan was referenced.

All outcomes are recorded. Acceptance criteria remain human-readable evidence;
v0.1 does not claim semantic verification.

## Local approval API

`eve daemon --addr 127.0.0.1:4317` and `eve dev` share the same runtime:

- `GET /api/plan-requests?status=pending_approval`
- `GET /api/plan-requests/{id}`
- `GET /api/plan-requests/events`
- `POST /api/plan-requests/{id}/approve`
- `POST /api/plan-requests/{id}/reject`

Mutations require `expectedRevision`. Revision or repository-context conflicts
return `409`; invalid edited content returns `422`. SSE sends queue updates and
heartbeats. Clients refetch the full queue after reconnect.

Approve and reject are intentionally absent from MCP and CLI. The localhost API
is a trusted-local UX boundary for the YC-demo milestone, not protection
against malicious local software.
