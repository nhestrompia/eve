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

var requiredTopLevelFields = []string{
	"eve",
	"metadata",
	"intent",
	"outcome",
	"behavior",
	"decisions",
	"risks",
	"verification",
	"sessions",
	"timeline",
	"relationships",
	"implementation",
	"extensions",
}

var topLevelFieldKinds = map[string]byte{
	"eve":            '{',
	"metadata":       '{',
	"intent":         '"',
	"outcome":        '"',
	"behavior":       '{',
	"decisions":      '[',
	"risks":          '[',
	"verification":   '[',
	"sessions":       '[',
	"timeline":       '[',
	"relationships":  '{',
	"implementation": '{',
	"extensions":     '{',
}

var metadataStatuses = map[string]struct{}{
	"draft":     {},
	"active":    {},
	"completed": {},
	"archived":  {},
}

var verificationStatuses = map[string]struct{}{
	"passed":    {},
	"failed":    {},
	"skipped":   {},
	"pending":   {},
	"generated": {},
	"approved":  {},
}

var terminalRepositoryStatuses = map[string]struct{}{
	"merged":    {},
	"completed": {},
	"archived":  {},
}

func Parse(data []byte) (*Evolution, error) {
	if err := requireTopLevelFields(data); err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var evolution Evolution
	if err := dec.Decode(&evolution); err != nil {
		return nil, fmt.Errorf("parse evolution: %w", err)
	}
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); err != io.EOF {
		return nil, errors.New("parse evolution: unexpected trailing JSON value")
	}

	evolution = normalizeEvolution(evolution)

	if err := Validate(&evolution); err != nil {
		return nil, err
	}

	return &evolution, nil
}

func LoadFile(path string) (*Evolution, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load evolution file %q: %w", path, err)
	}
	return Parse(data)
}

func Validate(evolution *Evolution) error {
	if evolution == nil {
		return errors.New("evolution is nil")
	}

	var problems []string

	if evolution.EVE.Version != ProtocolVersion {
		problems = append(problems, fmt.Sprintf("eve.version must be %d", ProtocolVersion))
	}
	if _, ok := metadataStatuses[evolution.Metadata.Status]; !ok {
		problems = append(problems, "metadata.status must be one of draft, active, completed, archived")
	}

	for i, verification := range evolution.Verification {
		if _, ok := verificationStatuses[verification.Status]; !ok {
			problems = append(problems, fmt.Sprintf("verification[%d].status must be one of passed, failed, skipped, pending, generated, approved", i))
		}
	}

	if evolution.Metadata.Status == "completed" {
		for name, repository := range evolution.Implementation.Repositories {
			if _, ok := terminalRepositoryStatuses[repository.Status]; !ok {
				problems = append(problems, fmt.Sprintf("implementation.repositories[%q].status must be terminal for a completed evolution", name))
			}
		}
	}

	if evolution.Implementation.FilesChanged < 0 {
		problems = append(problems, "implementation.files_changed must be non-negative")
	}
	if evolution.Implementation.Insertions < 0 {
		problems = append(problems, "implementation.insertions must be non-negative")
	}
	if evolution.Implementation.Deletions < 0 {
		problems = append(problems, "implementation.deletions must be non-negative")
	}

	for name, raw := range evolution.Extensions {
		if !json.Valid(raw) {
			problems = append(problems, fmt.Sprintf("extensions[%q] must contain valid JSON", name))
		}
	}

	if len(problems) > 0 {
		return ValidationError{Problems: problems}
	}

	return nil
}

func CanonicalJSON(evolution *Evolution) ([]byte, error) {
	if err := Validate(evolution); err != nil {
		return nil, err
	}

	normalized := normalizeEvolution(*evolution)
	normalized.Decisions = normalizeRawMessageSlice(normalized.Decisions)
	normalized.Risks = normalizeRawMessageSlice(normalized.Risks)
	normalized.Extensions = normalizeRawMessages(normalized.Extensions)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(normalized); err != nil {
		return nil, fmt.Errorf("canonicalize evolution: %w", err)
	}

	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

type ValidationError struct {
	Problems []string
}

func (err ValidationError) Error() string {
	return "invalid evolution: " + strings.Join(err.Problems, "; ")
}

func requireTopLevelFields(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse evolution: %w", err)
	}

	var missing []string
	var invalid []string
	for _, field := range requiredTopLevelFields {
		value, ok := raw[field]
		if !ok {
			missing = append(missing, field)
			continue
		}
		if got := firstJSONByte(value); got != topLevelFieldKinds[field] {
			invalid = append(invalid, fmt.Sprintf("%s must be %s", field, topLevelKindName(topLevelFieldKinds[field])))
		}
	}

	if len(missing) > 0 {
		return ValidationError{Problems: []string{"missing top-level field(s): " + strings.Join(missing, ", ")}}
	}
	if len(invalid) > 0 {
		return ValidationError{Problems: invalid}
	}

	return nil
}

func normalizeRawMessages(values map[string]json.RawMessage) map[string]json.RawMessage {
	if values == nil {
		return map[string]json.RawMessage{}
	}

	normalized := make(map[string]json.RawMessage, len(values))
	for name, raw := range values {
		var buf bytes.Buffer
		if err := json.Compact(&buf, raw); err != nil {
			normalized[name] = raw
			continue
		}
		normalized[name] = append(json.RawMessage(nil), buf.Bytes()...)
	}
	return normalized
}

func normalizeRawMessageSlice(values []json.RawMessage) []json.RawMessage {
	if values == nil {
		return []json.RawMessage{}
	}

	normalized := make([]json.RawMessage, len(values))
	for i, raw := range values {
		var buf bytes.Buffer
		if err := json.Compact(&buf, raw); err != nil {
			normalized[i] = raw
			continue
		}
		normalized[i] = append(json.RawMessage(nil), buf.Bytes()...)
	}
	return normalized
}

func normalizeEvolution(evolution Evolution) Evolution {
	if evolution.Decisions == nil {
		evolution.Decisions = []json.RawMessage{}
	}
	if evolution.Risks == nil {
		evolution.Risks = []json.RawMessage{}
	}
	if evolution.Verification == nil {
		evolution.Verification = []Verification{}
	}
	if evolution.Sessions == nil {
		evolution.Sessions = []Session{}
	}
	if evolution.Timeline == nil {
		evolution.Timeline = []TimelineEntry{}
	}
	if evolution.Extensions == nil {
		evolution.Extensions = map[string]json.RawMessage{}
	}
	return evolution
}

func firstJSONByte(raw json.RawMessage) byte {
	for _, b := range raw {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			return b
		}
	}
	return 0
}

func topLevelKindName(kind byte) string {
	switch kind {
	case '{':
		return "an object"
	case '[':
		return "an array"
	case '"':
		return "a string"
	default:
		return "the expected JSON type"
	}
}
