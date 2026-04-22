package shared

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// testModel is a simple model struct used for testing marshaling functions.
type testModel struct {
	ID      types.String  `tfsdk:"id"`
	Name    types.String  `tfsdk:"name"`
	Port    types.Int64   `tfsdk:"port"`
	Enabled types.Bool    `tfsdk:"enabled"`
	Rate    types.Float64 `tfsdk:"rate"`
	Tags    types.List    `tfsdk:"tags"`
}

func TestBuildTfsdkFieldIndex(t *testing.T) {
	model := &testModel{}

	v, index := BuildTfsdkFieldIndex(model)

	if v.Kind() != reflect.Struct {
		t.Errorf("Expected struct kind, got %s", v.Kind())
	}

	expectedFields := []string{"id", "name", "port", "enabled", "rate", "tags"}
	for _, field := range expectedFields {
		if _, ok := index[field]; !ok {
			t.Errorf("Expected field %q in index", field)
		}
	}

	if len(index) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(index))
	}
}

func TestBuildTfsdkFieldIndex_NonPointer(t *testing.T) {
	model := testModel{}

	v, index := BuildTfsdkFieldIndex(model)

	if v.Kind() != reflect.Struct {
		t.Errorf("Expected struct kind, got %s", v.Kind())
	}
	if len(index) != 6 {
		t.Errorf("Expected 6 fields, got %d", len(index))
	}
}

// testModelWithDeprecated embeds testModel and adds deprecated aliases,
// mirroring the pattern used for connector deprecated field support.
type testModelWithDeprecated struct {
	testModel
	OldName types.String `tfsdk:"old_name"`
}

func TestBuildTfsdkFieldIndex_EmbeddedStruct(t *testing.T) {
	model := &testModelWithDeprecated{}

	v, index := BuildTfsdkFieldIndex(model)

	if v.Kind() != reflect.Struct {
		t.Errorf("Expected struct kind, got %s", v.Kind())
	}

	// Should find both embedded fields and direct fields
	expectedFields := []string{"id", "name", "port", "enabled", "rate", "tags", "old_name"}
	for _, field := range expectedFields {
		if _, ok := index[field]; !ok {
			t.Errorf("Expected field %q in index", field)
		}
	}

	if len(index) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(index))
	}

	// Verify we can access embedded fields via FieldByIndex
	namePath := index["name"]
	nameField := v.FieldByIndex(namePath)
	if !nameField.IsValid() {
		t.Error("Expected valid field for 'name'")
	}

	// Verify we can access the outer deprecated field
	oldNamePath := index["old_name"]
	oldNameField := v.FieldByIndex(oldNamePath)
	if !oldNameField.IsValid() {
		t.Error("Expected valid field for 'old_name'")
	}
}

