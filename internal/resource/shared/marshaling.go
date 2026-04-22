// Package shared provides common reflection-based marshaling utilities
// used by both connector and transform base resources.
package shared

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/helper"
)

// BuildTfsdkFieldIndex builds a map from tfsdk struct tag to field index path
// for quick lookup during reflection-based marshaling. It recurses into
// anonymous (embedded) struct fields so promoted fields are discoverable.
func BuildTfsdkFieldIndex(model any) (reflect.Value, map[string][]int) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	tfsdkToField := make(map[string][]int, v.Type().NumField())
	buildFieldIndex(v.Type(), nil, tfsdkToField)

	return v, tfsdkToField
}

func buildFieldIndex(t reflect.Type, prefix []int, out map[string][]int) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		path := append(slices.Clone(prefix), i)

		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			buildFieldIndex(field.Type, path, out)
			continue
		}

		tag := field.Tag.Get("tfsdk")
		if tag != "" && tag != "-" {
			out[tag] = path
		}
	}
}

// ExtractValueFunc is a hook that allows callers to handle additional types
// beyond the core set. Return (value, true) if handled, (nil, false) to
// fall through to the default warning.
type ExtractValueFunc func(ctx context.Context, fieldValue reflect.Value) (any, bool)

// ExtractTerraformValue extracts the Go value from a Terraform types value.
// Handles String, Int64, Bool, Float64, and List. The optional extraFn hook
// handles additional types (e.g., jsontypes.Normalized, map types).
func ExtractTerraformValue(ctx context.Context, fieldValue reflect.Value, extraFn ExtractValueFunc) any {
	switch v := fieldValue.Interface().(type) {
	case types.String:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		return v.ValueString()

	case types.Int64:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		return v.ValueInt64()

	case types.Bool:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		return v.ValueBool()

	case types.Float64:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		return v.ValueFloat64()

	case types.List:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		var result []string
		for _, elem := range v.Elements() {
			if strVal, ok := elem.(types.String); ok {
				result = append(result, strVal.ValueString())
			}
		}
		return result

	default:
		if extraFn != nil {
			if val, handled := extraFn(ctx, fieldValue); handled {
				return val
			}
		}
		tflog.Warn(ctx, fmt.Sprintf("Unknown Terraform type: %T", v))
		return nil
	}
}

// SetValueFunc is a hook that allows callers to handle additional types
// beyond the core set. Return true if handled.
type SetValueFunc func(ctx context.Context, cfg map[string]any, apiField string, fieldValue reflect.Value) bool

// SetTerraformValue sets a Terraform types value from an API config map value.
// Handles String, Int64, Bool, Float64, and List. The optional extraFn hook
// handles additional types (e.g., jsontypes.Normalized, map types).
func SetTerraformValue(ctx context.Context, cfg map[string]any, apiField string, fieldValue reflect.Value, extraFn SetValueFunc) {
	fieldType := fieldValue.Type()

	switch fieldType {
	case reflect.TypeOf(types.String{}):
		tfVal := helper.GetTfCfgString(ctx, cfg, apiField)
		fieldValue.Set(reflect.ValueOf(tfVal))

	case reflect.TypeOf(types.Int64{}):
		tfVal := helper.GetTfCfgInt64(ctx, cfg, apiField)
		fieldValue.Set(reflect.ValueOf(tfVal))

	case reflect.TypeOf(types.Bool{}):
		tfVal := helper.GetTfCfgBool(ctx, cfg, apiField)
		fieldValue.Set(reflect.ValueOf(tfVal))

	case reflect.TypeOf(types.Float64{}):
		tfVal := helper.GetTfCfgFloat64(ctx, cfg, apiField)
		fieldValue.Set(reflect.ValueOf(tfVal))

	case reflect.TypeOf(types.List{}):
		tfVal := helper.GetTfCfgListString(ctx, cfg, apiField)
		fieldValue.Set(reflect.ValueOf(tfVal))

	default:
		if extraFn != nil && extraFn(ctx, cfg, apiField, fieldValue) {
			return
		}
		tflog.Warn(ctx, fmt.Sprintf("Unknown field type for '%s': %s", apiField, fieldType))
	}
}

// ModelToConfigMap converts a model struct to an API config map using field
// mappings and reflection. The optional extraFn hook handles additional types.
func ModelToConfigMap(ctx context.Context, model any, mappings map[string]string, extraFn ExtractValueFunc) (map[string]any, error) {
	configMap := make(map[string]any)

	v, tfsdkToField := BuildTfsdkFieldIndex(model)

	for tfAttr, apiField := range mappings {
		fieldPath, ok := tfsdkToField[tfAttr]
		if !ok {
			tflog.Warn(ctx, fmt.Sprintf("Field mapping for '%s' not found in model", tfAttr))
			continue
		}

		fieldValue := v.FieldByIndex(fieldPath)
		apiValue := ExtractTerraformValue(ctx, fieldValue, extraFn)
		configMap[apiField] = apiValue
	}

	return configMap, nil
}

// ConfigMapToModel updates a model struct from an API config map using field
// mappings and reflection. The optional extraFn hook handles additional types.
func ConfigMapToModel(ctx context.Context, cfg map[string]any, model any, mappings map[string]string, extraFn SetValueFunc) {
	v, tfsdkToField := BuildTfsdkFieldIndex(model)

	for tfAttr, apiField := range mappings {
		fieldPath, ok := tfsdkToField[tfAttr]
		if !ok {
			continue
		}

		fieldValue := v.FieldByIndex(fieldPath)
		if !fieldValue.CanSet() {
			continue
		}

		SetTerraformValue(ctx, cfg, apiField, fieldValue, extraFn)
	}
}

// GetStringField gets a string value from a model field by name using reflection.
func GetStringField(model any, fieldName string) string {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return ""
	}

	if strVal, ok := field.Interface().(types.String); ok {
		return strVal.ValueString()
	}

	return ""
}

// SetStringField sets a types.String value on a model field by name using reflection.
func SetStringField(model any, fieldName string, value string) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() || !field.CanSet() {
		return
	}

	field.Set(reflect.ValueOf(types.StringValue(value)))
}
