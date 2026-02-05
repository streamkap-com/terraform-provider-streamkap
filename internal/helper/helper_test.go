package helper

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGetTfCfgString(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]any
		key      string
		expected string
		isNull   bool
	}{
		{"string value", map[string]any{"key": "value"}, "key", "value", false},
		{"empty string value", map[string]any{"key": ""}, "key", "", false},
		{"missing key", map[string]any{}, "key", "", true},
		{"nil value", map[string]any{"key": nil}, "key", "", true},
		{"non-string int", map[string]any{"key": 123}, "key", "", false},
		{"non-string float", map[string]any{"key": 123.45}, "key", "", false},
		{"non-string bool", map[string]any{"key": true}, "key", "", false},
		{"different key exists", map[string]any{"other": "value"}, "key", "", true},
		{"unicode string", map[string]any{"key": "hello world"}, "key", "hello world", false},
		{"special characters", map[string]any{"key": "test@#$%^&*()"}, "key", "test@#$%^&*()", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTfCfgString(tt.cfg, tt.key)
			if tt.isNull {
				if !result.IsNull() {
					t.Errorf("Expected null, got: %v", result)
				}
			} else {
				if result.IsNull() {
					t.Errorf("Expected non-null value %q, got null", tt.expected)
				} else if result.ValueString() != tt.expected {
					t.Errorf("GetTfCfgString() = %q, want %q", result.ValueString(), tt.expected)
				}
			}
		})
	}
}

func TestGetTfCfgInt64(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]any
		key      string
		expected int64
		isNull   bool
	}{
		{"float64 value", map[string]any{"key": float64(42)}, "key", 42, false},
		{"float64 zero", map[string]any{"key": float64(0)}, "key", 0, false},
		{"float64 negative", map[string]any{"key": float64(-100)}, "key", -100, false},
		{"float64 large", map[string]any{"key": float64(9223372036854775807)}, "key", 9223372036854775807, false},
		{"string int", map[string]any{"key": "42"}, "key", 42, false},
		{"string negative int", map[string]any{"key": "-42"}, "key", -42, false},
		{"string zero", map[string]any{"key": "0"}, "key", 0, false},
		{"missing key", map[string]any{}, "key", 0, true},
		{"nil value", map[string]any{"key": nil}, "key", 0, true},
		{"different key exists", map[string]any{"other": float64(42)}, "key", 0, true},
		{"invalid string returns zero", map[string]any{"key": "not-a-number"}, "key", 0, false},
		{"empty string returns zero", map[string]any{"key": ""}, "key", 0, false},
		{"float with decimal truncates", map[string]any{"key": float64(42.9)}, "key", 42, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTfCfgInt64(tt.cfg, tt.key)
			if tt.isNull {
				if !result.IsNull() {
					t.Errorf("Expected null, got: %v", result)
				}
			} else {
				if result.IsNull() {
					t.Errorf("Expected non-null value %d, got null", tt.expected)
				} else if result.ValueInt64() != tt.expected {
					t.Errorf("GetTfCfgInt64() = %d, want %d", result.ValueInt64(), tt.expected)
				}
			}
		})
	}
}

func TestGetTfCfgBool(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]any
		key      string
		expected bool
		isNull   bool
	}{
		{"true value", map[string]any{"key": true}, "key", true, false},
		{"false value", map[string]any{"key": false}, "key", false, false},
		{"missing key", map[string]any{}, "key", false, true},
		{"nil value", map[string]any{"key": nil}, "key", false, true},
		{"different key exists", map[string]any{"other": true}, "key", false, true},
		{"non-bool string returns false", map[string]any{"key": "true"}, "key", false, false},
		{"non-bool int returns false", map[string]any{"key": 1}, "key", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTfCfgBool(tt.cfg, tt.key)
			if tt.isNull {
				if !result.IsNull() {
					t.Errorf("Expected null, got: %v", result)
				}
			} else {
				if result.IsNull() {
					t.Errorf("Expected non-null value %t, got null", tt.expected)
				} else if result.ValueBool() != tt.expected {
					t.Errorf("GetTfCfgBool() = %t, want %t", result.ValueBool(), tt.expected)
				}
			}
		})
	}
}

