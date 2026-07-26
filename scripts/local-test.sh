#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

TMPDIR_VALUE="${TMPDIR:-/tmp}"

run() {
  printf '\n==> %s\n' "$*"
  "$@"
}

run go test ./...
run env EVE_EXPECT_CLI_VERSION=9.9.9-local go test -run TestRunVersion ./cmd/eve -ldflags="-X github.com/nhestrompia/eve.CLIVersion=9.9.9-local"

run npm --prefix npm/eve test
run npm --prefix npm/eve run pack:check

run npm --prefix ui ci
run npm --prefix ui test
run npm --prefix ui run build

if [[ "${SKIP_MACOS_APPROVAL:-}" == "1" ]]; then
  printf '\n==> Skipping macOS approval app because SKIP_MACOS_APPROVAL=1\n'
elif [[ "$(uname -s)" == "Darwin" ]]; then
  if ! command -v xcodebuild >/dev/null 2>&1; then
    printf '\nerror: xcodebuild is required for local macOS approval app verification\n' >&2
    exit 1
  fi
  run xcodebuild -project macos/EVEApproval/EVEApproval.xcodeproj -scheme EVEApproval -destination 'platform=macOS' -derivedDataPath "$TMPDIR_VALUE/eveapproval-derived" test CODE_SIGNING_ALLOWED=NO
  run xcodebuild -project macos/EVEApproval/EVEApproval.xcodeproj -scheme EVEApproval -destination 'platform=macOS' -configuration Release -derivedDataPath "$TMPDIR_VALUE/eveapproval-derived" build CODE_SIGNING_ALLOWED=NO
else
  printf '\n==> Skipping macOS approval app on %s\n' "$(uname -s)"
fi
