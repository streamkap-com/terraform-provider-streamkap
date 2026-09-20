package smt

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/shared"
)

func buildChain(t *testing.T) *ChainSchema {
	t.Helper()
	_, cat := loadCorpus(t)
	cs, err := BuildChain(cat)
	require.NoError(t, err)
	return cs
}

func nestedChildType(t *testing.T, cs *ChainSchema) types.ObjectType {
	t.Helper()
	return cs.EntryType.AttributeTypes()["contract_fixture_nested"].(types.ObjectType)
}

// apiRoundTrip is what the wire does to a value: JSON out, JSON in.
func apiRoundTrip(t *testing.T, v any) any {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var back any
	require.NoError(t, json.Unmarshal(raw, &back))
	return back
}

func objectWith(t *testing.T, ot types.ObjectType, set map[string]attr.Value) types.Object {
	t.Helper()
	values := map[string]attr.Value{}
	for name, typ := range ot.AttributeTypes() {
		if v, ok := set[name]; ok {
			values[name] = v
		} else {
			values[name] = nullOf(typ)
		}
	}
	return types.ObjectValueMust(ot.AttributeTypes(), values)
}

func strList(items ...string) types.List {
	elems := make([]attr.Value, 0, len(items))
	for _, s := range items {
		elems = append(elems, types.StringValue(s))
	}
	return types.ListValueMust(types.StringType, elems)
}

// Every value state the corpus can express must survive provider -> wire ->
// provider unchanged: null, empty, zero, false, "", nullable scalars and
// ordered object rows. Unknown must never be silently flattened.
func TestValueFidelity_RoundTrip(t *testing.T) {
	ctx := context.Background()
	cs := buildChain(t)
	child := nestedChildType(t, cs)
	outputType := child.AttributeTypes()["output"].(types.ObjectType)
	routesType := child.AttributeTypes()["routes"].(types.ListType)
	rowType := routesType.ElemType.(types.ObjectType)

	row := func(key, label, endpoint string) types.Object {
		return objectWith(t, rowType, map[string]attr.Value{
			"key": types.StringValue(key), "label": types.StringValue(label), "endpoint": types.StringValue(endpoint),
		})
	}

	cases := map[string]attr.Value{
		"whole chain null":   types.ListNull(cs.EntryType),
		"empty chain":        types.ListValueMust(cs.EntryType, nil),
		"nested object null": objectWith(t, child, map[string]attr.Value{"output": types.ObjectNull(outputType.AttributeTypes())}),
		"nested object known": objectWith(t, child, map[string]attr.Value{"output": objectWith(t, outputType, map[string]attr.Value{
			"prefix": types.StringValue(""), "label": types.StringValue("l"), "retry_limit": types.Int64Value(0),
		})}),
		"nullable scalar null":  objectWith(t, child, map[string]attr.Value{"fallback": types.StringNull()}),
		"nullable scalar known": objectWith(t, child, map[string]attr.Value{"fallback": types.StringValue("x")}),
		"zero false empty": objectWith(t, child, map[string]attr.Value{
			"retries": types.Int64Value(0), "enabled": types.BoolValue(false),
			"output": objectWith(t, outputType, map[string]attr.Value{"prefix": types.StringValue("")}),
		}),
		"empty scalar list": objectWith(t, child, map[string]attr.Value{"labels": strList()}),
		"null scalar list":  objectWith(t, child, map[string]attr.Value{"labels": types.ListNull(types.StringType)}),
		"scalar list":       objectWith(t, child, map[string]attr.Value{"labels": strList("b", "a", "c")}),
		"empty object list": objectWith(t, child, map[string]attr.Value{"routes": types.ListValueMust(rowType, nil)}),
		"ordered object list": objectWith(t, child, map[string]attr.Value{"routes": types.ListValueMust(rowType, []attr.Value{
			row("row~west/1", "west", "other.invalid"), row("row-east", "east", "example.invalid"),
		})}),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			wire, err := ToWire(in, path.Root("v"))
			require.NoError(t, err)
			out, diags := FromWire(ctx, in.Type(ctx), apiRoundTrip(t, wire), path.Root("v"))
			require.False(t, diags.HasError(), "%v", diags)
			require.True(t, in.Equal(out), "in=%s out=%s", in, out)
			require.Equal(t, in.IsNull(), out.IsNull())
		})
	}

	// Distinct states stay distinct: the empty list and the null list are
	// different values on both sides of the wire.
	empty, _ := ToWire(cases["empty scalar list"], path.Root("v"))
	null, _ := ToWire(cases["null scalar list"], path.Root("v"))
	require.Equal(t, []any{}, empty.(map[string]any)["labels"])
	require.Nil(t, null.(map[string]any)["labels"])

	// Unknown is refused, at the top and inside a row, naming the path.
	_, err := ToWire(types.ListUnknown(cs.EntryType), path.Root(AttrChain))
	require.EqualError(t, err, "ToWire: smt_chain is unknown")
	deepUnknown := objectWith(t, child, map[string]attr.Value{"routes": types.ListValueMust(rowType, []attr.Value{
		objectWith(t, rowType, map[string]attr.Value{"key": types.StringValue("k"), "endpoint": types.StringUnknown()}),
	})})
	_, err = ToWire(deepUnknown, path.Root("v"))
	require.EqualError(t, err, "ToWire: v.routes[0].endpoint is unknown")
}

