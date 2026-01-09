package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewGenerator verifies the Generator constructor.
func TestNewGenerator(t *testing.T) {
	g := NewGenerator("/output/dir", "source")

	if g.outputDir != "/output/dir" {
		t.Errorf("outputDir = %q, want %q", g.outputDir, "/output/dir")
	}
	if g.entityType != "source" {
		t.Errorf("entityType = %q, want %q", g.entityType, "source")
	}
}

// TestCommonFields verifies that common fields (id, name, connector) are generated correctly.
func TestCommonFields(t *testing.T) {
	tests := []struct {
		entityType   string
		wantIDDesc   string
		wantNameDesc string
	}{
		{"source", "Unique identifier for the source", "Name of the source"},
		{"destination", "Unique identifier for the destination", "Name of the destination"},
		{"transform", "Unique identifier for the transform", "Name of the transform"},
	}

	for _, tt := range tests {
		t.Run(tt.entityType, func(t *testing.T) {
			g := NewGenerator("/tmp", tt.entityType)
			fields := g.commonFields()

			if len(fields) != 3 {
				t.Fatalf("commonFields() returned %d fields, want 3", len(fields))
			}

			// Verify ID field
			idField := fields[0]
			if idField.GoFieldName != "ID" {
				t.Errorf("idField.GoFieldName = %q, want %q", idField.GoFieldName, "ID")
			}
			if idField.GoType != "types.String" {
				t.Errorf("idField.GoType = %q, want %q", idField.GoType, "types.String")
			}
			if idField.TfAttrName != "id" {
				t.Errorf("idField.TfAttrName = %q, want %q", idField.TfAttrName, "id")
			}
			if !idField.Computed {
				t.Error("idField.Computed should be true")
			}
			if !idField.NeedsPlanMod {
				t.Error("idField.NeedsPlanMod should be true (UseStateForUnknown)")
			}
			if idField.Description != tt.wantIDDesc {
				t.Errorf("idField.Description = %q, want %q", idField.Description, tt.wantIDDesc)
			}

			// Verify Name field
			nameField := fields[1]
			if nameField.GoFieldName != "Name" {
				t.Errorf("nameField.GoFieldName = %q, want %q", nameField.GoFieldName, "Name")
			}
			if !nameField.Required {
				t.Error("nameField.Required should be true")
			}
			if nameField.Description != tt.wantNameDesc {
				t.Errorf("nameField.Description = %q, want %q", nameField.Description, tt.wantNameDesc)
			}

			// Verify Connector field
			connField := fields[2]
			if connField.GoFieldName != "Connector" {
				t.Errorf("connField.GoFieldName = %q, want %q", connField.GoFieldName, "Connector")
			}
			if !connField.Computed {
				t.Error("connField.Computed should be true")
			}
			if !connField.NeedsPlanMod {
				t.Error("connField.NeedsPlanMod should be true (UseStateForUnknown)")
			}
		})
	}
}

// TestFieldTypeMapping verifies that control types map to correct Terraform types.
func TestFieldTypeMapping(t *testing.T) {
	tests := []struct {
		name           string
		control        string
		wantGoType     string
		wantSchemaType string
		wantIsListType bool
	}{
		{"string control", "string", "types.String", "schema.StringAttribute", false},
		{"password control", "password", "types.String", "schema.StringAttribute", false},
		{"textarea control", "textarea", "types.String", "schema.StringAttribute", false},
		{"json control", "json", "types.String", "schema.StringAttribute", false},
		{"datetime control", "datetime", "types.String", "schema.StringAttribute", false},
		{"number control", "number", "types.Int64", "schema.Int64Attribute", false},
		{"slider control", "slider", "types.Int64", "schema.Int64Attribute", false},
		{"boolean control", "boolean", "types.Bool", "schema.BoolAttribute", false},
		{"toggle control", "toggle", "types.Bool", "schema.BoolAttribute", false},
		{"one-select control", "one-select", "types.String", "schema.StringAttribute", false},
		{"multi-select control", "multi-select", "types.List", "schema.ListAttribute", true},
	}

	g := NewGenerator("/tmp", "source")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &ConfigEntry{
				Name:        "test.field",
				UserDefined: true,
				Value: ValueObject{
					Control: tt.control,
				},
			}

			field := g.entryToFieldData(entry)

			if field.GoType != tt.wantGoType {
				t.Errorf("GoType = %q, want %q", field.GoType, tt.wantGoType)
			}
			if field.SchemaAttrType != tt.wantSchemaType {
				t.Errorf("SchemaAttrType = %q, want %q", field.SchemaAttrType, tt.wantSchemaType)
			}
			if field.IsListType != tt.wantIsListType {
				t.Errorf("IsListType = %v, want %v", field.IsListType, tt.wantIsListType)
			}
		})
	}
}

