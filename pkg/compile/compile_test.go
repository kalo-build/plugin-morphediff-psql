package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kalo-build/plugin-morphediff-psql/internal/testutils"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/compile"
	"github.com/stretchr/testify/suite"
)

type CompileTestSuite struct {
	suite.Suite
	TestDirPath            string
	TestGroundTruthDirPath string
}

func TestCompileTestSuite(t *testing.T) {
	suite.Run(t, new(CompileTestSuite))
}

func (suite *CompileTestSuite) SetupTest() {
	suite.TestDirPath = testutils.GetTestDirPath()
	suite.TestGroundTruthDirPath = filepath.Join(suite.TestDirPath, "expected")
}

func (suite *CompileTestSuite) TearDownTest() {
	suite.TestDirPath = ""
}

func (suite *CompileTestSuite) TestMorphediffToPostgreSQL() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	inputPath := filepath.Join(suite.TestDirPath, "morphe-diff.yaml")
	outputPath := workingDirPath

	config := compile.DefaultMorphediffCompileConfig(inputPath, outputPath)
	config.MigrationConfig.UseTimestamp = false

	err := compile.MorphediffToPostgreSQL(config)
	suite.Nil(err)

	// Verify output files exist
	expectedFiles := []string{
		"add_model_user.sql",
		"remove_model_legacy_product.sql",
		"add_field_email.sql",
		"remove_field_legacy_id.sql",
		"modify_field_name.sql",
		"add_enum_user_role.sql",
		"add_entity_user_profile.sql",
		"remove_entity_legacy_view.sql",
		"add_relationship_profile.sql",
		"remove_relationship_legacy_profile.sql",
	}

	for _, fileName := range expectedFiles {
		generatedPath := filepath.Join(outputPath, fileName)
		expectedPath := filepath.Join(suite.TestGroundTruthDirPath, fileName)

		suite.FileExists(generatedPath, "Generated file should exist: %s", fileName)
		suite.FileExists(expectedPath, "Expected file should exist: %s", fileName)

		// Compare file contents
		generatedContent, err := os.ReadFile(generatedPath)
		suite.Nil(err)

		expectedContent, err := os.ReadFile(expectedPath)
		suite.Nil(err)

		suite.Equal(string(expectedContent), string(generatedContent),
			"Generated file %s should match expected output", fileName)
	}
}

func (suite *CompileTestSuite) TestMorphediffToPostgreSQL_EmptyDiff() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working-empty")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	// Create an empty diff file
	emptyDiffPath := filepath.Join(suite.TestDirPath, "empty-diff.yaml")
	emptyDiffContent := `metadata:
  spec_version: KA:MD1:YAML1
  source:
    version: base
    timestamp: "2025-01-15T10:30:00Z"
  target:
    version: head
    timestamp: "2025-01-20T14:45:00Z"
  summary:
    total_changes: 0
    breaking: 0
    additive: 0
    safe: 0
  generated_at: "2025-01-20T15:00:00Z"
changes: []
`
	suite.Nil(os.WriteFile(emptyDiffPath, []byte(emptyDiffContent), 0644))
	defer os.Remove(emptyDiffPath)

	config := compile.DefaultMorphediffCompileConfig(emptyDiffPath, workingDirPath)
	config.MigrationConfig.UseTimestamp = false

	err := compile.MorphediffToPostgreSQL(config)
	suite.Nil(err)

	// Should have no output files
	entries, err := os.ReadDir(workingDirPath)
	suite.Nil(err)
	suite.Len(entries, 0, "Should generate no migration files for empty diff")
}

func (suite *CompileTestSuite) TestMorphediffToPostgreSQL_CustomSchema() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working-custom")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	inputPath := filepath.Join(suite.TestDirPath, "morphe-diff.yaml")
	outputPath := workingDirPath

	config := compile.DefaultMorphediffCompileConfig(inputPath, outputPath)
	config.FormatConfig.SchemaName = "custom_schema"
	config.MigrationConfig.UseTimestamp = false

	err := compile.MorphediffToPostgreSQL(config)
	suite.Nil(err)

	// Verify schema name is used in model SQL
	modelPath := filepath.Join(outputPath, "add_model_user.sql")
	suite.FileExists(modelPath)

	modelContent, err := os.ReadFile(modelPath)
	suite.Nil(err)
	suite.Contains(string(modelContent), "custom_schema.users", "Should use custom schema name in model")
	suite.NotContains(string(modelContent), "public.users", "Should not use default schema in model")

	// Verify schema name is used in entity view SQL
	entityPath := filepath.Join(outputPath, "add_entity_user_profile.sql")
	suite.FileExists(entityPath)

	entityContent, err := os.ReadFile(entityPath)
	suite.Nil(err)
	entityStr := string(entityContent)
	suite.Contains(entityStr, "custom_schema.user_profile_entities", "Should use custom schema name in entity view")
	suite.Contains(entityStr, "custom_schema.users", "Should use custom schema in FROM clause")
	suite.Contains(entityStr, "custom_schema.contact_infos", "Should use custom schema in JOIN clause")
	suite.NotContains(entityStr, "public.", "Should not use default schema in entity view")

	// Verify schema name is used in entity drop SQL
	dropPath := filepath.Join(outputPath, "remove_entity_legacy_view.sql")
	suite.FileExists(dropPath)

	dropContent, err := os.ReadFile(dropPath)
	suite.Nil(err)
	suite.Contains(string(dropContent), "custom_schema.legacy_view_entities", "Should use custom schema in DROP VIEW")
}

