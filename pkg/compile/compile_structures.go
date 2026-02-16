package compile

import (
	"fmt"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/formatdef"
)

// CompileStructureDeltas processes all structure-related changes and generates output.
// Structures in Morphe are value objects (no identity). PostgreSQL does not have a
// direct equivalent, so migrations are minimal placeholders.
func CompileStructureDeltas(config *MorphediffCompileConfig, changes []diffdef.Change, writer *MorphediffWriter) error {
	migrationContents := make(map[string][]byte)

	for _, change := range changes {
		var content []byte
		var err error

		switch change.Operation {
		case diffdef.OperationAdd:
			content, err = generateAddStructureMigration(*config, change)
		case diffdef.OperationRemove:
			content, err = generateRemoveStructureMigration(*config, change)
		case diffdef.OperationModify:
			content, err = generateModifyStructureMigration(*config, change)
		case diffdef.OperationRename:
			content, err = generateRenameStructureMigration(*config, change)
		case diffdef.OperationDeprecate:
			content, err = generateDeprecateStructureMigration(*config, change)
		default:
			return fmt.Errorf("unsupported operation for structure: %s", change.Operation)
		}

		if err != nil {
			return fmt.Errorf("failed to generate migration for structure change: %w", err)
		}

		migrationName := generateMigrationName(config, "structure", change)
		migrationContents[migrationName] = content
	}

	return writer.WriteAllMigrations(migrationContents)
}

// generateAddStructureMigration generates a placeholder migration for adding a structure.
// Structures are typically embedded in models/entities and do not have their own table.
func generateAddStructureMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	structureName := change.Target["structure"]
	if structureName == "" {
		return nil, fmt.Errorf("structure name not found in change target")
	}

	cb.Comment("Add structure: %s", structureName)
	cb.Comment("Classification: %s", change.Classification)
	cb.Comment("Structures are value objects; no dedicated table is created.")
	cb.Comment("Columns are added to parent model/entity tables as needed.")

	return cb.Build(), nil
}

// generateRemoveStructureMigration generates a placeholder migration for removing a structure
func generateRemoveStructureMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	structureName := change.Target["structure"]
	if structureName == "" {
		return nil, fmt.Errorf("structure name not found in change target")
	}

	cb.Comment("Remove structure: %s", structureName)
	cb.Comment("Classification: %s", change.Classification)
	cb.Comment("Embedded columns should be removed from parent tables separately.")

	return cb.Build(), nil
}

// generateModifyStructureMigration generates a placeholder migration for modifying a structure
func generateModifyStructureMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	structureName := change.Target["structure"]
	if structureName == "" {
		return nil, fmt.Errorf("structure name not found in change target")
	}

	cb.Comment("Modify structure: %s", structureName)
	cb.Comment("Classification: %s", change.Classification)

	return cb.Build(), nil
}

// generateRenameStructureMigration generates a placeholder migration for renaming a structure
func generateRenameStructureMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	oldName := change.Target["structure"]
	newName := change.RenamedTo
	if oldName == "" || newName == "" {
		return nil, fmt.Errorf("structure names not found in change")
	}

	cb.Comment("Rename structure: %s -> %s", oldName, newName)
	cb.Comment("Classification: %s", change.Classification)

	return cb.Build(), nil
}

// generateDeprecateStructureMigration generates a placeholder migration for deprecating a structure
func generateDeprecateStructureMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	structureName := change.Target["structure"]
	if structureName == "" {
		return nil, fmt.Errorf("structure name not found in change target")
	}

	cb.Comment("Deprecate structure: %s", structureName)
	if change.Reason != "" {
		cb.Comment("Reason: %s", change.Reason)
	}

	return cb.Build(), nil
}