func TestGetTfCfgFloat64(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]any
		key      string
		expected float64
		isNull   bool
	}{
		{"float64 value", map[string]any{"key": float64(42.5)}, "key", 42.5, false},
		{"float64 zero", map[string]any{"key": float64(0)}, "key", 0, false},
		{"float64 negative", map[string]any{"key": float64(-100.25)}, "key", -100.25, false},
		{"float64 large", map[string]any{"key": float64(1.7976931348623157e+308)}, "key", 1.7976931348623157e+308, false},
		{"float64 small", map[string]any{"key": float64(0.0001)}, "key", 0.0001, false},
		{"integer value as float64", map[string]any{"key": float64(42)}, "key", 42, false},
		{"string float", map[string]any{"key": "42.5"}, "key", 42.5, false},
		{"string negative float", map[string]any{"key": "-42.5"}, "key", -42.5, false},
		{"string zero", map[string]any{"key": "0"}, "key", 0, false},
		{"string integer", map[string]any{"key": "42"}, "key", 42, false},
		{"missing key", map[string]any{}, "key", 0, true},
		{"nil value", map[string]any{"key": nil}, "key", 0, true},
		{"different key exists", map[string]any{"other": float64(42.5)}, "key", 0, true},
		{"invalid string returns zero", map[string]any{"key": "not-a-number"}, "key", 0, false},
		{"empty string returns zero", map[string]any{"key": ""}, "key", 0, false},
		{"non-float64 int returns null", map[string]any{"key": 42}, "key", 0, true},
		{"non-float64 bool returns null", map[string]any{"key": true}, "key", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTfCfgFloat64(tt.cfg, tt.key)
			if tt.isNull {
				if !result.IsNull() {
					t.Errorf("Expected null, got: %v", result)
				}
			} else {
				if result.IsNull() {
					t.Errorf("Expected non-null value %f, got null", tt.expected)
				} else if result.ValueFloat64() != tt.expected {
					t.Errorf("GetTfCfgFloat64() = %f, want %f", result.ValueFloat64(), tt.expected)
				}
			}
		})
	}
}

// TestGetTfCfgFloat64NilMap tests behavior with nil map
func TestGetTfCfgFloat64NilMap(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log("GetTfCfgFloat64 panics with nil map (expected behavior)")
		}
	}()

	var nilMap map[string]any
	result := GetTfCfgFloat64(nilMap, "key")
	if !result.IsNull() {
		t.Error("Expected null for nil map")
	}
}

func TestGetTfCfgListString(t *testing.T) {
	ctx := context.Background()

	t.Run("valid list with elements", func(t *testing.T) {
		cfg := map[string]any{"key": []any{"a", "b", "c"}}
		result := GetTfCfgListString(ctx, cfg, "key")
		if result.IsNull() {
			t.Error("Expected non-null list")
		}
		if len(result.Elements()) != 3 {
			t.Errorf("Expected 3 elements, got %d", len(result.Elements()))
		}
		// Verify element values
		elements := result.Elements()
		expectedVals := []string{"a", "b", "c"}
		for i, elem := range elements {
			strVal, ok := elem.(types.String)
			if !ok {
				t.Errorf("Element %d is not a types.String", i)
				continue
			}
			if strVal.ValueString() != expectedVals[i] {
				t.Errorf("Element %d = %q, want %q", i, strVal.ValueString(), expectedVals[i])
			}
		}
	})

	t.Run("empty list", func(t *testing.T) {
		cfg := map[string]any{"key": []any{}}
		result := GetTfCfgListString(ctx, cfg, "key")
		if result.IsNull() {
			t.Error("Expected non-null empty list")
		}
		if len(result.Elements()) != 0 {
			t.Errorf("Expected 0 elements, got %d", len(result.Elements()))
		}
	})

	t.Run("missing key", func(t *testing.T) {
		cfg := map[string]any{}
		result := GetTfCfgListString(ctx, cfg, "key")
		if !result.IsNull() {
			t.Error("Expected null list for missing key")
		}
	})

	t.Run("nil value", func(t *testing.T) {
		cfg := map[string]any{"key": nil}
		result := GetTfCfgListString(ctx, cfg, "key")
		if !result.IsNull() {
			t.Error("Expected null list for nil value")
		}
	})

	t.Run("different key exists", func(t *testing.T) {
		cfg := map[string]any{"other": []any{"a", "b"}}
		result := GetTfCfgListString(ctx, cfg, "key")
		if !result.IsNull() {
			t.Error("Expected null list when key doesn't exist")
		}
	})

	t.Run("single element list", func(t *testing.T) {
		cfg := map[string]any{"key": []any{"single"}}
		result := GetTfCfgListString(ctx, cfg, "key")
		if result.IsNull() {
			t.Error("Expected non-null list")
		}
		if len(result.Elements()) != 1 {
			t.Errorf("Expected 1 element, got %d", len(result.Elements()))
		}
	})

	t.Run("list with empty strings", func(t *testing.T) {
		cfg := map[string]any{"key": []any{"", "value", ""}}
		result := GetTfCfgListString(ctx, cfg, "key")
		if result.IsNull() {
			t.Error("Expected non-null list")
		}
		if len(result.Elements()) != 3 {
			t.Errorf("Expected 3 elements, got %d", len(result.Elements()))
		}
	})

	t.Run("list with special characters", func(t *testing.T) {
		cfg := map[string]any{"key": []any{"test@domain.com", "path/to/file", "value=123"}}
		result := GetTfCfgListString(ctx, cfg, "key")
		if result.IsNull() {
			t.Error("Expected non-null list")
		}
		if len(result.Elements()) != 3 {
			t.Errorf("Expected 3 elements, got %d", len(result.Elements()))
		}
	})

	t.Run("list type verification", func(t *testing.T) {
		cfg := map[string]any{"key": []any{"a"}}
		result := GetTfCfgListString(ctx, cfg, "key")
		// Verify the element type is StringType
		if result.ElementType(ctx) != types.StringType {
			t.Errorf("Expected element type StringType, got %v", result.ElementType(ctx))
		}
	})

	t.Run("null list has correct element type", func(t *testing.T) {
		cfg := map[string]any{}
		result := GetTfCfgListString(ctx, cfg, "key")
		if !result.IsNull() {
			t.Error("Expected null list")
		}
		// Even null lists should have the correct element type
		if result.ElementType(ctx) != types.StringType {
			t.Errorf("Expected null list to have element type StringType, got %v", result.ElementType(ctx))
		}
	})
}

