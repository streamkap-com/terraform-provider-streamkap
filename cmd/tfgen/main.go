// Package main provides the tfgen CLI tool for generating Terraform provider
// schemas from backend configuration files.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var backendPath, output, entityType string

	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate Terraform provider schemas from backend configs",
		Long: `Generate Terraform provider schema code by parsing backend
configuration.latest.json files from the Streamkap Python backend repository.

This command reads the configuration schemas for sources, destinations,
and transforms, then generates the corresponding Terraform provider code.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Backend path: %s\n", backendPath)
			fmt.Printf("Output: %s\n", output)
			fmt.Printf("Entity type: %s\n", entityType)
			fmt.Println("\n[Placeholder] Generation not yet implemented.")
		},
	}

	generateCmd.Flags().StringVar(&backendPath, "backend-path", "", "Path to backend repository (required)")
	generateCmd.Flags().StringVar(&output, "output", "internal/generated", "Output directory for generated code")
	generateCmd.Flags().StringVar(&entityType, "entity-type", "all", "Entity type to generate: sources, destinations, transforms, or all")
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
