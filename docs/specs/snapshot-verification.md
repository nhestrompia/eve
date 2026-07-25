# Snapshot Verification: Required Checks, Trust Model, and Execution Lifecycle

## Problem Statement

EVE snapshots currently record validation claims supplied by an agent. EVE does not execute the recorded command, capture its exit status independently, or distinguish an agent-reported result from an EVE-executed result. A reader therefore cannot tell whether a green result is durable evidence or an unverified assertion.

Allowing an agent to choose arbitrary commands at snapshot completion would not close this gap. The same actor making the claim could choose a trivial command, omit difficult checks, or alter a command while still obtaining an EVE-branded result. Likewise, allowing independently passing checks from different suite runs to be combined would let a snapshot appear green even though no complete run passed.

EVE needs a verification model in which:

- repository policy defines the commands and complete set of checks;
- EVE explicitly executes those checks against a committed Git state;
- execution evidence is durable, bounded, attributable only to an unauthenticated actor claim, and bound to its exact policy and environment;
- agent-reported and historical validation remain visible without being presented as EVE-executed evidence; and
- incomplete or failed verification never prevents EVE from recording honest product history.

The first version is tamper-evident, not tamper-resistant. It protects against optimistic, mistaken, and shortcut-taking agents. It does not protect against a deliberately deceptive agent with repository, Git, and shell access. Tamper-resistance requires a trust anchor outside the agent's authority, such as signed policy, protected-branch state, or human approval.

## Solution

Repositories declare named checks, suites, and Git-context verification profiles in EVE configuration. Agents cannot submit executable strings as EVE-executed checks. They request an explicit suite execution, and EVE resolves the applicable profile and command definitions from committed repository policy.

Suite execution uses an asynchronous lifecycle. Each suite run produces canonical evidence tied to the implementation commit, configuration blob, server-derived Git reference context, resolved profile, and executor fingerprint. Every required check must pass within one eligible suite run for a snapshot to receive the aggregate `required_checks_passed` state. Results from different runs are never assembled into a green aggregate.

Snapshot completion remains an evidence-recording operation rather than a CI gate. It always records a structurally valid snapshot, even when verification is absent, incomplete, or failed. EVE computes and stores one of five aggregate states: `not_configured`, `not_run`, `incomplete`, `failed`, or `required_checks_passed`.

Legacy validation is interpreted at read time as `legacy_unattributed`. New agent-supplied validation is explicitly `reported_by_agent`. Neither provenance contributes to `required_checks_passed`.

The UI describes the narrow fact EVE can prove: "Required checks passed." It does not claim that EVE verified the product. Verification details disclose the executed commit, policy, reference context, environment, run-record digest, check results, output truncation, policy changes, and the tamper-evident trust boundary.

## User Stories

