package compile

import (
	"fmt"
	"time"
)

// MorphediffCompileConfig contains all configuration for compiling Morphe Diff to PostgreSQL
type MorphediffCompileConfig struct {
	// Input path to morphe diff YAML file
	InputPath string

	// Output path for generated files
	OutputPath string

	// Format-specific configuration
	FormatConfig PostgreSQLConfig

	// MigrationConfig controls migration naming
	MigrationConfig MigrationNamingConfig
}

// MigrationNamingConfig controls how migrations are named
type MigrationNamingConfig struct {
	// UseTimestamp adds a timestamp prefix to migration names (default: true)
	// Format: YYYYMMDDHHMMSS_operation_type_name.sql
	UseTimestamp bool

	// Timestamp to use for this batch of migrations (set automatically if not provided)
	Timestamp time.Time

	// Counter for generating sequential numbers within a batch
	counter int
}

// NextSequence returns the next sequence number for a migration within a batch
func (c *MigrationNamingConfig) NextSequence() int {
	c.counter++
	return c.counter
}

// ResetCounter resets the sequence counter (call before processing a new batch)
func (c *MigrationNamingConfig) ResetCounter() {
	c.counter = 0
}

// PostgreSQLConfig contains PostgreSQL-specific configuration options
type PostgreSQLConfig struct {
	// SchemaName is the PostgreSQL schema name (default: "public")
	SchemaName string

	// UseBigSerial uses BIGSERIAL instead of SERIAL for auto-increment columns
	UseBigSerial bool

	// EnablePersistence enables persistence-related features (default: true)
	EnablePersistence bool
}

// DefaultMorphediffCompileConfig creates a default configuration
func DefaultMorphediffCompileConfig(
	diffInputPath string,
	baseOutputDirPath string,
) MorphediffCompileConfig {
	return MorphediffCompileConfig{
		InputPath:  diffInputPath,
		OutputPath: baseOutputDirPath,
		FormatConfig: PostgreSQLConfig{
			SchemaName:        "public",
			UseBigSerial:      false,
			EnablePersistence: true,
		},
		MigrationConfig: MigrationNamingConfig{
			UseTimestamp: true,
			Timestamp:    time.Now(),
		},
	}
}

// Validate checks if the configuration is valid
func (config MorphediffCompileConfig) Validate() error {
	if config.InputPath == "" {
		return fmt.Errorf("input path is required")
	}

	if config.OutputPath == "" {
		return fmt.Errorf("output path is required")
	}

	// Validate PostgreSQL-specific configuration
	if config.FormatConfig.SchemaName == "" {
		return fmt.Errorf("schema name cannot be empty")
	}

	return nil
}