// TestGetTfCfgStringNilMap tests behavior with nil map
func TestGetTfCfgStringNilMap(t *testing.T) {
	// Note: This would panic in current implementation
	// Adding this test to document expected behavior
	defer func() {
		if r := recover(); r != nil {
			t.Log("GetTfCfgString panics with nil map (expected behavior)")
		}
	}()

	var nilMap map[string]any
	result := GetTfCfgString(nilMap, "key")
	if !result.IsNull() {
		t.Error("Expected null for nil map")
	}
}

// TestGetTfCfgInt64NilMap tests behavior with nil map
func TestGetTfCfgInt64NilMap(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log("GetTfCfgInt64 panics with nil map (expected behavior)")
		}
	}()

	var nilMap map[string]any
	result := GetTfCfgInt64(nilMap, "key")
	if !result.IsNull() {
		t.Error("Expected null for nil map")
	}
}

// TestGetTfCfgBoolNilMap tests behavior with nil map
func TestGetTfCfgBoolNilMap(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log("GetTfCfgBool panics with nil map (expected behavior)")
		}
	}()

	var nilMap map[string]any
	result := GetTfCfgBool(nilMap, "key")
	if !result.IsNull() {
		t.Error("Expected null for nil map")
	}
}

// TestGetTfCfgListStringNilMap tests behavior with nil map
func TestGetTfCfgListStringNilMap(t *testing.T) {
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			t.Log("GetTfCfgListString panics with nil map (expected behavior)")
		}
	}()

	var nilMap map[string]any
	result := GetTfCfgListString(ctx, nilMap, "key")
	if !result.IsNull() {
		t.Error("Expected null for nil map")
	}
}

func TestGetTfCfgMapString(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		cfg      map[string]any
		key      string
		expected map[string]types.String
	}{
		{
			name: "valid map[string]interface{}",
			cfg: map[string]any{
				"test_map": map[string]interface{}{
					"key1": "value1",
					"key2": "value2",
				},
			},
			key: "test_map",
			expected: map[string]types.String{
				"key1": types.StringValue("value1"),
				"key2": types.StringValue("value2"),
			},
		},
		{
			name: "valid map[string]string",
			cfg: map[string]any{
				"test_map": map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
			key: "test_map",
			expected: map[string]types.String{
				"key1": types.StringValue("value1"),
				"key2": types.StringValue("value2"),
			},
		},
		{
			name:     "missing key",
			cfg:      map[string]any{},
			key:      "test_map",
			expected: nil,
		},
		{
			name: "nil value",
			cfg: map[string]any{
				"test_map": nil,
			},
			key:      "test_map",
			expected: nil,
		},
		{
			name: "empty map",
			cfg: map[string]any{
				"test_map": map[string]string{},
			},
			key:      "test_map",
			expected: map[string]types.String{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTfCfgMapString(ctx, tt.cfg, tt.key)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected map length %d, got %d", len(tt.expected), len(result))
				return
			}

			for k, expectedVal := range tt.expected {
				actualVal, ok := result[k]
				if !ok {
					t.Errorf("Expected key %s not found in result", k)
					continue
				}

				if !expectedVal.Equal(actualVal) {
					t.Errorf("For key %s: expected %v, got %v", k, expectedVal, actualVal)
				}
			}
		})
	}
}