1. As a snapshot reader, I want to distinguish EVE-executed checks from agent-reported claims, so that I know which evidence was independently captured by EVE.
2. As a snapshot reader, I want a green aggregate only when every required check passed in one run, so that partial evidence cannot appear complete.
3. As a snapshot reader, I want the exact commit shown with each suite run, so that I know which implementation state was checked.
4. As a snapshot reader, I want the resolved verification profile and required checks shown, so that I understand what the aggregate result covers.
5. As a snapshot reader, I want the executor environment shown, so that I do not treat results from materially different environments as interchangeable.
6. As a snapshot reader, I want policy changes and requirement reductions disclosed, so that a passing result cannot hide a weakened suite.
7. As a snapshot reader, I want evidence-file digest mismatches surfaced as tampering, so that modified run records never retain a green presentation.
8. As a snapshot reader, I want historical validation labeled `legacy_unattributed`, so that old agent-supplied claims are not retroactively presented as EVE-executed.
9. As a snapshot reader, I want the trust boundary explained in-product, so that I do not mistake repository-local, tamper-evident evidence for externally trusted attestation.
10. As a developer, I want repository maintainers to predeclare named checks, so that agents cannot choose arbitrary command strings at completion time.
11. As a developer, I want checks invoked by ID rather than command text, so that command injection cannot be introduced through a suite request.
12. As a developer, I want verification profiles resolved independently of descriptive snapshot type, so that labeling work as a feature or bug fix cannot lower the required checks.
13. As a developer, I want release-related Git context to select a stricter profile, so that release evidence can be stronger without taxing every ordinary change.
14. As a developer, I want to run a stricter suite than required, so that I can voluntarily escalate evidence without gaining a de-escalation path.
15. As a developer, I want suite execution to start explicitly, so that snapshot completion never launches a long-running or side-effectful command unexpectedly.
16. As a developer, I want a suite start operation to return immediately, so that long checks do not hold an MCP request open.
17. As a developer, I want to inspect suite progress, so that I can see queued, running, and terminal checks.
18. As a developer, I want to cancel a suite, so that hung or obsolete work can stop while still leaving a durable terminal record.
19. As a developer, I want timeouts and output limits enforced per check, so that a bad check cannot block completion indefinitely or produce unbounded evidence.
20. As a developer, I want output truncation marked with byte counts and digests, so that bounded evidence does not silently pretend to be complete output.
21. As a developer, I want configured environment inheritance to be explicit, so that checks do not accidentally receive or disclose every secret in the parent process.
22. As a developer, I want the latest eligible terminal run to be authoritative, so that a newer failed or cancelled rerun is not hidden behind an older pass.
23. As a developer, I want an in-progress rerun not to displace the latest terminal evidence, so that starting a run does not temporarily erase the current status.
24. As a developer, I want dirty-tree and mid-run drift detected, so that EVE never binds a passing result to code other than the recorded commit.
25. As a developer, I want snapshot creation to succeed when verification is missing or failed, so that product history is not lost merely because evidence is incomplete.
26. As a developer, I want `not_configured` distinguished from `not_run`, so that I know whether to configure policy or execute an existing suite.
27. As a developer, I want EVE to suggest likely checks without writing or approving them automatically, so that trustworthy onboarding remains low-friction and human-controlled.
28. As a maintainer, I want run records committed with snapshots, so that a fresh clone can substantiate the aggregate result.
29. As a maintainer, I want full logs kept out of Git by default, so that evidence does not create uncontrolled repository growth or secret exposure.
30. As a maintainer, I want terminal run records immutable, so that retries and reruns append history rather than rewriting it.
31. As a maintainer, I want only EVE-owned new run evidence exempted from implementation cleanliness checks, so that modifying committed evidence remains visible.
32. As a maintainer, I want configuration schema evolution to be explicit, so that existing repositories receive a controlled migration rather than a silent reinterpretation.
33. As an agent integrator, I want public MCP tools for starting, polling, and cancelling suite runs, so that agent clients can manage long-running verification predictably.
34. As a CLI user, I want one convenience command that starts and follows a suite, so that interactive use remains simple without weakening the asynchronous API.
35. As a future EVE operator, I want actor data represented as an unauthenticated claim, so that authenticated identity can be added later without rewriting the model or overstating today's guarantees.

## Implementation Decisions

### Trust boundary and terminology

- v1 is explicitly tamper-evident and not tamper-resistant.
- The aggregate passing label is `required_checks_passed`; presentation copy is "Required checks passed."
- `executed_by_eve` is per-check provenance, not a claim that EVE verified the product.
- Repository branch, tag, and configuration signals are server-derived and independent of snapshot payload fields. They are not described as outside a malicious agent's control.
- The verification detail surface permanently explains that the result reflects repository-configured checks executed locally and does not resist an adversarial actor with repository write access.

### Snapshot type and verification profile

- `snapshot.type` remains descriptive product metadata. It has no verification authority.
- `verification.profile` is resolved from committed profile rules and server-derived Git reference context.
- Descriptive snapshot type never selects, lowers, or overrides a profile.
- Profile rules support branch and tag glob matching plus exactly one default profile.
- The caller's branch and tags are captured before execution. Detached execution environments do not attempt to infer a branch from the checked-out commit.
- The captured reference context includes branch, matching tags, matched rule, and resolved profile.
- Snapshot completion recomputes reference context. A mismatch makes the prior run ineligible for the current aggregate.

### Repository configuration

