package eve

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var snapshotTypes = map[string]struct{}{
	"feature":    {},
	"bugfix":     {},
	"experiment": {},
	"refactor":   {},
	"release":    {},
}

var riskSeverities = map[string]struct{}{
	"low":    {},
	"medium": {},
	"high":   {},
}

var timelinePhases = map[string]struct{}{
	"planning":       {},
	"implementation": {},
	"validation":     {},
	"review":         {},
	"release":        {},
}

var validationStatuses = map[string]struct{}{
	"passed":  {},
	"failed":  {},
	"skipped": {},
}

var artifactTypes = map[string]struct{}{
	"screenshot":   {},
	"video":        {},
	"preview":      {},
	"url":          {},
	"note":         {},
	"log":          {},
	"conversation": {},
}

func ParseSnapshot(data []byte) (*Snapshot, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var snapshot Snapshot
	if err := dec.Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("parse snapshot: %w", err)
	}
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("parse snapshot: unexpected trailing JSON value")
	}

	legacySchema := snapshot.SchemaVersion == "" || snapshot.SchemaVersion == "0.1.0"
	snapshot = normalizeSnapshot(snapshot)
	for i := range snapshot.Validation {
		if strings.TrimSpace(snapshot.Validation[i].Provenance) == "" {
			if legacySchema {
				snapshot.Validation[i].Provenance = "legacy_unattributed"
			} else {
				snapshot.Validation[i].Provenance = "reported_by_agent"
			}
		}
	}
	if err := ValidateSnapshot(&snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func LoadSnapshotFile(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load snapshot file %q: %w", path, err)
	}
	return ParseSnapshot(data)
}

func ValidateSnapshot(snapshot *Snapshot) error {
	if snapshot == nil {
		return errors.New("snapshot is nil")
	}

	var problems []string
	if strings.TrimSpace(snapshot.ID) == "" {
		problems = append(problems, "id is required")
	}
	if snapshot.SchemaVersion != SnapshotSchemaVersion && snapshot.SchemaVersion != "0.1.0" {
		problems = append(problems, fmt.Sprintf("schemaVersion must be %q or %q", SnapshotSchemaVersion, "0.1.0"))
	}
	if strings.TrimSpace(snapshot.Title) == "" {
		problems = append(problems, "title is required")
	}
	if _, ok := snapshotTypes[snapshot.Type]; !ok {
		problems = append(problems, "type must be one of feature, bugfix, experiment, refactor, release")
	}
	if strings.TrimSpace(snapshot.Summary) == "" {
		problems = append(problems, "summary is required")
	}
	if strings.TrimSpace(snapshot.CreatedAt) == "" {
		problems = append(problems, "createdAt is required")
	}
	if strings.TrimSpace(snapshot.Implementation.GitState) == "" {
		problems = append(problems, "implementation.gitState is required")
	}
	if snapshot.Implementation.Commits == nil {
		problems = append(problems, "implementation.commits is required")
	}
	for i, risk := range snapshot.Risks {
		if strings.TrimSpace(risk.Title) == "" {
			problems = append(problems, fmt.Sprintf("risks[%d].title is required", i))
		}
		if _, ok := riskSeverities[risk.Severity]; !ok {
			problems = append(problems, fmt.Sprintf("risks[%d].severity must be one of low, medium, high", i))
		}
	}
	for i, entry := range snapshot.Timeline {
		if _, ok := timelinePhases[entry.Phase]; !ok {
			problems = append(problems, fmt.Sprintf("timeline[%d].phase must be one of planning, implementation, validation, review, release", i))
		}
		if strings.TrimSpace(entry.Title) == "" {
			problems = append(problems, fmt.Sprintf("timeline[%d].title is required", i))
		}
	}
	for i, decision := range snapshot.Decisions {
		if strings.TrimSpace(decision.Title) == "" {
			problems = append(problems, fmt.Sprintf("decisions[%d].title is required", i))
		}
	}
	for i, validation := range snapshot.Validation {
		if strings.TrimSpace(validation.Command) == "" {
			problems = append(problems, fmt.Sprintf("validation[%d].command is required", i))
		}
		if _, ok := validationStatuses[validation.Status]; !ok {
			problems = append(problems, fmt.Sprintf("validation[%d].status must be one of passed, failed, skipped", i))
		}
		if validation.Provenance != "" {
			validProvenance := map[string]bool{"executed_by_eve": true, "reported_by_agent": true, "legacy_unattributed": true}
			if !validProvenance[validation.Provenance] {
				problems = append(problems, fmt.Sprintf("validation[%d].provenance has invalid value %q", i, validation.Provenance))
			}
		}
	}
	if snapshot.Verification != nil {
		validStatuses := map[string]bool{
			"not_configured": true, "not_run": true, "incomplete": true,
			"failed": true, "required_checks_passed": true,
		}
		if !validStatuses[snapshot.Verification.Status] {
			problems = append(problems, fmt.Sprintf("verification.status has invalid value %q", snapshot.Verification.Status))
		}
	}
	for i, artifact := range snapshot.Artifacts {
		if _, ok := artifactTypes[artifact.Type]; !ok {
			problems = append(problems, fmt.Sprintf("artifacts[%d].type must be one of screenshot, video, preview, url, note, log, conversation", i))
		}
		if strings.TrimSpace(artifact.URI) == "" && strings.TrimSpace(artifact.Path) == "" && strings.TrimSpace(artifact.URL) == "" {
			problems = append(problems, fmt.Sprintf("artifacts[%d] requires uri, path, or url", i))
		}
	}

	if len(problems) > 0 {
		return ValidationError{Problems: problems}
	}
	return nil
}

func CanonicalSnapshotJSON(snapshot *Snapshot) ([]byte, error) {
	if snapshot == nil {
		return nil, errors.New("snapshot is nil")
	}
	normalized := normalizeSnapshot(*snapshot)
	if err := ValidateSnapshot(&normalized); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(normalized); err != nil {
		return nil, fmt.Errorf("canonicalize snapshot: %w", err)
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

type ValidationError struct {
	Problems []string
}

func (err ValidationError) Error() string {
	return "invalid snapshot: " + strings.Join(err.Problems, "; ")
}

func normalizeSnapshot(snapshot Snapshot) Snapshot {
	if snapshot.SchemaVersion == "" || snapshot.SchemaVersion == "0.1.0" {
		snapshot.SchemaVersion = SnapshotSchemaVersion
	}
	if snapshot.Relationships.Corrects == nil {
		snapshot.Relationships.Corrects = []string{}
	}
	if snapshot.Relationships.Supersedes == nil {
		snapshot.Relationships.Supersedes = []string{}
	}
	if snapshot.Relationships.Reverts == nil {
		snapshot.Relationships.Reverts = []string{}
	}
	if snapshot.Relationships.DependsOn == nil {
		snapshot.Relationships.DependsOn = []string{}
	}
	if snapshot.Relationships.Related == nil {
		snapshot.Relationships.Related = []string{}
	}
	if snapshot.Risks == nil {
		snapshot.Risks = []Risk{}
	}
	if snapshot.Timeline == nil {
		snapshot.Timeline = []TimelineEntry{}
	}
	if snapshot.Decisions == nil {
		snapshot.Decisions = []Decision{}
	}
	if snapshot.Validation == nil {
		snapshot.Validation = []Validation{}
	}
	if snapshot.Artifacts == nil {
		snapshot.Artifacts = []Artifact{}
	}
	if snapshot.Implementation.Commits == nil {
		snapshot.Implementation.Commits = []string{}
	}
	return snapshot
}
