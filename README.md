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
POST /api/repos/{repoId}/open-editor
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

### Use EVE MCP from Codex

Codex can start EVE as a local stdio MCP server. Add this to a trusted
project-scoped `.codex/config.toml` or to `~/.codex/config.toml`:

```toml
[mcp_servers.eve]
command = "go"
args = ["run", "./cmd/eve", "mcp-stdio"]
cwd = "/Users/nhestrompia/Documents/eve"
startup_timeout_sec = 20
tool_timeout_sec = 120
```

Or add it with the Codex CLI:

```sh
codex mcp add eve -- go run ./cmd/eve mcp-stdio
```

Use `/mcp` inside Codex to confirm the server is connected.

If `eve dev` is already running, Codex can also connect over Streamable HTTP:

```toml
[mcp_servers.eve]
url = "http://localhost:4317/mcp"
startup_timeout_sec = 20
tool_timeout_sec = 120
```

### Use EVE MCP from Claude Code

For a personal/project-local setup, add the stdio server:

```sh
claude mcp add --transport stdio eve -- go run ./cmd/eve mcp-stdio
```

For a team-shared project config, create `.mcp.json` in the repository root:

```json
{
  "mcpServers": {
    "eve": {
      "command": "go",
      "args": ["run", "./cmd/eve", "mcp-stdio"]
    }
  }
}
```

Claude Code prompts for approval before using project-scoped `.mcp.json`
servers. Use `/mcp` inside Claude Code or `claude mcp list` to check status.

If `eve dev` is already running, Claude Code can connect over HTTP:

```sh
claude mcp add --transport http eve http://localhost:4317/mcp
```

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
