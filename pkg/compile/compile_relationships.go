package compile

import (
	"fmt"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/formatdef"
)

// CompileRelationshipDeltas processes all relationship-related changes and generates output
func CompileRelationshipDeltas(config *MorphediffCompileConfig, changes []diffdef.Change, writer *MorphediffWriter) error {
	migrationContents := make(map[string][]byte)

	for _, change := range changes {
		var content []byte
		var err error

		switch change.Operation {
		case diffdef.OperationAdd:
			content, err = generateAddRelationshipMigration(*config, change)
		case diffdef.OperationRemove:
			content, err = generateRemoveRelationshipMigration(*config, change)
		case diffdef.OperationModify:
			content, err = generateModifyRelationshipMigration(*config, change)
		default:
			return fmt.Errorf("unsupported operation for relationship: %s", change.Operation)
		}

		if err != nil {
			return fmt.Errorf("failed to generate migration for relationship change: %w", err)
		}

		migrationName := generateMigrationName(config, "relationship", change)
		migrationContents[migrationName] = content
	}

	return writer.WriteAllMigrations(migrationContents)
}

// generateAddRelationshipMigration generates PostgreSQL migration for adding a relationship (foreign key)
func generateAddRelationshipMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	relationshipName := change.Target["relationship"]
	modelName := change.Target["model"]
	if relationshipName == "" || modelName == "" {
		return nil, fmt.Errorf("relationship or model name not found in change target")
	}

	// Use ToTableName for pluralized, snake_case table names (matches plugin-morphe-psql-types)
	tableName := formatdef.ToTableName(modelName)
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Add relationship: %s to model %s", relationshipName, modelName)
	cb.Comment("Classification: %s", change.Classification)

	// Check if this is a polymorphic relationship
	relationType := ""
	if change.Definition != nil {
		if rt, ok := change.Definition["type"].(string); ok {
			relationType = rt
		}
	}

	// Handle polymorphic relationships (ForOnePoly, ForManyPoly)
	if formatdef.IsForwardPolymorphicType(relationType) {
		return generateAddPolymorphicRelationshipMigration(cb, config, change, tableName, relationshipName)
	}

	// Extract target model from definition for regular relationships
	targetModel := ""
	if change.Definition != nil {
		if target, ok := change.Definition["target"].(string); ok {
			targetModel = target
		}
	}

	if targetModel != "" {
		targetTableName := formatdef.ToTableName(targetModel)
		fkColumnName := formatdef.ToForeignKeyColumnName(relationshipName)
		fkName := formatdef.ToForeignKeyConstraintName(tableName, relationshipName)

		// Add foreign key column if it doesn't exist (for HasOne, BelongsTo relationships)
		cb.Line("ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS %s INTEGER;", schemaName, tableName, fkColumnName)
		cb.Line("ALTER TABLE %s.%s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s.%s(id);",
			schemaName, tableName, fkName, fkColumnName, schemaName, targetTableName)
	} else {
		cb.Comment("TODO: Relationship target not specified, manual migration may be required")
	}

	return cb.Build(), nil
}

