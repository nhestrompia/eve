# Changelog

All notable changes to eve will be documented here.

This project uses Git tags for releases.

## Unreleased

## 0.7.0 - 2026-07-28

- Reduced idle CPU usage by replacing stale UI polling and runtime fan-out loops
  with event-driven refreshes.
- Added context-first pull request review views for inspecting EVE-linked product
  history during PR review.
- Fixed recorded artifact rendering and pull request Snapshot context linkage.
- Kept history-only pull request Snapshots current after their branches merge.

## 0.6.0 - 2026-07-28

- Kept the local dashboard live after product-history changes by refreshing UI
  queries when Plans and Snapshots change.
- Fixed pending Snapshot resolution for branches that were merged before the
  Snapshot was recorded.
- Required agent workflows to declare Plans before snapshot-worthy product
  changes, and updated generated agent instructions with the shorter release
  guidance.
- Renamed the local web app title to `eve UI`.

## 0.5.0 - 2026-07-27

- Fixed Snapshots and Plans dashboard filters so page-local search, status,
  agent, repository, date, filter, sort, pagination, and snapshot graph
  navigation controls update the visible rows.

## 0.4.0 - 2026-07-26

- Added `scripts/prepare-release.sh` for preparing npm package metadata and
  changelog release sections locally.

## 0.3.0 - 2026-07-26

- Added `eve` as the default local launcher for the UI, API, HTTP MCP endpoint,
  browser, and installed macOS approval app.
- Added `eve kill` to stop the local runtime for the configured localhost port.
- Added Planned Snapshot documentation for Plan approval, MCP tools, local API
  endpoints, CLI flags, and Snapshot conformance records.
- Added `scripts/local-test.sh` for running repository verification locally
  without documentation-site checks.
- Fixed release builds so tagged deployments inject the tag version into the
  `eve version` output used by npm installer verification.
- Added the `npx @nhestrompia/eve@latest install` flow for checksummed,
  platform-specific global CLI installation and MCP setup.
- Added automatic `AGENTS.md` and `CLAUDE.md` instruction bootstrap, managed
  instruction commands, and repository diagnostics through `eve doctor`.
- Added a Fumadocs-powered documentation website.
- Added repository metadata for licensing, contribution guidance, security reporting, CI, dependency checks, and releases.
