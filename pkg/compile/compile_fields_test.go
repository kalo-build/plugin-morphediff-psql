package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/compile"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
	"github.com/stretchr/testify/suite"
)

type CompileFieldsTestSuite struct {
	suite.Suite
	Config  compile.MorphediffCompileConfig
	Writer  *compile.MorphediffWriter
	TempDir string
}

func TestCompileFieldsTestSuite(t *testing.T) {
	suite.Run(t, new(CompileFieldsTestSuite))
}

func (suite *CompileFieldsTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "compile-fields-test-*")
	suite.Nil(err)
	suite.TempDir = tempDir

	suite.Config = compile.DefaultMorphediffCompileConfig("", tempDir)
	suite.Config.MigrationConfig.UseTimestamp = false
	suite.Writer = compile.NewMorphediffWriter(tempDir)
}

func (suite *CompileFieldsTestSuite) TearDownTest() {
	if suite.TempDir != "" {
		os.RemoveAll(suite.TempDir)
	}
}

func (suite *CompileFieldsTestSuite) TestCompileFieldDeltas_AddField() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeField,
			Target: map[string]string{
				"field":   "Email",
				"model": "User",
			},
			Definition: map[string]interface{}{
				"type":     "string",
				"required": true,
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileFieldDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "add_field_email.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TABLE public.users ADD COLUMN IF NOT EXISTS email TEXT NOT NULL")
}

func (suite *CompileFieldsTestSuite) TestCompileFieldDeltas_AddField_Nullable() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeField,
			Target: map[string]string{
				"field":   "PhoneNumber",
				"model": "User",
			},
			Definition: map[string]interface{}{
				"type":     "string",
				"required": false,
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileFieldDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "add_field_phone_number.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TABLE public.users ADD COLUMN IF NOT EXISTS phone_number TEXT NULL")
}

func (suite *CompileFieldsTestSuite) TestCompileFieldDeltas_AddField_IntegerType() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeField,
			Target: map[string]string{
				"field":   "Age",
				"model": "User",
			},
			Definition: map[string]interface{}{
				"type":     "integer",
				"required": true,
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileFieldDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "add_field_age.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TABLE public.users ADD COLUMN IF NOT EXISTS age INTEGER NOT NULL")
}

func (suite *CompileFieldsTestSuite) TestCompileFieldDeltas_RemoveField() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationRemove,
			Type:      diffdef.TypeField,
			Target: map[string]string{
				"field":   "LegacyID",
				"model": "User",
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileFieldDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "remove_field_legacy_id.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TABLE public.users DROP COLUMN IF EXISTS legacy_id")
}

func (suite *CompileFieldsTestSuite) TestCompileFieldDeltas_ModifyField_TypeChange() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationModify,
			Type:      diffdef.TypeField,
			Target: map[string]string{
				"field":   "Age",
				"model": "User",
			},
			Changes: map[string]interface{}{
				"type": map[string]interface{}{
					"after":  "integer",
					"before": "string",
				},
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileFieldDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "modify_field_age.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TABLE public.users ALTER COLUMN age TYPE INTEGER")
}

func (suite *CompileFieldsTestSuite) TestCompileFieldDeltas_ModifyField_RequiredChange() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationModify,
			Type:      diffdef.TypeField,
			Target: map[string]string{
				"field":   "Email",
				"model": "User",
			},
			Changes: map[string]interface{}{
				"required": map[string]interface{}{
					"after":  true,
					"before": false,
				},
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileFieldDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "modify_field_email.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TABLE public.users ALTER COLUMN email SET NOT NULL")
}

func (suite *CompileFieldsTestSuite) TestCompileFieldDeltas_ModifyField_RequiredChange_ToNullable() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationModify,
			Type:      diffdef.TypeField,
			Target: map[string]string{
				"field":   "Email",
				"model": "User",
			},
			Changes: map[string]interface{}{
				"required": map[string]interface{}{
					"after":  false,
					"before": true,
				},
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileFieldDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "modify_field_email.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TABLE public.users ALTER COLUMN email DROP NOT NULL")
}

func (suite *CompileFieldsTestSuite) TestCompileFieldDeltas_RenameField() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationRename,
			Type:      diffdef.TypeField,
			Target: map[string]string{
				"field": "OldName",
				"model": "User",
			},
			RenamedTo:      "NewName",
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileFieldDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "rename_field_new_name.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TABLE public.users RENAME COLUMN old_name TO new_name")
}

func (suite *CompileFieldsTestSuite) TestCompileFieldDeltas_TypeMapping() {
	testCases := []struct {
		morpheType string
		expected   string
	}{
		{"string", "TEXT"},
		{"uuid", "TEXT"},
		{"integer", "INTEGER"},
		{"float", "REAL"},
		{"boolean", "BOOLEAN"},
		{"time", "TIMESTAMP"},
		{"date", "TIMESTAMP"},
		{"timestamp", "TIMESTAMP"},
		{"json", "JSONB"},
		{"unknown", "TEXT"}, // Default fallback
	}

	for _, tc := range testCases {
		changes := []diffdef.Change{
			{
				Operation: diffdef.OperationAdd,
				Type:      diffdef.TypeField,
				Target: map[string]string{
					"field":   "TestField",
					"model": "User",
				},
				Definition: map[string]interface{}{
					"type":     tc.morpheType,
					"required": true,
				},
				Classification: diffdef.ClassificationAdditive,
			},
		}

		// Clean temp dir for each test
		os.RemoveAll(suite.TempDir)
		os.MkdirAll(suite.TempDir, 0755)

		err := compile.CompileFieldDeltas(&suite.Config, changes, suite.Writer)
		suite.Nil(err)

		content, err := os.ReadFile(filepath.Join(suite.TempDir, "add_field_test_field.sql"))
		suite.Nil(err)
		contentStr := string(content)

		suite.Contains(contentStr, tc.expected,
			"Type %s should map to %s", tc.morpheType, tc.expected)
	}
}

func (suite *CompileFieldsTestSuite) TestCompileFieldDeltas_InvalidOperation() {
	changes := []diffdef.Change{
		{
			Operation: "invalid",
			Type:      diffdef.TypeField,
			Target: map[string]string{
				"field":   "Email",
				"model": "User",
			},
		},
	}

	err := compile.CompileFieldDeltas(&suite.Config, changes, suite.Writer)
	suite.NotNil(err)
	suite.Contains(err.Error(), "unsupported operation")
}

func (suite *CompileFieldsTestSuite) TestCompileFieldDeltas_MissingFieldName() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeField,
			Target: map[string]string{
				"model": "User",
			},
		},
	}

	err := compile.CompileFieldDeltas(&suite.Config, changes, suite.Writer)
	suite.NotNil(err)
	suite.Contains(err.Error(), "field name not found")
}

func (suite *CompileFieldsTestSuite) TestCompileFieldDeltas_CustomSchema() {
	suite.Config.FormatConfig.SchemaName = "custom_schema"

	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeField,
			Target: map[string]string{
				"field":   "Email",
				"model": "User",
			},
			Definition: map[string]interface{}{
				"type":     "string",
				"required": true,
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileFieldDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "add_field_email.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "custom_schema.users")
	suite.NotContains(contentStr, "public.users")
}
