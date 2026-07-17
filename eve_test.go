package eve

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestParseValidSnapshot(t *testing.T) {
	snapshot, err := ParseSnapshot([]byte(validSnapshotJSON()))
	if err != nil {
		t.Fatalf("ParseSnapshot returned error: %v", err)
	}
	if snapshot.SchemaVersion != SnapshotSchemaVersion {
		t.Fatalf("schemaVersion = %q, want %q", snapshot.SchemaVersion, SnapshotSchemaVersion)
	}
	if snapshot.Type != "feature" {
		t.Fatalf("type = %q, want feature", snapshot.Type)
	}
	if len(snapshot.Relationships.Corrects) != 0 {
		t.Fatalf("relationships were not normalized")
	}
}

func TestParseLegacySnapshotLabelsValidationAsUnattributed(t *testing.T) {
	legacy := validSnapshotJSON()
	snapshot, err := ParseSnapshot([]byte(legacy))
	if err != nil {
		t.Fatalf("ParseSnapshot returned error: %v", err)
	}
	if snapshot.Validation[0].Provenance != "legacy_unattributed" {
		t.Fatalf("provenance = %q, want legacy_unattributed", snapshot.Validation[0].Provenance)
	}
}

func TestParseSchemaLessHistoricalSnapshotAsLegacy(t *testing.T) {
	historical := strings.Replace(validSnapshotJSON(), "  \"schemaVersion\": \"0.1.0\",\n", "", 1)
	snapshot, err := ParseSnapshot([]byte(historical))
	if err != nil {
		t.Fatalf("ParseSnapshot returned error: %v", err)
	}
	if snapshot.SchemaVersion != SnapshotSchemaVersion || snapshot.Validation[0].Provenance != "legacy_unattributed" {
		t.Fatalf("historical snapshot migration = %#v", snapshot)
	}
}

func TestParseCurrentSnapshotLabelsMissingValidationProvenanceAsAgentReported(t *testing.T) {
	current := strings.Replace(validSnapshotJSON(), `"schemaVersion": "0.1.0"`, `"schemaVersion": "0.2.0"`, 1)
	snapshot, err := ParseSnapshot([]byte(current))
	if err != nil {
		t.Fatalf("ParseSnapshot returned error: %v", err)
	}
	if snapshot.Validation[0].Provenance != "reported_by_agent" {
		t.Fatalf("provenance = %q, want reported_by_agent", snapshot.Validation[0].Provenance)
	}
}

func TestParseRejectsUnknownFields(t *testing.T) {
	input := strings.Replace(validSnapshotJSON(), `"createdAt":`, `"unexpected": true, "createdAt":`, 1)
	_, err := ParseSnapshot([]byte(input))
	if err == nil {
		t.Fatal("ParseSnapshot succeeded, want unknown field error")
	}
}

func TestValidateRejectsInvalidTypeAndStatus(t *testing.T) {
	snapshot := mustParseSnapshot(t, validSnapshotJSON())
	snapshot.Type = "docs"
	snapshot.Validation[0].Status = "pending"
	err := ValidateSnapshot(snapshot)
	if err == nil {
		t.Fatal("ValidateSnapshot succeeded, want errors")
	}
	for _, want := range []string{"type must be one of", "validation[0].status"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
	}
}

func TestCanonicalSnapshotJSONNormalizesCollections(t *testing.T) {
	snapshot := &Snapshot{
		ID:            "snap_test",
		SchemaVersion: SnapshotSchemaVersion,
		Title:         "Snapshot runtime",
		Type:          "feature",
		Summary:       "Snapshots are canonical.",
		Implementation: Implementation{
			Branch:   "main",
			GitState: "abc123",
			Commits:  []string{},
			Dirty:    false,
		},
		CreatedAt: "2026-07-03T12:00:00Z",
	}
	canonical, err := CanonicalSnapshotJSON(snapshot)
	if err != nil {
		t.Fatalf("CanonicalSnapshotJSON returned error: %v", err)
	}
	got := string(canonical)
	for _, want := range []string{
		`"relationships":{"corrects":[],"supersedes":[],"reverts":[],"dependsOn":[],"related":[]}`,
		`"risks":[]`,
		`"timeline":[]`,
		`"decisions":[]`,
		`"validation":[]`,
		`"artifacts":[]`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("canonical JSON = %s, want %s", got, want)
		}
	}
	reparsed := mustParseSnapshot(t, got)
	second, err := CanonicalSnapshotJSON(reparsed)
	if err != nil {
		t.Fatalf("second CanonicalSnapshotJSON returned error: %v", err)
	}
	if string(second) != got {
		t.Fatalf("canonical JSON not stable:\nfirst: %s\nsecond: %s", got, second)
	}
}

func TestSnapshotSchemaDocumentIsValidJSON(t *testing.T) {
	data, err := os.ReadFile("schema/eve.snapshot.v0.schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if !json.Valid(data) {
		t.Fatal("schema/eve.snapshot.v0.schema.json is not valid JSON")
	}
}

func mustParseSnapshot(t *testing.T, input string) *Snapshot {
	t.Helper()
	snapshot, err := ParseSnapshot([]byte(input))
	if err != nil {
		t.Fatalf("ParseSnapshot returned error: %v", err)
	}
	return snapshot
}

func validSnapshotJSON() string {
	return `{
  "id": "snap_123",
  "schemaVersion": "0.1.0",
  "title": "Add GitHub OAuth",
  "type": "feature",
  "summary": "Users can now sign in with GitHub.",
  "userVisibleChange": "The login screen now includes a GitHub sign-in option.",
  "relationships": {
    "corrects": [],
    "supersedes": [],
    "reverts": [],
    "dependsOn": [],
    "related": []
  },
  "risks": [
    {
      "title": "OAuth callback misconfiguration",
      "severity": "medium",
      "mitigation": "Covered by callback validation tests."
    }
  ],
  "timeline": [
    {
      "phase": "validation",
      "title": "Verified OAuth tests",
      "summary": "Ran focused auth tests."
    }
  ],
  "decisions": [
    {
      "title": "Use provider-owned OAuth flow",
      "rationale": "Avoids password handling."
    }
  ],
  "validation": [
    {
      "command": "go test ./...",
      "status": "passed",
      "output": "ok"
    }
  ],
  "artifacts": [
    {
      "type": "screenshot",
      "path": ".eve/artifacts/snap_123/login.png",
      "mimeType": "image/png",
      "description": "Login screen"
    }
  ],
  "implementation": {
    "branch": "main",
    "gitState": "abc123",
    "baseCommit": "def456",
    "commits": ["abc123"],
    "dirty": false
  },
  "createdAt": "2026-07-03T15:00:00Z"
}`
}