func (suite *CompileTestSuite) TestMorphediffToPostgreSQL_EntityOnlyDiff() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working-entity-only")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	entityDiffPath := filepath.Join(suite.TestDirPath, "entity-only-diff.yaml")
	entityDiffContent := `metadata:
  spec_version: KA:MD1:YAML1
  source:
    version: base
    timestamp: "2025-01-15T10:30:00Z"
  target:
    version: head
    timestamp: "2025-01-20T14:45:00Z"
  summary:
    total_changes: 2
    breaking: 1
    additive: 1
    safe: 0
    by_type:
      entity: 2
  generated_at: "2025-01-20T15:00:00Z"
changes:
  # Add a new entity with resolved data (multi-model join)
  - operation: add
    type: entity
    target:
      entity: OrderSummary
    definition:
      fields:
        ID:
          type: Order.ID
        CustomerName:
          type: Order.Customer.Name
        Total:
          type: Order.Total
      identifiers:
        primary: ID
      resolved:
        root_model: Order
        field_sources:
          ID:
            model: Order
            field: ID
            type: UUID
          CustomerName:
            model: Customer
            field: Name
            type: String
          Total:
            model: Order
            field: Total
            type: Float
        joins:
          - from_model: Order
            relationship: Customer
            relationship_type: ForOne
            to_model: Customer
    classification: additive

  # Remove an entity entirely
  - operation: remove
    type: entity
    target:
      entity: LegacyReport
    reason: Entity no longer needed
    classification: breaking
`
	suite.Nil(os.WriteFile(entityDiffPath, []byte(entityDiffContent), 0644))
	defer os.Remove(entityDiffPath)

	config := compile.DefaultMorphediffCompileConfig(entityDiffPath, workingDirPath)
	config.MigrationConfig.UseTimestamp = false

	err := compile.MorphediffToPostgreSQL(config)
	suite.Nil(err)

	// Verify all entity migration files exist
	expectedFiles := []string{
		"add_entity_order_summary.sql",
		"remove_entity_legacy_report.sql",
	}
	for _, fileName := range expectedFiles {
		suite.FileExists(filepath.Join(workingDirPath, fileName), "Should generate: %s", fileName)
	}

	// Verify add entity produces CREATE OR REPLACE VIEW with proper JOINs
	addContent, err := os.ReadFile(filepath.Join(workingDirPath, "add_entity_order_summary.sql"))
	suite.Nil(err)
	addSQL := string(addContent)
	suite.Contains(addSQL, "CREATE OR REPLACE VIEW", "Should generate CREATE OR REPLACE VIEW")
	suite.Contains(addSQL, "order_summary_entities", "View name should be order_summary_entities")
	suite.Contains(addSQL, "FROM public.orders", "Should SELECT FROM orders table")
	suite.Contains(addSQL, "LEFT JOIN public.customers", "Should JOIN customers table")
	suite.Contains(addSQL, "customers.name", "Should select customer name")
	suite.Contains(addSQL, "orders.total", "Should select order total")
	suite.Contains(addSQL, "orders.id", "Should select order ID")

	// Verify remove entity produces DROP VIEW
	removeContent, err := os.ReadFile(filepath.Join(workingDirPath, "remove_entity_legacy_report.sql"))
	suite.Nil(err)
	removeSQL := string(removeContent)
	suite.Contains(removeSQL, "DROP VIEW IF EXISTS", "Should generate DROP VIEW")
	suite.Contains(removeSQL, "legacy_report_entities", "View name should be legacy_report_entities")
	suite.Contains(removeSQL, "CASCADE", "Should include CASCADE")
}