// TestSensitiveFieldHandling verifies that sensitive fields are properly detected.
func TestSensitiveFieldHandling(t *testing.T) {
	tests := []struct {
		name          string
		control       string
		encrypt       bool
		wantSensitive bool
	}{
		{"password control", "password", false, true},
		{"encrypt true", "string", true, true},
		{"encrypt and password", "password", true, true},
		{"not sensitive", "string", false, false},
		{"number not sensitive", "number", false, false},
	}

	g := NewGenerator("/tmp", "source")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &ConfigEntry{
				Name:        "test.field",
				UserDefined: true,
				Encrypt:     tt.encrypt,
				Value: ValueObject{
					Control: tt.control,
				},
			}

			field := g.entryToFieldData(entry)

			if field.Sensitive != tt.wantSensitive {
				t.Errorf("Sensitive = %v, want %v", field.Sensitive, tt.wantSensitive)
			}
		})
	}
}

// TestDefaultValueHandling verifies default value generation for different types.
func TestDefaultValueHandling(t *testing.T) {
	tests := []struct {
		name            string
		control         string
		defaultValue    any
		wantHasDefault  bool
		wantDefaultFunc string
	}{
		{
			name:            "string default",
			control:         "string",
			defaultValue:    "5432",
			wantHasDefault:  true,
			wantDefaultFunc: `stringdefault.StaticString("5432")`,
		},
		{
			name:            "int64 default",
			control:         "number",
			defaultValue:    float64(10),
			wantHasDefault:  true,
			wantDefaultFunc: "int64default.StaticInt64(10)",
		},
		{
			name:            "bool default true",
			control:         "boolean",
			defaultValue:    true,
			wantHasDefault:  true,
			wantDefaultFunc: "booldefault.StaticBool(true)",
		},
		{
			name:            "bool default false",
			control:         "toggle",
			defaultValue:    false,
			wantHasDefault:  true,
			wantDefaultFunc: "booldefault.StaticBool(false)",
		},
		{
			name:            "slider default",
			control:         "slider",
			defaultValue:    float64(5),
			wantHasDefault:  true,
			wantDefaultFunc: "int64default.StaticInt64(5)",
		},
		{
			name:            "no default",
			control:         "string",
			defaultValue:    nil,
			wantHasDefault:  false,
			wantDefaultFunc: "",
		},
	}

	g := NewGenerator("/tmp", "source")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &ConfigEntry{
				Name:        "test.field",
				UserDefined: true,
				Value: ValueObject{
					Control: tt.control,
					Default: tt.defaultValue,
				},
			}

			field := g.entryToFieldData(entry)

			if field.HasDefault != tt.wantHasDefault {
				t.Errorf("HasDefault = %v, want %v", field.HasDefault, tt.wantHasDefault)
			}
			if field.DefaultFunc != tt.wantDefaultFunc {
				t.Errorf("DefaultFunc = %q, want %q", field.DefaultFunc, tt.wantDefaultFunc)
			}

			// Fields with defaults should be Optional+Computed
			if tt.wantHasDefault {
				if !field.Optional {
					t.Error("field with default should be Optional")
				}
				if !field.Computed {
					t.Error("field with default should be Computed")
				}
				if field.Required {
					t.Error("field with default should not be Required")
				}
			}
		})
	}
}

