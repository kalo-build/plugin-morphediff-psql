package compile

import (
	"fmt"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/formatdef"
)

// CompileModelDeltas processes all model-related changes and generates output
func CompileModelDeltas(config *MorphediffCompileConfig, changes []diffdef.Change, writer *MorphediffWriter) error {
	migrationContents := make(map[string][]byte)

	for _, change := range changes {
		var content []byte
		var err error

		switch change.Operation {
		case diffdef.OperationAdd:
			content, err = generateAddModelMigration(*config, change)
		case diffdef.OperationRemove:
			content, err = generateRemoveModelMigration(*config, change)
		case diffdef.OperationModify:
			content, err = generateModifyModelMigration(*config, change)
		case diffdef.OperationRename:
			content, err = generateRenameModelMigration(*config, change)
		case diffdef.OperationDeprecate:
			content, err = generateDeprecateModelMigration(*config, change)
		default:
			return fmt.Errorf("unsupported operation for model: %s", change.Operation)
		}

		if err != nil {
			return fmt.Errorf("failed to generate migration for model change: %w", err)
		}

		// Generate migration name with timestamp
		migrationName := generateMigrationName(config, "model", change)
		migrationContents[migrationName] = content
	}

	return writer.WriteAllMigrations(migrationContents)
}

// generateAddModelMigration generates PostgreSQL migration for adding a model (table)
func generateAddModelMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("    ")

	modelName := change.Target["model"]
	if modelName == "" {
		return nil, fmt.Errorf("model name not found in change target")
	}

	// Use ToTableName for pluralized, snake_case table names (matches plugin-morphe-psql-types)
	tableName := formatdef.ToTableName(modelName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Add model: %s", modelName)
	cb.Line("CREATE TABLE IF NOT EXISTS %s.%s (", schemaName, tableName)
	cb.Indent()

	// Add primary key column (id)
	idType := "SERIAL"
	if config.FormatConfig.UseBigSerial {
		idType = "BIGSERIAL"
	}
	cb.Line("id %s PRIMARY KEY,", idType)

	// Add created_at and updated_at timestamps if persistence is enabled
	if config.FormatConfig.EnablePersistence {
		cb.Line("created_at TIMESTAMP NOT NULL DEFAULT NOW(),")
		cb.Line("updated_at TIMESTAMP NOT NULL DEFAULT NOW()")
	}

	cb.Dedent()
	cb.Line(");")
	cb.Line("")

	// Add index on created_at if persistence is enabled
	if config.FormatConfig.EnablePersistence {
		cb.Line("CREATE INDEX IF NOT EXISTS idx_%s_created_at ON %s.%s(created_at);", tableName, schemaName, tableName)
	}

	return cb.Build(), nil
}

// generateRemoveModelMigration generates PostgreSQL migration for removing a model (table)
func generateRemoveModelMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	modelName := change.Target["model"]
	if modelName == "" {
		return nil, fmt.Errorf("model name not found in change target")
	}

	// Use ToTableName for pluralized, snake_case table names (matches plugin-morphe-psql-types)
	tableName := formatdef.ToTableName(modelName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Remove model: %s", modelName)
	cb.Comment("Classification: %s", change.Classification)
	cb.Line("DROP TABLE IF EXISTS %s.%s CASCADE;", schemaName, tableName)

	return cb.Build(), nil
}

// generateModifyModelMigration generates PostgreSQL migration for modifying a model
func generateModifyModelMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	modelName := change.Target["model"]
	if modelName == "" {
		return nil, fmt.Errorf("model name not found in change target")
	}

	// Use ToTableName for pluralized, snake_case table names (matches plugin-morphe-psql-types)
	tableName := formatdef.ToTableName(modelName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Modify model: %s", modelName)
	cb.Comment("Classification: %s", change.Classification)

	// TODO: Process change.Changes to generate specific ALTER TABLE statements
	// For now, add a placeholder
	if change.Changes != nil {
		cb.Comment("TODO: Process specific changes from change.Changes map")
		cb.Comment("Table: %s.%s", schemaName, tableName)
		cb.Comment("Changes: %v", change.Changes)
	}

	return cb.Build(), nil
}

// generateRenameModelMigration generates PostgreSQL migration for renaming a model (table)
func generateRenameModelMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	oldName := change.Target["model"]
	newName := change.RenamedTo
	if oldName == "" || newName == "" {
		return nil, fmt.Errorf("model names not found in change")
	}

	// Use ToTableName for pluralized, snake_case table names (matches plugin-morphe-psql-types)
	oldTableName := formatdef.ToTableName(oldName)
	newTableName := formatdef.ToTableName(newName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Rename model: %s -> %s", oldName, newName)
	cb.Comment("Classification: %s", change.Classification)
	cb.Line("ALTER TABLE %s.%s RENAME TO %s;", schemaName, oldTableName, newTableName)

	return cb.Build(), nil
}

// generateDeprecateModelMigration generates PostgreSQL migration for deprecating a model
func generateDeprecateModelMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	modelName := change.Target["model"]
	if modelName == "" {
		return nil, fmt.Errorf("model name not found in change target")
	}

	// Use ToTableName for pluralized, snake_case table names (matches plugin-morphe-psql-types)
	tableName := formatdef.ToTableName(modelName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Deprecate model: %s", modelName)
	if change.Reason != "" {
		cb.Comment("Reason: %s", change.Reason)
	}

	// Add a comment to the table indicating it's deprecated
	cb.Line("COMMENT ON TABLE %s.%s IS 'DEPRECATED: %s';", schemaName, tableName, change.Reason)

	return cb.Build(), nil
}

// generateMigrationName creates a standardized migration file name with timestamp
// Format: YYYYMMDDHHMMSS_NNN_operation_type_name.sql
// Example: 20260127120000_001_add_relationship_owner.sql
func generateMigrationName(config *MorphediffCompileConfig, artifactType string, change diffdef.Change) string {
	// For rename operations, use RenamedTo as the artifact name
	var artifactName string
	if change.Operation == diffdef.OperationRename && change.RenamedTo != "" {
		artifactName = change.RenamedTo
	} else {
		// Use the spec-correct type-specific target key
		switch artifactType {
		case "model":
			artifactName = change.Target["model"]
		case "enum":
			artifactName = change.Target["enum"]
		case "entity":
			artifactName = change.Target["entity"]
		case "structure":
			artifactName = change.Target["structure"]
		case "field":
			artifactName = change.Target["field"]
		case "relationship":
			artifactName = change.Target["relationship"]
		}

		// Fallback to RenamedTo or unknown
		if artifactName == "" {
			if change.RenamedTo != "" {
				artifactName = change.RenamedTo
			} else {
				artifactName = "unknown"
			}
		}
	}

	// Convert to snake_case for PostgreSQL file naming
	artifactNameSnake := formatdef.ToSnakeCase(artifactName)

	// Build base name: operation_type_name
	baseName := fmt.Sprintf("%s_%s_%s", change.Operation, artifactType, artifactNameSnake)

	// Add timestamp prefix if enabled
	if config.MigrationConfig.UseTimestamp {
		timestamp := config.MigrationConfig.Timestamp.Format("20060102150405")
		seq := config.MigrationConfig.NextSequence()
		return fmt.Sprintf("%s_%03d_%s", timestamp, seq, baseName)
	}

	return baseName
}
