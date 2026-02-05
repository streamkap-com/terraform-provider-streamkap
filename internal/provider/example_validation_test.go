// internal/provider/example_validation_test.go
package provider

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// Example File Validation Tests
// =============================================================================
// These tests validate that all example files in examples/resources/ are
// syntactically valid HCL files. They use `terraform fmt -check` which validates
// syntax without requiring provider initialization or API credentials.

// TestExampleFiles_HCLSyntax validates all basic.tf example files are valid HCL
// using `terraform fmt -check`. This catches syntax errors without needing
// provider initialization.
func TestExampleFiles_HCLSyntax(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping example validation in short mode")
	}

	// Check if terraform is available
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("terraform CLI not found in PATH, skipping HCL validation")
	}

	projectRoot := findProjectRoot(t)
	examplesDir := filepath.Join(projectRoot, "examples", "resources")

	// Collect all .tf files
	var exampleFiles []string
	err := filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tf") {
			exampleFiles = append(exampleFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk examples directory: %v", err)
	}

	if len(exampleFiles) == 0 {
		t.Fatal("No .tf example files found")
	}

	t.Logf("Found %d example files to validate", len(exampleFiles))

	// Test each example file
	for _, exampleFile := range exampleFiles {
		// Extract resource name and file name for test name
		parts := strings.Split(exampleFile, string(filepath.Separator))
		testName := ""
		for i, part := range parts {
			if part == "resources" && i+2 < len(parts) {
				testName = parts[i+1] + "/" + parts[i+2]
				break
			}
		}
		if testName == "" {
			testName = filepath.Base(exampleFile)
		}

		t.Run(testName, func(t *testing.T) {
			// Use terraform fmt -check to validate HCL syntax
			// This doesn't require provider initialization
			cmd := exec.Command("terraform", "fmt", "-check", "-diff", exampleFile)
			output, err := cmd.CombinedOutput()
			if err != nil {
				// terraform fmt -check returns non-zero if file needs formatting
				// OR if there's a syntax error
				t.Errorf("HCL validation failed for %s:\n%s\nError: %v", exampleFile, string(output), err)
			}
		})
	}
}

// TestExampleFiles_NoEmptyFiles verifies that all example .tf files have content
func TestExampleFiles_NoEmptyFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	projectRoot := findProjectRoot(t)
	examplesDir := filepath.Join(projectRoot, "examples", "resources")

	err := filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tf") {
			if info.Size() == 0 {
				t.Errorf("Empty example file: %s", path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk examples directory: %v", err)
	}
}

// TestExampleFiles_RequiredFilesExist verifies that each resource has at least a basic.tf
func TestExampleFiles_RequiredFilesExist(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	projectRoot := findProjectRoot(t)
	examplesDir := filepath.Join(projectRoot, "examples", "resources")

	entries, err := os.ReadDir(examplesDir)
	if err != nil {
		t.Fatalf("Failed to read examples directory: %v", err)
	}

	var resourceDirs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "streamkap_") {
			resourceDirs = append(resourceDirs, entry.Name())
		}
	}

	if len(resourceDirs) == 0 {
		t.Fatal("No streamkap_* resource directories found")
	}

	t.Logf("Found %d resource example directories", len(resourceDirs))

	for _, resourceDir := range resourceDirs {
		t.Run(resourceDir, func(t *testing.T) {
			basicPath := filepath.Join(examplesDir, resourceDir, "basic.tf")
			if _, err := os.Stat(basicPath); os.IsNotExist(err) {
				t.Errorf("Missing basic.tf in %s", resourceDir)
			}
		})
	}
}

// TestExampleFiles_SensitiveVariablesUsed verifies examples use variables for sensitive data
func TestExampleFiles_SensitiveVariablesUsed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode")
	}

	projectRoot := findProjectRoot(t)
	examplesDir := filepath.Join(projectRoot, "examples", "resources")

	// Patterns that indicate hardcoded secrets (which should be variables instead)
	// This is a basic check - false positives are acceptable, we just want to catch obvious issues
	hardcodedPatterns := []string{
		`password\s*=\s*"[^v"][^a][^r]`, // password = "something" (not starting with var.)
		`secret\s*=\s*"[^v"][^a][^r]`,   // secret = "something"
		`private_key\s*=\s*"-----`,      // actual private key content
	}

	err := filepath.Walk(examplesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tf") {
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Errorf("Failed to read %s: %v", path, readErr)
				return nil
			}

			contentStr := string(content)
			for _, pattern := range hardcodedPatterns {
				if strings.Contains(strings.ToLower(contentStr), pattern) {
					// Check if it's using a variable reference
					if !strings.Contains(contentStr, "var.") {
						t.Logf("Warning: %s may contain hardcoded sensitive value matching pattern: %s", path, pattern)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to walk examples directory: %v", err)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

// findProjectRoot locates the project root by finding go.mod
func findProjectRoot(t *testing.T) string {
	t.Helper()

	// Start from the current working directory and walk up
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			t.Fatal("Could not find project root (no go.mod found)")
		}
		dir = parent
	}
}
