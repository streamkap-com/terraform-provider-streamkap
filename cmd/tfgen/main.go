// Package main provides the tfgen CLI tool for generating Terraform provider
// schemas from backend configuration files.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// EntityConfig holds the configuration for discovering connectors of a specific entity type.
type EntityConfig struct {
	Type      string // "source", "destination", "transform"
	PluginDir string // relative path from backend root to plugins directory
}

// knownEntities defines the supported entity types and their plugin directories.
var knownEntities = []EntityConfig{
	{Type: "source", PluginDir: "app/sources/plugins"},
	{Type: "destination", PluginDir: "app/destinations/plugins"},
	{Type: "transform", PluginDir: "app/transforms/plugins"},
}

func main() {
	var backendPath, output, entityType, connector string

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate Terraform provider schemas from backend configs",
		Long: `Generate Terraform provider schema code by parsing backend
configuration.latest.json files from the Streamkap Python backend repository.

This command reads the configuration schemas for sources, destinations,
and transforms, then generates the corresponding Terraform provider code.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(backendPath, output, entityType, connector)
		},
	}

	generateCmd.Flags().StringVar(&backendPath, "backend-path", "", "Path to backend repository (required)")
	generateCmd.Flags().StringVar(&output, "output", "internal/generated", "Output directory for generated code")
	generateCmd.Flags().StringVar(&entityType, "entity-type", "all", "Entity type to generate: sources, destinations, transforms, or all")
	generateCmd.Flags().StringVar(&connector, "connector", "", "Specific connector to generate (e.g., postgresql). If empty, generates all connectors.")
	if err := generateCmd.MarkFlagRequired("backend-path"); err != nil {
		fmt.Fprintf(os.Stderr, "Error marking flag required: %v\n", err)
		os.Exit(1)
	}

	rootCmd := &cobra.Command{
		Use:   "tfgen",
		Short: "Terraform provider code generator for Streamkap",
		Long: `tfgen is a code generation tool for the Streamkap Terraform provider.

It parses backend configuration.latest.json files and generates
Terraform provider schema code, reducing manual boilerplate and
ensuring consistency with the backend API.`,
	}

	rootCmd.AddCommand(generateCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// runGenerate executes the generation process.
func runGenerate(backendPath, output, entityType, connector string) error {
	fmt.Printf("Backend path: %s\n", backendPath)
	fmt.Printf("Output: %s\n", output)
	fmt.Printf("Entity type: %s\n", entityType)
	if connector != "" {
		fmt.Printf("Connector: %s\n", connector)
	}
	fmt.Println()

	// Validate backend path exists
	if _, err := os.Stat(backendPath); os.IsNotExist(err) {
		return fmt.Errorf("backend path does not exist: %s", backendPath)
	}

	// Load override configuration
	// The overrides file is located relative to the executable, find it using runtime
	overridesPath := findOverridesPath()
	overrides, err := LoadOverrides(overridesPath)
	if err != nil {
		return fmt.Errorf("failed to load overrides from %s: %w", overridesPath, err)
	}
	if len(overrides.FieldOverrides) > 0 {
		fmt.Printf("Loaded %d field overrides from %s\n\n", len(overrides.FieldOverrides), overridesPath)
	}

	// Load deprecation configuration
	deprecationsPath := findDeprecationsPath()
	deprecations, err := LoadDeprecations(deprecationsPath)
	if err != nil {
		return fmt.Errorf("failed to load deprecations from %s: %w", deprecationsPath, err)
	}
	if len(deprecations.DeprecatedFields) > 0 {
		fmt.Printf("Loaded %d deprecated field definitions from %s\n\n", len(deprecations.DeprecatedFields), deprecationsPath)
	}

	// Determine which entity types to process
	entitiesToProcess := knownEntities
	if entityType != "all" {
		entitiesToProcess = filterEntities(entityType)
		if len(entitiesToProcess) == 0 {
			return fmt.Errorf("unknown entity type: %s (valid: sources, destinations, transforms, all)", entityType)
		}
	}

	// Process each entity type
	var totalGenerated int
	for _, entity := range entitiesToProcess {
		count, err := processEntity(backendPath, output, entity, connector, overrides, deprecations)
		if err != nil {
			return fmt.Errorf("failed to process %s: %w", entity.Type, err)
		}
		totalGenerated += count
	}

	fmt.Printf("\nGeneration complete! Generated %d schema files.\n", totalGenerated)
	return nil
}

// findOverridesPath locates the overrides.json file.
// It searches in the following order:
// 1. Current working directory
// 2. Directory containing the executable
// 3. cmd/tfgen directory (for development)
func findOverridesPath() string {
	candidates := []string{
		"overrides.json",
		"cmd/tfgen/overrides.json",
	}

	// Get executable directory
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "overrides.json"))
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default to current directory (LoadOverrides handles missing file gracefully)
	return "overrides.json"
}

// findDeprecationsPath locates the deprecations.json file.
// It searches in the following order:
// 1. Current working directory
// 2. Directory containing the executable
// 3. cmd/tfgen directory (for development)
func findDeprecationsPath() string {
	candidates := []string{
		"deprecations.json",
		"cmd/tfgen/deprecations.json",
	}

	// Get executable directory
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "deprecations.json"))
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default to current directory (LoadDeprecations handles missing file gracefully)
	return "deprecations.json"
}

// filterEntities returns entities matching the given type filter.
func filterEntities(entityType string) []EntityConfig {
	// Normalize plural forms
	normalized := strings.TrimSuffix(entityType, "s")

	var filtered []EntityConfig
	for _, e := range knownEntities {
		if e.Type == normalized || e.Type+"s" == entityType {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// processEntity processes all connectors for a given entity type.
func processEntity(backendPath, output string, entity EntityConfig, specificConnector string, overrides *OverrideConfig, deprecations *DeprecationConfig) (int, error) {
	pluginDir := filepath.Join(backendPath, entity.PluginDir)

	// Check if plugin directory exists
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		fmt.Printf("Skipping %s: plugin directory not found at %s\n", entity.Type, pluginDir)
		return 0, nil
	}

	// List connector directories
	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read plugin directory %s: %w", pluginDir, err)
	}

	generator := NewGeneratorWithConfig(output, entity.Type, overrides, deprecations)
	var count int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		connectorCode := entry.Name()

		// Skip common/shared directories
		if connectorCode == "__pycache__" || connectorCode == "common" || strings.HasPrefix(connectorCode, "_") {
			continue
		}

		// Filter to specific connector if requested
		if specificConnector != "" && connectorCode != specificConnector {
			continue
		}

		// Look for configuration.latest.json
		configPath := filepath.Join(pluginDir, connectorCode, "configuration.latest.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			fmt.Printf("Skipping %s %s: no configuration.latest.json found\n", entity.Type, connectorCode)
			continue
		}

		// Parse the config
		config, err := ParseConnectorConfig(configPath)
		if err != nil {
			fmt.Printf("Warning: failed to parse %s: %v\n", configPath, err)
			continue
		}

		// Generate the schema
		fmt.Printf("Generating %s_%s.go...\n", entity.Type, connectorCode)
		if err := generator.Generate(config, connectorCode); err != nil {
			return count, fmt.Errorf("failed to generate %s %s: %w", entity.Type, connectorCode, err)
		}
		count++
	}

	return count, nil
}
