package compile_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/compile"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
	"github.com/stretchr/testify/suite"
)

type CompileModelsTestSuite struct {
	suite.Suite
	Config  compile.MorphediffCompileConfig
	Writer  *compile.MorphediffWriter
	TempDir string
}

func TestCompileModelsTestSuite(t *testing.T) {
	suite.Run(t, new(CompileModelsTestSuite))
}

func (suite *CompileModelsTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "compile-models-test-*")
	suite.Nil(err)
	suite.TempDir = tempDir

	suite.Config = compile.DefaultMorphediffCompileConfig("", tempDir)
	suite.Config.MigrationConfig.UseTimestamp = false
	suite.Writer = compile.NewMorphediffWriter(tempDir)
}

func (suite *CompileModelsTestSuite) TearDownTest() {
	if suite.TempDir != "" {
		os.RemoveAll(suite.TempDir)
	}
}

func (suite *CompileModelsTestSuite) TestCompileModelDeltas_AddModel() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeModel,
			Target: map[string]string{
				"model": "User",
			},
			Definition: map[string]interface{}{
				"fields": map[string]interface{}{},
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileModelDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	// Verify file was created
	migrationPath := filepath.Join(suite.TempDir, "add_model_user.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "CREATE TABLE IF NOT EXISTS public.users")
	suite.Contains(contentStr, "id SERIAL PRIMARY KEY")
	suite.Contains(contentStr, "created_at TIMESTAMP")
	suite.Contains(contentStr, "updated_at TIMESTAMP")
}

func (suite *CompileModelsTestSuite) TestCompileModelDeltas_RemoveModel() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationRemove,
			Type:      diffdef.TypeModel,
			Target: map[string]string{
				"model": "LegacyProduct",
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileModelDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "remove_model_legacy_product.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "DROP TABLE IF EXISTS public.legacy_products CASCADE")
}

func (suite *CompileModelsTestSuite) TestCompileModelDeltas_RenameModel() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationRename,
			Type:      diffdef.TypeModel,
			Target: map[string]string{
				"model": "OldUser",
			},
			RenamedTo:      "NewUser",
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileModelDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "rename_model_new_user.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TABLE public.old_users RENAME TO new_users")
}

func (suite *CompileModelsTestSuite) TestCompileModelDeltas_DeprecateModel() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationDeprecate,
			Type:      diffdef.TypeModel,
			Target: map[string]string{
				"model": "LegacyModel",
			},
			Reason:         "Replaced by new model",
			Classification: diffdef.ClassificationSafe,
		},
	}

	err := compile.CompileModelDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "deprecate_model_legacy_model.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "COMMENT ON TABLE public.legacy_models")
	suite.Contains(contentStr, "DEPRECATED")
	suite.Contains(contentStr, "Replaced by new model")
}

func (suite *CompileModelsTestSuite) TestCompileModelDeltas_ModifyModel() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationModify,
			Type:      diffdef.TypeModel,
			Target: map[string]string{
				"model": "User",
			},
			Changes: map[string]interface{}{
				"fields": map[string]interface{}{},
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileModelDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "modify_model_user.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "-- Modify model: User")
}

func (suite *CompileModelsTestSuite) TestCompileModelDeltas_MultipleChanges() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeModel,
			Target: map[string]string{
				"model": "User",
			},
			Definition:     map[string]interface{}{},
			Classification: diffdef.ClassificationAdditive,
		},
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeModel,
			Target: map[string]string{
				"model": "Product",
			},
			Definition:     map[string]interface{}{},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileModelDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	// Verify both files were created
	suite.FileExists(filepath.Join(suite.TempDir, "add_model_user.sql"))
	suite.FileExists(filepath.Join(suite.TempDir, "add_model_product.sql"))
}

func (suite *CompileModelsTestSuite) TestCompileModelDeltas_UseBigSerial() {
	suite.Config.FormatConfig.UseBigSerial = true

	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeModel,
			Target: map[string]string{
				"model": "User",
			},
			Definition:     map[string]interface{}{},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileModelDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "add_model_user.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "id BIGSERIAL PRIMARY KEY")
	suite.NotContains(contentStr, "id SERIAL PRIMARY KEY")
}

func (suite *CompileModelsTestSuite) TestCompileModelDeltas_NoPersistence() {
	suite.Config.FormatConfig.EnablePersistence = false

	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeModel,
			Target: map[string]string{
				"model": "User",
			},
			Definition:     map[string]interface{}{},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileModelDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "add_model_user.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.NotContains(contentStr, "created_at")
	suite.NotContains(contentStr, "updated_at")
	suite.NotContains(contentStr, "idx_users_created_at")
}

func (suite *CompileModelsTestSuite) TestCompileModelDeltas_CustomSchema() {
	suite.Config.FormatConfig.SchemaName = "custom_schema"

	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeModel,
			Target: map[string]string{
				"model": "User",
			},
			Definition:     map[string]interface{}{},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileModelDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "add_model_user.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "custom_schema.users")
	suite.NotContains(contentStr, "public.users")
}

func (suite *CompileModelsTestSuite) TestCompileModelDeltas_InvalidOperation() {
	changes := []diffdef.Change{
		{
			Operation: "invalid",
			Type:      diffdef.TypeModel,
			Target: map[string]string{
				"model": "User",
			},
		},
	}

	err := compile.CompileModelDeltas(&suite.Config, changes, suite.Writer)
	suite.NotNil(err)
	suite.Contains(err.Error(), "unsupported operation")
}

func (suite *CompileModelsTestSuite) TestCompileModelDeltas_MissingModelName() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeModel,
			Target:    map[string]string{},
		},
	}

	err := compile.CompileModelDeltas(&suite.Config, changes, suite.Writer)
	suite.NotNil(err)
	suite.Contains(err.Error(), "model name not found")
}

func (suite *CompileModelsTestSuite) TestGenerateMigrationName() {
	change := diffdef.Change{
		Operation: diffdef.OperationAdd,
		Type:      diffdef.TypeModel,
		Target: map[string]string{
			"model": "User",
		},
	}

	// Access the private function through a test helper
	// Since generateMigrationName is private, we test it indirectly
	// by checking the generated file name
	err := compile.CompileModelDeltas(&suite.Config, []diffdef.Change{change}, suite.Writer)
	suite.Nil(err)

	// Check that file was created with expected naming pattern
	files, err := os.ReadDir(suite.TempDir)
	suite.Nil(err)

	found := false
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "add_model_") && strings.HasSuffix(file.Name(), ".sql") {
			found = true
			suite.Contains(file.Name(), "user")
		}
	}
	suite.True(found, "Should find migration file with correct naming pattern")
}
