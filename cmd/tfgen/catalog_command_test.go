package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validCommandCatalog = `{
  "contract_version": 1,
  "producer": "test",
  "digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "types": [{
    "type_id": "mask_field",
    "schema_version": 1,
    "codec_version": 1,
    "publishable": true,
    "authorable": true,
    "role": "transform",
    "hosts": ["destinations"],
    "data_schema": {
      "type": "object",
      "properties": {"field": {"type": "string"}},
      "required": ["field"],
      "additionalProperties": false
    },
	"consumer_requirements": {
	  "version": 1,
	  "schema_keywords": ["additionalProperties", "properties", "required", "type"],
	  "schema_types": ["object", "string"],
	  "schema_formats": [],
	  "ui_elements": [],
	  "ui_controls": [],
	  "secret_addressing": []
	},
    "secret_paths": [],
    "row_keys": {},
    "examples": {"read_projection": {"field": "email"}}
  }]
}`

func writeCommandCatalog(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "catalog.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCatalogCommandWritesOnlyTheValidatedArtifactAndProvenance(t *testing.T) {
	input := writeCommandCatalog(t, validCommandCatalog)
	output := t.TempDir()
	cmd, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetArgs([]string{"catalog", "--output", output, "--smt-catalog", input, "--backend-revision", "backend-abc123"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("catalog command: %v", err)
	}
	copied, err := os.ReadFile(filepath.Join(output, smtCatalogJSONName))
	if err != nil {
		t.Fatal(err)
	}
	if string(copied) != validCommandCatalog {
		t.Error("catalog command must preserve the exact artifact bytes")
	}
	provenance, err := os.ReadFile(filepath.Join(output, smtCatalogGoName))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"backend-abc123", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "mask_field"} {
		if !strings.Contains(string(provenance), want) {
			t.Errorf("generated provenance lacks %q", want)
		}
	}
	entries, err := os.ReadDir(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("catalog command wrote %d files, want only catalog and provenance", len(entries))
	}
}

func TestCatalogCommandRefusesInvalidInputsBeforeWriting(t *testing.T) {
	tests := []struct {
		name     string
		catalog  string
		revision string
	}{
		{name: "unsupported envelope version", catalog: strings.Replace(validCommandCatalog, `"contract_version": 1`, `"contract_version": 2`, 1), revision: "backend-abc123"},
		{name: "unsupported schema", catalog: strings.Replace(validCommandCatalog, `"type": "string"`, `"oneOf": [{"type": "string"}, {"type": "integer"}]`, 1), revision: "backend-abc123"},
		{name: "missing provenance", catalog: validCommandCatalog},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := writeCommandCatalog(t, tt.catalog)
			output := filepath.Join(t.TempDir(), "generated")
			args := []string{"catalog", "--output", output, "--smt-catalog", input}
			if tt.revision != "" {
				args = append(args, "--backend-revision", tt.revision)
			}
			cmd, err := newRootCommand()
			if err != nil {
				t.Fatal(err)
			}
			cmd.SetArgs(args)

			if err := cmd.Execute(); err == nil {
				t.Fatal("catalog command accepted invalid input")
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatalf("validation failure created output: %v", err)
			}
		})
	}
}

// The provenance describes the set the chain schema is built from, which
// includes predicates, not only the publishable transforms.
func TestCatalogProvenanceCountsEveryAuthorableRole(t *testing.T) {
	predicateCatalog := strings.Replace(validCommandCatalog, `"role": "transform"`, `"role": "predicate"`, 1)
	output := t.TempDir()
	cmd, err := newRootCommand()
	if err != nil {
		t.Fatal(err)
	}
	cmd.SetArgs([]string{"catalog", "--output", output, "--smt-catalog", writeCommandCatalog(t, predicateCatalog), "--backend-revision", "backend-abc123"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("catalog command: %v", err)
	}
	provenance, err := os.ReadFile(filepath.Join(output, smtCatalogGoName))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"//\tdestinations authorable types: 1", "//\t  - mask_field"} {
		if !strings.Contains(string(provenance), want) {
			t.Errorf("generated provenance lacks %q:\n%s", want, provenance)
		}
	}
}
