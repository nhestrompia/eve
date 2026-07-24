package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifyInstructionData(t *testing.T) {
	target := instructionTargets[0]
	oldTemplate := strings.ReplaceAll(canonicalInstructionTemplateV1, `version="1"`, `version="0"`)
	templates := map[int]string{0: oldTemplate, 1: canonicalInstructionTemplateV1}
	cases := []struct {
		name    string
		data    string
		state   instructionState
		version int
	}{
		{name: "missing", data: "# Existing\n", state: instructionMissing},
		{name: "current", data: canonicalInstructionTemplateV1, state: instructionCurrent, version: 1},
		{name: "current crlf", data: strings.ReplaceAll(canonicalInstructionTemplateV1, "\n", "\r\n"), state: instructionCurrent, version: 1},
		{name: "current crlf file", data: strings.ReplaceAll(canonicalInstructionTemplateV1+"\n", "\n", "\r\n"), state: instructionCurrent, version: 1},
		{name: "stale", data: oldTemplate, state: instructionStale},
		{name: "modified", data: strings.Replace(canonicalInstructionTemplateV1, "completed product work", "important product work", 1), state: instructionModified, version: 1},
		{name: "unknown version", data: strings.Replace(canonicalInstructionTemplateV1, `version="1"`, `version="2"`, 1), state: instructionModified, version: 2},
		{name: "only start", data: `<!-- eve:instructions:start version="1" -->`, state: instructionMalformed},
		{name: "only end", data: `<!-- eve:instructions:end -->`, state: instructionMalformed},
		{name: "duplicate", data: canonicalInstructionTemplateV1 + "\n" + canonicalInstructionTemplateV1, state: instructionMalformed},
		{name: "nested", data: `<!-- eve:instructions:start version="1" -->
<!-- eve:instructions:start version="1" -->
<!-- eve:instructions:end -->`, state: instructionMalformed},
		{name: "bad version", data: `<!-- eve:instructions:start version="x" -->
<!-- eve:instructions:end -->`, state: instructionMalformed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inspection, err := classifyInstructionData(target, "AGENTS.md", []byte(tc.data), 0o644, templates, 1)
			if err != nil {
				t.Fatalf("classify: %v", err)
			}
			if inspection.State != tc.state || inspection.Version != tc.version {
				t.Fatalf("state/version = %s/%d, want %s/%d", inspection.State, inspection.Version, tc.state, tc.version)
			}
		})
	}
}

