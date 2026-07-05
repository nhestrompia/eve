# Security Policy

## Supported Versions

eve is pre-1.0. Security fixes target the latest version on `main` unless a release branch is explicitly maintained.

## Reporting a Vulnerability

Please report security issues privately to the repository owner instead of opening a public issue.

Include:

- A description of the issue
- Reproduction steps or proof of concept
- Affected commands, APIs, or files
- Any known impact or workaround

Do not include live secrets or private data in reports.

## Local Runtime Assumption

eve currently assumes trusted local usage. Keep `eve dev` bound to localhost and only connect trusted MCP clients.
