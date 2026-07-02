# EVE

EVE records product evolution alongside Git implementation history.

Git stages code. EVE stages product meaning.

This repository contains the RFC-0001 reference shape for an Evolution, a JSON
Schema 2020-12 contract, a Go library, and a CLI.

## Library

```go
evolution, err := eve.Parse(data)
if err != nil {
    return err
}

if err := eve.Validate(evolution); err != nil {
    return err
}

canonical, err := eve.CanonicalJSON(evolution)
```

Public package APIs:

- `Parse([]byte) (*Evolution, error)`
- `Validate(*Evolution) error`
- `CanonicalJSON(*Evolution) ([]byte, error)`
- `LoadFile(path string) (*Evolution, error)`

## CLI Workflow

```sh
go run ./cmd/eve init
git commit -m "Implement Enterprise SSO"
go run ./cmd/eve add \
  --title "Enterprise SSO" \
  --type feature \
  --behavior-added "Organizations can log in via Okta" \
  --outcome "Organizations can authenticate with Okta." \
  --verification "passed: go test ./..." \
  --session codex:session_912 \
  --session-source transcript.jsonl \
  --implementation HEAD \
  --sanitize
go run ./cmd/eve status
go run ./cmd/eve commit
git add .eve/
git commit -m "EV-001 Enterprise SSO"
```

Manual staging commands:

```sh
go run ./cmd/eve add title "Enterprise SSO" --type feature
go run ./cmd/eve add behavior --added "Organizations can log in via Okta"
go run ./cmd/eve add verification --status passed --reference "go test ./..."
go run ./cmd/eve add session codex:session_912 --source transcript.jsonl --sanitize
go run ./cmd/eve add outcome "Organizations can authenticate with Okta."
go run ./cmd/eve add implementation --snapshot HEAD --commit HEAD --repository eve --status merged
```

Read committed Evolutions:

```sh
go run ./cmd/eve list
go run ./cmd/eve show EV-001
go run ./cmd/eve timeline EV-001
go run ./cmd/eve graph
go run ./cmd/eve search okta
```

Navigate product snapshots:

```sh
go run ./cmd/eve snapshot EV-001
go run ./cmd/eve checkout EV-001
```

`eve checkout` refuses to run when the Git working tree has uncommitted changes.

`implementation.commits` records commits contributed by an Evolution.
`implementation.snapshot` records the resolved Git commit SHA that represents
the repository state after the Evolution became true. `eve checkout` uses
`implementation.snapshot`.

Protocol tools:

```sh
go run ./cmd/eve validate evolution.json
go run ./cmd/eve canonicalize evolution.json
go run ./cmd/eve version
```

## Storage

Initialized structure:

```text
.eve/
  config.json
  staged/
  evolutions/
  sessions/
```

Committed product history:

```text
.eve/config.json
.eve/evolutions/EV-001.json
.eve/sessions/EV-001/
  codex-session-912.md
  codex-session-912.jsonl
  manifest.json
```

Local/generated state:

```text
.eve/staged/
.eve/public/
```

## Scope

EVE does not specify synchronization, auth, review policy, verification policy,
deployment, or UI. Git remains the source of truth for implementation.
