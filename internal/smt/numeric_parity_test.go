package smt

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

const (
	minSafeInteger = int64(-9007199254740991)
	maxSafeInteger = int64(9007199254740991)
)

func TestNumericParity_IntegerCodecDomain(t *testing.T) {
	ctx := context.Background()

	for _, value := range []int64{minSafeInteger, 0, maxSafeInteger} {
		wire, err := ToWire(types.Int64Value(value), path.Root("count"))
		require.NoError(t, err)
		require.Equal(t, value, wire)

		decoded, diags := FromWire(ctx, types.Int64Type, float64(value), path.Root("count"))
		require.False(t, diags.HasError(), "%v", diags)
		require.Equal(t, value, decoded.(types.Int64).ValueInt64())
	}

	for name, value := range map[string]int64{
		"below safe minimum": minSafeInteger - 1,
		"above safe maximum": maxSafeInteger + 1,
	} {
		t.Run("write "+name, func(t *testing.T) {
			wire, err := ToWire(types.Int64Value(value), path.Root("count"))
			require.Error(t, err)
			require.Nil(t, wire, "an unsafe integer must not reach the request body")
		})
		for representation, raw := range map[string]any{"json number": float64(value), "native integer": value} {
			t.Run("read "+name+" from "+representation, func(t *testing.T) {
				decoded, diags := FromWire(ctx, types.Int64Type, raw, path.Root("count"))
				require.True(t, diags.HasError())
				require.True(t, decoded.IsNull(), "an unsafe integer must not be cast into provider state")
			})
		}
	}

	for name, value := range map[string]float64{
		"fractional":        1.5,
		"positive infinity": math.Inf(1),
		"negative infinity": math.Inf(-1),
		"not a number":      math.NaN(),
	} {
		t.Run(name, func(t *testing.T) {
			decoded, diags := FromWire(ctx, types.Int64Type, value, path.Root("count"))
			require.True(t, diags.HasError())
			require.True(t, decoded.IsNull(), "an invalid integer must not be cast into provider state")
		})
	}
}

func TestNumericParity_NumberAndStringCodecDomain(t *testing.T) {
	ctx := context.Background()
	for _, value := range []float64{-1e300, -1e100, 0, 1e100, 1e300, float64(maxSafeInteger + 1)} {
		wire, err := ToWire(types.Float64Value(value), path.Root("ratio"))
		require.NoError(t, err)
		require.Equal(t, value, wire, "finite number values are IEEE 754 values, not integer-domain values")

		decoded, diags := FromWire(ctx, types.Float64Type, value, path.Root("ratio"))
		require.False(t, diags.HasError(), "%v", diags)
		require.Equal(t, value, decoded.(types.Float64).ValueFloat64())
	}

	for name, value := range map[string]float64{
		"positive infinity": math.Inf(1),
		"negative infinity": math.Inf(-1),
	} {
		t.Run("write "+name, func(t *testing.T) {
			wire, err := ToWire(types.Float64Value(value), path.Root("ratio"))
			require.Error(t, err)
			require.Nil(t, wire, "a non-finite number must not reach the request body")
		})
		t.Run("read "+name, func(t *testing.T) {
			decoded, diags := FromWire(ctx, types.Float64Type, value, path.Root("ratio"))
			require.True(t, diags.HasError())
			require.True(t, decoded.IsNull(), "a non-finite number must not enter provider state")
		})
	}
	require.Panics(t, func() { types.Float64Value(math.NaN()) }, "the framework must refuse NaN before ToWire")
	decoded, diags := FromWire(ctx, types.Float64Type, math.NaN(), path.Root("ratio"))
	require.True(t, diags.HasError())
	require.True(t, decoded.IsNull(), "NaN from the wire must not enter provider state")

	lossless := "90071992547409931234567890.00000000000000000001"
	wire, err := ToWire(types.StringValue(lossless), path.Root("literal"))
	require.NoError(t, err)
	require.Equal(t, lossless, wire)
	decoded, diags = FromWire(ctx, types.StringType, lossless, path.Root("literal"))
	require.False(t, diags.HasError(), "%v", diags)
	require.Equal(t, lossless, decoded.(types.String).ValueString())
}

