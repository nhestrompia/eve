package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var supportedMCPClients = map[string]struct{}{
	"codex":    {},
	"claude":   {},
	"opencode": {},
}

func runInstallMCP(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("install-mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)
	clientsValue := fs.String("clients", "codex,claude,opencode", "comma-separated MCP clients to configure")
	eveBin := fs.String("eve-bin", "", "absolute path to the installed eve binary")
	cwd := fs.String("cwd", "", "repository working directory to pass to eve mcp-stdio")
	home := fs.String("home", "", "home directory containing client config files")
	install := fs.Bool("install", false, "run go install ./cmd/eve before writing MCP config")
	dryRun := fs.Bool("dry-run", false, "print planned config changes without writing files")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve install-mcp takes no positional arguments")
		return 2
	}

	configHome, err := installMCPHome(*home)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if *install {
		if err := goInstallEve(); err != nil {
			fmt.Fprintf(stderr, "install eve: %v\n", err)
			return 1
		}
	}
	binValue := strings.TrimSpace(*eveBin)
	if *install && binValue == "" && strings.TrimSpace(os.Getenv("EVE_BIN")) == "" {
		if installed, err := installedGoBinaryPath("eve"); err == nil {
			binValue = installed
		}
	}
	bin, err := resolveEveBinary(binValue)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	clients, err := parseMCPClients(*clientsValue)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}

	installer := mcpConfigInstaller{
		home:   configHome,
		eveBin: bin,
		cwd:    strings.TrimSpace(*cwd),
		dryRun: *dryRun,
	}
	for _, client := range clients {
		var path string
		switch client {
		case "codex":
			path, err = installer.installCodex()
		case "claude":
			path, err = installer.installClaude()
		case "opencode":
			path, err = installer.installOpenCode()
		}
		if err != nil {
			fmt.Fprintf(stderr, "%s: %v\n", client, err)
			return 1
		}
		action := "Configured"
		if *dryRun {
			action = "Would configure"
		}
		fmt.Fprintf(stdout, "%s %s MCP server in %s\n", action, client, path)
	}
	if *install {
		fmt.Fprintf(stdout, "Installed eve binary at %s\n", bin)
	}
	return 0
}

func installMCPHome(value string) (string, error) {
	if strings.TrimSpace(value) != "" {
		return filepath.Abs(strings.TrimSpace(value))
	}
	return os.UserHomeDir()
}

func goInstallEve() error {
	cmd := exec.Command("go", "install", "./cmd/eve")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go install ./cmd/eve: %w\n%s", err, output)
	}
	return nil
}

func resolveEveBinary(value string) (string, error) {
	if strings.TrimSpace(value) != "" {
		return requireAbsExecutable(strings.TrimSpace(value))
	}
	if env := strings.TrimSpace(os.Getenv("EVE_BIN")); env != "" {
		return requireAbsExecutable(env)
	}
	if path, err := exec.LookPath("eve"); err == nil {
		return requireAbsExecutable(path)
	}
	if path, err := installedGoBinaryPath("eve"); err == nil {
		return requireAbsExecutable(path)
	}
	return "", errors.New("could not find an installed eve binary; run go install ./cmd/eve first or pass --eve-bin /absolute/path/to/eve")
}

func requireAbsExecutable(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("eve binary %q is not accessible: %w", abs, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("eve binary %q is a directory", abs)
	}
	return abs, nil
}

func installedGoBinaryPath(name string) (string, error) {
	output, err := exec.Command("go", "env", "GOBIN", "GOPATH").Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	gobin := ""
	gopath := ""
	if len(lines) > 0 {
		gobin = strings.TrimSpace(lines[0])
	}
	if len(lines) > 1 {
		gopath = strings.TrimSpace(lines[1])
	}
	if gobin != "" {
		return filepath.Join(gobin, name), nil
	}
	if gopath != "" {
		return filepath.Join(gopath, "bin", name), nil
	}
	return "", errors.New("go env did not return GOBIN or GOPATH")
}