- Verification policy extends the repository's existing `.eve/config.json` contract.
- The canonical configuration field spelling is camelCase, matching the current runtime configuration contract.
- Repository configuration advances to schema version 3. Readers provide a controlled migration from supported older shapes, including the legacy checked-in shape, and never silently treat an unknown version as current.
- Check IDs and suite IDs are stable repository-defined identifiers.
- Each check defines a non-empty argument vector, repository-relative working directory, positive timeout, accepted exit codes, output excerpt limit, static environment values, and an allowlist of parent environment variable names to inherit.
- Commands do not use a shell by default. Shell operators have no special meaning inside an argument vector.
- A check working directory cannot escape the repository root.
- Every suite member must reference a defined check. Empty suites, duplicate check IDs within a suite, missing profiles, multiple defaults, and rules referencing undefined profiles are configuration errors.
- An explicitly requested suite must contain every check required by the resolved profile. A smaller or incomparable suite is rejected. There is no de-escalation override.
- EVE may provide `checks suggest` functionality that inspects conventional project metadata and prints proposed configuration. It does not write or approve commands automatically.

The following shape encodes the configuration contract; it is a schema decision rather than executable configuration:

```json
{
  "schemaVersion": 3,
  "snapshotSchema": "0.2.0",
  "verification": {
    "checks": {
      "unit": {
        "argv": ["go", "test", "./..."],
        "workingDirectory": ".",
        "timeoutSeconds": 900,
        "successExitCodes": [0],
        "outputLimitBytes": 1000000,
        "inheritEnvironment": ["PATH", "HOME", "TMPDIR"],
        "environment": { "CI": "true" }
      }
    },
    "suites": { "change": ["unit"] },
    "profileRules": [{ "default": "change" }]
  }
}
```

### Execution environment and output safety

- EVE supplies a minimal platform environment required to launch a process. Additional parent variables are inherited only when named by the check definition.
- Static values in the committed `environment` object are configuration, not a secret store. Secrets may enter only through named inherited variables.
- Inherited environment values are never stored in run metadata. Metadata stores inherited variable names only.
- EVE redacts exact values of inherited variables from stdout and stderr before any excerpt, full local log, or digest is persisted.
- Output digests are computed over the redacted complete stream, not raw secret-bearing bytes.
- Stdout and stderr remain distinct.
- Canonical run evidence stores bounded excerpts, original redacted byte counts, truncation flags, and digests.
- Full redacted logs are optional local artifacts and are not required to substantiate the stored exit outcome. They are not committed by default and may be unavailable after cloning.
- Check execution uses the configured timeout and accepted exit-code set. A zero accepted exit code is a convention, not a hard-coded assumption.

### Asynchronous execution API

- `start_suite` accepts repository context, a commit SHA, an optional stricter suite ID, and an optional actor claim. It returns a unique run ID immediately.
- `get_suite_run` returns suite status, timing, resolution evidence, and per-check progress/results.
- `cancel_suite` requests termination and returns the latest run state. Cancellation is idempotent.
- The CLI `run-suite` command is convenience behavior built on start and poll operations. The MCP API never holds a call open for suite duration.
- Suite runs execute every configured check in order even after an ordinary check failure, producing a complete evidence set. Explicit cancellation or executor failure stops remaining work and assigns durable terminal outcomes.
- Cancellation terminates the process group, waits a bounded grace period, and force-terminates remaining processes when necessary.
- Milestone 1 serializes active suite execution per repository using a cross-process repository lock. It does not rely on authenticated caller identity.

### Suite and check state machines

- Suite status is one of `queued`, `running`, `completed`, `cancelled`, or `invalidated`.
- Check status is one of `pending`, `running`, `passed`, `failed`, `timed_out`, `execution_error`, `cancelled`, or `invalidated`.
- Cancellation writes `cancelled` for the in-flight check and every required check that did not start. Nothing disappears without a terminal outcome.
- Drift invalidates the suite and affected check attempts. Invalidated runs are retained for audit but are ineligible for aggregate selection.
- Run files may be updated atomically while queued or running. A terminal run record is immutable.
- Retrying or rerunning always creates a new run ID.
- An in-progress run does not displace the latest eligible terminal run.
- Once a newer non-invalidated run becomes terminal, it is authoritative for its matching resolution context, including when it failed or was cancelled. Older passing evidence remains visible but is not current.

### Commit and drift binding