// The corpus read projection is the backend's canonical read shape; it must
// decode into the typed child and encode back to the same bytes.
func TestValueFidelity_ReadProjection(t *testing.T) {
	ctx := context.Background()
	_, cat := loadCorpus(t)
	cs, err := BuildChain(cat)
	require.NoError(t, err)
	for _, id := range cat.TypeIDs() {
		t.Run(id, func(t *testing.T) {
			child := cs.EntryType.AttributeTypes()[id]
			example := cat[id].Examples.ReadProjection
			v, diags := FromWire(ctx, child, example, path.Root(id))
			require.False(t, diags.HasError(), "%v", diags)
			require.False(t, v.IsNull())
			back, err := ToWire(v, path.Root(id))
			require.NoError(t, err)
			require.Equal(t, apiRoundTrip(t, example), apiRoundTrip(t, back))
		})
	}
}

// Null and unknown must stay distinguishable through the framework's own
// plan container, not only in Go memory.
func TestValueFidelity_NullVsUnknownThroughPlan(t *testing.T) {
	ctx := context.Background()
	cs := buildChain(t)
	child := nestedChildType(t, cs)
	outputType := child.AttributeTypes()["output"].(types.ObjectType)

	entry := objectWith(t, cs.EntryType, map[string]attr.Value{
		attrKey:  types.StringValue("a"),
		attrType: types.StringValue("contract_fixture_nested"),
		attrID:   types.StringUnknown(),
		"contract_fixture_nested": objectWith(t, child, map[string]attr.Value{
			"fallback": types.StringUnknown(),
			"labels":   types.ListUnknown(types.StringType),
			"output":   types.ObjectNull(outputType.AttributeTypes()),
			"routes":   types.ListNull(child.AttributeTypes()["routes"].(types.ListType).ElemType),
		}),
	})
	s := schema.Schema{Attributes: map[string]schema.Attribute{AttrChain: cs.Attribute}}
	plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	require.False(t, plan.SetAttribute(ctx, path.Root(AttrChain), types.ListValueMust(cs.EntryType, []attr.Value{entry})).HasError())

	var got types.List
	require.False(t, plan.GetAttribute(ctx, path.Root(AttrChain), &got).HasError())
	obj := got.Elements()[0].(types.Object).Attributes()
	require.True(t, obj[attrID].IsUnknown() && !obj[attrID].IsNull())
	cfg := obj["contract_fixture_nested"].(types.Object).Attributes()
	require.True(t, cfg["fallback"].IsUnknown() && !cfg["fallback"].IsNull())
	require.True(t, cfg["labels"].IsUnknown() && !cfg["labels"].IsNull())
	require.True(t, cfg["output"].IsNull() && !cfg["output"].IsUnknown())
	require.True(t, cfg["routes"].IsNull() && !cfg["routes"].IsUnknown())
	require.True(t, cfg["retries"].IsNull(), "an attribute never set is null, not unknown")

	unknownChain := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
	require.False(t, unknownChain.SetAttribute(ctx, path.Root(AttrChain), types.ListUnknown(cs.EntryType)).HasError())
	require.False(t, unknownChain.GetAttribute(ctx, path.Root(AttrChain), &got).HasError())
	require.True(t, got.IsUnknown() && !got.IsNull())
}

// The released reflection marshaler is not a candidate for the chain: it
// collapses null and unknown into one nil and drops every non-string list
// element, so object rows vanish silently.
func TestReleasedMarshalerCollapsesStates(t *testing.T) {
	ctx := context.Background()
	cs := buildChain(t)
	rowType := nestedChildType(t, cs).AttributeTypes()["routes"].(types.ListType).ElemType.(types.ObjectType)

	nullList := shared.ExtractTerraformValue(ctx, reflect.ValueOf(types.ListNull(types.StringType)), nil)
	unknownList := shared.ExtractTerraformValue(ctx, reflect.ValueOf(types.ListUnknown(types.StringType)), nil)
	require.Nil(t, nullList)
	require.Nil(t, unknownList, "unknown is indistinguishable from null")

	rows := types.ListValueMust(rowType, []attr.Value{objectWith(t, rowType, map[string]attr.Value{"key": types.StringValue("row-east")})})
	got := shared.ExtractTerraformValue(ctx, reflect.ValueOf(rows), nil)
	require.Len(t, got, 0, "object rows are dropped: %#v", got)

	obj := shared.ExtractTerraformValue(ctx, reflect.ValueOf(objectWith(t, rowType, nil)), nil)
	require.Nil(t, obj, "objects are not handled at all")
}

// Homogeneous map gate: the framework carries typed maps through the same
// conversion, including an explicit empty map distinct from null.
func TestValueFidelity_MapGate(t *testing.T) {
	ctx := context.Background()
	var node JSONSchema
	require.NoError(t, json.Unmarshal([]byte(`{"type":"object","additionalProperties":{"type":"object","properties":{"url":{"type":"string"},"weight":{"type":"integer"}}}}`), &node))
	a, err := BuildMapAttribute(&node, "/m")
	require.NoError(t, err)
	mt := a.GetType().(basetypes.MapType)
	et := mt.ElemType.(types.ObjectType)

	cases := map[string]attr.Value{
		"null map":  types.MapNull(et),
		"empty map": types.MapValueMust(et, map[string]attr.Value{}),
		"two keys": types.MapValueMust(et, map[string]attr.Value{
			"a": objectWith(t, et, map[string]attr.Value{"url": types.StringValue("a.invalid"), "weight": types.Int64Value(0)}),
			"b": objectWith(t, et, map[string]attr.Value{"url": types.StringNull(), "weight": types.Int64Value(2)}),
		}),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			wire, err := ToWire(in, path.Root("m"))
			require.NoError(t, err)
			out, diags := FromWire(ctx, mt, apiRoundTrip(t, wire), path.Root("m"))
			require.False(t, diags.HasError(), "%v", diags)
			require.True(t, in.Equal(out), "in=%s out=%s", in, out)
		})
	}
}
