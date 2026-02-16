package compile

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
)

// MorphediffToPostgreSQL compiles a Morphe Diff document to PostgreSQL migrations
func MorphediffToPostgreSQL(config MorphediffCompileConfig) error {
	// Load the Morphe Diff document
	diffDoc, err := loadDiffDocument(config.InputPath)
	if err != nil {
		return fmt.Errorf("failed to load morphe diff document: %w", err)
	}

	// Initialize the writer
	writer := NewMorphediffWriter(config.OutputPath)

	// Reset migration counter for this batch
	config.MigrationConfig.ResetCounter()

	// Process all changes in the diff document
	fmt.Printf("Processing %d changes (%d breaking, %d additive, %d safe)...\n",
		diffDoc.Metadata.Summary.TotalChanges,
		diffDoc.Metadata.Summary.Breaking,
		diffDoc.Metadata.Summary.Additive,
		diffDoc.Metadata.Summary.Safe)

	if config.MigrationConfig.UseTimestamp {
		fmt.Printf("Migration timestamp: %s\n", config.MigrationConfig.Timestamp.Format("20060102150405"))
	}

	// Group changes by type and operation for efficient processing
	changesByType := groupChangesByType(diffDoc.Changes)

	// Process each artifact type (pass pointer to share counter state)
	if changes, ok := changesByType[diffdef.TypeModel]; ok {
		fmt.Println("Compiling model changes...")
		if err := CompileModelDeltas(&config, changes, writer); err != nil {
			return fmt.Errorf("failed to compile model deltas: %w", err)
		}
	}

	if changes, ok := changesByType[diffdef.TypeEntity]; ok {
		fmt.Println("Compiling entity changes...")
		if err := CompileEntityDeltas(&config, changes, writer); err != nil {
			return fmt.Errorf("failed to compile entity deltas: %w", err)
		}
	}

	if changes, ok := changesByType[diffdef.TypeEnum]; ok {
		fmt.Println("Compiling enum changes...")
		if err := CompileEnumDeltas(&config, changes, writer); err != nil {
			return fmt.Errorf("failed to compile enum deltas: %w", err)
		}
	}

	if changes, ok := changesByType[diffdef.TypeStructure]; ok {
		fmt.Println("Compiling structure changes...")
		if err := CompileStructureDeltas(&config, changes, writer); err != nil {
			return fmt.Errorf("failed to compile structure deltas: %w", err)
		}
	}

	if changes, ok := changesByType[diffdef.TypeField]; ok {
		fmt.Println("Compiling field changes...")
		if err := CompileFieldDeltas(&config, changes, writer); err != nil {
			return fmt.Errorf("failed to compile field deltas: %w", err)
		}
	}

	if changes, ok := changesByType[diffdef.TypeRelationship]; ok {
		fmt.Println("Compiling relationship changes...")
		if err := CompileRelationshipDeltas(&config, changes, writer); err != nil {
			return fmt.Errorf("failed to compile relationship deltas: %w", err)
		}
	}

	return nil
}

// loadDiffDocument loads a morphe diff YAML file
func loadDiffDocument(path string) (*diffdef.DiffDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read diff file: %w", err)
	}

	var diffDoc diffdef.DiffDocument
	if err := yaml.Unmarshal(data, &diffDoc); err != nil {
		return nil, fmt.Errorf("failed to parse diff YAML: %w", err)
	}

	return &diffDoc, nil
}

// groupChangesByType groups changes by artifact type
func groupChangesByType(changes []diffdef.Change) map[string][]diffdef.Change {
	grouped := make(map[string][]diffdef.Change)
	for _, change := range changes {
		grouped[change.Type] = append(grouped[change.Type], change)
	}
	return grouped
}