func TestExtractTerraformValue(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		value    any
		expected any
	}{
		{"string value", types.StringValue("hello"), "hello"},
		{"string null", types.StringNull(), nil},
		{"string unknown", types.StringUnknown(), nil},
		{"int64 value", types.Int64Value(42), int64(42)},
		{"int64 null", types.Int64Null(), nil},
		{"bool true", types.BoolValue(true), true},
		{"bool false", types.BoolValue(false), false},
		{"bool null", types.BoolNull(), nil},
		{"float64 value", types.Float64Value(3.14), 3.14},
		{"float64 null", types.Float64Null(), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fieldValue := reflect.ValueOf(tt.value)
			result := ExtractTerraformValue(ctx, fieldValue, nil)
			if result != tt.expected {
				t.Errorf("ExtractTerraformValue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractTerraformValue_WithHook(t *testing.T) {
	ctx := context.Background()

	hook := func(_ context.Context, fieldValue reflect.Value) (any, bool) {
		if s, ok := fieldValue.Interface().(string); ok {
			return "hooked:" + s, true
		}
		return nil, false
	}

	// Hook should be called for unrecognized types
	fieldValue := reflect.ValueOf("raw-string")
	result := ExtractTerraformValue(ctx, fieldValue, hook)
	if result != "hooked:raw-string" {
		t.Errorf("Expected hook result, got %v", result)
	}
}

func TestSetTerraformValue(t *testing.T) {
	ctx := context.Background()
	cfg := map[string]any{
		"name":    "test",
		"port":    float64(5432),
		"enabled": true,
		"rate":    3.14,
	}

	t.Run("string", func(t *testing.T) {
		var val types.String
		fieldValue := reflect.ValueOf(&val).Elem()
		SetTerraformValue(ctx, cfg, "name", fieldValue, nil)
		if val.ValueString() != "test" {
			t.Errorf("Expected 'test', got %q", val.ValueString())
		}
	})

	t.Run("int64", func(t *testing.T) {
		var val types.Int64
		fieldValue := reflect.ValueOf(&val).Elem()
		SetTerraformValue(ctx, cfg, "port", fieldValue, nil)
		if val.ValueInt64() != 5432 {
			t.Errorf("Expected 5432, got %d", val.ValueInt64())
		}
	})

	t.Run("bool", func(t *testing.T) {
		var val types.Bool
		fieldValue := reflect.ValueOf(&val).Elem()
		SetTerraformValue(ctx, cfg, "enabled", fieldValue, nil)
		if val.ValueBool() != true {
			t.Errorf("Expected true, got %t", val.ValueBool())
		}
	})

	t.Run("float64", func(t *testing.T) {
		var val types.Float64
		fieldValue := reflect.ValueOf(&val).Elem()
		SetTerraformValue(ctx, cfg, "rate", fieldValue, nil)
		if val.ValueFloat64() != 3.14 {
			t.Errorf("Expected 3.14, got %f", val.ValueFloat64())
		}
	})

	t.Run("missing key returns null", func(t *testing.T) {
		var val types.String
		fieldValue := reflect.ValueOf(&val).Elem()
		SetTerraformValue(ctx, cfg, "nonexistent", fieldValue, nil)
		if !val.IsNull() {
			t.Errorf("Expected null, got %v", val)
		}
	})
}

func TestModelToConfigMap(t *testing.T) {
	ctx := context.Background()
	mappings := map[string]string{
		"name":    "config.name",
		"port":    "config.port",
		"enabled": "config.enabled",
	}

	model := &testModel{
		ID:      types.StringValue("123"),
		Name:    types.StringValue("test-source"),
		Port:    types.Int64Value(5432),
		Enabled: types.BoolValue(true),
		Rate:    types.Float64Null(),
		Tags:    types.ListNull(types.StringType),
	}

	configMap, err := ModelToConfigMap(ctx, model, mappings, nil)
	if err != nil {
		t.Fatalf("ModelToConfigMap() error: %v", err)
	}

	if configMap["config.name"] != "test-source" {
		t.Errorf("Expected name 'test-source', got %v", configMap["config.name"])
	}
	if configMap["config.port"] != int64(5432) {
		t.Errorf("Expected port 5432, got %v", configMap["config.port"])
	}
	if configMap["config.enabled"] != true {
		t.Errorf("Expected enabled true, got %v", configMap["config.enabled"])
	}
	// ID should NOT be in configMap because it's not in the mappings
	if _, ok := configMap["id"]; ok {
		t.Error("ID should not be in config map (not in mappings)")
	}
}

func TestModelToConfigMap_MissingField(t *testing.T) {
	ctx := context.Background()
	mappings := map[string]string{
		"nonexistent_field": "config.ghost",
	}

	model := &testModel{Name: types.StringValue("test")}

	configMap, err := ModelToConfigMap(ctx, model, mappings, nil)
	if err != nil {
		t.Fatalf("ModelToConfigMap() error: %v", err)
	}

	// Missing field should be skipped, not cause error
	if _, ok := configMap["config.ghost"]; ok {
		t.Error("Missing field should be skipped")
	}
}

func TestConfigMapToModel(t *testing.T) {
	ctx := context.Background()
	mappings := map[string]string{
		"name":    "config.name",
		"port":    "config.port",
		"enabled": "config.enabled",
		"rate":    "config.rate",
	}

	cfg := map[string]any{
		"config.name":    "test-source",
		"config.port":    float64(5432),
		"config.enabled": true,
		"config.rate":    3.14,
	}

	model := &testModel{}
	ConfigMapToModel(ctx, cfg, model, mappings, nil)

	if model.Name.ValueString() != "test-source" {
		t.Errorf("Expected name 'test-source', got %q", model.Name.ValueString())
	}
	if model.Port.ValueInt64() != 5432 {
		t.Errorf("Expected port 5432, got %d", model.Port.ValueInt64())
	}
	if model.Enabled.ValueBool() != true {
		t.Errorf("Expected enabled true, got %t", model.Enabled.ValueBool())
	}
	if model.Rate.ValueFloat64() != 3.14 {
		t.Errorf("Expected rate 3.14, got %f", model.Rate.ValueFloat64())
	}
}

func TestGetStringField(t *testing.T) {
	model := &testModel{
		Name: types.StringValue("test-name"),
	}

	if got := GetStringField(model, "Name"); got != "test-name" {
		t.Errorf("GetStringField(Name) = %q, want 'test-name'", got)
	}

	if got := GetStringField(model, "Nonexistent"); got != "" {
		t.Errorf("GetStringField(Nonexistent) = %q, want ''", got)
	}
}

func TestSetStringField(t *testing.T) {
	model := &testModel{}

	SetStringField(model, "Name", "new-name")
	if model.Name.ValueString() != "new-name" {
		t.Errorf("SetStringField(Name) = %q, want 'new-name'", model.Name.ValueString())
	}

	// Setting nonexistent field should not panic
	SetStringField(model, "Nonexistent", "value")
}

func TestGetSetStringField_NonPointer(t *testing.T) {
	// GetStringField should work with non-pointer models
	model := testModel{Name: types.StringValue("test")}
	if got := GetStringField(model, "Name"); got != "test" {
		t.Errorf("GetStringField(non-pointer) = %q, want 'test'", got)
	}

	// SetStringField needs pointer to actually modify
	SetStringField(&model, "Name", "modified")
	if model.Name.ValueString() != "modified" {
		t.Errorf("SetStringField(pointer) = %q, want 'modified'", model.Name.ValueString())
	}
}
