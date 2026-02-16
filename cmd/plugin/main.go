package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kalo "github.com/kalo-build/kalo-sdk-go"
	"github.com/kalo-build/plugin-morphediff-psql/pkg/compile"
)

// PluginConfig represents the configuration passed to the plugin by Kalo CLI
type PluginConfig struct {
	// New store-based paths (mounted by CLI)
	Stores map[string]StoreConfig `json:"stores,omitempty"`

	// Legacy direct paths (for backward compatibility)
	InputPath  string `json:"inputPath,omitempty"`
	OutputPath string `json:"outputPath,omitempty"`

	// Plugin-specific config
	Config  map[string]interface{} `json:"config,omitempty"`
	Verbose bool                   `json:"verbose,omitempty"`
}

// StoreConfig represents a store configuration from Kalo CLI
type StoreConfig struct {
	ID        uint32 `json:"id"`
	Type      string `json:"type"`
	MountPath string `json:"mountPath,omitempty"`
}

// Exit codes
const (
	ExitSuccess         = 0
	ExitCompileFailed   = 1
	ExitMissingConfig   = 3
	ExitInvalidConfig   = 4
	ExitInputPathError  = 12
	ExitOutputPathError = 13
)

// logInfo prints info messages only when verbose mode is enabled
func logInfo(verbose bool, format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stdout, format+"\n", args...)
	}
}

func main() {
	// Check command line arguments
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: plugin-morphediff-psql <config>")
		fmt.Fprintln(os.Stderr, "  config: JSON string with store configurations")
		os.Exit(ExitMissingConfig)
	}

	// Parse configuration
	rawConfig := os.Args[1]
	var config PluginConfig
	if err := json.Unmarshal([]byte(rawConfig), &config); err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing config JSON:", err)
		os.Exit(ExitInvalidConfig)
	}

	// Determine paths - prefer store mounts, fall back to legacy paths
	var inputPath, outputPath string

	// Check for store-based configuration (new approach)
	if config.Stores != nil {
		for _, store := range config.Stores {
			if store.MountPath == "/input" {
				inputPath = "/input"
			}
			if store.MountPath == "/output" {
				outputPath = "/output"
			}
		}
	}

	// Fall back to legacy direct paths
	if inputPath == "" && config.InputPath != "" {
		inputPath = config.InputPath
	}
	if outputPath == "" && config.OutputPath != "" {
		outputPath = config.OutputPath
	}

	// Validate required paths
	if inputPath == "" {
		fmt.Fprintln(os.Stderr, "Error: input path is required")
		os.Exit(ExitInputPathError)
	}

	if outputPath == "" {
		fmt.Fprintln(os.Stderr, "Error: output path is required")
		os.Exit(ExitOutputPathError)
	}

	// If inputPath is a directory (from store mount), find the diff file(s)
	inputFileInfo, err := os.Stat(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error accessing input path '%s': %v\n", inputPath, err)
		os.Exit(ExitInputPathError)
	}

	var diffFilePath string
	if inputFileInfo.IsDir() {
		// Look for .yaml files in the directory
		entries, err := os.ReadDir(inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input directory '%s': %v\n", inputPath, err)
			os.Exit(ExitInputPathError)
		}

		for _, entry := range entries {
			if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml")) {
				diffFilePath = filepath.Join(inputPath, entry.Name())
				break // Use first YAML file found
			}
		}

		if diffFilePath == "" {
			fmt.Fprintf(os.Stderr, "Error: no YAML files found in input directory '%s'\n", inputPath)
			os.Exit(ExitInputPathError)
		}
	} else {
		diffFilePath = inputPath
	}

	// Get verbose from nested config if present
	verbose := config.Verbose
	if config.Config != nil {
		if v, ok := config.Config["verbose"].(bool); ok {
			verbose = v
		}
	}

	logInfo(verbose, "Processing Morphe Diff document from: '%s'", diffFilePath)
	logInfo(verbose, "Output PostgreSQL migrations to: '%s'", outputPath)

	// Initialize the compile configuration
	logInfo(verbose, "Initializing compile configuration...")
	morphediffConfig := compile.DefaultMorphediffCompileConfig(
		diffFilePath,
		outputPath,
	)

	// Use host system time for migration timestamps
	// This bypasses WASI limitations where real-time clock access is restricted
	morphediffConfig.MigrationConfig.Timestamp = kalo.System.Now()

	// Parse format-specific configuration from config.Config
	if config.Config != nil {
		if schemaName, ok := config.Config["schemaName"].(string); ok {
			morphediffConfig.FormatConfig.SchemaName = schemaName
		}
		if useBigSerial, ok := config.Config["useBigSerial"].(bool); ok {
			morphediffConfig.FormatConfig.UseBigSerial = useBigSerial
		}
	}

	// Validate configuration
	if err := morphediffConfig.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, "Invalid configuration:", err)
		os.Exit(ExitInvalidConfig)
	}

	// Run compilation
	logInfo(verbose, "Starting compilation process...")
	if err := compile.MorphediffToPostgreSQL(morphediffConfig); err != nil {
		fmt.Fprintln(os.Stderr, "Compilation failed:", err)
		os.Exit(ExitCompileFailed)
	}

	logInfo(verbose, "Compilation completed successfully")
	os.Exit(ExitSuccess)
}
