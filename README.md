# EVE

EVE records product evolution alongside Git implementation history.

This repository contains the RFC-0001 reference shape for an Evolution, a
JSON Schema 2020-12 contract, a Go library, and a small CLI.

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

## CLI

```sh
go run ./cmd/eve validate evolution.json
go run ./cmd/eve canonicalize evolution.json
go run ./cmd/eve version
```

Exit codes:

- `0`: success
- `1`: validation failure
- `2`: usage or file I/O error

## Scope

EVE does not specify storage, synchronization, auth, review policy,
verification policy, deployment, or UI. Git remains the source of truth for
implementation.
