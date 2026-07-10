package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nhestrompia/eve"
)

func runDoctor(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(stderr, "eve doctor takes no arguments")
		return 2
	}
	repo, err := resolveRepo(repoRequest{})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	fmt.Fprintln(stdout, "EVE doctor")
	fatal := false

	fmt.Fprintln(stdout, "\nRepository")
	if err := validateDoctorRepository(repo); err != nil {
		fatal = true
		fmt.Fprintf(stdout, "  ✗ %v\n", err)
	} else {
		fmt.Fprintln(stdout, "  ✓ .eve directory exists")
		fmt.Fprintln(stdout, "  ✓ Configuration is valid")
		fmt.Fprintln(stdout, "  ✓ Snapshot storage directories exist")
	}

	fmt.Fprintln(stdout, "\nMCP")
	home, homeErr := os.UserHomeDir()
	configured := []string(nil)
	var configWarnings []string
	if homeErr != nil {
		configWarnings = append(configWarnings, homeErr.Error())
	} else {
		configured, configWarnings = detectMCPConfigurations(home)
	}
	if len(configured) == 0 {
		fmt.Fprintln(stdout, "  ⚠ No EVE configuration found in the default Codex, Claude Code, or opencode config files")
		fmt.Fprintln(stdout, "    Run `eve install-mcp`; manually configured clients may still work.")
	} else {
		fmt.Fprintf(stdout, "  ✓ EVE server configured for %s\n", strings.Join(configured, ", "))
	}
	for _, warning := range configWarnings {
		fmt.Fprintf(stdout, "  ⚠ %s\n", warning)
	}
	for _, tool := range []string{"complete_snapshot", "skip_snapshot"} {
		if mcpToolAvailable(tool) {
			fmt.Fprintf(stdout, "  ✓ %s available\n", tool)
		} else {
			fatal = true
			fmt.Fprintf(stdout, "  ✗ %s is missing from the EVE MCP server\n", tool)
		}
	}

	fmt.Fprintln(stdout, "\nAgent instructions")
	for _, target := range instructionTargets {
		inspection, err := inspectInstructionTarget(repo, target)
		if err != nil {
			fatal = true
			fmt.Fprintf(stdout, "  ✗ %s: %v\n", target.Filename, err)
			continue
		}
		if inspection.State == instructionCurrent {
			fmt.Fprintf(stdout, "  ✓ %s contains current EVE instructions\n", target.Filename)
			continue
		}
		fatal = true
		switch inspection.State {
		case instructionMissing:
			fmt.Fprintf(stdout, "  ⚠ %s is missing EVE instructions\n", target.Filename)
			fmt.Fprintf(stdout, "    Run `eve instructions install --target %s`.\n", target.Name)
		case instructionStale:
			fmt.Fprintf(stdout, "  ⚠ %s uses EVE instruction version %d; current version is %d\n", target.Filename, inspection.Version, currentInstructionVersion)
			fmt.Fprintln(stdout, "    Run `eve instructions install` to update it.")
		case instructionModified:
			fmt.Fprintf(stdout, "  ⚠ %s contains a modified EVE instruction block\n", target.Filename)
			fmt.Fprintln(stdout, "    Run `eve instructions diff` to inspect it.")
			fmt.Fprintln(stdout, "    Run `eve instructions install --force` to replace it.")
		case instructionMalformed:
			fmt.Fprintf(stdout, "  ✗ %s contains malformed EVE instruction markers\n", target.Filename)
			fmt.Fprintln(stdout, "    Resolve the markers manually, then run `eve instructions status`.")
		}
	}

	fmt.Fprintln(stdout, "\nSnapshot activity")
	if _, err := os.Stat(repo.snapshotsDir()); err == nil {
		snapshots, listErr := repo.listSnapshots("")
		if listErr != nil {
			fatal = true
			fmt.Fprintf(stdout, "  ✗ %v\n", listErr)
		} else if len(snapshots) == 0 {
			fmt.Fprintln(stdout, "  ⚠ No Snapshot has been created yet")
		} else {
			fmt.Fprintf(stdout, "  ✓ %d Snapshot(s) recorded\n", len(snapshots))
		}
	} else if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintln(stdout, "  ⚠ Snapshot storage is not initialized")
	} else {
		fatal = true
		fmt.Fprintf(stdout, "  ✗ inspect snapshot storage: %v\n", err)
	}

	if fatal {
		return 1
	}
	return 0
}

func validateDoctorRepository(repo repository) error {
	info, err := os.Stat(repo.eveDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New(".eve directory is missing; run `eve init`")
		}
		return fmt.Errorf("inspect .eve directory: %w", err)
	}
	if !info.IsDir() {
		return errors.New(".eve is not a directory")
	}
	config, err := repo.loadConfig()
	if err != nil {
		return err
	}
	if config.SchemaVersion != configFileVersion {
		return fmt.Errorf("configuration schemaVersion is %d; expected %d", config.SchemaVersion, configFileVersion)
	}
	if config.SnapshotSchema != eve.SnapshotSchemaVersion {
		return fmt.Errorf("configuration snapshotSchema is %q; expected %q", config.SnapshotSchema, eve.SnapshotSchemaVersion)
	}
	for _, dir := range []string{repo.snapshotsDir(), repo.skipsDir(), repo.artifactsDir(), repo.cacheDir()} {
		info, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("inspect %s: %w", filepath.Base(dir), err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}
	}
	return nil
}

func detectMCPConfigurations(home string) ([]string, []string) {
	var configured []string
	var warnings []string

	codexPath := filepath.Join(home, ".codex", "config.toml")
	if data, err := os.ReadFile(codexPath); err == nil {
		if tomlTableContains(string(data), "[mcp_servers.eve]", "mcp-stdio") {
			configured = append(configured, "Codex")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		warnings = append(warnings, fmt.Sprintf("could not inspect %s: %v", codexPath, err))
	}

	claudePath := filepath.Join(home, ".claude.json")
	if found, err := jsonMCPConfigured(claudePath, []string{"mcpServers", "eve"}, "args"); err != nil {
		warnings = append(warnings, fmt.Sprintf("could not inspect %s: %v", claudePath, err))
	} else if found {
		configured = append(configured, "Claude Code")
	}

	opencodePath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if found, err := jsonMCPConfigured(opencodePath, []string{"mcp", "eve"}, "command"); err != nil {
		warnings = append(warnings, fmt.Sprintf("could not inspect %s: %v", opencodePath, err))
	} else if found {
		configured = append(configured, "opencode")
	}
	return configured, warnings
}

func tomlTableContains(data string, table string, value string) bool {
	lines := strings.Split(data, "\n")
	inTable := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			if inTable {
				return false
			}
			inTable = trimmed == table
			continue
		}
		if inTable && strings.Contains(trimmed, value) {
			return true
		}
	}
	return false
}

func jsonMCPConfigured(path string, keys []string, commandKey string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return false, err
	}
	var current any = root
	for _, key := range keys {
		object, ok := current.(map[string]any)
		if !ok {
			return false, nil
		}
		current, ok = object[key]
		if !ok {
			return false, nil
		}
	}
	server, ok := current.(map[string]any)
	if !ok {
		return false, nil
	}
	return containsMCPStdio(server[commandKey]), nil
}

func containsMCPStdio(value any) bool {
	switch typed := value.(type) {
	case string:
		return strings.Contains(typed, "mcp-stdio")
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok && text == "mcp-stdio" {
				return true
			}
		}
	}
	return false
}

func mcpToolAvailable(name string) bool {
	for _, tool := range mcpTools() {
		if tool["name"] == name {
			return true
		}
	}
	return false
}