func TestInstructionInstallPreservesContentModeAndIsIdempotent(t *testing.T) {
	repoRoot := initTempGitRepo(t)
	repo := repoFromRoot(repoRoot)
	path := filepath.Join(repoRoot, "AGENTS.md")
	original := []byte("# Team rules\n\nKeep this text unchanged.\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}

	result := installInstructionTarget(repo, instructionTargets[0], false)
	if result.Err != nil || result.Action != "updated" {
		t.Fatalf("install result = %#v", result)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if !bytes.HasPrefix(data, original) || strings.Count(string(data), "<!-- eve:instructions:start") != 1 {
		t.Fatalf("updated content = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat AGENTS.md: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}

	before := append([]byte(nil), data...)
	result = installInstructionTarget(repo, instructionTargets[0], false)
	if result.Err != nil || result.Action != "current" {
		t.Fatalf("second install result = %#v", result)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("idempotent install changed AGENTS.md")
	}
}

func TestInstructionInstallProtectsModifiedAndMalformedBlocks(t *testing.T) {
	repoRoot := initTempGitRepo(t)
	repo := repoFromRoot(repoRoot)
	path := filepath.Join(repoRoot, "AGENTS.md")
	modified := strings.Replace(canonicalInstructionTemplateV2, "approve implementation plans", "review implementation plans", 1) + "\n"
	if err := os.WriteFile(path, []byte(modified), 0o644); err != nil {
		t.Fatalf("write modified block: %v", err)
	}

	result := installInstructionTarget(repo, instructionTargets[0], false)
	if result.Err == nil || !strings.Contains(result.Err.Error(), "modified") {
		t.Fatalf("modified install result = %#v", result)
	}
	if got := readTextFile(t, path); got != modified {
		t.Fatalf("modified block was overwritten: %q", got)
	}
	result = installInstructionTarget(repo, instructionTargets[0], true)
	if result.Err != nil || result.Action != "updated" {
		t.Fatalf("forced result = %#v", result)
	}
	if got := readTextFile(t, path); !strings.Contains(got, "approve implementation plans") {
		t.Fatalf("forced content = %q", got)
	}

	malformed := "before\n<!-- eve:instructions:start version=\"1\" -->\n"
	if err := os.WriteFile(path, []byte(malformed), 0o644); err != nil {
		t.Fatalf("write malformed block: %v", err)
	}
	result = installInstructionTarget(repo, instructionTargets[0], true)
	if result.Err == nil || !strings.Contains(result.Err.Error(), "malformed") {
		t.Fatalf("malformed forced result = %#v", result)
	}
	if got := readTextFile(t, path); got != malformed {
		t.Fatalf("malformed block was overwritten: %q", got)
	}
}

func TestInstructionInstallUpgradesExactKnownStaleBlock(t *testing.T) {
	repoRoot := initTempGitRepo(t)
	oldTemplate := strings.ReplaceAll(canonicalInstructionTemplateV1, `version="1"`, `version="0"`)
	previous, existed := instructionTemplates[0]
	instructionTemplates[0] = oldTemplate
	defer func() {
		if existed {
			instructionTemplates[0] = previous
		} else {
			delete(instructionTemplates, 0)
		}
	}()
	path := filepath.Join(repoRoot, "AGENTS.md")
	original := "# Existing\n\n" + oldTemplate + "\n\nAfter block.\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write stale block: %v", err)
	}
	result := installInstructionTarget(repoFromRoot(repoRoot), instructionTargets[0], false)
	if result.Err != nil || result.Action != "updated" || result.Inspection.State != instructionStale {
		t.Fatalf("stale result = %#v", result)
	}
	updated := readTextFile(t, path)
	if !strings.HasPrefix(updated, "# Existing\n\n") || !strings.HasSuffix(updated, "\n\nAfter block.\n") || !strings.Contains(updated, `version="2"`) {
		t.Fatalf("updated stale content = %q", updated)
	}
}

func TestInstructionInstallRejectsSymlink(t *testing.T) {
	repoRoot := initTempGitRepo(t)
	target := filepath.Join(repoRoot, "real.md")
	if err := os.WriteFile(target, []byte("real\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(repoRoot, "AGENTS.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	result := installInstructionTarget(repoFromRoot(repoRoot), instructionTargets[0], true)
	if result.Err == nil || !strings.Contains(result.Err.Error(), "not a regular file") {
		t.Fatalf("symlink result = %#v", result)
	}
	if got := readTextFile(t, target); got != "real\n" {
		t.Fatalf("symlink target changed: %q", got)
	}
}

func TestInitInstructionFlagsAndPartialFailure(t *testing.T) {
	t.Run("conflicting flags", func(t *testing.T) {
		repo := initTempGitRepo(t)
		t.Chdir(repo)
		var stdout, stderr bytes.Buffer
		if code := run([]string{"init", "--no-agent-instructions", "--instructions-only"}, &stdout, &stderr); code != 2 {
			t.Fatalf("code = %d, stdout = %s stderr = %s", code, stdout.String(), stderr.String())
		}
	})

	t.Run("skip instructions", func(t *testing.T) {
		repo := initTempGitRepo(t)
		t.Chdir(repo)
		var stdout, stderr bytes.Buffer
		if code := run([]string{"init", "--no-agent-instructions"}, &stdout, &stderr); code != 0 {
			t.Fatalf("code = %d, stderr = %s", code, stderr.String())
		}
		if _, err := os.Stat(filepath.Join(repo, "AGENTS.md")); !os.IsNotExist(err) {
			t.Fatalf("AGENTS.md should be absent, err = %v", err)
		}
		if _, err := os.Stat(filepath.Join(repo, ".eve", "config.json")); err != nil {
			t.Fatalf("config missing: %v", err)
		}
	})

	t.Run("instructions only", func(t *testing.T) {
		repo := initTempGitRepo(t)
		t.Chdir(repo)
		var stdout, stderr bytes.Buffer
		if code := run([]string{"init", "--instructions-only"}, &stdout, &stderr); code != 0 {
			t.Fatalf("code = %d, stderr = %s", code, stderr.String())
		}
		for _, name := range []string{"AGENTS.md", "CLAUDE.md"} {
			if _, err := os.Stat(filepath.Join(repo, name)); err != nil {
				t.Fatalf("%s missing: %v", name, err)
			}
		}
		if _, err := os.Stat(filepath.Join(repo, ".eve")); !os.IsNotExist(err) {
			t.Fatalf(".eve should be absent, err = %v", err)
		}
	})

	t.Run("partial failure", func(t *testing.T) {
		repo := initTempGitRepo(t)
		t.Chdir(repo)
		if err := os.Mkdir(filepath.Join(repo, "CLAUDE.md"), 0o755); err != nil {
			t.Fatalf("mkdir CLAUDE.md: %v", err)
		}
		var stdout, stderr bytes.Buffer
		if code := run([]string{"init"}, &stdout, &stderr); code != 1 {
			t.Fatalf("code = %d, stdout = %s stderr = %s", code, stdout.String(), stderr.String())
		}
		if _, err := os.Stat(filepath.Join(repo, ".eve", "config.json")); err != nil {
			t.Fatalf("config missing after partial failure: %v", err)
		}
		if _, err := os.Stat(filepath.Join(repo, "AGENTS.md")); err != nil {
			t.Fatalf("AGENTS.md missing after partial failure: %v", err)
		}
	})
}

func TestInstructionsCLIStatusDiffTargetForceAndTrackedOutput(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# Existing\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	gitRun(t, repo, "add", "AGENTS.md")
	gitRun(t, repo, "commit", "-m", "add agent rules")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"instructions", "install", "--target", "agents"}, &stdout, &stderr); code != 0 {
		t.Fatalf("install code = %d stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Modified tracked files:") || !strings.Contains(stdout.String(), "AGENTS.md") {
		t.Fatalf("install stdout = %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(repo, "CLAUDE.md")); !os.IsNotExist(err) {
		t.Fatalf("CLAUDE.md should not be installed, err = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"instructions", "status"}, &stdout, &stderr); code != 1 || !strings.Contains(stdout.String(), "CLAUDE.md") || !strings.Contains(stdout.String(), "Missing") {
		t.Fatalf("status code/stdout = %d/%q stderr = %q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"instructions", "diff"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "+++ canonical/CLAUDE.md") || !strings.Contains(stdout.String(), "+<!-- eve:instructions:start") {
		t.Fatalf("diff code/stdout = %d/%q stderr = %q", code, stdout.String(), stderr.String())
	}

	modified := strings.Replace(canonicalInstructionTemplateV2, "approve implementation plans", "review implementation plans", 1)
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte(modified), 0o644); err != nil {
		t.Fatalf("write modified: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"instructions", "install", "--target", "agents"}, &stdout, &stderr); code != 1 {
		t.Fatalf("modified install code = %d", code)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"instructions", "install", "--target", "agents", "--force"}, &stdout, &stderr); code != 0 {
		t.Fatalf("force code = %d stderr = %s", code, stderr.String())
	}
}

func TestInitPreservesModifiedBlockWithWarning(t *testing.T) {
	repo := initTempGitRepo(t)
	t.Chdir(repo)
	modified := strings.Replace(canonicalInstructionTemplateV2, "approve implementation plans", "review implementation plans", 1)
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte(modified), 0o644); err != nil {
		t.Fatalf("write modified: %v", err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"init"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "preserved existing content") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if got := readTextFile(t, filepath.Join(repo, "AGENTS.md")); got != modified {
		t.Fatalf("modified content changed: %q", got)
	}
}

func TestDoctorDiagnostics(t *testing.T) {
	t.Run("healthy with warnings", func(t *testing.T) {
		repo := initTempGitRepo(t)
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Chdir(repo)
		mustRun(t, []string{"init"})
		var stdout, stderr bytes.Buffer
		if code := run([]string{"doctor"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "No EVE configuration found") {
			t.Fatalf("warning-only doctor code/stdout = %d/%q stderr = %q", code, stdout.String(), stderr.String())
		}
		configPath := filepath.Join(home, ".codex", "config.toml")
		if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
			t.Fatalf("mkdir config: %v", err)
		}
		if err := os.WriteFile(configPath, []byte("[mcp_servers.eve]\ncommand = \"eve\"\nargs = [\"mcp-stdio\"]\n"), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		stdout.Reset()
		stderr.Reset()
		if code := run([]string{"doctor"}, &stdout, &stderr); code != 0 {
			t.Fatalf("code = %d stdout = %s stderr = %s", code, stdout.String(), stderr.String())
		}
		for _, want := range []string{"Configuration is valid", "Codex", "complete_snapshot available", "AGENTS.md contains current", "No Snapshot has been created yet"} {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("stdout = %q, want %q", stdout.String(), want)
			}
		}
		head := gitOutputForTest(t, repo, "rev-parse", "HEAD")
		writeSnapshot(t, repo, sampleSnapshot("snap_doctor", "Doctor Snapshot", head))
		stdout.Reset()
		stderr.Reset()
		if code := run([]string{"doctor"}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "1 Snapshot(s) recorded") {
			t.Fatalf("snapshot doctor code/stdout = %d/%q stderr = %q", code, stdout.String(), stderr.String())
		}
	})

	t.Run("invalid repository and missing instructions", func(t *testing.T) {
		repo := initTempGitRepo(t)
		t.Setenv("HOME", t.TempDir())
		t.Chdir(repo)
		if err := os.MkdirAll(filepath.Join(repo, ".eve"), 0o755); err != nil {
			t.Fatalf("mkdir .eve: %v", err)
		}
		if err := os.WriteFile(filepath.Join(repo, ".eve", "config.json"), []byte(`{"schemaVersion":999}`), 0o644); err != nil {
			t.Fatalf("write config: %v", err)
		}
		var stdout, stderr bytes.Buffer
		if code := run([]string{"doctor"}, &stdout, &stderr); code != 1 {
			t.Fatalf("code = %d stdout = %s stderr = %s", code, stdout.String(), stderr.String())
		}
		if !strings.Contains(stdout.String(), "schemaVersion") || !strings.Contains(stdout.String(), "missing EVE instructions") {
			t.Fatalf("stdout = %q", stdout.String())
		}
	})
}

func TestDetectMCPConfigurations(t *testing.T) {
	home := t.TempDir()
	files := map[string]string{
		filepath.Join(home, ".codex", "config.toml"): `[mcp_servers.eve]
command = "eve"
args = ["mcp-stdio"]
`,
		filepath.Join(home, ".claude.json"):                         `{"mcpServers":{"eve":{"command":"eve","args":["mcp-stdio"]}}}`,
		filepath.Join(home, ".config", "opencode", "opencode.json"): `{"mcp":{"eve":{"type":"local","command":["eve","mcp-stdio"]}}}`,
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	configured, warnings := detectMCPConfigurations(home)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %#v", warnings)
	}
	if got := strings.Join(configured, ","); got != "Codex,Claude Code,opencode" {
		t.Fatalf("configured = %q", got)
	}
}