func TestNumericParity_NestedListAndNullableCodecMatrix(t *testing.T) {
	ctx := context.Background()
	valueType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"count":   types.Int64Type,
		"literal": types.StringType,
		"ratios":  types.ListType{ElemType: types.Float64Type},
		"steps":   types.ListType{ElemType: types.Int64Type},
	}}
	raw := map[string]any{
		"count":   float64(maxSafeInteger),
		"literal": "000900719925474099300",
		"ratios":  []any{1e100, nil, 1e300},
		"steps":   []any{float64(minSafeInteger), nil, float64(maxSafeInteger)},
	}
	original := map[string]any{
		"count":   raw["count"],
		"literal": raw["literal"],
		"ratios":  append([]any(nil), raw["ratios"].([]any)...),
		"steps":   append([]any(nil), raw["steps"].([]any)...),
	}

	decoded, diags := FromWire(ctx, valueType, raw, path.Root("numeric"))
	require.False(t, diags.HasError(), "%v", diags)
	wire, err := ToWire(decoded, path.Root("numeric"))
	require.NoError(t, err)
	require.Equal(t, apiRoundTrip(t, original), apiRoundTrip(t, wire))
	require.Equal(t, original, raw, "conversion must not normalize or mutate the response")

	invalid := map[string]any{
		"count":   float64(maxSafeInteger),
		"literal": raw["literal"],
		"ratios":  raw["ratios"],
		"steps":   []any{float64(minSafeInteger), 2.5},
	}
	decoded, diags = FromWire(ctx, valueType, invalid, path.Root("numeric"))
	require.True(t, diags.HasError())
	require.Contains(t, diags[0].Detail(), "expected an integer in the exact JSON integer domain")
	require.True(t, decoded.(types.Object).Attributes()["steps"].(types.List).Elements()[1].IsNull())
	require.Equal(t, 2.5, invalid["steps"].([]any)[1], "refusal must not normalize the source value")
}

func TestNumericParity_SchemaDefaults(t *testing.T) {
	defaultValue := func(t *testing.T, kind, literal string) (attr.Value, error) {
		t.Helper()
		var node JSONSchema
		require.NoError(t, json.Unmarshal([]byte(`{"type":"object","properties":{"value":{"type":"`+kind+`","default":`+literal+`}},"additionalProperties":false}`), &node))
		child, err := BuildConfigChild(&TypeSpec{TypeID: "numeric", DataSchema: &node})
		if err != nil {
			return nil, err
		}
		value, ok := DefaultValue(child.Attribute.Attributes["value"])
		require.True(t, ok)
		return value, nil
	}

	for _, value := range []int64{minSafeInteger, maxSafeInteger} {
		got, err := defaultValue(t, "integer", strconv.FormatInt(value, 10))
		require.NoError(t, err)
		require.Equal(t, value, got.(types.Int64).ValueInt64())
	}
	for _, literal := range []string{"1.0", "1e0"} {
		got, err := defaultValue(t, "integer", literal)
		require.NoError(t, err)
		require.Equal(t, int64(1), got.(types.Int64).ValueInt64())
	}
	for _, literal := range []string{"-9007199254740992", "9007199254740992", "1.5"} {
		got, err := defaultValue(t, "integer", literal)
		require.Error(t, err)
		require.Nil(t, got)
	}
	for _, tc := range []struct {
		kind    string
		literal string
	}{{"integer", `"5"`}, {"number", `"5"`}} {
		got, err := defaultValue(t, tc.kind, tc.literal)
		require.Error(t, err)
		require.Nil(t, got)
	}

	got, err := defaultValue(t, "number", "1e300")
	require.NoError(t, err)
	require.Equal(t, 1e300, got.(types.Float64).ValueFloat64())
	got, err = defaultValue(t, "number", "1e400")
	require.Error(t, err)
	require.Nil(t, got)
}

