# Codex Working Rule

After completing and verifying a product change in this repository, commit the work twice:

1. Commit the implementation changes to Git.
2. Record the product change with EVE using `eve add` and `eve commit`, then commit the generated `.eve/` record to Git.

Run the relevant verification before creating the EVE record, and include that verification in the Evolution.

## EVE Product History

- Before non-trivial product code changes, call `declare_plan` and wait for a locked Plan.
- Reuse the same `planRequestId` after timeout, cancellation, or restart; recover with `get_plan_request` when needed.
- After implementation and verification, call `complete_snapshot` with the locked Plan ID/revision, or `skip_snapshot` if no Snapshot is warranted.
