package compile

import (
	"fmt"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/formatdef"
)

// CompileEnumDeltas processes all enum-related changes and generates output
func CompileEnumDeltas(config *MorphediffCompileConfig, changes []diffdef.Change, writer *MorphediffWriter) error {
	migrationContents := make(map[string][]byte)

	for _, change := range changes {
		var content []byte
		var err error

		switch change.Operation {
		case diffdef.OperationAdd:
			content, err = generateAddEnumMigration(*config, change)
		case diffdef.OperationRemove:
			content, err = generateRemoveEnumMigration(*config, change)
		case diffdef.OperationModify:
			content, err = generateModifyEnumMigration(*config, change)
		case diffdef.OperationRename:
			content, err = generateRenameEnumMigration(*config, change)
		case diffdef.OperationDeprecate:
			content, err = generateDeprecateEnumMigration(*config, change)
		default:
			return fmt.Errorf("unsupported operation for enum: %s", change.Operation)
		}

		if err != nil {
			return fmt.Errorf("failed to generate migration for enum change: %w", err)
		}

		migrationName := generateMigrationName(config, "enum", change)
		migrationContents[migrationName] = content
	}

	return writer.WriteAllMigrations(migrationContents)
}

// generateAddEnumMigration generates PostgreSQL migration for adding an enum (CREATE TYPE)
func generateAddEnumMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	enumName := change.Target["enum"]
	if enumName == "" {
		return nil, fmt.Errorf("enum name not found in change target")
	}

	typeName := formatdef.ToSnakeCase(enumName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Add enum: %s", enumName)
	cb.Comment("Classification: %s", change.Classification)

	// Extract enum entries from definition
	entries := []string{}
	if change.Definition != nil {
		if entriesList, ok := change.Definition["entries"].([]interface{}); ok {
			for _, entry := range entriesList {
				if entryMap, ok := entry.(map[string]interface{}); ok {
					if name, ok := entryMap["name"].(string); ok {
						entries = append(entries, fmt.Sprintf("'%s'", formatdef.ToSnakeCase(name)))
					}
				} else if entryStr, ok := entry.(string); ok {
					entries = append(entries, fmt.Sprintf("'%s'", formatdef.ToSnakeCase(entryStr)))
				}
			}
		}
	}

	if len(entries) == 0 {
		entries = []string{"'value1'", "'value2'"} // Placeholder
		cb.Comment("TODO: Add actual enum values from definition")
	}

	cb.Line("CREATE TYPE %s.%s AS ENUM (", schemaName, typeName)
	cb.Indent()
	for i, entry := range entries {
		if i < len(entries)-1 {
			cb.Line("%s,", entry)
		} else {
			cb.Line("%s", entry)
		}
	}
	cb.Dedent()
	cb.Line(");")

	return cb.Build(), nil
}

// generateRemoveEnumMigration generates PostgreSQL migration for removing an enum (DROP TYPE)
func generateRemoveEnumMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	enumName := change.Target["enum"]
	if enumName == "" {
		return nil, fmt.Errorf("enum name not found in change target")
	}

	typeName := formatdef.ToSnakeCase(enumName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Remove enum: %s", enumName)
	cb.Comment("Classification: %s", change.Classification)
	cb.Line("DROP TYPE IF EXISTS %s.%s CASCADE;", schemaName, typeName)

	return cb.Build(), nil
}

// generateModifyEnumMigration generates PostgreSQL migration for modifying an enum
func generateModifyEnumMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	enumName := change.Target["enum"]
	if enumName == "" {
		return nil, fmt.Errorf("enum name not found in change target")
	}

	typeName := formatdef.ToSnakeCase(enumName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Modify enum: %s", enumName)
	cb.Comment("Classification: %s", change.Classification)

	// Process changes from change.Changes map
	if change.Changes != nil {
		// Handle entry additions/removals
		if entriesChange, ok := change.Changes["entries"].(map[string]interface{}); ok {
			// Add new entries
			if added, ok := entriesChange["added"].([]interface{}); ok {
				for _, entry := range added {
					if entryStr, ok := entry.(string); ok {
						cb.Line("ALTER TYPE %s.%s ADD VALUE IF NOT EXISTS '%s';",
							schemaName, typeName, formatdef.ToSnakeCase(entryStr))
					}
				}
			}
			// Note: Removing enum values requires recreating the type in PostgreSQL
			if removed, ok := entriesChange["removed"].([]interface{}); ok && len(removed) > 0 {
				cb.Comment("WARNING: Removing enum values requires recreating the type")
				cb.Comment("TODO: Implement enum value removal (requires type recreation)")
			}
		}
	}

	return cb.Build(), nil
}

// generateRenameEnumMigration generates PostgreSQL migration for renaming an enum (ALTER TYPE)
func generateRenameEnumMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	oldName := change.Target["enum"]
	newName := change.RenamedTo
	if oldName == "" || newName == "" {
		return nil, fmt.Errorf("enum names not found in change")
	}

	oldTypeName := formatdef.ToSnakeCase(oldName)
	newTypeName := formatdef.ToSnakeCase(newName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Rename enum: %s -> %s", oldName, newName)
	cb.Comment("Classification: %s", change.Classification)
	cb.Line("ALTER TYPE %s.%s RENAME TO %s;", schemaName, oldTypeName, newTypeName)

	return cb.Build(), nil
}

// generateDeprecateEnumMigration generates PostgreSQL migration for deprecating an enum
func generateDeprecateEnumMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	enumName := change.Target["enum"]
	if enumName == "" {
		return nil, fmt.Errorf("enum name not found in change target")
	}

	typeName := formatdef.ToSnakeCase(enumName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Deprecate enum: %s", enumName)
	if change.Reason != "" {
		cb.Comment("Reason: %s", change.Reason)
	}

	// Add a comment to the type indicating it's deprecated
	cb.Line("COMMENT ON TYPE %s.%s IS 'DEPRECATED: %s';", schemaName, typeName, change.Reason)

	return cb.Build(), nil
}