// generateAddPolymorphicRelationshipMigration handles adding polymorphic relationships (ForOnePoly, ForManyPoly)
func generateAddPolymorphicRelationshipMigration(cb *formatdef.ContentBuilder, config MorphediffCompileConfig, change diffdef.Change, tableName, relationshipName string) ([]byte, error) {
	schemaName := config.FormatConfig.SchemaName

	// Extract the "for" targets (the models this polymorphic relationship can point to)
	var forTargets []string
	if change.Definition != nil {
		if forList, ok := change.Definition["for"].([]interface{}); ok {
			for _, f := range forList {
				if target, ok := f.(string); ok {
					forTargets = append(forTargets, target)
				}
			}
		}
	}

	// Generate polymorphic column names
	typeColumnName := formatdef.ToPolymorphicTypeColumnName(relationshipName)
	idColumnName := formatdef.ToPolymorphicIdColumnName(relationshipName)
	indexName := formatdef.ToPolymorphicIndexName(tableName, relationshipName)

	cb.Comment("Polymorphic relationship - can reference: %v", forTargets)

	// Add polymorphic type column (stores the model name, e.g., "User" or "Organization")
	cb.Line("ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS %s VARCHAR(255) NOT NULL;",
		schemaName, tableName, typeColumnName)

	// Add polymorphic ID column (stores the foreign key ID)
	cb.Line("ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS %s INTEGER NOT NULL;",
		schemaName, tableName, idColumnName)

	// Add composite index for efficient lookups on type + id
	cb.Line("CREATE INDEX IF NOT EXISTS %s ON %s.%s(%s, %s);",
		indexName, schemaName, tableName, typeColumnName, idColumnName)

	// Add a check constraint to ensure the type is one of the valid values
	if len(forTargets) > 0 {
		cb.Line("")
		cb.Comment("Optional: Add check constraint to validate polymorphic type")
		validTypes := ""
		for i, target := range forTargets {
			if i > 0 {
				validTypes += ", "
			}
			validTypes += fmt.Sprintf("'%s'", target)
		}
		constraintName := fmt.Sprintf("chk_%s_%s_type", formatdef.ToSnakeCase(tableName), formatdef.ToSnakeCase(relationshipName))
		cb.Line("ALTER TABLE %s.%s ADD CONSTRAINT %s CHECK (%s IN (%s));",
			schemaName, tableName, constraintName, typeColumnName, validTypes)
	}

	return cb.Build(), nil
}

// generateRemoveRelationshipMigration generates PostgreSQL migration for removing a relationship (foreign key)
func generateRemoveRelationshipMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	relationshipName := change.Target["relationship"]
	modelName := change.Target["model"]
	if relationshipName == "" || modelName == "" {
		return nil, fmt.Errorf("relationship or model name not found in change target")
	}

	// Use ToTableName for pluralized, snake_case table names (matches plugin-morphe-psql-types)
	tableName := formatdef.ToTableName(modelName)
	schemaName := config.FormatConfig.SchemaName
	fkName := formatdef.ToForeignKeyConstraintName(tableName, relationshipName)

	cb.Comment("Remove relationship: %s from model %s", relationshipName, modelName)
	cb.Comment("Classification: %s", change.Classification)

	// Check if this was a polymorphic relationship by looking at the definition
	wasPolymorphic := false
	if change.Definition != nil {
		if relType, ok := change.Definition["type"].(string); ok {
			wasPolymorphic = formatdef.IsForwardPolymorphicType(relType)
		}
	}

	if wasPolymorphic {
		// Remove polymorphic columns and index
		typeColumnName := formatdef.ToPolymorphicTypeColumnName(relationshipName)
		idColumnName := formatdef.ToPolymorphicIdColumnName(relationshipName)
		indexName := formatdef.ToPolymorphicIndexName(tableName, relationshipName)
		constraintName := fmt.Sprintf("chk_%s_%s_type", formatdef.ToSnakeCase(tableName), formatdef.ToSnakeCase(relationshipName))

		cb.Comment("Removing polymorphic relationship columns and constraints")
		cb.Line("DROP INDEX IF EXISTS %s.%s;", schemaName, indexName)
		cb.Line("ALTER TABLE %s.%s DROP CONSTRAINT IF EXISTS %s;", schemaName, tableName, constraintName)
		cb.Line("ALTER TABLE %s.%s DROP COLUMN IF EXISTS %s;", schemaName, tableName, typeColumnName)
		cb.Line("ALTER TABLE %s.%s DROP COLUMN IF EXISTS %s;", schemaName, tableName, idColumnName)
	} else {
		// Regular relationship - drop FK constraint
		cb.Line("ALTER TABLE %s.%s DROP CONSTRAINT IF EXISTS %s CASCADE;", schemaName, tableName, fkName)
		// Also try to drop the FK column
		fkColumnName := formatdef.ToForeignKeyColumnName(relationshipName)
		cb.Line("ALTER TABLE %s.%s DROP COLUMN IF EXISTS %s;", schemaName, tableName, fkColumnName)
	}

	return cb.Build(), nil
}

