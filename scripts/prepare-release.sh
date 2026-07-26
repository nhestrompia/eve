#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="${1:-}"

if [[ -z "$VERSION" ]]; then
  printf 'usage: %s <version>\n' "${0##*/}" >&2
  exit 2
fi

VERSION="${VERSION#v}"
if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  printf 'error: version must be semver, got %q\n' "$VERSION" >&2
  exit 2
fi

RELEASE_DATE="${RELEASE_DATE:-$(date +%Y-%m-%d)}"

cd "$ROOT"

npm --prefix npm/eve version "$VERSION" --no-git-tag-version --allow-same-version >/dev/null

node - "$VERSION" "$RELEASE_DATE" <<'NODE'
const fs = require("fs");

const [version, releaseDate] = process.argv.slice(2);
const changelogPath = "CHANGELOG.md";
const changelog = fs.readFileSync(changelogPath, "utf8");

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

if (new RegExp(`^## ${escapeRegExp(version)}(?:\\s|$)`, "m").test(changelog)) {
  throw new Error(`CHANGELOG.md already contains a ${version} release section`);
}

const heading = "## Unreleased";
const headingStart = changelog.indexOf(heading);
if (headingStart === -1) {
  throw new Error("CHANGELOG.md is missing an Unreleased section");
}

const bodyStart = headingStart + heading.length;
const nextHeadingStart = changelog.indexOf("\n## ", bodyStart);
const beforeHeading = changelog.slice(0, headingStart);
const unreleasedBody =
  nextHeadingStart === -1 ? changelog.slice(bodyStart) : changelog.slice(bodyStart, nextHeadingStart);
const afterUnreleased = nextHeadingStart === -1 ? "" : changelog.slice(nextHeadingStart);
const releaseNotes = unreleasedBody.trim();

if (!releaseNotes) {
  throw new Error("CHANGELOG.md Unreleased section has no release notes");
}

const nextChangelog = `${beforeHeading}${heading}\n\n## ${version} - ${releaseDate}\n\n${releaseNotes}\n${afterUnreleased}`;
fs.writeFileSync(changelogPath, nextChangelog.endsWith("\n") ? nextChangelog : `${nextChangelog}\n`);
NODE

printf 'Prepared eve release v%s for %s.\n' "$VERSION" "$RELEASE_DATE"
printf 'Next release steps: commit these changes, record the release with EVE, commit the .eve record, tag v%s, and push the tag.\n' "$VERSION"
