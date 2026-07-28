## EVE Product History

- Before non-trivial product code changes, call `declare_plan` and wait for a locked Plan.
- Reuse the same `planRequestId` after timeout, cancellation, or restart; recover with `get_plan_request` when needed.
- After implementation and verification, call `complete_snapshot` with the locked Plan ID/revision, or `skip_snapshot` if no Snapshot is warranted.