- Suite execution requires the requested commit to equal repository HEAD at start.
- The implementation tree must satisfy the clean-tree precondition before execution, excluding only EVE-owned new run-evidence files that pass canonical validation.
- EVE captures HEAD, tracked tree state, configuration blob, and reference context before execution and rechecks them afterward.
- A HEAD change, tracked-file change, configuration mismatch, or reference-context mismatch invalidates the run.
- Gitignored output does not affect drift. New nonignored untracked output is reported and prevents an unqualified clean result unless the output is recognized as an EVE-owned evidence file.
- Dirty-tree agent claims may be recorded as `reported_by_agent`; they never contribute to `required_checks_passed`.
- Pre-commit verification is deferred. v1 only certifies committed content.

### Executor fingerprint and run selection

- Each run records an executor fingerprint containing EVE version, operating system, architecture, and detected relevant runtime/toolchain versions.
- Detection rules are deterministic and recorded. Failure to detect an optional runtime is represented explicitly rather than silently omitted.
- Executor fingerprints prevent cache or dedup equivalence across materially different environments.
- Snapshot completion does not match runs against the completion process's environment. It groups eligible terminal runs by their own complete resolution evidence and deterministically selects the most recently completed eligible run, breaking timestamp ties by run ID.
- The snapshot stores the selected executor fingerprint so readers know which environment produced the aggregate.
- v1 does not allow one suite aggregate to combine check attempts from different executor fingerprints or run IDs.

### Canonical run evidence

- Each suite run has one canonical record under `.eve/runs/`, named by run ID.
- A run record contains repository/commit identity, configuration blob hash, reference context, resolved profile, resolved suite/check definitions, executor fingerprint, actor claim, suite lifecycle, per-check outcomes, timestamps, exit codes, bounded redacted output evidence, and policy-resolution evidence.
- Actor attribution is stored as `actorClaim` with `provenance: "unauthenticated"`. It is never presented as verified identity.
- Run records use atomic file replacement while nonterminal and become immutable at a terminal state.
- Run history is append-only at the run level. New attempts create new files.
- Snapshot verification stores the selected run ID and the canonical digest of the selected terminal run record.
- Readers recalculate the run-record digest. A missing record or mismatch is an evidence-integrity failure and can never render as `required_checks_passed`.
- Cleanliness exemptions apply only to new, schema-valid EVE run records associated with known local run operations. Modifications to already committed run records are never exempt.
- The selected run record and snapshot are committed together in the normal EVE record commit so that a clone can substantiate the aggregate.

### Policy-change and downgrade disclosure

- Snapshot completion compares verification policy at the snapshot implementation base commit with policy at the implementation commit.
- The snapshot stores previous and current configuration blob hashes plus a structured difference for the resolved profile: checks added, checks removed, profile introduced/removed, and whether requirements were reduced.
- Policy change does not falsify the aggregate computed under current policy, but it changes presentation. `required_checks_passed` with a policy change must not render identically to a stable-policy pass.
- A requirements reduction receives a stronger warning than an additive change.
- When no prior policy exists, the snapshot records policy introduction rather than a downgrade.
- Policy comparison follows Git ancestry and implementation range, not wall-clock snapshot creation order.

### Snapshot schema and completion

- Verification was introduced in snapshot schema `0.2.0`; new snapshots use
  `0.3.0`, which retains this verification shape and adds Plan conformance.
- Snapshot verification is a server-computed object. Agent input cannot set aggregate state, selected run ID, executed provenance, policy diff, configuration hash, or executor fingerprint.
- The verification object contains aggregate state, profile, required check IDs, executed check IDs, selected run ID/digest when present, resolution evidence, policy-change evidence, and integrity state.
- Agent-provided validation remains a separate set of reported claims and requires `reported_by_agent` provenance. The server rejects attempts to label agent input `executed_by_eve`.
- `complete_snapshot` always writes a structurally valid snapshot regardless of verification outcome. Malformed input, unsupported schema/configuration, filesystem failure, and existing explicit dirty-tree policy remain real errors rather than fabricated verification states.
- Existing intentional dirty-snapshot behavior remains explicit. A dirty snapshot can never receive `required_checks_passed`.
- Snapshot aggregate state is one of:
  - `not_configured`: the resolved profile has no valid configured suite;
  - `not_run`: a suite is configured but no eligible terminal run exists;
  - `incomplete`: some required evidence is missing or stale and no known failure dominates;
  - `failed`: any required check in the authoritative evaluation has a terminal nonpassing outcome, including cancellation or timeout;
  - `required_checks_passed`: every required check passed in one eligible run.