func parseMCPClients(value string) ([]string, error) {
	seen := map[string]bool{}
	var clients []string
	for _, part := range strings.Split(value, ",") {
		client := strings.ToLower(strings.TrimSpace(part))
		if client == "" {
			continue
		}
		if client == "all" {
			client = "codex,claude,opencode"
			for _, nested := range strings.Split(client, ",") {
				if !seen[nested] {
					seen[nested] = true
					clients = append(clients, nested)
				}
			}
			continue
		}
		if _, ok := supportedMCPClients[client]; !ok {
			return nil, fmt.Errorf("unsupported MCP client %q; supported clients: codex, claude, opencode", client)
		}
		if !seen[client] {
			seen[client] = true
			clients = append(clients, client)
		}
	}
	if len(clients) == 0 {
		return nil, errors.New("at least one MCP client is required")
	}
	return clients, nil
}

type mcpConfigInstaller struct {
	home   string
	eveBin string
	cwd    string
	dryRun bool
}

func (installer mcpConfigInstaller) installCodex() (string, error) {
	path := filepath.Join(installer.home, ".codex", "config.toml")
	args := []string{"mcp-stdio"}
	if installer.cwd != "" {
		args = append(args, "--cwd", installer.cwd)
	}
	block := []string{
		"[mcp_servers.eve]",
		"command = " + tomlQuote(installer.eveBin),
		"args = " + tomlArray(args),
		"startup_timeout_sec = 20",
		"tool_timeout_sec = 120",
		"",
	}
	if installer.dryRun {
		return path, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return path, writeFileEnsuringDir(path, []byte(strings.Join(block, "\n")))
	}
	if err != nil {
		return path, err
	}
	updated := upsertTOMLTable(string(data), "[mcp_servers.eve]", strings.Join(block, "\n"))
	return path, os.WriteFile(path, []byte(updated), 0o644)
}

func (installer mcpConfigInstaller) installClaude() (string, error) {
	path := filepath.Join(installer.home, ".claude.json")
	args := []string{"mcp-stdio"}
	if installer.cwd != "" {
		args = append(args, "--cwd", installer.cwd)
	} else {
		args = append(args, "--cwd", "${CLAUDE_PROJECT_DIR:-.}")
	}
	server := map[string]any{
		"type":    "stdio",
		"command": installer.eveBin,
		"args":    args,
		"env":     map[string]any{},
	}
	return path, installer.upsertJSON(path, []string{"mcpServers", "eve"}, server)
}

func (installer mcpConfigInstaller) installOpenCode() (string, error) {
	path := filepath.Join(installer.home, ".config", "opencode", "opencode.json")
	server := map[string]any{
		"type":    "local",
		"command": []string{installer.eveBin, "mcp-stdio"},
		"enabled": true,
	}
	if installer.cwd != "" {
		server["command"] = []string{installer.eveBin, "mcp-stdio", "--cwd", installer.cwd}
	}
	return path, installer.upsertJSON(path, []string{"mcp", "eve"}, server)
}

func (installer mcpConfigInstaller) upsertJSON(path string, keys []string, value any) error {
	if installer.dryRun {
		return nil
	}
	root := map[string]any{}
	data, err := os.ReadFile(path)
	if err == nil && len(bytesTrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &root); err != nil {
			return err
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	current := root
	for _, key := range keys[:len(keys)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[key] = next
		}
		current = next
	}
	current[keys[len(keys)-1]] = value
	data, err = json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return writeFileEnsuringDir(path, append(data, '\n'))
}

func writeFileEnsuringDir(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func upsertTOMLTable(data string, table string, block string) string {
	lines := strings.Split(data, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == table {
			start = i
			break
		}
	}
	if start == -1 {
		trimmed := strings.TrimRight(data, "\n")
		if trimmed == "" {
			return block
		}
		return trimmed + "\n\n" + block
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			end = i
			break
		}
	}
	replacement := strings.Split(strings.TrimRight(block, "\n"), "\n")
	updated := append([]string{}, lines[:start]...)
	updated = append(updated, replacement...)
	updated = append(updated, lines[end:]...)
	return strings.TrimRight(strings.Join(updated, "\n"), "\n") + "\n"
}

func tomlQuote(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + replacer.Replace(value) + `"`
}

func tomlArray(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, tomlQuote(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func bytesTrimSpace(data []byte) []byte {
	return []byte(strings.TrimSpace(string(data)))
}