func (suite *CompileTestSuite) TestMorphediffToPostgreSQL_EntityFieldChangeRegeneratesView() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working-entity-field")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	fieldDiffPath := filepath.Join(suite.TestDirPath, "entity-field-diff.yaml")
	fieldDiffContent := `metadata:
  spec_version: KA:MD1:YAML1
  source:
    version: base
    timestamp: "2025-01-15T10:30:00Z"
  target:
    version: head
    timestamp: "2025-01-20T14:45:00Z"
  summary:
    total_changes: 1
    breaking: 1
    additive: 0
    safe: 0
    by_type:
      field: 1
  generated_at: "2025-01-20T15:00:00Z"
changes:
  # Modify a field on an entity (type change) - triggers full view regeneration
  - operation: modify
    type: field
    target:
      entity: UserProfile
      field: Email
    changes:
      type:
        before: User.Email
        after: User.ContactInfo.Email
    entity_snapshot:
      fields:
        ID:
          type: User.ID
        Email:
          type: User.ContactInfo.Email
      identifiers:
        primary: ID
      resolved:
        root_model: User
        field_sources:
          ID:
            model: User
            field: ID
            type: AutoIncrement
          Email:
            model: ContactInfo
            field: Email
            type: String
        joins:
          - from_model: User
            relationship: ContactInfo
            relationship_type: HasOne
            to_model: ContactInfo
    classification: breaking
`
	suite.Nil(os.WriteFile(fieldDiffPath, []byte(fieldDiffContent), 0644))
	defer os.Remove(fieldDiffPath)

	config := compile.DefaultMorphediffCompileConfig(fieldDiffPath, workingDirPath)
	config.MigrationConfig.UseTimestamp = false

	err := compile.MorphediffToPostgreSQL(config)
	suite.Nil(err)

	// Entity field modification should produce a view regeneration file
	migrationName := "modify_entity_user_profile_field_email.sql"
	migrationPath := filepath.Join(workingDirPath, migrationName)
	suite.FileExists(migrationPath, "Should generate entity field modification migration: %s", migrationName)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	sql := string(content)

	// View regeneration from entity_snapshot should produce CREATE OR REPLACE VIEW
	suite.Contains(sql, "CREATE OR REPLACE VIEW", "Should regenerate entity view")
	suite.Contains(sql, "user_profile_entities", "View name should be user_profile_entities")
	suite.Contains(sql, "FROM public.users", "Should SELECT FROM root model table")
	suite.Contains(sql, "LEFT JOIN public.contact_infos", "Should JOIN contact_infos via resolved joins")
	suite.Contains(sql, "contact_infos.email", "Should select email from joined table")
	suite.Contains(sql, "users.id", "Should select ID from root model")
}

