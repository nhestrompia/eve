# EVE installer

Install the EVE product-history CLI for the current user:

```sh
npx @nhestrompia/eve@latest install
```

The installer downloads the matching EVE binary from the corresponding GitHub
Release, verifies it against `SHA256SUMS`, installs it under the user's home
directory, confirms the binary version, and configures Codex, Claude Code, and
opencode MCP settings with the absolute binary path.

Options:

```sh
npx @nhestrompia/eve@latest install --clients codex,claude
npx @nhestrompia/eve@latest install --no-mcp
npx @nhestrompia/eve@latest install --install-dir /custom/bin
```

After installation, open a Git repository and run:

```sh
eve init
eve doctor
```

EVE installs to `~/.local/bin/eve` on macOS and Linux and to
`%LOCALAPPDATA%\EVE\bin\eve.exe` on Windows unless `--install-dir` or
`EVE_INSTALL_DIR` is provided. The installer prints PATH guidance when needed.
