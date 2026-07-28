<!-- eve:instructions:start version="3" -->

## EVE Product History

This repository uses EVE to approve implementation plans and record completed
product work.

Create an EVE Plan before making non-trivial, Snapshot-worthy code changes:

1. Call the EVE `declare_plan` tool with a caller-stable
   `planRequestId`, goal, acceptance criteria, allowed path globs,
   milestones, and verification suite when applicable.
2. Wait for EVE to return a locked Plan ID and revision before modifying code.
3. If the call times out, is cancelled, or the agent restarts, recover with
   `get_plan_request` or call `declare_plan` again using
   the same request ID. Never replace a pending request with a new ID just to
   avoid waiting.
4. After implementation and verification, pass the locked Plan ID and revision
   to `complete_snapshot`.

When you complete a coherent unit of product work, call the EVE
`complete_snapshot` tool before ending the task.

Create a Snapshot for work such as:

- A feature or user-visible improvement
- A bug fix
- A meaningful refactor
- A migration
- An experiment
- A release-related change

Do not create a Snapshot for trivial work such as:

- Formatting-only changes
- A variable rename with no behavior change
- Lint-only fixes
- Temporary debugging changes
- Work that was started but not completed

When no Snapshot is warranted, call `skip_snapshot` and include a short reason.

The Snapshot should reflect the completed task and include the relevant
behavior changes, validation, commits, screenshots, decisions, risks,
relationships, and session references when available.

<!-- eve:instructions:end -->
