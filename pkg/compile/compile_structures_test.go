package compile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kalo-build/plugin-morphediff-psql/pkg/compile"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/diffdef"
	"github.com/stretchr/testify/suite"
)

type CompileStructuresTestSuite struct {
	suite.Suite
	Config  compile.MorphediffCompileConfig
	Writer  *compile.MorphediffWriter
	TempDir string
}

func TestCompileStructuresTestSuite(t *testing.T) {
	suite.Run(t, new(CompileStructuresTestSuite))
}

func (suite *CompileStructuresTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "compile-structures-test-*")
	suite.Nil(err)
	suite.TempDir = tempDir

	suite.Config = compile.DefaultMorphediffCompileConfig("", tempDir)
	suite.Config.MigrationConfig.UseTimestamp = false
	suite.Writer = compile.NewMorphediffWriter(tempDir)
}

func (suite *CompileStructuresTestSuite) TearDownTest() {
	if suite.TempDir != "" {
		os.RemoveAll(suite.TempDir)
	}
}

func (suite *CompileStructuresTestSuite) TestCompileStructureDeltas_AddStructure() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationAdd,
			Type:      diffdef.TypeStructure,
			Target: map[string]string{
				"structure": "Address",
			},
			Classification: diffdef.ClassificationAdditive,
		},
	}

	err := compile.CompileStructureDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "add_structure_address.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "-- Add structure: Address")
}

func (suite *CompileStructuresTestSuite) TestCompileStructureDeltas_RemoveStructure() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationRemove,
			Type:      diffdef.TypeStructure,
			Target: map[string]string{
				"structure": "LegacyAddress",
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileStructureDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "remove_structure_legacy_address.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "-- Remove structure: LegacyAddress")
}

func (suite *CompileStructuresTestSuite) TestCompileStructureDeltas_ModifyStructure() {
	changes := []diffdef.Change{
		{
			Operation: diffdef.OperationModify,
			Type:      diffdef.TypeStructure,
			Target: map[string]string{
				"structure": "Address",
			},
			Classification: diffdef.ClassificationBreaking,
		},
	}

	err := compile.CompileStructureDeltas(&suite.Config, changes, suite.Writer)
	suite.Nil(err)

	migrationPath := filepath.Join(suite.TempDir, "modify_structure_address.sql")
	suite.FileExists(migrationPath)

	content, err := os.ReadFile(migrationPath)
	suite.Nil(err)
	contentStr := string(content)

	suite.Contains(contentStr, "-- Modify structure: Address")
}

func (suite *CompileStructuresTestSuite) TestCompileStructureDeltas_InvalidOperation() {
	changes := []diffdef.Change{
		{
			Operation: "invalid",
			Type:      diffdef.TypeStructure,
			Target: map[string]string{
				"structure": "Address",
			},
		},
	}

	err := compile.CompileStructureDeltas(&suite.Config, changes, suite.Writer)
	suite.NotNil(err)
	suite.Contains(err.Error(), "unsupported operation")
}
