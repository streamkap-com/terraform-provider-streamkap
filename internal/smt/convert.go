package smt

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ToWire converts a known framework value into its JSON-shaped Go value.
// Null becomes nil at every depth; an empty list stays an empty slice; an
// unknown anywhere is an error, because nothing unknown may reach the API.
// This is the replacement for shared.ExtractTerraformValue, which returns
// nil for both null and unknown and drops every non-string list element.
func ToWire(v attr.Value, p path.Path) (any, error) {
	if v == nil || v.IsNull() {
		return nil, nil
	}
	if v.IsUnknown() {
		return nil, fmt.Errorf("ToWire: %s is unknown", p)
	}
	switch t := v.(type) {
	case types.String:
		return t.ValueString(), nil
	case types.Int64:
		if !isJSONSafeInteger(t.ValueInt64()) {
			return nil, fmt.Errorf("ToWire: %s is outside the exact JSON integer domain", p)
		}
		return t.ValueInt64(), nil
	case types.Float64:
		if !isFiniteNumber(t.ValueFloat64()) {
			return nil, fmt.Errorf("ToWire: %s is not a finite number", p)
		}
		return t.ValueFloat64(), nil
	case types.Bool:
		return t.ValueBool(), nil
	case types.List:
		out := make([]any, 0, len(t.Elements()))
		for i, e := range t.Elements() {
			w, err := ToWire(e, p.AtListIndex(i))
			if err != nil {
				return nil, err
			}
			out = append(out, w)
		}
		return out, nil
	case types.Map:
		out := make(map[string]any, len(t.Elements()))
		for k, e := range t.Elements() {
			w, err := ToWire(e, p.AtMapKey(k))
			if err != nil {
				return nil, err
			}
			out[k] = w
		}
		return out, nil
	case types.Object:
		out := make(map[string]any, len(t.Attributes()))
		for k, e := range t.Attributes() {
			w, err := ToWire(e, p.AtName(k))
			if err != nil {
				return nil, err
			}
			out[k] = w
		}
		return out, nil
	}
	return nil, fmt.Errorf("ToWire: %s has unsupported type %T", p, v)
}

// FromWire converts a JSON-shaped Go value into a framework value of type t.
// nil is null; a key an object omits is null; an empty JSON array is an
// empty, known list. Unknown never appears: the API only returns known data.
func FromWire(ctx context.Context, t attr.Type, raw any, p path.Path) (attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	if raw == nil {
		return nullOf(t), nil
	}
	switch tt := t.(type) {
	case basetypes.StringType:
		s, ok := raw.(string)
		if !ok {
			diags.AddAttributeError(p, "Wire type mismatch", fmt.Sprintf("expected string, got %T", raw))
			return nullOf(t), diags
		}
		return types.StringValue(s), nil
	case basetypes.Int64Type:
		switch n := raw.(type) {
		case float64:
			value, ok := jsonSafeIntegerFromFloat(n)
			if !ok {
				diags.AddAttributeError(p, "Wire type mismatch", "expected an integer in the exact JSON integer domain")
				return nullOf(t), diags
			}
			return types.Int64Value(value), nil
		case int64:
			if !isJSONSafeInteger(n) {
				diags.AddAttributeError(p, "Wire type mismatch", "expected an integer in the exact JSON integer domain")
				return nullOf(t), diags
			}
			return types.Int64Value(n), nil
		case int:
			value := int64(n)
			if !isJSONSafeInteger(value) {
				diags.AddAttributeError(p, "Wire type mismatch", "expected an integer in the exact JSON integer domain")
				return nullOf(t), diags
			}
			return types.Int64Value(value), nil
		}
		diags.AddAttributeError(p, "Wire type mismatch", fmt.Sprintf("expected integer, got %T", raw))
		return nullOf(t), diags
	case basetypes.Float64Type:
		switch n := raw.(type) {
		case float64:
			if !isFiniteNumber(n) {
				diags.AddAttributeError(p, "Wire type mismatch", "expected a finite number")
				return nullOf(t), diags
			}
			return types.Float64Value(n), nil
		case int64:
			return types.Float64Value(float64(n)), nil
		case int:
			return types.Float64Value(float64(n)), nil
		}
		diags.AddAttributeError(p, "Wire type mismatch", fmt.Sprintf("expected number, got %T", raw))
		return nullOf(t), diags
	case basetypes.BoolType:
		b, ok := raw.(bool)
		if !ok {
			diags.AddAttributeError(p, "Wire type mismatch", fmt.Sprintf("expected boolean, got %T", raw))
			return nullOf(t), diags
		}
		return types.BoolValue(b), nil
	case basetypes.ListType:
		items, ok := raw.([]any)
		if !ok {
			diags.AddAttributeError(p, "Wire type mismatch", fmt.Sprintf("expected array, got %T", raw))
			return nullOf(t), diags
		}
		elems := make([]attr.Value, 0, len(items))
		for i, item := range items {
			v, d := FromWire(ctx, tt.ElemType, item, p.AtListIndex(i))
			diags.Append(d...)
			elems = append(elems, v)
		}
		v, d := types.ListValue(tt.ElemType, elems)
		diags.Append(d...)
		return v, diags
	case basetypes.MapType:
		items, ok := raw.(map[string]any)
		if !ok {
			diags.AddAttributeError(p, "Wire type mismatch", fmt.Sprintf("expected object, got %T", raw))
			return nullOf(t), diags
		}
		elems := make(map[string]attr.Value, len(items))
		for k, item := range items {
			v, d := FromWire(ctx, tt.ElemType, item, p.AtMapKey(k))
			diags.Append(d...)
			elems[k] = v
		}
		v, d := types.MapValue(tt.ElemType, elems)
		diags.Append(d...)
		return v, diags
	case basetypes.ObjectType:
		fields, ok := raw.(map[string]any)
		if !ok {
			diags.AddAttributeError(p, "Wire type mismatch", fmt.Sprintf("expected object, got %T", raw))
			return nullOf(t), diags
		}
		attrTypes := tt.AttributeTypes()
		names := make([]string, 0, len(attrTypes))
		for n := range attrTypes {
			names = append(names, n)
		}
		sort.Strings(names)
		values := make(map[string]attr.Value, len(attrTypes))
		for _, n := range names {
			v, d := FromWire(ctx, attrTypes[n], fields[n], p.AtName(n))
			diags.Append(d...)
			values[n] = v
		}
		v, d := types.ObjectValue(attrTypes, values)
		diags.Append(d...)
		return v, diags
	}
	diags.AddAttributeError(p, "Unsupported framework type", t.String())
	return nil, diags
}

func nullOf(t attr.Type) attr.Value {
	switch tt := t.(type) {
	case basetypes.StringType:
		return types.StringNull()
	case basetypes.Int64Type:
		return types.Int64Null()
	case basetypes.Float64Type:
		return types.Float64Null()
	case basetypes.BoolType:
		return types.BoolNull()
	case basetypes.ListType:
		return types.ListNull(tt.ElemType)
	case basetypes.MapType:
		return types.MapNull(tt.ElemType)
	case basetypes.ObjectType:
		return types.ObjectNull(tt.AttributeTypes())
	}
	return nil
}
