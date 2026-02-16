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

	// Verify schema name is used in generated SQL
	generatedPath := filepath.Join(outputPath, "add_model_user.sql")
	suite.FileExists(generatedPath)

	content, err := os.ReadFile(generatedPath)
	suite.Nil(err)

	contentStr := string(content)
	suite.Contains(contentStr, "custom_schema.users", "Should use custom schema name")
	suite.NotContains(contentStr, "public.users", "Should not use default schema")
}
