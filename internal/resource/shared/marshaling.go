// Package shared provides common reflection-based marshaling utilities
// used by both connector and transform base resources.
package shared

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
		// Multiple tfsdk attributes can map to the same API field — this is how
		// deprecated aliases work (e.g. both `insert_static_key_field_2` and
		// `transforms_insert_static_key2_static_field` target
		// `transforms.InsertStaticKey2.static.field`). ConflictsWith validators
		// ensure only one side is user-set, but Go's map-iteration order is
		// randomized. Without the non-nil guard below, a nil from the unset
		// alias can overwrite the user's real value and silently blank the
		// field on Create/Update.
		if apiValue != nil {
			configMap[apiField] = apiValue
		} else if _, alreadySet := configMap[apiField]; !alreadySet {
			configMap[apiField] = nil
		}
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

// CaptureStringFields snapshots the current types.String values of the named
// tfsdk attributes from model, keyed by tfsdk name. tfsdk names that don't
// resolve to a types.String field are skipped. Used with
// PreserveKnownStringFields to carry user-supplied values across an API
// round-trip that would otherwise overwrite them.
func CaptureStringFields(model any, tfsdkNames []string) map[string]types.String {
	if len(tfsdkNames) == 0 {
		return nil
	}

	v, tfsdkToField := BuildTfsdkFieldIndex(model)

	out := make(map[string]types.String, len(tfsdkNames))
	for _, name := range tfsdkNames {
		fieldPath, ok := tfsdkToField[name]
		if !ok {
			continue
		}
		if s, ok := v.FieldByIndex(fieldPath).Interface().(types.String); ok {
			out[name] = s
		}
	}
	return out
}

// PreserveKnownStringFields writes captured values back into model for every
// field whose captured value is known (not null, not unknown), overwriting
// whatever a prior step set.
//
// This implements write-only semantics for sensitive credential fields. The
// Streamkap backend does not faithfully echo every secret back on
// Create/Update: even with secret_returned=true it returns null for a secret
// whose stored value is absent or decrypts to "null" (e.g. a
// conditionally-disabled passphrase — app/utils/entity_searches.py). The
// user-configured value is authoritative, so we restore it. Without this,
// Terraform aborts the apply with "Provider produced inconsistent result after
// apply: <field>: inconsistent values for sensitive attribute" whenever the
// planned value is known but the echo differs. Captured null/unknown values are
// left untouched so an unset Optional+Computed secret still defers to the echo.
func PreserveKnownStringFields(model any, captured map[string]types.String) {
	if len(captured) == 0 {
		return
	}

	v, tfsdkToField := BuildTfsdkFieldIndex(model)

	for name, val := range captured {
		if val.IsNull() || val.IsUnknown() {
			continue
		}
		fieldPath, ok := tfsdkToField[name]
		if !ok {
			continue
		}
		field := v.FieldByIndex(fieldPath)
		if field.CanSet() && field.Type() == reflect.TypeOf(types.String{}) {
			field.Set(reflect.ValueOf(val))
		}
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

// GetStringSliceField reads a Set[String] model field and returns it as a
// []string. Returns nil for null/unknown values; nil marshals to JSON `null`
// (the request struct fields deliberately do NOT use omitempty — see the
// comment on api.Source.Tags), which the backend's
// `app/utils/entity_changes.py::upsert_entities` interprets as "do not change
// tags". An empty (but non-nil) set is returned as `[]string{}` so the caller
// can distinguish "no tags" (clear) from "leave alone".
func GetStringSliceField(ctx context.Context, model any, fieldName string) []string {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return nil
	}

	setVal, ok := field.Interface().(types.Set)
	if !ok {
		return nil
	}
	if setVal.IsNull() || setVal.IsUnknown() {
		return nil
	}

	out := make([]string, 0, len(setVal.Elements()))
	for _, e := range setVal.Elements() {
		if s, ok := e.(types.String); ok {
			out = append(out, s.ValueString())
		}
	}
	return out
}

// SetStringSliceField writes a []string to a Set[String] model field by name.
// Nil input becomes a Null set (preserves "unset" semantics on the way back to
// state); a non-nil empty slice becomes an empty (but non-null) Set.
func SetStringSliceField(model any, fieldName string, values []string) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() || !field.CanSet() {
		return
	}

	if values == nil {
		field.Set(reflect.ValueOf(types.SetNull(types.StringType)))
		return
	}

	elems := make([]attr.Value, len(values))
	for i, s := range values {
		elems[i] = types.StringValue(s)
	}
	setVal, diags := types.SetValue(types.StringType, elems)
	if diags.HasError() {
		// Fall back to null on construction error (shouldn't happen for
		// String elements but defensive — the alternative is to surface
		// diags up through the call chain, which this helper signature
		// doesn't support).
		field.Set(reflect.ValueOf(types.SetNull(types.StringType)))
		return
	}
	field.Set(reflect.ValueOf(setVal))
}