- Aggregate precedence is `failed`, then `incomplete`, then `not_run`, then `required_checks_passed`. `not_configured` is resolved before run evaluation.
- `required_checks_passed` requires all required checks from the same selected run. No cross-run assembly is permitted.
- A stricter suite satisfies a profile only when the required check set is a subset of checks executed and passed within that same run.

The following shape encodes the snapshot verification contract:

```json
{
  "verification": {
    "status": "required_checks_passed",
    "profile": "change",
    "requiredChecks": ["unit"],
    "ranChecks": ["unit"],
    "selectedRunId": "run_123",
    "runRecordDigest": "sha256:...",
    "configBlobHash": "...",
    "executorFingerprint": { "eve": "0.3.0", "os": "darwin", "arch": "arm64", "toolchains": { "go": "1.25.0" } },
    "refContext": { "branch": "main", "matchingTags": [], "matchedRule": "default", "resolvedProfile": "change" },
    "policyChange": { "changed": false, "requirementsReduced": false, "addedChecks": [], "removedChecks": [] },
    "integrity": "matched"
  }
}
```

### Read-time legacy migration

- Snapshot schemas `0.1.0` and `0.2.0` remain readable.
- Historical validation without provenance is interpreted as `legacy_unattributed` at read time.
- Historical files are not rewritten merely to add provenance.
- Legacy `passed` values never contribute to the new aggregate or render as `executed_by_eve`.
- New-record parsing removes the automatic status inference that turns an unspecified result into `passed`.
- API and UI adapters expose legacy provenance consistently without mutating canonical history.

### UI and CLI presentation

- Aggregate presentation distinguishes Not configured, Not run, Incomplete, Failed, and Required checks passed.
- Failed is visually dominant when failure and missing evidence coexist.
- Agent-reported claims and legacy evidence occupy separate provenance sections; they are not alternative shades of the executed-check aggregate.
- The verification detail shows the selected run, commit, profile, required suite, check outcomes, exit codes, timestamps, executor fingerprint, configuration hash, reference context, output truncation, evidence integrity, and policy-change details.
- Missing or mismatched run evidence displays an integrity error and removes any green state.
- Policy changes add visible qualifying copy to the aggregate. A requirement reduction is never hidden in a tooltip alone.
- The trust-boundary disclosure is accessible on pointer, keyboard, and touch surfaces and is also available as inline explanatory text in verification details.
- `checks suggest` explains that suggestions are unapproved configuration proposals.

### Milestones

- Milestone 1 delivers configuration, profile resolution, reference context, asynchronous suite execution in the caller's clean checkout, cross-process repository serialization, evidence persistence, aggregation, schema evolution, migration, API/CLI/UI presentation, and acceptance coverage.
- Milestone 2 adds per-invocation Git worktree isolation, dependency/bootstrap strategy, cache reuse, cleanup after cancellation/crash, and concurrency without a shared HEAD.
- Worktree behavior must preserve the same evidence, profile, drift, and aggregation contracts. Isolation is an execution implementation change, not a new trust tier.

## Testing Decisions

- Tests assert public behavior rather than helper implementation. The primary seam is the existing runtime/MCP harness: call the public suite lifecycle and snapshot tools, then inspect API responses and durable canonical records.
- CLI tests cover `run-suite` polling behavior, cancellation signals, diagnostics, suggestions, and actionable error messages.
- UI tests cover aggregate labels, failed-state precedence, provenance separation, integrity mismatches, policy-change disclosure, and accessible trust-boundary copy.
- Schema tests cover supported configuration versions, controlled migration, invalid references, path escape rejection, unknown versions, historical `0.1.0`/`0.2.0` reads, and `0.3.0` canonicalization.
- Process tests use bounded fixture commands that deterministically pass, fail, time out, emit output, handle cancellation, modify tracked files, and create untracked output.
- Tests do not depend on network access or real package registries.
- Existing runtime API/MCP tests, snapshot canonicalization tests, dirty-tree tests, and UI verification component tests are the prior art to extend.

Acceptance behavior includes:

