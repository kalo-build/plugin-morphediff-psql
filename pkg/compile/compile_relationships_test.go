package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/compile"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
	"github.com/stretchr/testify/suite"
)

type CompileRelationshipsTestSuite struct {
	suite.Suite
	Config  compile.MorphediffCompileConfig
	Writer  *compile.MorphediffWriter
	TempDir string
}

func TestCompileRelationshipsTestSuite(t *testing.T) {
	suite.Run(t, new(CompileRelationshipsTestSuite))
}

func (suite *CompileRelationshipsTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "compile-relationships-test-*")
	suite.Nil(err)
	suite.TempDir = tempDir

	suite.Config = compile.DefaultMorphediffCompileConfig("", tempDir)
	suite.Config.MigrationConfig.UseTimestamp = false
	suite.Writer = compile.NewMorphediffWriter(tempDir)
}

func (suite *CompileRelationshipsTestSuite) TearDownTest() {
	if suite.TempDir != "" {
		os.RemoveAll(suite.TempDir)
	}
}

func (suite *CompileRelationshipsTestSuite) TestCompileRelationshipDeltas_AddRelationship() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeRelationship,
			Target: map[string]string{
				"relationship": "Profile",
				"model":        "User",
			},
			Definition: map[string]interface{}{
				"target": "UserProfile",
				"type":   "HasOne",
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileRelationshipDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "add_relationship_profile.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TABLE public.users ADD COLUMN IF NOT EXISTS profile_id INTEGER")
	suite.Contains(contentStr, "ALTER TABLE public.users ADD CONSTRAINT fk_users_profile FOREIGN KEY")
	suite.Contains(contentStr, "REFERENCES public.user_profiles(id)")
}

func (suite *CompileRelationshipsTestSuite) TestCompileRelationshipDeltas_AddRelationship_NoTarget() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeRelationship,
			Target: map[string]string{
				"relationship": "Profile",
				"model":        "User",
			},
			Definition:     map[string]interface{}{},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileRelationshipDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "add_relationship_profile.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "TODO: Relationship target not specified")
}

func (suite *CompileRelationshipsTestSuite) TestCompileRelationshipDeltas_RemoveRelationship() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationRemove,
			Type:      diffdef.TypeRelationship,
			Target: map[string]string{
				"relationship": "LegacyProfile",
				"model":        "User",
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileRelationshipDeltas(&suite.Config, changes, suite.Writer)
	if err != nil {
		suite.T().Logf("CompileRelationshipDeltas error: %v", err)
	}
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "remove_relationship_legacy_profile.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "ALTER TABLE public.users DROP CONSTRAINT IF EXISTS fk_users_legacy_profile CASCADE")
}

func (suite *CompileRelationshipsTestSuite) TestCompileRelationshipDeltas_ModifyRelationship_TypeChange() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationModify,
			Type:      diffdef.TypeRelationship,
			Target: map[string]string{
				"relationship": "Address",
				"model":        "User",
			},
			Changes: map[string]interface{}{
				"type": map[string]interface{}{
					"after":  "HasManyPoly",
					"before": "HasMany",
				},
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileRelationshipDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "modify_relationship_address.sql"))
	suite.Nil(err)
	contentStr := string(content)

	// For HasManyPoly (inverse side of polymorphic), we drop the FK and column
	suite.Contains(contentStr, "ALTER TABLE public.users DROP CONSTRAINT IF EXISTS fk_users_address CASCADE")
	suite.Contains(contentStr, "Relationship changed from HasMany to HasManyPoly (polymorphic)")
	suite.Contains(contentStr, "This is the inverse side of a polymorphic relationship")
}

func (suite *CompileRelationshipsTestSuite) TestCompileRelationshipDeltas_ModifyRelationship_ThroughChange() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationModify,
			Type:      diffdef.TypeRelationship,
			Target: map[string]string{
				"relationship": "Commentable",
				"model":        "Comment",
			},
			Changes: map[string]interface{}{
				"through": map[string]interface{}{
					"after":  "Owner",
					"before": "",
				},
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileRelationshipDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "modify_relationship_commentable.sql"))
	suite.Nil(err)
	contentStr := string(content)

	// For through changes without type change to polymorphic, we just note the through
	suite.Contains(contentStr, "Relationship now uses through: Owner")
}

func (suite *CompileRelationshipsTestSuite) TestCompileRelationshipDeltas_InvalidOperation() {
	changes := []diffdef.Change{
		{
			Operation: "invalid",
			Type:      diffdef.TypeRelationship,
			Target: map[string]string{
				"relationship": "Profile",
				"model":        "User",
			},
		},
	}

	err := compile.CompileRelationshipDeltas(&suite.Config, changes, suite.Writer)
	suite.NotNil(err)
	suite.Contains(err.Error(), "unsupported operation")
}

func (suite *CompileRelationshipsTestSuite) TestCompileRelationshipDeltas_MissingRelationshipName() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeRelationship,
			Target: map[string]string{
				"model": "User",
			},
		},
	}

	err := compile.CompileRelationshipDeltas(&suite.Config, changes, suite.Writer)
	suite.NotNil(err)
	suite.Contains(err.Error(), "relationship or model name not found")
}

func (suite *CompileRelationshipsTestSuite) TestCompileRelationshipDeltas_AddPolymorphicRelationship() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeRelationship,
			Target: map[string]string{
				"relationship": "Owner",
				"model":        "Plugin",
			},
			Definition: map[string]interface{}{
				"type": "ForOnePoly",
				"for":  []interface{}{"User", "Organization"},
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileRelationshipDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "add_relationship_owner.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	// Should have polymorphic columns
	suite.Contains(contentStr, "Polymorphic relationship")
	suite.Contains(contentStr, "ADD COLUMN IF NOT EXISTS owner_type VARCHAR(255)")
	suite.Contains(contentStr, "ADD COLUMN IF NOT EXISTS owner_id INTEGER")
	suite.Contains(contentStr, "CREATE INDEX IF NOT EXISTS idx_plugins_owner")
	suite.Contains(contentStr, "CHECK (owner_type IN ('User', 'Organization'))")
}

func (suite *CompileRelationshipsTestSuite) TestCompileRelationshipDeltas_CustomSchema() {
	suite.Config.FormatConfig.SchemaName = "custom_schema"

	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeRelationship,
			Target: map[string]string{
				"relationship": "Profile",
				"model":        "User",
			},
			Definition: map[string]interface{}{
				"target": "UserProfile",
				"type":   "HasOne",
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileRelationshipDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	content, err := os.ReadFile(filepath.Join(suite.TempDir, "add_relationship_profile.sql"))
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "custom_schema.users")
	suite.NotContains(contentStr, "public.users")
}
