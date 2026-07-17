package eve

const (
	SnapshotSchemaVersion = "0.2.0"
	CLIVersion            = "0.2.0"
)

type Snapshot struct {
	ID                string          `json:"id"`
	SchemaVersion     string          `json:"schemaVersion"`
	Title             string          `json:"title"`
	Type              string          `json:"type"`
	Summary           string          `json:"summary"`
	UserVisibleChange string          `json:"userVisibleChange,omitempty"`
	Relationships     Relationships   `json:"relationships"`
	Risks             []Risk          `json:"risks"`
	Timeline          []TimelineEntry `json:"timeline"`
	Decisions         []Decision      `json:"decisions"`
	Validation        []Validation    `json:"validation"`
	Verification      *Verification   `json:"verification,omitempty"`
	Artifacts         []Artifact      `json:"artifacts"`
	Implementation    Implementation  `json:"implementation"`
	CreatedAt         string          `json:"createdAt"`
}

type Relationships struct {
	Corrects   []string `json:"corrects"`
	Supersedes []string `json:"supersedes"`
	Reverts    []string `json:"reverts"`
	DependsOn  []string `json:"dependsOn"`
	Related    []string `json:"related"`
}

type Risk struct {
	Title      string `json:"title"`
	Severity   string `json:"severity"`
	Mitigation string `json:"mitigation,omitempty"`
}

type TimelineEntry struct {
	Phase      string `json:"phase"`
	Title      string `json:"title"`
	Summary    string `json:"summary,omitempty"`
	OccurredAt string `json:"occurredAt,omitempty"`
}

type Decision struct {
	Title     string `json:"title"`
	Rationale string `json:"rationale,omitempty"`
}

type Validation struct {
	Command    string `json:"command"`
	Status     string `json:"status"`
	Output     string `json:"output,omitempty"`
	Provenance string `json:"provenance,omitempty"`
}

type Verification struct {
	Status              string                    `json:"status"`
	Profile             string                    `json:"profile,omitempty"`
	Suite               string                    `json:"suite,omitempty"`
	RequiredChecks      []string                  `json:"requiredChecks,omitempty"`
	RanChecks           []string                  `json:"ranChecks,omitempty"`
	CheckResults        []VerificationCheckResult `json:"checkResults,omitempty"`
	SelectedRunID       string                    `json:"selectedRunId,omitempty"`
	RunStartedAt        string                    `json:"runStartedAt,omitempty"`
	RunCompletedAt      string                    `json:"runCompletedAt,omitempty"`
	RunRecordDigest     string                    `json:"runRecordDigest,omitempty"`
	ConfigBlobHash      string                    `json:"configBlobHash,omitempty"`
	ExecutorFingerprint map[string]string         `json:"executorFingerprint,omitempty"`
	RefContext          map[string]any            `json:"refContext,omitempty"`
	PolicyChange        *PolicyChange             `json:"policyChange,omitempty"`
	Integrity           string                    `json:"integrity,omitempty"`
}

type VerificationCheckResult struct {
	CheckID      string `json:"checkId"`
	Status       string `json:"status"`
	ExitCode     int    `json:"exitCode"`
	StartedAt    string `json:"startedAt,omitempty"`
	CompletedAt  string `json:"completedAt,omitempty"`
	Output       string `json:"output,omitempty"`
	OutputBytes  int    `json:"outputBytes,omitempty"`
	OutputDigest string `json:"outputDigest,omitempty"`
	Truncated    bool   `json:"truncated,omitempty"`
}

type PolicyChange struct {
	Changed             bool     `json:"changed"`
	RequirementsReduced bool     `json:"requirementsReduced"`
	PolicyIntroduced    bool     `json:"policyIntroduced,omitempty"`
	ProfileIntroduced   bool     `json:"profileIntroduced,omitempty"`
	ProfileRemoved      bool     `json:"profileRemoved,omitempty"`
	PreviousConfigHash  string   `json:"previousConfigHash,omitempty"`
	CurrentConfigHash   string   `json:"currentConfigHash,omitempty"`
	AddedChecks         []string `json:"addedChecks,omitempty"`
	RemovedChecks       []string `json:"removedChecks,omitempty"`
}

type Artifact struct {
	Type        string `json:"type"`
	URI         string `json:"uri,omitempty"`
	Path        string `json:"path,omitempty"`
	URL         string `json:"url,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Description string `json:"description,omitempty"`
}

type Implementation struct {
	Branch     string   `json:"branch"`
	GitState   string   `json:"gitState"`
	BaseCommit string   `json:"baseCommit,omitempty"`
	Commits    []string `json:"commits"`
	Dirty      bool     `json:"dirty"`
}
