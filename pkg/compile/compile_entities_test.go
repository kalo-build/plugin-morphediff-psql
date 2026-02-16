package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/compile"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
	"github.com/stretchr/testify/suite"
)

type CompileEntitiesTestSuite struct {
	suite.Suite
	Config  compile.MorphediffCompileConfig
	Writer  *compile.MorphediffWriter
	TempDir string
}

func TestCompileEntitiesTestSuite(t *testing.T) {
	suite.Run(t, new(CompileEntitiesTestSuite))
}

func (suite *CompileEntitiesTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "compile-entities-test-*")
	suite.Nil(err)
	suite.TempDir = tempDir

	suite.Config = compile.DefaultMorphediffCompileConfig("", tempDir)
	suite.Config.MigrationConfig.UseTimestamp = false
	suite.Writer = compile.NewMorphediffWriter(tempDir)
}

func (suite *CompileEntitiesTestSuite) TearDownTest() {
	if suite.TempDir != "" {
		os.RemoveAll(suite.TempDir)
	}
}

func (suite *CompileEntitiesTestSuite) TestCompileEntityDeltas_AddEntity_WithResolved() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeEntity,
			Target: map[string]string{
				"entity": "UserProfile",
			},
			Definition: map[string]interface{}{
				"fields": map[string]interface{}{
					"ID": map[string]interface{}{
						"type": "User.ID",
					},
					"Email": map[string]interface{}{
						"type": "User.ContactInfo.Email",
					},
				},
				"identifiers": map[string]interface{}{
					"primary": []string{"ID"},
				},
				"resolved": map[string]interface{}{
					"root_model": "User",
					"field_sources": map[string]interface{}{
						"ID": map[string]interface{}{
							"model": "User",
							"field": "ID",
							"type":  "AutoIncrement",
						},
						"Email": map[string]interface{}{
							"model": "ContactInfo",
							"field": "Email",
							"type":  "String",
						},
					},
					"joins": []interface{}{
						map[string]interface{}{
							"from_model":        "User",
							"relationship":      "ContactInfo",
							"relationship_type": "HasOne",
							"to_model":          "ContactInfo",
						},
					},
				},
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileEntityDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "add_entity_user_profile.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "CREATE OR REPLACE VIEW")
	suite.Contains(contentStr, "user_profile_entities")
	suite.Contains(contentStr, "FROM public.users")
	suite.Contains(contentStr, "LEFT JOIN public.contact_infos")
	suite.Contains(contentStr, "users.id")
	suite.Contains(contentStr, "contact_infos.email")
}

func (suite *CompileEntitiesTestSuite) TestCompileEntityDeltas_AddEntity_WithoutResolved() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeEntity,
			Target: map[string]string{
				"entity": "UserProfile",
			},
			Definition: map[string]interface{}{
				"fields": map[string]interface{}{
					"ID": map[string]interface{}{
						"type": "User.ID",
					},
				},
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileEntityDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "add_entity_user_profile.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	// Without resolved data, a placeholder is generated
	suite.Contains(contentStr, "Add entity view: UserProfile")
	suite.Contains(contentStr, "Resolved entity definition not available")
}

func (suite *CompileEntitiesTestSuite) TestCompileEntityDeltas_RemoveEntity() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationRemove,
			Type:      diffdef.TypeEntity,
			Target: map[string]string{
				"entity": "LegacyEntity",
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileEntityDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "remove_entity_legacy_entity.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "DROP VIEW IF EXISTS public.legacy_entity_entities CASCADE")
}

func (suite *CompileEntitiesTestSuite) TestCompileEntityDeltas_ModifyEntity_WithSnapshot() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationModify,
			Type:      diffdef.TypeEntity,
			Target: map[string]string{
				"entity": "UserProfile",
			},
			EntitySnapshot: map[string]interface{}{
				"fields": map[string]interface{}{
					"ID": map[string]interface{}{
						"type": "User.ID",
					},
					"FullName": map[string]interface{}{
						"type": "User.Name",
					},
				},
				"identifiers": map[string]interface{}{
					"primary": []string{"ID"},
				},
				"resolved": map[string]interface{}{
					"root_model": "User",
					"field_sources": map[string]interface{}{
						"ID": map[string]interface{}{
							"model": "User",
							"field": "ID",
							"type":  "AutoIncrement",
						},
						"FullName": map[string]interface{}{
							"model": "User",
							"field": "Name",
							"type":  "String",
						},
					},
				},
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileEntityDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "modify_entity_user_profile.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "CREATE OR REPLACE VIEW")
	suite.Contains(contentStr, "user_profile_entities")
	suite.Contains(contentStr, "users.name AS full_name")
}

func (suite *CompileEntitiesTestSuite) TestCompileEntityDeltas_RenameEntity() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationRename,
			Type:      diffdef.TypeEntity,
			Target: map[string]string{
				"entity": "OldProfile",
			},
			RenamedTo:      "NewProfile",
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileEntityDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "rename_entity_new_profile.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER VIEW")
	suite.Contains(contentStr, "old_profile_entities")
	suite.Contains(contentStr, "RENAME TO new_profile_entities")
}

func (suite *CompileEntitiesTestSuite) TestCompileEntityDeltas_DeprecateEntity() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationDeprecate,
			Type:      diffdef.TypeEntity,
			Target: map[string]string{
				"entity": "LegacyView",
			},
			Reason:         "Use NewView instead",
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileEntityDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "deprecate_entity_legacy_view.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "COMMENT ON VIEW")
	suite.Contains(contentStr, "legacy_view_entities")
	suite.Contains(contentStr, "DEPRECATED")
}

func (suite *CompileEntitiesTestSuite) TestCompileEntityDeltas_InvalidOperation() {
	changes := []diffdef.Change{
		{
			Operation: "invalid",
			Type:      diffdef.TypeEntity,
			Target: map[string]string{
				"entity": "UserProfile",
			},
		},
	}

	err := compile.CompileEntityDeltas(&suite.Config, changes, suite.Writer)
	suite.NotNil(err)
	suite.Contains(err.Error(), "unsupported operation")
}

func (suite *CompileEntitiesTestSuite) TestCompileEntityDeltas_AddEntity_NoJoins() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeEntity,
			Target: map[string]string{
				"entity": "SimpleProfile",
			},
			Definition: map[string]interface{}{
				"fields": map[string]interface{}{
					"ID": map[string]interface{}{
						"type": "User.ID",
					},
					"Name": map[string]interface{}{
						"type": "User.Name",
					},
				},
				"resolved": map[string]interface{}{
					"root_model": "User",
					"field_sources": map[string]interface{}{
						"ID": map[string]interface{}{
							"model": "User",
							"field": "ID",
							"type":  "AutoIncrement",
						},
						"Name": map[string]interface{}{
							"model": "User",
							"field": "Name",
							"type":  "String",
						},
					},
				},
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileEntityDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "add_entity_simple_profile.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "CREATE OR REPLACE VIEW")
	suite.Contains(contentStr, "simple_profile_entities")
	suite.Contains(contentStr, "FROM public.users")
	// No LEFT JOIN should appear
	suite.NotContains(contentStr, "LEFT JOIN")
}