1. A repository with no configured suite produces `not_configured`; a configured suite with no eligible terminal run produces `not_run`.
2. A suite start returns a run ID immediately while polling reports queued/running state.
3. Every terminal path—pass, failure, timeout, execution error, cancellation, or invalidation—produces durable evidence.
4. Cancelling a run marks in-flight and unstarted required checks `cancelled` and makes the suite terminal.
5. One failed required check plus one missing/cancelled check produces aggregate `failed`, not `incomplete`.
6. Passing checks from different runs can never assemble `required_checks_passed`.
7. An in-progress rerun leaves the prior terminal state current; once the rerun terminates, its result becomes authoritative.
8. A newer cancelled or failed run displaces an older passing run for the same resolution context while preserving history.
9. An invalidated run is retained but never selected.
10. A stricter suite can satisfy a lighter profile only when every lighter-profile check passed in the same run.
11. An explicitly requested suite that omits a required check is rejected before execution.
12. Changing descriptive snapshot type does not affect profile resolution.
13. Branch/tag context is captured outside detached execution and a later mismatch prevents stale reuse.
14. A new commit after a run makes that run ineligible for the new snapshot commit.
15. Dirty state at suite start refuses EVE execution; dirty agent claims remain reported-only.
16. Tracked-file or HEAD drift during execution invalidates the run.
17. New nonignored untracked output is reported according to the drift contract; Gitignored output does not invalidate.
18. Output limits produce marked truncation with correct redacted byte counts and digests.
19. Secret values inherited through configured environment names do not appear in excerpts, local logs, or persisted digests' source bytes.
20. A fresh clone can substantiate a passing snapshot from its committed snapshot and run record without local cache data.
21. Changing a terminal run record causes a digest mismatch and removes the passing presentation.
22. Modifying a committed run record remains visible to cleanliness checks; only valid new EVE evidence receives a narrow exemption.
23. A policy change in the implementation range is disclosed with structured added/removed checks.
24. A policy reduction never renders identically to a stable-policy pass.
25. Snapshot completion writes records for `not_configured`, `not_run`, `incomplete`, `failed`, and `required_checks_passed` when all structural preconditions hold.
26. Malformed snapshot input and filesystem failure remain real errors rather than verification outcomes.
27. Historical snapshots read as `legacy_unattributed` without changing their files.
28. New agent-reported validation cannot set executed provenance and unspecified status never infers `passed`.
29. Runs from different executor fingerprints are never deduplicated or combined.
30. Snapshot completion deterministically selects one eligible terminal run and stores that run's fingerprint and digest.
31. Concurrent Milestone 1 starts in the same repository serialize through a cross-process lock without corrupting run records.
32. Configuration suggestion prints proposals but does not modify repository policy.

## Out of Scope

- Defending against a deliberately deceptive agent with repository, Git, and shell access.
- Signed verification policy, protected-branch attestation, human approval gates, or other external trust anchors.
- Pre-commit or dirty-working-tree EVE execution.
- Treating agent-reported or legacy validation as EVE-executed evidence.
- Shell command strings supplied through suite-execution APIs.
- Hidden suite execution from snapshot completion.
- Combining check attempts across suite runs or executor fingerprints.
- Cross-machine run deduplication or a shared execution service.
- Authenticated human or agent identity.
- Cross-agent or cross-human invalidation notifications.
- Hosted team execution infrastructure or a hosted snapshot viewer.
- Git worktree isolation in Milestone 1.
- Automatic writing or approval of suggested repository checks.
- Committing unbounded full command logs.

## Further Notes

- This feature is evidence capture, not a replacement for CI or release protection. External systems may later decide that `required_checks_passed` is a merge or release requirement, but EVE v1 records the state rather than enforcing deployment policy.
- The repository remains Git-native and local-first. Canonical snapshot and selected run evidence travel through Git; execution remains local.
- Configuration, run-record, and snapshot schemas should be documented alongside their machine-readable schemas before release.
- The implementation should preserve the existing two-commit EVE workflow: implementation first, then the snapshot and selected run evidence together in the EVE record commit.
- The first implementation ticket should prove the full tracer path with one configured check: configuration resolution, asynchronous execution, durable run evidence, snapshot aggregation, clone-readable evidence, and UI provenance. Additional outcomes, policy diffs, cancellation, and Milestone 2 isolation should layer onto that vertical slice.
