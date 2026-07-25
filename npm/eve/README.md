# eve installer

Install the eve product-history CLI for the current user:

```sh
npx @nhestrompia/eve@latest install
```

The installer downloads the matching eve binary from the corresponding GitHub
Release, verifies it against `SHA256SUMS`, installs it under the user's home
directory, confirms the binary version, and configures Codex, Claude Code, and
opencode MCP settings with the absolute binary path.

On macOS, the installer also downloads the checksummed `eve-macos-app.zip`
release asset and installs the native approval app at `~/Applications/eve.app`
unless `EVE_APP_INSTALL_PATH` is set. On Linux and Windows, app installation is
skipped because the approval app is macOS-only for now.

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

eve installs to `~/.local/bin/eve` on macOS and Linux and to
`%LOCALAPPDATA%\EVE\bin\eve.exe` on Windows unless `--install-dir` or
`EVE_INSTALL_DIR` is provided. The installer prints PATH guidance when needed.