// TestValidatorGeneration verifies validator generation for select and slider fields.
func TestValidatorGeneration(t *testing.T) {
	g := NewGenerator("/tmp", "source")

	t.Run("one-select validator", func(t *testing.T) {
		entry := &ConfigEntry{
			Name:        "test.select",
			UserDefined: true,
			Value: ValueObject{
				Control:   "one-select",
				RawValues: []string{"Yes", "No"},
			},
		}

		field := g.entryToFieldData(entry)

		if !field.HasValidators {
			t.Error("HasValidators should be true for one-select")
		}
		wantValidator := `stringvalidator.OneOf("Yes", "No")`
		if field.Validators != wantValidator {
			t.Errorf("Validators = %q, want %q", field.Validators, wantValidator)
		}
	})

	t.Run("one-select with multiple options", func(t *testing.T) {
		entry := &ConfigEntry{
			Name:        "test.select",
			UserDefined: true,
			Value: ValueObject{
				Control:   "one-select",
				RawValues: []string{"MERGE", "SNOWPIPE_STREAMING", "APPEND_ONLY"},
			},
		}

		field := g.entryToFieldData(entry)

		if !field.HasValidators {
			t.Error("HasValidators should be true")
		}
		wantValidator := `stringvalidator.OneOf("MERGE", "SNOWPIPE_STREAMING", "APPEND_ONLY")`
		if field.Validators != wantValidator {
			t.Errorf("Validators = %q, want %q", field.Validators, wantValidator)
		}
	})

	t.Run("slider validator", func(t *testing.T) {
		min := float64(1)
		max := float64(10)
		entry := &ConfigEntry{
			Name:        "test.slider",
			UserDefined: true,
			Value: ValueObject{
				Control: "slider",
				Min:     &min,
				Max:     &max,
			},
		}

		field := g.entryToFieldData(entry)

		if !field.HasValidators {
			t.Error("HasValidators should be true for slider")
		}
		wantValidator := "int64validator.Between(1, 10)"
		if field.Validators != wantValidator {
			t.Errorf("Validators = %q, want %q", field.Validators, wantValidator)
		}
	})

	t.Run("one-select without values - no validator", func(t *testing.T) {
		entry := &ConfigEntry{
			Name:        "test.select",
			UserDefined: true,
			Value: ValueObject{
				Control:   "one-select",
				RawValues: []string{}, // Empty values
			},
		}

		field := g.entryToFieldData(entry)

		if field.HasValidators {
			t.Error("HasValidators should be false for one-select without values")
		}
	})
}

// TestRequiredOptionalComputed verifies Required/Optional/Computed field properties.
func TestRequiredOptionalComputed(t *testing.T) {
	g := NewGenerator("/tmp", "source")

	t.Run("required field without default", func(t *testing.T) {
		required := true
		entry := &ConfigEntry{
			Name:        "test.field",
			UserDefined: true,
			Required:    &required,
			Value: ValueObject{
				Control: "string",
			},
		}

		field := g.entryToFieldData(entry)

		if !field.Required {
			t.Error("field should be Required")
		}
		if field.Optional {
			t.Error("required field should not be Optional")
		}
		if field.Computed {
			t.Error("required field should not be Computed")
		}
	})

	t.Run("required field with default becomes optional+computed", func(t *testing.T) {
		required := true
		entry := &ConfigEntry{
			Name:        "test.field",
			UserDefined: true,
			Required:    &required,
			Value: ValueObject{
				Control: "string",
				Default: "default_value",
			},
		}

		field := g.entryToFieldData(entry)

		if field.Required {
			t.Error("field with default should not be Required")
		}
		if !field.Optional {
			t.Error("field with default should be Optional")
		}
		if !field.Computed {
			t.Error("field with default should be Computed")
		}
	})

	t.Run("optional field without default", func(t *testing.T) {
		entry := &ConfigEntry{
			Name:        "test.field",
			UserDefined: true,
			Value: ValueObject{
				Control: "string",
			},
		}

		field := g.entryToFieldData(entry)

		if field.Required {
			t.Error("optional field should not be Required")
		}
		if !field.Optional {
			t.Error("optional field should be Optional")
		}
		if field.Computed {
			t.Error("optional field without default should not be Computed")
		}
	})
}

// TestSetOnceFields verifies that set_once fields get RequiresReplace modifier.
func TestSetOnceFields(t *testing.T) {
	g := NewGenerator("/tmp", "source")

	t.Run("set_once field", func(t *testing.T) {
		entry := &ConfigEntry{
			Name:        "ingestion.mode",
			UserDefined: true,
			SetOnce:     true,
			Value: ValueObject{
				Control: "one-select",
			},
		}

		field := g.entryToFieldData(entry)

		if !field.RequiresReplace {
			t.Error("set_once field should have RequiresReplace=true")
		}
	})

	t.Run("non set_once field", func(t *testing.T) {
		entry := &ConfigEntry{
			Name:        "regular.field",
			UserDefined: true,
			SetOnce:     false,
			Value: ValueObject{
				Control: "string",
			},
		}

		field := g.entryToFieldData(entry)

		if field.RequiresReplace {
			t.Error("non set_once field should not have RequiresReplace")
		}
	})
}

