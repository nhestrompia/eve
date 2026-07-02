package eve

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestParseValidEvolution(t *testing.T) {
	evolution, err := Parse([]byte(validEvolutionJSON()))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if evolution.EVE.Version != ProtocolVersion {
		t.Fatalf("version = %d, want %d", evolution.EVE.Version, ProtocolVersion)
	}
	if evolution.Metadata.Type != "custom-feature" {
		t.Fatalf("metadata type = %q, want producer-defined value", evolution.Metadata.Type)
	}
	if got := string(evolution.Extensions["acme"]); !strings.Contains(got, "rollout") {
		t.Fatalf("extensions not preserved: %s", got)
	}
}

func TestParseRejectsMissingTopLevelFields(t *testing.T) {
	_, err := Parse([]byte(`{"eve":{"version":1}}`))
	if err == nil {
		t.Fatal("Parse succeeded, want missing field error")
	}
	if !strings.Contains(err.Error(), "missing top-level field") {
		t.Fatalf("error = %q, want missing top-level field", err.Error())
	}
}

func TestParseRejectsUnknownTopLevelFields(t *testing.T) {
	input := strings.Replace(validEvolutionJSON(), `"extensions": {`, `"unexpected": true, "extensions": {`, 1)
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("Parse succeeded, want unknown field error")
	}
}

func TestParseRejectsWrongTopLevelFieldKind(t *testing.T) {
	input := strings.Replace(validEvolutionJSON(), `"decisions": [],`, `"decisions": null,`, 1)
	_, err := Parse([]byte(input))
	if err == nil {
		t.Fatal("Parse succeeded, want wrong top-level kind error")
	}
	if !strings.Contains(err.Error(), "decisions must be an array") {
		t.Fatalf("error = %q, want decisions array error", err.Error())
	}
}

func TestValidateRejectsInvalidMetadataStatus(t *testing.T) {
	evolution := mustParse(t, validEvolutionJSON())
	evolution.Metadata.Status = "done"

	err := Validate(evolution)
	if err == nil {
		t.Fatal("Validate succeeded, want invalid metadata status")
	}
	if !strings.Contains(err.Error(), "metadata.status") {
		t.Fatalf("error = %q, want metadata.status", err.Error())
	}
}

func TestValidateRejectsInvalidVerificationStatus(t *testing.T) {
	evolution := mustParse(t, validEvolutionJSON())
	evolution.Verification[0].Status = "ok"

	err := Validate(evolution)
	if err == nil {
		t.Fatal("Validate succeeded, want invalid verification status")
	}
	if !strings.Contains(err.Error(), "verification[0].status") {
		t.Fatalf("error = %q, want verification status", err.Error())
	}
}

func TestValidateRejectsCompletedEvolutionWithNonTerminalRepository(t *testing.T) {
	evolution := mustParse(t, validEvolutionJSON())
	evolution.Metadata.Status = "completed"
	evolution.Implementation.Repositories["web"] = Repository{Status: "open"}

	err := Validate(evolution)
	if err == nil {
		t.Fatal("Validate succeeded, want non-terminal repository status error")
	}
	if !strings.Contains(err.Error(), `implementation.repositories["web"].status`) {
		t.Fatalf("error = %q, want repository status", err.Error())
	}
}

func TestCanonicalJSONIsCompactAndStable(t *testing.T) {
	evolution := mustParse(t, validEvolutionJSON())
	canonical, err := CanonicalJSON(evolution)
	if err != nil {
		t.Fatalf("CanonicalJSON returned error: %v", err)
	}

	if strings.Contains(string(canonical), "\n") {
		t.Fatalf("canonical JSON contains newline: %q", string(canonical))
	}
	if strings.Contains(string(canonical), `\u003c`) {
		t.Fatalf("canonical JSON escaped HTML characters: %q", string(canonical))
	}
	if !json.Valid(canonical) {
		t.Fatalf("canonical JSON is invalid: %q", string(canonical))
	}

	reparsed := mustParse(t, string(canonical))
	second, err := CanonicalJSON(reparsed)
	if err != nil {
		t.Fatalf("second CanonicalJSON returned error: %v", err)
	}
	if string(second) != string(canonical) {
		t.Fatalf("canonical JSON not stable:\nfirst:  %s\nsecond: %s", canonical, second)
	}
}

