package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "protocol v1") {
		t.Fatalf("stdout = %q, want protocol version", stdout.String())
	}
}

func TestRunValidateValidFile(t *testing.T) {
	path := writeTempEvolution(t, validCLIJSON())
	var stdout, stderr bytes.Buffer

	code := run([]string{"validate", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "valid") {
		t.Fatalf("stdout = %q, want valid", stdout.String())
	}
}

func TestRunValidateInvalidFile(t *testing.T) {
	path := writeTempEvolution(t, `{"eve":{"version":2}}`)
	var stdout, stderr bytes.Buffer

	code := run([]string{"validate", path}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "invalid evolution") {
		t.Fatalf("stderr = %q, want validation error", stderr.String())
	}
}

func TestRunValidateUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"validate"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestRunValidateFileIOError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"validate", filepath.Join(t.TempDir(), "missing.json")}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func TestRunCanonicalize(t *testing.T) {
	path := writeTempEvolution(t, validCLIJSON())
	var stdout, stderr bytes.Buffer

	code := run([]string{"canonicalize", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if strings.Contains(strings.TrimSpace(stdout.String()), "\n") {
		t.Fatalf("stdout contains embedded newline: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"eve":{"version":1}`) {
		t.Fatalf("stdout = %q, want canonical JSON", stdout.String())
	}
}

func TestRunCanonicalizeFileIOError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"canonicalize", filepath.Join(t.TempDir(), "missing.json")}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
}

func writeTempEvolution(t *testing.T, input string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "evolution.json")
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatalf("write temp evolution: %v", err)
	}
	return path
}

func validCLIJSON() string {
	return `{
  "eve": {"version": 1},
  "metadata": {"status": "active", "type": "custom"},
  "intent": "Document work.",
  "outcome": "Work is documented.",
  "behavior": {},
  "decisions": [],
  "risks": [],
  "verification": [{"status": "passed"}],
  "sessions": [],
  "timeline": [],
  "relationships": {},
  "implementation": {"repositories": {"web": {"status": "merged"}}},
  "extensions": {"acme": {"rollout": "25%"}}
}`
}
