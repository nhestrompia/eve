package eve

import "encoding/json"

const (
	ProtocolVersion = 1
	CLIVersion      = "0.1.0"
)

type Evolution struct {
	EVE            EVEHeader                  `json:"eve"`
	Metadata       Metadata                   `json:"metadata"`
	Intent         string                     `json:"intent"`
	Outcome        string                     `json:"outcome"`
	Behavior       Behavior                   `json:"behavior"`
	Decisions      []json.RawMessage          `json:"decisions"`
	Risks          []json.RawMessage          `json:"risks"`
	Verification   []Verification             `json:"verification"`
	Sessions       []Session                  `json:"sessions"`
	Timeline       []TimelineEntry            `json:"timeline"`
	Relationships  Relationships              `json:"relationships"`
	Implementation Implementation             `json:"implementation"`
	Extensions     map[string]json.RawMessage `json:"extensions"`
}

type EVEHeader struct {
	Version int `json:"version"`
}

type Metadata struct {
	ID        string `json:"id,omitempty"`
	Title     string `json:"title,omitempty"`
	Type      string `json:"type,omitempty"`
	Status    string `json:"status"`
	CreatedBy string `json:"created_by,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type Behavior struct {
	Added   []BehaviorClaim `json:"added,omitempty"`
	Changed []BehaviorClaim `json:"changed,omitempty"`
	Removed []BehaviorClaim `json:"removed,omitempty"`
	Fixed   []BehaviorClaim `json:"fixed,omitempty"`
}

type BehaviorClaim struct {
	Description string    `json:"description"`
	Evidence    *Evidence `json:"evidence,omitempty"`
}

type Evidence struct {
	Commits []string `json:"commits,omitempty"`
	Files   []string `json:"files,omitempty"`
	Tests   []string `json:"tests,omitempty"`
}

type Verification struct {
	Type      string `json:"type,omitempty"`
	Status    string `json:"status"`
	Reference string `json:"reference,omitempty"`
}

type Session struct {
	Provider string `json:"provider,omitempty"`
	ID       string `json:"id,omitempty"`
	URI      string `json:"uri,omitempty"`
}

type TimelineEntry struct {
	Timestamp   string `json:"timestamp,omitempty"`
	Actor       *Actor `json:"actor,omitempty"`
	Event       string `json:"event,omitempty"`
	Description string `json:"description,omitempty"`
}

type Actor struct {
	Type     string `json:"type,omitempty"`
	Provider string `json:"provider,omitempty"`
	ID       string `json:"id,omitempty"`
}

type Relationships struct {
	Extends    []string `json:"extends,omitempty"`
	DependsOn  []string `json:"depends_on,omitempty"`
	Corrects   []string `json:"corrects,omitempty"`
	Supersedes []string `json:"supersedes,omitempty"`
	Reverts    []string `json:"reverts,omitempty"`
	Related    []string `json:"related,omitempty"`
}

type Implementation struct {
	Repositories map[string]Repository `json:"repositories,omitempty"`
	Commits      []string              `json:"commits,omitempty"`
	PullRequests []string              `json:"pull_requests,omitempty"`
	FilesChanged int                   `json:"files_changed,omitempty"`
	Insertions   int                   `json:"insertions,omitempty"`
	Deletions    int                   `json:"deletions,omitempty"`
}

type Repository struct {
	Status string `json:"status,omitempty"`
}