func TestCanonicalJSONNormalizesNilTopLevelCollections(t *testing.T) {
	evolution := &Evolution{
		EVE:      EVEHeader{Version: ProtocolVersion},
		Metadata: Metadata{Status: "active"},
	}

	canonical, err := CanonicalJSON(evolution)
	if err != nil {
		t.Fatalf("CanonicalJSON returned error: %v", err)
	}

	got := string(canonical)
	for _, want := range []string{
		`"decisions":[]`,
		`"risks":[]`,
		`"verification":[]`,
		`"sessions":[]`,
		`"timeline":[]`,
		`"extensions":{}`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("canonical JSON = %s, want %s", got, want)
		}
	}
}

func TestCanonicalJSONNormalizesRawMessages(t *testing.T) {
	evolution := &Evolution{
		EVE:       EVEHeader{Version: ProtocolVersion},
		Metadata:  Metadata{Status: "active"},
		Decisions: []json.RawMessage{json.RawMessage(`{ "name" : "decision" }`)},
		Risks:     []json.RawMessage{json.RawMessage(`{ "name" : "risk" }`)},
		Extensions: map[string]json.RawMessage{
			"acme": json.RawMessage(`{ "rollout" : "25%" }`),
		},
	}

	canonical, err := CanonicalJSON(evolution)
	if err != nil {
		t.Fatalf("CanonicalJSON returned error: %v", err)
	}

	got := string(canonical)
	for _, want := range []string{
		`"decisions":[{"name":"decision"}]`,
		`"risks":[{"name":"risk"}]`,
		`"extensions":{"acme":{"rollout":"25%"}}`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("canonical JSON = %s, want %s", got, want)
		}
	}
}

func TestSchemaDocumentIsValidJSON(t *testing.T) {
	data, err := os.ReadFile("schema/eve.v1.schema.json")
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if !json.Valid(data) {
		t.Fatal("schema/eve.v1.schema.json is not valid JSON")
	}
}

func mustParse(t *testing.T, input string) *Evolution {
	t.Helper()
	evolution, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	return evolution
}

func validEvolutionJSON() string {
	return `{
  "eve": {
    "version": 1
  },
  "metadata": {
    "id": "ev_182",
    "title": "Enterprise SSO",
    "type": "custom-feature",
    "status": "active",
    "created_by": "codex",
    "created_at": "2026-07-02T12:00:00Z",
    "updated_at": "2026-07-02T13:00:00Z"
  },
  "intent": "Add enterprise SSO support.",
  "outcome": "Organizations can authenticate using Okta.",
  "behavior": {
    "added": [
      {
        "description": "Organizations can authenticate using Okta",
        "evidence": {
          "commits": ["81ab92"],
          "files": ["auth/providers/okta.ts"],
          "tests": ["auth.okta.test.ts"]
        }
      }
    ],
    "changed": [],
    "removed": [],
    "fixed": []
  },
  "decisions": [],
  "risks": [],
  "verification": [
    {
      "type": "unit-tests",
      "status": "passed",
      "reference": "npm test"
    }
  ],
  "sessions": [
    {
      "provider": "codex",
      "id": "session_912",
      "uri": "codex://session/912"
    }
  ],
  "timeline": [
    {
      "timestamp": "2026-07-02T12:10:00Z",
      "actor": {
        "type": "agent",
        "provider": "codex",
        "id": "session_912"
      },
      "event": "implementation_started",
      "description": "Started implementing SAML provider support."
    }
  ],
  "relationships": {
    "extends": ["ev_140"],
    "depends_on": [],
    "corrects": [],
    "supersedes": [],
    "reverts": [],
    "related": []
  },
  "implementation": {
    "repositories": {
      "web": {
        "status": "merged"
      }
    },
    "commits": ["81ab2f"],
    "pull_requests": ["412"],
    "files_changed": 214,
    "insertions": 8421,
    "deletions": 3122
  },
  "extensions": {
    "acme": {
      "rollout": "25%",
      "feature_flag": "<enterprise-sso>"
    }
  }
}`
}