func (suite *CompileTestSuite) TestMorphediffToPostgreSQL_EntityMixedChanges() {
	workingDirPath := filepath.Join(suite.TestDirPath, "working-entity-mixed")
	suite.Nil(os.MkdirAll(workingDirPath, 0755))
	defer os.RemoveAll(workingDirPath)

	mixedDiffPath := filepath.Join(suite.TestDirPath, "entity-mixed-diff.yaml")
	mixedDiffContent := `metadata:
  spec_version: KA:MD1:YAML1
  source:
    version: base
    timestamp: "2025-01-15T10:30:00Z"
  target:
    version: head
    timestamp: "2025-01-20T14:45:00Z"
  summary:
    total_changes: 4
    breaking: 1
    additive: 3
    safe: 0
    by_type:
      entity: 1
      field: 2
      relationship: 1
  generated_at: "2025-01-20T15:00:00Z"
changes:
  # Add a new entity
  - operation: add
    type: entity
    target:
      entity: InvoiceSummary
    definition:
      fields:
        ID:
          type: Invoice.ID
        Amount:
          type: Invoice.Amount
      identifiers:
        primary: ID
      resolved:
        root_model: Invoice
        field_sources:
          ID:
            model: Invoice
            field: ID
            type: UUID
          Amount:
            model: Invoice
            field: Amount
            type: Float
    classification: additive

  # Add a field to the same entity (triggers view regeneration via entity_snapshot)
  - operation: add
    type: field
    target:
      entity: InvoiceSummary
      field: CustomerEmail
    definition:
      type: Invoice.Customer.Email
    entity_snapshot:
      fields:
        ID:
          type: Invoice.ID
        Amount:
          type: Invoice.Amount
        CustomerEmail:
          type: Invoice.Customer.Email
      identifiers:
        primary: ID
      resolved:
        root_model: Invoice
        field_sources:
          ID:
            model: Invoice
            field: ID
            type: UUID
          Amount:
            model: Invoice
            field: Amount
            type: Float
          CustomerEmail:
            model: Customer
            field: Email
            type: String
        joins:
          - from_model: Invoice
            relationship: Customer
            relationship_type: ForOne
            to_model: Customer
    classification: additive

  # Add another field to the entity
  - operation: add
    type: field
    target:
      entity: InvoiceSummary
      field: DueDate
    definition:
      type: Invoice.DueDate
    entity_snapshot:
      fields:
        ID:
          type: Invoice.ID
        Amount:
          type: Invoice.Amount
        CustomerEmail:
          type: Invoice.Customer.Email
        DueDate:
          type: Invoice.DueDate
      identifiers:
        primary: ID
      resolved:
        root_model: Invoice
        field_sources:
          ID:
            model: Invoice
            field: ID
            type: UUID
          Amount:
            model: Invoice
            field: Amount
            type: Float
          CustomerEmail:
            model: Customer
            field: Email
            type: String
          DueDate:
            model: Invoice
            field: DueDate
            type: Time
        joins:
          - from_model: Invoice
            relationship: Customer
            relationship_type: ForOne
            to_model: Customer
    classification: additive

  # Remove a relationship from the entity
  - operation: remove
    type: relationship
    target:
      entity: InvoiceSummary
      relationship: Vendor
    entity_snapshot:
      fields:
        ID:
          type: Invoice.ID
        Amount:
          type: Invoice.Amount
        CustomerEmail:
          type: Invoice.Customer.Email
        DueDate:
          type: Invoice.DueDate
      identifiers:
        primary: ID
      resolved:
        root_model: Invoice
        field_sources:
          ID:
            model: Invoice
            field: ID
            type: UUID
          Amount:
            model: Invoice
            field: Amount
            type: Float
          CustomerEmail:
            model: Customer
            field: Email
            type: String
          DueDate:
            model: Invoice
            field: DueDate
            type: Time
        joins:
          - from_model: Invoice
            relationship: Customer
            relationship_type: ForOne
            to_model: Customer
    classification: breaking
`
	suite.Nil(os.WriteFile(mixedDiffPath, []byte(mixedDiffContent), 0644))
	defer os.Remove(mixedDiffPath)

	config := compile.DefaultMorphediffCompileConfig(mixedDiffPath, workingDirPath)
	config.MigrationConfig.UseTimestamp = false

	err := compile.MorphediffToPostgreSQL(config)
	suite.Nil(err)

	// Entity-level add should create a view
	addEntityPath := filepath.Join(workingDirPath, "add_entity_invoice_summary.sql")
	suite.FileExists(addEntityPath)
	addContent, err := os.ReadFile(addEntityPath)
	suite.Nil(err)
	suite.Contains(string(addContent), "CREATE OR REPLACE VIEW", "Entity add should create a view")
	suite.Contains(string(addContent), "invoice_summary_entities", "Should use entity view name")

	// Entity field adds should each produce a view regeneration with unique filenames
	fieldAddPath1 := filepath.Join(workingDirPath, "add_entity_invoice_summary_field_customer_email.sql")
	suite.FileExists(fieldAddPath1, "Should generate field add migration for CustomerEmail")
	f1Content, err := os.ReadFile(fieldAddPath1)
	suite.Nil(err)
	suite.Contains(string(f1Content), "CREATE OR REPLACE VIEW", "Entity field add should regenerate view")
	suite.Contains(string(f1Content), "customers.email", "Regenerated view should include new field")
	suite.Contains(string(f1Content), "LEFT JOIN public.customers", "Regenerated view should include join")

	fieldAddPath2 := filepath.Join(workingDirPath, "add_entity_invoice_summary_field_due_date.sql")
	suite.FileExists(fieldAddPath2, "Should generate field add migration for DueDate")
	f2Content, err := os.ReadFile(fieldAddPath2)
	suite.Nil(err)
	suite.Contains(string(f2Content), "CREATE OR REPLACE VIEW", "Entity field add should regenerate view")
	suite.Contains(string(f2Content), "invoices.due_date", "Regenerated view should include due_date")

	// Entity relationship remove should produce a view regeneration
	relRemovePath := filepath.Join(workingDirPath, "remove_entity_invoice_summary_relationship_vendor.sql")
	suite.FileExists(relRemovePath, "Should generate relationship remove migration")
	relContent, err := os.ReadFile(relRemovePath)
	suite.Nil(err)
	suite.Contains(string(relContent), "CREATE OR REPLACE VIEW", "Relationship remove should regenerate view from snapshot")
}
