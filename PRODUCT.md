# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

EVE is used by software developers working with AI coding agents. The primary user reviews an agent's proposed work, controls its allowed scope and verification requirements, and preserves a trustworthy product history without leaving the development workflow.

## Product Purpose

EVE makes consequential agent work explicit before implementation and records verified product changes afterward. Success means a developer can understand what an agent intends to change, approve or redirect it at the right moment, and later recover the decisions, evidence, and resulting Git state.

## Positioning

EVE joins a human-approved, revisioned implementation plan to a verified product Snapshot. Approval is not a transient chat gesture: it becomes durable repository evidence that can be compared with the completed change.

## Operating Context

EVE runs locally alongside Git repositories and AI coding agents. Agents use its MCP tools; developers use the CLI, macOS approval app, or embedded web dashboard. Repositories keep shared product history under `.eve/`, while runtime coordination and device credentials remain private user state.

## Capabilities and Constraints

- Plans have durable requests, revisions, explicit scope, milestones, suites, and optimistic revision checks.
- A human can approve a proposal unchanged, approve an edited human revision, or reject it with feedback.
- Completed work is recorded as a Snapshot tied to Git state and verification evidence.
- The runtime binds only to localhost. Optional phone access is private to Tailscale Serve and the configured Tailscale identity.
- The first phone host is macOS; the phone client is a Home Screen PWA for iOS and iPadOS 16.4 or newer.
- EVE has no hosted relay, public ingress, analytics service, or silent background notification channel.
- A sleeping or offline host cannot originate a new phone notification until it wakes and reconnects.

## Brand Commitments

The product name is EVE. Product language is direct, calm, and specific about state, responsibility, and recovery. Existing EVE wordmark, neutral surfaces, slate typography, indigo action color, and restrained status colors remain the visual authority for product extensions.

## Evidence on Hand

The repository contains the working Go CLI/runtime, React dashboard, plan and Snapshot schemas, tests, the EVE wordmark at `ui/public/eve.svg`, and durable example history under `.eve/`. Future work must not fabricate customer claims, usage metrics, or hosted-service guarantees.

## Product Principles

- Ask for human judgment at consequential boundaries.
- Preserve decisions as inspectable repository evidence.
- Keep local work private by default and make remote exposure explicit.
- Prefer recoverable, revision-safe transitions over optimistic shortcuts.
- Explain failures with a concrete next action.

## Accessibility & Inclusion

Interactive product surfaces must remain keyboard operable, screen-reader labelled, responsive, and usable with platform text sizing and reduced motion. Touch interfaces use at least 44-point targets and account for iPhone safe areas.
