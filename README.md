# eveEVE

eve records completed product snapshots alongside Git implementation history.

Git records implementation checkpoints. eve records the completed product unit:
one feature, bug fix, experiment, refactor, or release.

## Library

```go
snapshot, err := eve.ParseSnapshot(data)
if err != nil {
    return err
}

if err := eve.ValidateSnapshot(snapshot); err != nil {
    return err
}

canonical, err := eve.CanonicalSnapshotJSON(snapshot)
```

Public package APIs:

- `ParseSnapshot([]byte) (*Snapshot, error)`
- `ValidateSnapshot(*Snapshot) error`
- `CanonicalSnapshotJSON(*Snapshot) ([]byte, error)`
- `LoadSnapshotFile(path string) (*Snapshot, error)`

## CLI

```sh
go run ./cmd/eve init
go run ./cmd/eve dev
go run ./cmd/eve snapshot snap_123
go run ./cmd/eve checkout snap_123
go run ./cmd/eve checkout --force snap_123
go run ./cmd/eve validate .eve/snapshots/snap_123.json
go run ./cmd/eve canonicalize .eve/snapshots/snap_123.json
```

`eve checkout` refuses to run when the Git working tree has uncommitted changes
unless `--force` is supplied.

## Runtime

`eve dev` starts the local Snapshot runtime:

```text
EVE Runtime
├── Web UI
├── Local API
├── MCP server
├── Repo watcher/cache refresh
└── Local index cache
```

The runtime binds to localhost only. The local cache under `.eve/cache/` is
rebuildable and not canonical.

Canonical product history lives in:

```text
.eve/snapshots/*.json
```

Artifacts are referenced from Snapshot JSON and stored as files or external
URIs:

```text
.eve/artifacts/<snapshot-id>/...
```

## Local API

```text
GET  /api/repos
GET  /api/repos/{repoId}
GET  /api/repos/{repoId}/snapshots
GET  /api/repos/{repoId}/snapshots/{snapshotId}
POST /api/repos/{repoId}/snapshots/{snapshotId}/checkout
POST /mcp
```

## MCP

Resources:

```text
eve://repos
eve://repos/{repoId}
eve://repos/{repoId}/snapshots
eve://repos/{repoId}/snapshots/{snapshotId}
```

Tools:

- `list_repos`
- `list_snapshots`
- `get_snapshot`
- `complete_snapshot`
- `skip_snapshot`
- `checkout_snapshot`

`complete_snapshot` accepts product meaning from agents. EVE derives Git facts
from the repository at completion time: branch, Git state, commits, and dirty
status.

## Storage

Initialized structure:

```text
.eve/
  config.json
  snapshots/
  artifacts/
  cache/
```

EVE no longer reads or writes `.eve/evolutions/` in the Snapshot-first runtime.
