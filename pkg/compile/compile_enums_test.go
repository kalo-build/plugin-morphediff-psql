package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/compile"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
	"github.com/stretchr/testify/suite"
)

type CompileEnumsTestSuite struct {
	suite.Suite
	Config  compile.MorphediffCompileConfig
	Writer  *compile.MorphediffWriter
	TempDir string
}

func TestCompileEnumsTestSuite(t *testing.T) {
	suite.Run(t, new(CompileEnumsTestSuite))
}

func (suite *CompileEnumsTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "compile-enums-test-*")
	suite.Nil(err)
	suite.TempDir = tempDir

	suite.Config = compile.DefaultMorphediffCompileConfig("", tempDir)
	suite.Config.MigrationConfig.UseTimestamp = false
	suite.Writer = compile.NewMorphediffWriter(tempDir)
}

func (suite *CompileEnumsTestSuite) TearDownTest() {
	if suite.TempDir != "" {
		os.RemoveAll(suite.TempDir)
	}
}

func (suite *CompileEnumsTestSuite) TestCompileEnumDeltas_AddEnum() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeEnum,
			Target: map[string]string{
				"enum": "UserRole",
			},
			Definition: map[string]interface{}{
				"entries": []interface{}{
					map[string]interface{}{"name": "Admin", "value": "ADMIN"},
					map[string]interface{}{"name": "Editor", "value": "EDITOR"},
					map[string]interface{}{"name": "Viewer", "value": "VIEWER"},
				},
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileEnumDeltas(&suite.Config, changes, suite.Writer)
	if err != nil {
		suite.T().Logf("CompileEnumDeltas error: %v", err)
	}
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "add_enum_user_role.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "CREATE TYPE public.user_role AS ENUM")
	suite.Contains(contentStr, "'admin'")
	suite.Contains(contentStr, "'editor'")
	suite.Contains(contentStr, "'viewer'")
}

func (suite *CompileEnumsTestSuite) TestCompileEnumDeltas_AddEnum_StringEntries() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeEnum,
			Target: map[string]string{
				"enum": "Status",
			},
			Definition: map[string]interface{}{
				"entries": []interface{}{
					"Active",
					"Inactive",
				},
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileEnumDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "add_enum_status.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "CREATE TYPE public.status AS ENUM")
	suite.Contains(contentStr, "'active'")
	suite.Contains(contentStr, "'inactive'")
}

func (suite *CompileEnumsTestSuite) TestCompileEnumDeltas_RemoveEnum() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationRemove,
			Type:      diffdef.TypeEnum,
			Target: map[string]string{
				"enum": "LegacyStatus",
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileEnumDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "remove_enum_legacy_status.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "DROP TYPE IF EXISTS public.legacy_status CASCADE")
}

func (suite *CompileEnumsTestSuite) TestCompileEnumDeltas_RenameEnum() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationRename,
			Type:      diffdef.TypeEnum,
			Target: map[string]string{
				"enum": "OldRole",
			},
			RenamedTo:      "NewRole",
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileEnumDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "rename_enum_new_role.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TYPE public.old_role RENAME TO new_role")
}

func (suite *CompileEnumsTestSuite) TestCompileEnumDeltas_ModifyEnum_AddEntries() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationModify,
			Type:      diffdef.TypeEnum,
			Target: map[string]string{
				"enum": "UserRole",
			},
			Changes: map[string]interface{}{
				"entries": map[string]interface{}{
					"added": []interface{}{"Moderator"},
				},
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileEnumDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "modify_enum_user_role.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TYPE public.user_role ADD VALUE IF NOT EXISTS 'moderator'")
}

func (suite *CompileEnumsTestSuite) TestCompileEnumDeltas_ModifyEnum_RemoveEntries() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationModify,
			Type:      diffdef.TypeEnum,
			Target: map[string]string{
				"enum": "UserRole",
			},
			Changes: map[string]interface{}{
				"entries": map[string]interface{}{
					"removed": []interface{}{"LegacyRole"},
				},
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileEnumDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "modify_enum_user_role.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "WARNING: Removing enum values requires recreating the type")
}

func (suite *CompileEnumsTestSuite) TestCompileEnumDeltas_DeprecateEnum() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationDeprecate,
			Type:      diffdef.TypeEnum,
			Target: map[string]string{
				"enum": "LegacyRole",
			},
			Reason:         "Use NewRole instead",
			Classification: diffdef.ClassificationSafe,
		},
	}

	err := compile.CompileEnumDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "deprecate_enum_legacy_role.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "COMMENT ON TYPE public.legacy_role")
	suite.Contains(contentStr, "DEPRECATED")
	suite.Contains(contentStr, "Use NewRole instead")
}

func (suite *CompileEnumsTestSuite) TestCompileEnumDeltas_InvalidOperation() {
	changes := []diffdef.Change{
		{
			Operation: "invalid",
			Type:      diffdef.TypeEnum,
			Target: map[string]string{
				"enum": "UserRole",
			},
		},
	}

	err := compile.CompileEnumDeltas(&suite.Config, changes, suite.Writer)
	suite.NotNil(err)
	suite.Contains(err.Error(), "unsupported operation")
}

func (suite *CompileEnumsTestSuite) TestCompileEnumDeltas_MissingEnumName() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeEnum,
			Target:    map[string]string{},
		},
	}

	err := compile.CompileEnumDeltas(&suite.Config, changes, suite.Writer)
	suite.NotNil(err)
	suite.Contains(err.Error(), "enum name not found")
}

func (suite *CompileEnumsTestSuite) TestCompileEnumDeltas_CustomSchema() {
	suite.Config.FormatConfig.SchemaName = "custom_schema"

	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeEnum,
			Target: map[string]string{
				"enum": "UserRole",
			},
			Definition: map[string]interface{}{
				"entries": []interface{}{
					map[string]interface{}{"name": "Admin", "value": "ADMIN"},
				},
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileEnumDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "add_enum_user_role.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "custom_schema.user_role")
	suite.NotContains(contentStr, "public.user_role")
}

func (suite *CompileEnumsTestSuite) TestCompileEnumDeltas_NoEntries() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeEnum,
			Target: map[string]string{
				"enum": "EmptyEnum",
			},
			Definition:     map[string]interface{}{},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileEnumDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "add_enum_empty_enum.sql"))
	suite.Nil(err)
	contentStr := string(content)

	// Should still create the type with placeholder values
	suite.Contains(contentStr, "CREATE TYPE")
	suite.Contains(contentStr, "TODO: Add actual enum values")
}
