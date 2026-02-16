package compile

import (
	"fmt"
	"strings"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/formatdef"
)

// CompileFieldDeltas processes all field-related changes and generates output
func CompileFieldDeltas(config *MorphediffCompileConfig, changes []diffdef.Change, writer *MorphediffWriter) error {
	migrationContents := make(map[string][]byte)

	for _, change := range changes {
		var content []byte
		var err error

		switch change.Operation {
		case diffdef.OperationAdd:
			content, err = generateAddFieldMigration(*config, change)
		case diffdef.OperationRemove:
			content, err = generateRemoveFieldMigration(*config, change)
		case diffdef.OperationModify:
			content, err = generateModifyFieldMigration(*config, change)
		case diffdef.OperationRename:
			content, err = generateRenameFieldMigration(*config, change)
		default:
			return fmt.Errorf("unsupported operation for field: %s", change.Operation)
		}

		if err != nil {
			return fmt.Errorf("failed to generate migration for field change: %w", err)
		}

		migrationName := generateMigrationName(config, "field", change)
		migrationContents[migrationName] = content
	}

	return writer.WriteAllMigrations(migrationContents)
}

// generateAddFieldMigration generates PostgreSQL migration for adding a field (column)
func generateAddFieldMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	fieldName := change.Target["field"]
	parentName := change.Target["model"]
	if fieldName == "" {
		return nil, fmt.Errorf("field name not found in change target")
	}

	// Use ToTableName for pluralized, snake_case table names (matches plugin-morphe-psql-types)
	tableName := formatdef.ToTableName(parentName)
	columnName := formatdef.ToSnakeCase(fieldName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Add field: %s to %s", fieldName, parentName)
	cb.Comment("Classification: %s", change.Classification)

	// Determine column type from change.Definition
	columnType := "TEXT" // Default type
	nullable := "NULL"
	if change.Definition != nil {
		if fieldType, ok := change.Definition["type"].(string); ok {
			columnType = mapMorpheTypeToPostgreSQL(fieldType)
		}
		if required, ok := change.Definition["required"].(bool); ok && required {
			nullable = "NOT NULL"
		}
	}

	cb.Line("ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS %s %s %s;", schemaName, tableName, columnName, columnType, nullable)

	return cb.Build(), nil
}

// mapMorpheTypeToPostgreSQL maps Morphe field types to PostgreSQL column types
func mapMorpheTypeToPostgreSQL(morpheType string) string {
	// Normalize to lowercase for comparison
	lowerType := strings.ToLower(morpheType)
	switch lowerType {
	case "string", "uuid", "protected", "sealed":
		return "TEXT"
	case "integer", "auto_increment":
		return "INTEGER"
	case "float":
		return "REAL"
	case "boolean":
		return "BOOLEAN"
	case "time", "date", "timestamp":
		return "TIMESTAMP"
	case "json":
		return "JSONB"
	default:
		return "TEXT" // Default fallback
	}
}

// generateRemoveFieldMigration generates PostgreSQL migration for removing a field (column)
func generateRemoveFieldMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	fieldName := change.Target["field"]
	parentName := change.Target["model"]
	if fieldName == "" {
		return nil, fmt.Errorf("field name not found in change target")
	}

	// Use ToTableName for pluralized, snake_case table names (matches plugin-morphe-psql-types)
	tableName := formatdef.ToTableName(parentName)
	columnName := formatdef.ToSnakeCase(fieldName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Remove field: %s from %s", fieldName, parentName)
	cb.Comment("Classification: %s", change.Classification)
	cb.Line("ALTER TABLE %s.%s DROP COLUMN IF EXISTS %s;", schemaName, tableName, columnName)

	return cb.Build(), nil
}

// generateModifyFieldMigration generates PostgreSQL migration for modifying a field (column)
func generateModifyFieldMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	fieldName := change.Target["field"]
	parentName := change.Target["model"]
	if fieldName == "" {
		return nil, fmt.Errorf("field name not found in change target")
	}

	// Use ToTableName for pluralized, snake_case table names (matches plugin-morphe-psql-types)
	tableName := formatdef.ToTableName(parentName)
	columnName := formatdef.ToSnakeCase(fieldName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Modify field: %s in %s", fieldName, parentName)
	cb.Comment("Classification: %s", change.Classification)

	// Process changes from change.Changes map
	if change.Changes != nil {
		// Handle type changes
		if typeChange, ok := change.Changes["type"].(map[string]interface{}); ok {
			if newType, ok := typeChange["after"].(string); ok {
				newColumnType := mapMorpheTypeToPostgreSQL(newType)
				cb.Line("ALTER TABLE %s.%s ALTER COLUMN %s TYPE %s;", schemaName, tableName, columnName, newColumnType)
			}
		}

		// Handle nullable changes
		if nullableChange, ok := change.Changes["required"].(map[string]interface{}); ok {
			if required, ok := nullableChange["after"].(bool); ok {
				if required {
					cb.Line("ALTER TABLE %s.%s ALTER COLUMN %s SET NOT NULL;", schemaName, tableName, columnName)
				} else {
					cb.Line("ALTER TABLE %s.%s ALTER COLUMN %s DROP NOT NULL;", schemaName, tableName, columnName)
				}
			}
		}
	}

	return cb.Build(), nil
}

// generateRenameFieldMigration generates PostgreSQL migration for renaming a field (column)
func generateRenameFieldMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	oldName := change.Target["field"]
	newName := change.RenamedTo
	parentName := change.Target["model"]
	if oldName == "" || newName == "" {
		return nil, fmt.Errorf("field names not found in change")
	}

	// Use ToTableName for pluralized, snake_case table names (matches plugin-morphe-psql-types)
	tableName := formatdef.ToTableName(parentName)
	oldColumnName := formatdef.ToSnakeCase(oldName)
	newColumnName := formatdef.ToSnakeCase(newName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Rename field: %s -> %s in %s", oldName, newName, parentName)
	cb.Comment("Classification: %s", change.Classification)
	cb.Line("ALTER TABLE %s.%s RENAME COLUMN %s TO %s;", schemaName, tableName, oldColumnName, newColumnName)

	return cb.Build(), nil
}
