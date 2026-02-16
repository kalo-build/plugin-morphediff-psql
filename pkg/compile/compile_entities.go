package compile

import (
	"fmt"
	"sort"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/formatdef"
)

// CompileEntityDeltas processes all entity-related changes and generates output.
// Entities in Morphe are non-persistent and resolve to underlying model tables.
// In PostgreSQL, entities map to SQL views rather than tables.
func CompileEntityDeltas(config *MorphediffCompileConfig, changes []diffdef.Change, writer *MorphediffWriter) error {
	migrationContents := make(map[string][]byte)

	for _, change := range changes {
		var content []byte
		var err error

		switch change.Operation {
		case diffdef.OperationAdd:
			content, err = generateAddEntityMigration(*config, change)
		case diffdef.OperationRemove:
			content, err = generateRemoveEntityMigration(*config, change)
		case diffdef.OperationModify:
			content, err = generateModifyEntityMigration(*config, change)
		case diffdef.OperationRename:
			content, err = generateRenameEntityMigration(*config, change)
		case diffdef.OperationDeprecate:
			content, err = generateDeprecateEntityMigration(*config, change)
		default:
			return fmt.Errorf("unsupported operation for entity: %s", change.Operation)
		}

		if err != nil {
			return fmt.Errorf("failed to generate migration for entity change: %w", err)
		}

		migrationName := generateMigrationName(config, "entity", change)
		migrationContents[migrationName] = content
	}

	return writer.WriteAllMigrations(migrationContents)
}

// generateAddEntityMigration generates a CREATE OR REPLACE VIEW migration for adding an entity
func generateAddEntityMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	entityName := change.Target["entity"]
	if entityName == "" {
		return nil, fmt.Errorf("entity name not found in change target")
	}

	// Try to build view DDL from the resolved block in the definition
	resolved := extractResolved(change.Definition)
	if resolved != nil {
		return buildViewSQL(config, entityName, change.Classification, resolved)
	}

	// Fallback: no resolved data available, generate a placeholder
	return buildEntityPlaceholder(config, "Add", entityName, change.Classification)
}

// generateRemoveEntityMigration generates a DROP VIEW migration for removing an entity
func generateRemoveEntityMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	entityName := change.Target["entity"]
	if entityName == "" {
		return nil, fmt.Errorf("entity name not found in change target")
	}

	cb := formatdef.NewContentBuilder("  ")
	viewName := formatdef.ToSnakeCase(entityName) + "_entities"
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Remove entity view: %s", entityName)
	cb.Comment("Classification: %s", change.Classification)
	cb.Line("DROP VIEW IF EXISTS %s.%s CASCADE;", schemaName, viewName)

	return cb.Build(), nil
}

// generateModifyEntityMigration generates a CREATE OR REPLACE VIEW migration.
// Since views must be fully regenerated, this uses the entity_snapshot.
func generateModifyEntityMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	entityName := change.Target["entity"]
	if entityName == "" {
		// For entity field/relationship changes, the entity name is in the target
		entityName = change.Target["entity"]
	}
	if entityName == "" {
		return nil, fmt.Errorf("entity name not found in change target")
	}

	// Use entity_snapshot if available (contains full post-change entity with resolution)
	if change.EntitySnapshot != nil {
		resolved := extractResolved(change.EntitySnapshot)
		if resolved != nil {
			return buildViewSQL(config, entityName, change.Classification, resolved)
		}
	}

	// Fallback
	return buildEntityPlaceholder(config, "Modify", entityName, change.Classification)
}

// generateRenameEntityMigration generates an ALTER VIEW RENAME migration
func generateRenameEntityMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	oldName := change.Target["entity"]
	newName := change.RenamedTo
	if oldName == "" || newName == "" {
		return nil, fmt.Errorf("entity names not found in change")
	}

	cb := formatdef.NewContentBuilder("  ")
	oldViewName := formatdef.ToSnakeCase(oldName) + "_entities"
	newViewName := formatdef.ToSnakeCase(newName) + "_entities"
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Rename entity view: %s -> %s", oldName, newName)
	cb.Comment("Classification: %s", change.Classification)
	cb.Line("ALTER VIEW %s.%s RENAME TO %s;", schemaName, oldViewName, newViewName)

	return cb.Build(), nil
}

// generateDeprecateEntityMigration generates a COMMENT ON VIEW migration
func generateDeprecateEntityMigration(config MorphediffCompileConfig, change diffdef.Change) ([]byte, error) {
	entityName := change.Target["entity"]
	if entityName == "" {
		return nil, fmt.Errorf("entity name not found in change target")
	}

	cb := formatdef.NewContentBuilder("  ")
	viewName := formatdef.ToSnakeCase(entityName) + "_entities"
	schemaName := config.FormatConfig.SchemaName

	cb.Comment("Deprecate entity view: %s", entityName)
	if change.Reason != "" {
		cb.Comment("Reason: %s", change.Reason)
	}
	cb.Line("COMMENT ON VIEW %s.%s IS 'DEPRECATED: %s';", schemaName, viewName, change.Reason)

	return cb.Build(), nil
}

// resolvedData holds the parsed resolved block from a definition or entity_snapshot
type resolvedData struct {
	RootModel    string
	FieldSources map[string]fieldSource
	Joins        []joinInfo
}

type fieldSource struct {
	Model string
	Field string
	Type  string
}

type joinInfo struct {
	FromModel        string
	Relationship     string
	RelationshipType string
	ToModel          string
}