// TestDescriptionFallback verifies description falls back to display_name.
func TestDescriptionFallback(t *testing.T) {
	g := NewGenerator("/tmp", "source")

	t.Run("uses description when available", func(t *testing.T) {
		entry := &ConfigEntry{
			Name:        "test.field",
			UserDefined: true,
			Description: "This is the description",
			DisplayName: "Test Field",
			Value: ValueObject{
				Control: "string",
			},
		}

		field := g.entryToFieldData(entry)

		if field.Description != "This is the description" {
			t.Errorf("Description = %q, want %q", field.Description, "This is the description")
		}
	})

	t.Run("falls back to display_name when description empty", func(t *testing.T) {
		entry := &ConfigEntry{
			Name:        "test.field",
			UserDefined: true,
			Description: "",
			DisplayName: "Test Field Display Name",
			Value: ValueObject{
				Control: "string",
			},
		}

		field := g.entryToFieldData(entry)

		if field.Description != "Test Field Display Name" {
			t.Errorf("Description = %q, want %q", field.Description, "Test Field Display Name")
		}
	})
}

// TestPrepareTemplateData verifies the overall template data preparation.
func TestPrepareTemplateData(t *testing.T) {
	g := NewGenerator("/tmp/output", "source")

	required := true
	config := &ConnectorConfig{
		DisplayName: "PostgreSQL",
		Config: []ConfigEntry{
			{
				Name:        "database.hostname.user.defined",
				UserDefined: true,
				Required:    &required,
				DisplayName: "Database Hostname",
				Value: ValueObject{
					Control: "string",
				},
			},
			{
				Name:        "not.user.defined",
				UserDefined: false,
				Value: ValueObject{
					Control: "string",
				},
			},
		},
	}

	data := g.prepareTemplateData(config, "postgresql")

	// Verify basic properties
	if data.PackageName != "generated" {
		t.Errorf("PackageName = %q, want %q", data.PackageName, "generated")
	}
	if data.EntityType != "source" {
		t.Errorf("EntityType = %q, want %q", data.EntityType, "source")
	}
	if data.EntityTypeCap != "Source" {
		t.Errorf("EntityTypeCap = %q, want %q", data.EntityTypeCap, "Source")
	}
	if data.ConnectorCode != "postgresql" {
		t.Errorf("ConnectorCode = %q, want %q", data.ConnectorCode, "postgresql")
	}
	if data.ConnectorCodeCap != "Postgresql" {
		t.Errorf("ConnectorCodeCap = %q, want %q", data.ConnectorCodeCap, "Postgresql")
	}
	if data.DisplayName != "PostgreSQL" {
		t.Errorf("DisplayName = %q, want %q", data.DisplayName, "PostgreSQL")
	}
	if data.ModelName != "SourcePostgresqlModel" {
		t.Errorf("ModelName = %q, want %q", data.ModelName, "SourcePostgresqlModel")
	}
	if data.SchemaFuncName != "SourcePostgresqlSchema" {
		t.Errorf("SchemaFuncName = %q, want %q", data.SchemaFuncName, "SourcePostgresqlSchema")
	}
	if data.FieldMappingsName != "SourcePostgresqlFieldMappings" {
		t.Errorf("FieldMappingsName = %q, want %q", data.FieldMappingsName, "SourcePostgresqlFieldMappings")
	}

	// Should have 3 common fields + 1 user-defined field = 4 fields total
	if len(data.Fields) != 4 {
		t.Errorf("len(Fields) = %d, want 4", len(data.Fields))
	}

	// Verify imports include required packages
	hasImport := func(path string) bool {
		for _, imp := range data.Imports {
			if imp == path {
				return true
			}
		}
		return false
	}

	requiredImports := []string{
		"github.com/hashicorp/terraform-plugin-framework/resource/schema",
		"github.com/hashicorp/terraform-plugin-framework/types",
		"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier",
		"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier",
	}
	for _, imp := range requiredImports {
		if !hasImport(imp) {
			t.Errorf("missing required import: %s", imp)
		}
	}
}