func TestNumericParity_CorpusAdmissionRejectsUnrepresentableLiterals(t *testing.T) {
	const valid = `{
  "contract_version": 1,
  "producer": "test",
  "digest": "sha256:test",
  "types": [{
    "type_id": "numeric",
    "schema_version": 1,
    "codec_version": 1,
    "publishable": false,
    "authorable": false,
    "role": "transform",
    "data_schema": {
      "type": "object",
      "properties": {
        "count": {"type": "integer", "minimum": -10, "maximum": 10, "exclusiveMinimum": -11, "exclusiveMaximum": 11, "multipleOf": 1, "const": 1, "enum": [2], "examples": [3], "default": 10},
        "nullable_count": {"anyOf": [
          {"type": "integer", "minimum": -10, "maximum": 10},
          {"type": "null"}
        ], "default": 4},
        "nullable_ratio": {"anyOf": [
          {"type": "number"},
          {"type": "null"}
        ], "default": null},
        "limits": {"type": "array", "items": {"anyOf": [
          {"type": "integer", "minimum": -9007199254740991, "maximum": 9007199254740991},
          {"type": "null"}
        ]}, "default": [5]},
        "window": {"type": "object", "properties": {
          "threshold": {"type": "integer", "minimum": -10, "maximum": 10}
        }, "additionalProperties": false, "default": {"threshold": 7}},
        "ratio": {"type": "number", "minimum": -1e300, "maximum": 1e300, "multipleOf": 1e-300, "default": 1e300},
        "literal": {"type": "string"}
      },
      "additionalProperties": false
    },
    "ui_schema": {
      "type": "Control",
      "scope": "#/properties/count",
      "rule": {"condition": {"type": "AND", "conditions": [
        {"scope": "#/properties/count", "schema": {"const": 6}},
        {"type": "OR", "conditions": [
          {"scope": "#/properties/ratio", "schema": {"const": 1e300}},
          {"scope": "#/properties/literal", "schema": {"const": "1e300"}}
        ]},
        {"scope": "#/properties/nullable_count", "schema": {"const": 5}},
        {"scope": "#/properties/nullable_ratio", "schema": {"const": 1e300}},
        {"scope": "#/properties/window", "schema": {"properties": {"threshold": {"const": 7}}}},
        {"scope": "#/properties/missing", "schema": {"const": 8}}
      ]}}
    },
    "consumer_requirements": {
      "version": 1,
      "schema_keywords": ["additionalProperties", "anyOf", "const", "default", "enum", "examples", "exclusiveMaximum", "exclusiveMinimum", "items", "maximum", "minimum", "multipleOf", "properties", "type"],
      "schema_types": ["array", "integer", "null", "number", "object", "string"],
      "schema_formats": [],
      "ui_elements": ["Control"],
      "ui_controls": [],
      "secret_addressing": []
    },
    "secret_paths": [],
    "row_keys": {},
    "examples": {"read_projection": {"count": 10, "nullable_count": 4, "nullable_ratio": null, "limits": [null], "window": {"threshold": 7}, "ratio": 1e300, "literal": "1e300"}}
  }]
}`

	corpus, err := ParseCorpus([]byte(valid))
	require.NoError(t, err)
	require.Len(t, corpus.Types, 1)

	cases := map[string]struct {
		from string
		to   string
		path string
	}{
		"unsafe integer default":       {`"default": 10`, `"default": 9007199254740992`, "/count"},
		"unsafe nested minimum":        {`"minimum": -9007199254740991`, `"minimum": -9007199254740992`, "/limits/*"},
		"unsafe nested maximum":        {`"maximum": 9007199254740991`, `"maximum": 9007199254740992`, "/limits/*"},
		"unsafe exclusive minimum":     {`"exclusiveMinimum": -11`, `"exclusiveMinimum": -9007199254740992`, "/count"},
		"unsafe exclusive maximum":     {`"exclusiveMaximum": 11`, `"exclusiveMaximum": 9007199254740992`, "/count"},
		"unsafe integer const":         {`"const": 1`, `"const": 9007199254740992`, "/count"},
		"unsafe integer enum":          {`"enum": [2]`, `"enum": [9007199254740992]`, "/count"},
		"unsafe integer example":       {`"examples": [3]`, `"examples": [9007199254740992]`, "/count"},
		"unsafe nullable default":      {`"default": 4`, `"default": 9007199254740992`, "/nullable_count"},
		"unsafe nested list default":   {`"default": [5]`, `"default": [9007199254740992]`, "/limits"},
		"unsafe nested object default": {`"default": {"threshold": 7}`, `"default": {"threshold": 9007199254740992}`, "/window/threshold"},
		"integer default as string":    {`"default": 10`, `"default": "10"`, "/count"},
		"number default as string":     {`"default": 1e300`, `"default": "1e300"`, "/ratio"},
		"unsafe partial condition":     {`"const": 6`, `"const": 9007199254740992`, "/ui_schema/rule/condition/conditions/0/schema"},
		"overflowing scoped number":    {`"const": 1e300`, `"const": 1e400`, "/ui_schema/rule/condition/conditions/1/conditions/0/schema"},
		"unsafe nullable condition":    {`"const": 5`, `"const": 9007199254740992`, "/ui_schema/rule/condition/conditions/2/schema"},
		"unsafe nested condition":      {`"const": 7`, `"const": 9007199254740992.0`, "/ui_schema/rule/condition/conditions/4/schema/threshold"},
		"unsafe unknown scope":         {`"const": 8`, `"const": 9007199254740992`, "/ui_schema/rule/condition/conditions/5/schema"},
		"bounded exponent":             {`"const": 1`, `"const": 1e100000000`, "/count"},
		"overflowing number bound":     {`"maximum": 1e300`, `"maximum": 1e400`, "/ratio"},
		"overflowing number multiple":  {`"multipleOf": 1e-300`, `"multipleOf": 1e400`, "/ratio"},
		"overflowing number default":   {`"default": 1e300`, `"default": 1e400`, "/ratio"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			mutated := strings.Replace(valid, tc.from, tc.to, 1)
			require.NotEqual(t, valid, mutated)
			_, err := ParseCorpus([]byte(mutated))
			require.Error(t, err)
			require.ErrorContains(t, err, tc.path)
		})
	}
}