// extractResolved parses the resolved block from a definition or entity_snapshot map
func extractResolved(data map[string]interface{}) *resolvedData {
	if data == nil {
		return nil
	}

	resolvedRaw, ok := data["resolved"]
	if !ok {
		return nil
	}
	resolvedMap, ok := resolvedRaw.(map[string]interface{})
	if !ok {
		return nil
	}

	result := &resolvedData{
		FieldSources: make(map[string]fieldSource),
	}

	if rootModel, ok := resolvedMap["root_model"].(string); ok {
		result.RootModel = rootModel
	}
	if result.RootModel == "" {
		return nil
	}

	// Parse field_sources
	if fsRaw, ok := resolvedMap["field_sources"]; ok {
		if fsMap, ok := fsRaw.(map[string]interface{}); ok {
			for fieldName, sourceRaw := range fsMap {
				if sourceMap, ok := sourceRaw.(map[string]interface{}); ok {
					fs := fieldSource{}
					if v, ok := sourceMap["model"].(string); ok {
						fs.Model = v
					}
					if v, ok := sourceMap["field"].(string); ok {
						fs.Field = v
					}
					if v, ok := sourceMap["type"].(string); ok {
						fs.Type = v
					}
					result.FieldSources[fieldName] = fs
				}
			}
		}
	}

	// Parse joins
	if joinsRaw, ok := resolvedMap["joins"]; ok {
		if joinsList, ok := joinsRaw.([]interface{}); ok {
			for _, joinRaw := range joinsList {
				if joinMap, ok := joinRaw.(map[string]interface{}); ok {
					ji := joinInfo{}
					if v, ok := joinMap["from_model"].(string); ok {
						ji.FromModel = v
					}
					if v, ok := joinMap["relationship"].(string); ok {
						ji.Relationship = v
					}
					if v, ok := joinMap["relationship_type"].(string); ok {
						ji.RelationshipType = v
					}
					if v, ok := joinMap["to_model"].(string); ok {
						ji.ToModel = v
					}
					result.Joins = append(result.Joins, ji)
				}
			}
		}
	}

	return result
}

// buildViewSQL generates a CREATE OR REPLACE VIEW statement from resolved entity data
func buildViewSQL(config MorphediffCompileConfig, entityName, classification string, resolved *resolvedData) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")
	schemaName := config.FormatConfig.SchemaName
	viewName := formatdef.ToSnakeCase(entityName) + "_entities"
	rootTable := formatdef.ToTableName(resolved.RootModel)

	cb.Comment("Entity view: %s", entityName)
	cb.Comment("Classification: %s", classification)
	cb.Line("CREATE OR REPLACE VIEW %s.%s AS", schemaName, viewName)
	cb.Line("SELECT")
	cb.Indent()

	// Build SELECT columns sorted by field name for deterministic output
	fieldNames := make([]string, 0, len(resolved.FieldSources))
	for name := range resolved.FieldSources {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	for i, fieldName := range fieldNames {
		source := resolved.FieldSources[fieldName]
		sourceTable := formatdef.ToTableName(source.Model)
		sourceColumn := formatdef.ToSnakeCase(source.Field)
		alias := formatdef.ToSnakeCase(fieldName)

		// If the column name matches the alias, no AS needed
		columnExpr := fmt.Sprintf("%s.%s", sourceTable, sourceColumn)
		if sourceColumn != alias {
			columnExpr = fmt.Sprintf("%s.%s AS %s", sourceTable, sourceColumn, alias)
		}

		if i < len(fieldNames)-1 {
			cb.Line("%s,", columnExpr)
		} else {
			cb.Line("%s", columnExpr)
		}
	}

	cb.Dedent()
	cb.Line("FROM %s.%s", schemaName, rootTable)

	// Add JOIN clauses
	for _, join := range resolved.Joins {
		joinTable := formatdef.ToTableName(join.ToModel)
		rootPK := formatdef.ToSnakeCase(resolved.RootModel) + "_id"

		// For HasOne/HasMany: the related table references the root table's PK
		// Convention: join on root_table.id = join_table.id (matching plugin-morphe-psql-types)
		cb.Line("LEFT JOIN %s.%s", schemaName, joinTable)
		cb.Indent()
		cb.Line("ON %s.id = %s.id;", rootTable, joinTable)
		cb.Dedent()
		_ = rootPK // reserved for future FK-based joins
	}

	// If no joins, terminate the FROM with a semicolon
	if len(resolved.Joins) == 0 {
		// Remove the trailing FROM line and re-add with semicolon
		// Actually, we need to append ; to the FROM line
		// The FROM was already added, just add the semicolon on a new empty line
		cb.Line(";")
	}

	return cb.Build(), nil
}

// buildEntityPlaceholder generates a comment-only placeholder when resolved data is unavailable
func buildEntityPlaceholder(config MorphediffCompileConfig, operation, entityName, classification string) ([]byte, error) {
	cb := formatdef.NewContentBuilder("  ")
	viewName := formatdef.ToSnakeCase(entityName) + "_entities"

	cb.Comment("%s entity view: %s", operation, entityName)
	cb.Comment("View name: %s.%s", config.FormatConfig.SchemaName, viewName)
	cb.Comment("Classification: %s", classification)
	cb.Comment("NOTE: Resolved entity definition not available.")
	cb.Comment("Regenerate this migration with an enriched MorpheDiff document.")

	return cb.Build(), nil
}