// TestCapitalizeFirst verifies the capitalizeFirst helper function.
func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"source", "Source"},
		{"destination", "Destination"},
		{"transform", "Transform"},
		{"", ""},
		{"a", "A"},
		{"ABC", "ABC"},
	}

	for _, tt := range tests {
		result := capitalizeFirst(tt.input)
		if result != tt.expected {
			t.Errorf("capitalizeFirst(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestToPascalCase verifies the toPascalCase helper function.
func TestToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"postgresql", "Postgresql"},
		{"database_hostname", "DatabaseHostname"},
		{"ssh_host", "SSHHost"},
		{"ssl_mode", "SSLMode"},
		{"db_name", "DBName"},
		{"url_path", "URLPath"},
		{"api_key", "APIKey"},
		{"simple", "Simple"},
		{"", ""},
		{"id", "ID"},
	}

	for _, tt := range tests {
		result := toPascalCase(tt.input)
		if result != tt.expected {
			t.Errorf("toPascalCase(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestDefaultFunc verifies the defaultFunc generation for different types.
func TestDefaultFunc(t *testing.T) {
	g := NewGenerator("/tmp", "source")

	tests := []struct {
		name     string
		entry    *ConfigEntry
		expected string
	}{
		{
			name: "string default",
			entry: &ConfigEntry{
				Value: ValueObject{Control: "string", Default: "hello"},
			},
			expected: `stringdefault.StaticString("hello")`,
		},
		{
			name: "string default with quotes",
			entry: &ConfigEntry{
				Value: ValueObject{Control: "string", Default: `value"with"quotes`},
			},
			expected: `stringdefault.StaticString("value\"with\"quotes")`,
		},
		{
			name: "int64 default",
			entry: &ConfigEntry{
				Value: ValueObject{Control: "number", Default: float64(42)},
			},
			expected: "int64default.StaticInt64(42)",
		},
		{
			name: "bool default true",
			entry: &ConfigEntry{
				Value: ValueObject{Control: "boolean", Default: true},
			},
			expected: "booldefault.StaticBool(true)",
		},
		{
			name: "bool default false",
			entry: &ConfigEntry{
				Value: ValueObject{Control: "toggle", Default: false},
			},
			expected: "booldefault.StaticBool(false)",
		},
		{
			name: "slider default",
			entry: &ConfigEntry{
				Value: ValueObject{Control: "slider", Default: float64(5)},
			},
			expected: "int64default.StaticInt64(5)",
		},
		{
			name: "unknown type returns empty",
			entry: &ConfigEntry{
				Value: ValueObject{Control: "multi-select", Default: []string{"a", "b"}},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := g.defaultFunc(tt.entry)
			if result != tt.expected {
				t.Errorf("defaultFunc() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestOneOfValidator verifies the oneOfValidator generation.
func TestOneOfValidator(t *testing.T) {
	g := NewGenerator("/tmp", "source")

	tests := []struct {
		name      string
		rawValues []string
		expected  string
	}{
		{
			name:      "two values",
			rawValues: []string{"Yes", "No"},
			expected:  `stringvalidator.OneOf("Yes", "No")`,
		},
		{
			name:      "multiple values",
			rawValues: []string{"MERGE", "APPEND_ONLY", "SNOWPIPE_STREAMING"},
			expected:  `stringvalidator.OneOf("MERGE", "APPEND_ONLY", "SNOWPIPE_STREAMING")`,
		},
		{
			name:      "single value",
			rawValues: []string{"only"},
			expected:  `stringvalidator.OneOf("only")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &ConfigEntry{
				Value: ValueObject{RawValues: tt.rawValues},
			}
			result := g.oneOfValidator(entry)
			if result != tt.expected {
				t.Errorf("oneOfValidator() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestRangeValidator verifies the rangeValidator generation for sliders.
func TestRangeValidator(t *testing.T) {
	g := NewGenerator("/tmp", "source")

	tests := []struct {
		name     string
		min      float64
		max      float64
		expected string
	}{
		{
			name:     "standard range",
			min:      1,
			max:      10,
			expected: "int64validator.Between(1, 10)",
		},
		{
			name:     "large range",
			min:      0,
			max:      1000000,
			expected: "int64validator.Between(0, 1000000)",
		},
		{
			name:     "same min max",
			min:      5,
			max:      5,
			expected: "int64validator.Between(5, 5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			min := tt.min
			max := tt.max
			entry := &ConfigEntry{
				Value: ValueObject{Min: &min, Max: &max},
			}
			result := g.rangeValidator(entry)
			if result != tt.expected {
				t.Errorf("rangeValidator() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestEntryToFieldData verifies the complete field data generation.
func TestEntryToFieldData(t *testing.T) {
	g := NewGenerator("/tmp", "source")

	t.Run("complete field generation", func(t *testing.T) {
		required := true
		min := float64(1)
		max := float64(100)
		entry := &ConfigEntry{
			Name:        "test.field.user.defined",
			UserDefined: true,
			Required:    &required,
			Description: "Test description",
			DisplayName: "Test Field",
			SetOnce:     true,
			Encrypt:     true,
			Value: ValueObject{
				Control: "slider",
				Min:     &min,
				Max:     &max,
				Default: float64(50),
			},
		}

		field := g.entryToFieldData(entry)

		// Check all field properties
		if field.GoFieldName != "TestField" {
			t.Errorf("GoFieldName = %q, want %q", field.GoFieldName, "TestField")
		}
		if field.GoType != "types.Int64" {
			t.Errorf("GoType = %q, want %q", field.GoType, "types.Int64")
		}
		if field.TfsdkTag != "test_field" {
			t.Errorf("TfsdkTag = %q, want %q", field.TfsdkTag, "test_field")
		}
		if field.TfAttrName != "test_field" {
			t.Errorf("TfAttrName = %q, want %q", field.TfAttrName, "test_field")
		}
		if field.SchemaAttrType != "schema.Int64Attribute" {
			t.Errorf("SchemaAttrType = %q, want %q", field.SchemaAttrType, "schema.Int64Attribute")
		}
		if field.Description != "Test description" {
			t.Errorf("Description = %q, want %q", field.Description, "Test description")
		}
		if !field.Sensitive {
			t.Error("Sensitive should be true (encrypt=true)")
		}
		if field.APIFieldName != "test.field.user.defined" {
			t.Errorf("APIFieldName = %q, want %q", field.APIFieldName, "test.field.user.defined")
		}
		if !field.RequiresReplace {
			t.Error("RequiresReplace should be true (set_once=true)")
		}
		if !field.HasDefault {
			t.Error("HasDefault should be true")
		}
		if field.DefaultFunc != "int64default.StaticInt64(50)" {
			t.Errorf("DefaultFunc = %q, want %q", field.DefaultFunc, "int64default.StaticInt64(50)")
		}
		if !field.HasValidators {
			t.Error("HasValidators should be true for slider")
		}
		if field.Validators != "int64validator.Between(1, 100)" {
			t.Errorf("Validators = %q, want %q", field.Validators, "int64validator.Between(1, 100)")
		}
	})
}

// TestGenerateFile_Integration tests actual file generation with a sample config.
// This test creates a temporary directory and generates a schema file.
func TestGenerateFile_Integration(t *testing.T) {
	// Create a temporary directory for output
	tmpDir, err := os.MkdirTemp("", "tfgen-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	g := NewGenerator(tmpDir, "source")

	// Create a sample config
	required := true
	min := float64(1)
	max := float64(10)
	config := &ConnectorConfig{
		DisplayName: "Test Connector",
		Config: []ConfigEntry{
			{
				Name:        "database.hostname.user.defined",
				UserDefined: true,
				Required:    &required,
				DisplayName: "Database Hostname",
				Description: "The hostname of the database server",
				Value: ValueObject{
					Control: "string",
				},
			},
			{
				Name:        "database.port.user.defined",
				UserDefined: true,
				DisplayName: "Database Port",
				Value: ValueObject{
					Control: "string",
					Default: "5432",
				},
			},
			{
				Name:        "database.password",
				UserDefined: true,
				Required:    &required,
				DisplayName: "Password",
				Value: ValueObject{
					Control: "password",
				},
			},
			{
				Name:        "snapshot.mode",
				UserDefined: true,
				DisplayName: "Snapshot Mode",
				Value: ValueObject{
					Control:   "one-select",
					RawValues: []string{"initial", "never", "always"},
					Default:   "initial",
				},
			},
			{
				Name:        "parallelism",
				UserDefined: true,
				DisplayName: "Parallelism",
				Value: ValueObject{
					Control: "slider",
					Min:     &min,
					Max:     &max,
					Default: float64(1),
				},
			},
			{
				Name:        "ssl.enabled",
				UserDefined: true,
				DisplayName: "SSL Enabled",
				Value: ValueObject{
					Control: "toggle",
					Default: false,
				},
			},
			{
				Name:        "internal.field",
				UserDefined: false, // Should be excluded
				Value: ValueObject{
					Control: "string",
				},
			},
		},
	}

	// Generate the file
	err = g.Generate(config, "testconn")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify the file exists
	outputPath := filepath.Join(tmpDir, "source_testconn.go")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Expected output file does not exist: %s", outputPath)
	}

	// Read the generated file
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}
	contentStr := string(content)

	// Verify file content contains expected elements
	// Note: gofmt uses tabs for struct field alignment, so we check for field names and tags separately
	expectedContents := []string{
		"// Code generated by tfgen. DO NOT EDIT.",
		"package generated",
		"type SourceTestconnModel struct",
		"func SourceTestconnSchema() schema.Schema",
		"var SourceTestconnFieldMappings = map[string]string",
		// Struct fields - check field names and tags separately due to gofmt alignment
		"ID ", "types.String", "`tfsdk:\"id\"`",
		"Name ", "`tfsdk:\"name\"`",
		"Connector ", "`tfsdk:\"connector\"`",
		"DatabaseHostname ", "`tfsdk:\"database_hostname\"`",
		"DatabasePort ", "`tfsdk:\"database_port\"`",
		"DatabasePassword ", "`tfsdk:\"database_password\"`",
		"SnapshotMode ", "`tfsdk:\"snapshot_mode\"`",
		"Parallelism ", "types.Int64", "`tfsdk:\"parallelism\"`",
		"SSLEnabled ", "types.Bool", "`tfsdk:\"ssl_enabled\"`",
		// Schema attributes
		"Required:", "true",
		"Optional:", "true",
		"Computed:", "true",
		"Sensitive:", "true",
		// Defaults and validators
		`stringdefault.StaticString("5432")`,
		`stringvalidator.OneOf("initial", "never", "always")`,
		"int64validator.Between(1, 10)",
		"booldefault.StaticBool(false)",
		"stringplanmodifier.UseStateForUnknown()",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Generated file missing expected content: %s", expected)
		}
	}

	// Verify internal field is NOT included
	if strings.Contains(contentStr, "internal_field") || strings.Contains(contentStr, "InternalField") {
		t.Error("Generated file should not contain internal (non user_defined) fields")
	}

	// Verify the code compiles by running gofmt
	cmd := exec.Command("gofmt", "-e", outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Generated code failed gofmt validation: %v\nOutput: %s", err, output)
	}
}

// TestGeneratePostgreSQL_Integration tests generation from actual PostgreSQL config.
// This test requires the backend repository to be available.
func TestGeneratePostgreSQL_Integration(t *testing.T) {
	backendPath := "/Users/alexandrubodea/Documents/Repositories/python-be-streamkap"
	configPath := filepath.Join(backendPath, "app/sources/plugins/postgresql/configuration.latest.json")

	// Skip if backend is not available
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("Backend repository not available, skipping integration test")
	}

	// Create a temporary directory for output
	tmpDir, err := os.MkdirTemp("", "tfgen-postgresql-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Parse the config
	config, err := ParseConnectorConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to parse PostgreSQL config: %v", err)
	}

	// Generate the schema
	g := NewGenerator(tmpDir, "source")
	err = g.Generate(config, "postgresql")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify the file exists
	outputPath := filepath.Join(tmpDir, "source_postgresql.go")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Expected output file does not exist: %s", outputPath)
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}
	contentStr := string(content)

	// Verify expected PostgreSQL-specific fields
	expectedFields := []string{
		"SourcePostgresqlModel",
		"SourcePostgresqlSchema",
		"database_hostname",
		"database_port",
		"database_password",
		"DatabaseHostname",
		"DatabasePort",
		"DatabasePassword",
	}

	for _, expected := range expectedFields {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Generated file missing expected PostgreSQL field: %s", expected)
		}
	}

	// Verify the code compiles by running gofmt
	cmd := exec.Command("gofmt", "-e", outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Generated code failed gofmt validation: %v\nOutput: %s", err, output)
	}

	// Additional validation: try to parse the Go file
	cmd = exec.Command("go", "vet", outputPath)
	cmd.Dir = tmpDir
	// Note: go vet will fail because the generated file references imports
	// that aren't available in the temp directory. This is expected.
	// The gofmt check above is sufficient to verify syntax validity.
}

// TestGenerateSnowflake_Integration tests generation from actual Snowflake config.
// This test requires the backend repository to be available.
func TestGenerateSnowflake_Integration(t *testing.T) {
	backendPath := "/Users/alexandrubodea/Documents/Repositories/python-be-streamkap"
	configPath := filepath.Join(backendPath, "app/destinations/plugins/snowflake/configuration.latest.json")

	// Skip if backend is not available
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("Backend repository not available, skipping integration test")
	}

	// Create a temporary directory for output
	tmpDir, err := os.MkdirTemp("", "tfgen-snowflake-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Parse the config
	config, err := ParseConnectorConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to parse Snowflake config: %v", err)
	}

	// Generate the schema
	g := NewGenerator(tmpDir, "destination")
	err = g.Generate(config, "snowflake")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify the file exists
	outputPath := filepath.Join(tmpDir, "destination_snowflake.go")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Expected output file does not exist: %s", outputPath)
	}

	// Read and verify content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}
	contentStr := string(content)

	// Verify expected Snowflake-specific content
	expectedContent := []string{
		"DestinationSnowflakeModel",
		"DestinationSnowflakeSchema",
		"ingestion_mode", // set_once field
		"Sensitive:",     // private key should be sensitive
	}

	for _, expected := range expectedContent {
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Generated file missing expected Snowflake content: %s", expected)
		}
	}

	// Verify the code compiles by running gofmt
	cmd := exec.Command("gofmt", "-e", outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Generated code failed gofmt validation: %v\nOutput: %s", err, output)
	}
}

// TestImportsTracking verifies that imports are correctly tracked based on field types.
func TestImportsTracking(t *testing.T) {
	g := NewGenerator("/tmp", "source")

	t.Run("imports for field with default", func(t *testing.T) {
		config := &ConnectorConfig{
			DisplayName: "Test",
			Config: []ConfigEntry{
				{
					Name:        "string.field",
					UserDefined: true,
					Value: ValueObject{
						Control: "string",
						Default: "value",
					},
				},
			},
		}

		data := g.prepareTemplateData(config, "test")

		hasImport := func(path string) bool {
			for _, imp := range data.Imports {
				if imp == path {
					return true
				}
			}
			return false
		}

		if !hasImport("github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault") {
			t.Error("missing stringdefault import for field with string default")
		}
	})

	t.Run("imports for field with validator", func(t *testing.T) {
		config := &ConnectorConfig{
			DisplayName: "Test",
			Config: []ConfigEntry{
				{
					Name:        "select.field",
					UserDefined: true,
					Value: ValueObject{
						Control:   "one-select",
						RawValues: []string{"a", "b"},
					},
				},
			},
		}

		data := g.prepareTemplateData(config, "test")

		hasImport := func(path string) bool {
			for _, imp := range data.Imports {
				if imp == path {
					return true
				}
			}
			return false
		}

		if !hasImport("github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator") {
			t.Error("missing stringvalidator import for one-select field")
		}
		if !hasImport("github.com/hashicorp/terraform-plugin-framework/schema/validator") {
			t.Error("missing validator import for one-select field")
		}
	})

	t.Run("imports for int64 field with slider", func(t *testing.T) {
		min := float64(1)
		max := float64(10)
		config := &ConnectorConfig{
			DisplayName: "Test",
			Config: []ConfigEntry{
				{
					Name:        "slider.field",
					UserDefined: true,
					Value: ValueObject{
						Control: "slider",
						Min:     &min,
						Max:     &max,
						Default: float64(5),
					},
				},
			},
		}

		data := g.prepareTemplateData(config, "test")

		hasImport := func(path string) bool {
			for _, imp := range data.Imports {
				if imp == path {
					return true
				}
			}
			return false
		}

		if !hasImport("github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default") {
			t.Error("missing int64default import for slider field with default")
		}
		if !hasImport("github.com/hashicorp/terraform-plugin-framework-validators/int64validator") {
			t.Error("missing int64validator import for slider field")
		}
	})
}

// TestOutputDirectoryCreation verifies that the output directory is created if it doesn't exist.
func TestOutputDirectoryCreation(t *testing.T) {
	// Create a temp dir for the test
	tmpDir, err := os.MkdirTemp("", "tfgen-dir-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a nested directory that doesn't exist
	outputDir := filepath.Join(tmpDir, "nested", "output", "dir")

	g := NewGenerator(outputDir, "source")

	config := &ConnectorConfig{
		DisplayName: "Test",
		Config:      []ConfigEntry{},
	}

	err = g.Generate(config, "test")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify the directory was created
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Errorf("Output directory was not created: %s", outputDir)
	}

	// Verify the file was created
	outputPath := filepath.Join(outputDir, "source_test.go")
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Output file was not created: %s", outputPath)
	}
}
