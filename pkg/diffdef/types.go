package diffdef

import "time"

// DiffDocument represents a complete morphe diff artifact
type DiffDocument struct {
	Metadata Metadata `yaml:"metadata"`
	Changes  []Change `yaml:"changes"`
}

// Metadata contains information about the diff context
type Metadata struct {
	SpecVersion string      `yaml:"spec_version"`
	Source      VersionInfo `yaml:"source"`
	Target      VersionInfo `yaml:"target"`
	Summary     Summary     `yaml:"summary"`
	GeneratedAt string      `yaml:"generated_at"`
	Generator   string      `yaml:"generator,omitempty"`
}

// VersionInfo describes a schema version
type VersionInfo struct {
	Version   string `yaml:"version"`
	Timestamp string `yaml:"timestamp"`
}

// Summary provides change statistics
type Summary struct {
	TotalChanges int            `yaml:"total_changes"`
	Breaking     int            `yaml:"breaking"`
	Additive     int            `yaml:"additive"`
	Safe         int            `yaml:"safe"`
	ByType       map[string]int `yaml:"by_type,omitempty"`
}

// Change represents a single delta operation
type Change struct {
	Operation      string                 `yaml:"operation"`
	Type           string                 `yaml:"type"`
	Target         map[string]string      `yaml:"target,omitempty"`
	Source         map[string]string      `yaml:"source,omitempty"`
	Destination    map[string]string      `yaml:"destination,omitempty"`
	Definition     map[string]interface{} `yaml:"definition,omitempty"`
	Changes        map[string]interface{} `yaml:"changes,omitempty"`
	EntitySnapshot map[string]interface{} `yaml:"entity_snapshot,omitempty"`
	RenamedTo      string                 `yaml:"renamed_to,omitempty"`
	Fingerprint    string                 `yaml:"fingerprint,omitempty"`
	Reason         string                 `yaml:"reason,omitempty"`
	Classification string                 `yaml:"classification"`
}

// Operation types
const (
	OperationAdd       = "add"
	OperationRemove    = "remove"
	OperationModify    = "modify"
	OperationRename    = "rename"
	OperationMove      = "move"
	OperationDeprecate = "deprecate"
)

// Artifact types
const (
	TypeModel        = "model"
	TypeEntity       = "entity"
	TypeEnum         = "enum"
	TypeStructure    = "structure"
	TypeField        = "field"
	TypeRelationship = "relationship"
	TypeEnumEntry    = "enum_entry"
)

// Classifications
const (
	ClassificationBreaking = "breaking"
	ClassificationAdditive = "additive"
	ClassificationSafe     = "safe"
)

// NewDiffDocument creates a new diff document with metadata
func NewDiffDocument(baseVersion, headVersion string) *DiffDocument {
	now := time.Now().UTC().Format(time.RFC3339)

	return &DiffDocument{
		Metadata: Metadata{
			SpecVersion: "KA:MD1:YAML1",
			Source: VersionInfo{
				Version:   baseVersion,
				Timestamp: now,
			},
			Target: VersionInfo{
				Version:   headVersion,
				Timestamp: now,
			},
			Summary: Summary{
				ByType: make(map[string]int),
			},
			GeneratedAt: now,
			Generator:   "morphe-git-morphediff@1.0.0",
		},
		Changes: make([]Change, 0),
	}
}

// AddChange adds a change and updates the summary
func (d *DiffDocument) AddChange(change Change) {
	d.Changes = append(d.Changes, change)
	d.Metadata.Summary.TotalChanges++

	// Update classification counts
	switch change.Classification {
	case ClassificationBreaking:
		d.Metadata.Summary.Breaking++
	case ClassificationAdditive:
		d.Metadata.Summary.Additive++
	case ClassificationSafe:
		d.Metadata.Summary.Safe++
	}

	// Update type counts
	d.Metadata.Summary.ByType[change.Type]++
}