// generateModifyRelationshipMigration generates PostgreSQL migration for modifying a relationship
func generateModifyRelationshipMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")

	relationshipName := change.Target["relationship"]
	modelName := change.Target["model"]
	if relationshipName == "" || modelName == "" {
		return nil, fmt.Errorf("relationship or model name not found in change target")
	}

	// Use ToTableName for pluralized, snake_case table names (matches plugin-morphe-psql-types)
	tableName := formatdef.ToTableName(modelName)
	schemaName := config.FormatConfig.SchemaName
	fkName := formatdef.ToForeignKeyConstraintName(tableName, relationshipName)

	cb.Comment("Modify relationship: %s in model %s", relationshipName, modelName)
	cb.Comment("Classification: %s", change.Classification)

	// Process changes from change.Changes map
	if change.Changes == nil {
		return cb.Build(), nil
	}

	// Extract type change information
	var beforeType, afterType string
	if typeChange, ok := change.Changes["type"].(map[string]interface{}); ok {
		if before, ok := typeChange["before"].(string); ok {
			beforeType = before
		}
		if after, ok := typeChange["after"].(string); ok {
			afterType = after
		}
	}

	// Extract through change information (for polymorphic relationships)
	var throughAfter string
	if throughChange, ok := change.Changes["through"].(map[string]interface{}); ok {
		if after, ok := throughChange["after"].(string); ok {
			throughAfter = after
		}
	}

	// Handle transition from regular to polymorphic relationship (e.g., HasMany -> HasManyPoly)
	if afterType != "" && formatdef.IsPolymorphicRelationType(afterType) && !formatdef.IsPolymorphicRelationType(beforeType) {
		cb.Comment("Relationship changed from %s to %s (polymorphic)", beforeType, afterType)

		// For HasManyPoly/HasOnePoly on the "inverse" side:
		// The polymorphic columns are stored on the related model, not this one.
		// This model no longer has a direct FK - it's accessed via the polymorphic junction.
		if afterType == "HasManyPoly" || afterType == "HasOnePoly" {
			cb.Comment("This is the inverse side of a polymorphic relationship.")
			cb.Comment("The polymorphic columns (%s_type, %s_id) are stored on the related model.",
				formatdef.ToSnakeCase(throughAfter), formatdef.ToSnakeCase(throughAfter))

			// Drop any existing FK constraint from this table (it's no longer a direct relationship)
			cb.Line("ALTER TABLE %s.%s DROP CONSTRAINT IF EXISTS %s CASCADE;",
				schemaName, tableName, fkName)

			// Also drop the old FK column if it exists
			fkColumnName := formatdef.ToForeignKeyColumnName(relationshipName)
			cb.Line("ALTER TABLE %s.%s DROP COLUMN IF EXISTS %s;",
				schemaName, tableName, fkColumnName)

			cb.Comment("")
			cb.Comment("No new columns needed on this table - access is via the '%s' polymorphic relationship", throughAfter)
			cb.Comment("on the related model which stores %s_type and %s_id columns.",
				formatdef.ToSnakeCase(throughAfter), formatdef.ToSnakeCase(throughAfter))
		}
	} else if afterType != "" {
		// Regular type change (non-polymorphic)
		cb.Comment("Relationship type changed from %s to %s", beforeType, afterType)
		cb.Line("ALTER TABLE %s.%s DROP CONSTRAINT IF EXISTS %s CASCADE;", schemaName, tableName, fkName)
		cb.Comment("TODO: Recreate foreign key constraint based on new relationship type")
	}

	// If there's a through change but no type change to polymorphic, just note it
	if throughAfter != "" && !formatdef.IsPolymorphicRelationType(afterType) {
		cb.Comment("Relationship now uses through: %s", throughAfter)
	}

	return cb.Build(), nil
}

