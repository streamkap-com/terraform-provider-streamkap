package smt

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func entry(t *testing.T, cs *ChainSchema, key, typeID, name string, child map[string]attr.Value, extra map[string]attr.Value) types.Object {
	t.Helper()
	set := map[string]attr.Value{
		attrKey:           types.StringValue(key),
		attrType:          types.StringValue(typeID),
		attrName:          types.StringValue(name),
		attrEnabled:       types.BoolValue(true),
		attrAlias:         types.StringValue("server-alias"),
		attrSchemaVersion: types.Int64Value(1),
	}
	if child != nil {
		set[typeID] = objectWith(t, cs.EntryType.AttributeTypes()[typeID].(types.ObjectType), child)
	}
	for k, v := range extra {
		set[k] = v
	}
	return objectWith(t, cs.EntryType, set)
}

// The DTO carries type, name, the enabled flag, the echoed id of a matched
// instance, the unwrapped config child and the key's secret operations. Key,
// alias and schema version never leave the provider.
func TestProject(t *testing.T) {
	cs := buildChain(t)
	nested := nestedChildType(t, cs)
	rowType := nested.AttributeTypes()["routes"].(types.ListType).ElemType.(types.ObjectType)

	child := map[string]attr.Value{
		"enabled": types.BoolValue(false), "retries": types.Int64Value(0), "fallback": types.StringNull(),
		"labels": strList(), "max_records": types.Int64Value(0),
		"output": objectWith(t, nested.AttributeTypes()["output"].(types.ObjectType), map[string]attr.Value{
			"prefix": types.StringValue(""), "label": types.StringValue(""), "retry_limit": types.Int64Value(0),
		}),
		"routes": types.ListValueMust(rowType, []attr.Value{
			objectWith(t, rowType, map[string]attr.Value{"key": types.StringValue("row-east"), "label": types.StringValue("east"), "endpoint": types.StringValue("example.invalid")}),
			objectWith(t, rowType, map[string]attr.Value{"key": types.StringValue("row~west/1"), "label": types.StringValue("west"), "endpoint": types.StringValue("other.invalid")}),
		}),
	}
	planned := types.ListValueMust(cs.EntryType, []attr.Value{
		entry(t, cs, "matched", "contract_fixture_nested", "East", child, map[string]attr.Value{attrID: types.StringValue("inst-1")}),
		entry(t, cs, "new", "contract_fixture_nested", "West", child, map[string]attr.Value{attrID: types.StringUnknown(), attrEnabled: types.BoolValue(false)}),
	})
	correlated := []Correlated{
		{Key: "matched", Type: "contract_fixture_nested", ID: "inst-1", Alias: "server-alias", SchemaVersion: 1, Matched: true},
		{Key: "new", Type: "contract_fixture_nested"},
	}
	ops := []SecretOp{
		{Key: "new", Pointer: "/routes/row-east/token", Value: "v"},
		{Key: "new", Pointer: "/routes/row~0west~11/token", Clear: true},
	}
	writes, diags := cs.Project(planned, correlated, ops)
	require.False(t, diags.HasError(), "%v", diags)
	require.Len(t, writes, 2)

	raw, err := json.Marshal(writes)
	require.NoError(t, err)
	var wire []map[string]any
	require.NoError(t, json.Unmarshal(raw, &wire))

	require.Equal(t, "inst-1", wire[0]["id"])
	require.Equal(t, "East", wire[0]["name"])
	require.Equal(t, true, wire[0]["enabled"])
	_, hasOps := wire[0]["secret_operations"]
	require.False(t, hasOps, "an instance without operations sends none")
	_, hasID := wire[1]["id"]
	require.False(t, hasID, "a new key sends no id")
	require.Equal(t, "West", wire[1]["name"])
	require.Equal(t, false, wire[1]["enabled"])
	require.Equal(t, []any{
		map[string]any{"pointer": "/routes/row-east/token", "operation": "replace", "value": "v"},
		map[string]any{"pointer": "/routes/row~0west~11/token", "operation": "clear"},
	}, wire[1]["secret_operations"], "operations ride with their instance")
	for _, w := range wire {
		require.Equal(t, "contract_fixture_nested", w["type"])
		for _, forbidden := range []string{"key", "alias", "schema_version", "secret_version", "predicate", "contract_fixture_nested", "contract_fixture_deep"} {
			_, has := w[forbidden]
			require.False(t, has, "%s must not be on the wire", forbidden)
		}
		require.Equal(t, apiRoundTrip(t, cs.Catalog["contract_fixture_nested"].Examples.ReadProjection), w["config"],
			"config is the unwrapped child and equals the corpus read projection")
	}
}

func TestProject_RejectsWrongOrMissingChild(t *testing.T) {
	cs := buildChain(t)
	deep := cs.EntryType.AttributeTypes()["contract_fixture_deep"].(types.ObjectType)

	missing := types.ListValueMust(cs.EntryType, []attr.Value{entry(t, cs, "a", "contract_fixture_nested", "", nil, nil)})
	_, diags := cs.Project(missing, []Correlated{{Key: "a", Type: "contract_fixture_nested"}}, nil)
	require.True(t, diags.HasError())
	require.Equal(t, "Missing config child", diags.Errors()[0].Summary())

	wrong := types.ListValueMust(cs.EntryType, []attr.Value{entry(t, cs, "a", "contract_fixture_nested", "", nil, map[string]attr.Value{
		"contract_fixture_deep": objectWith(t, deep, nil),
	})})
	_, diags = cs.Project(wrong, []Correlated{{Key: "a", Type: "contract_fixture_nested"}}, nil)
	require.True(t, diags.HasError())
	summaries := []string{}
	for _, d := range diags.Errors() {
		summaries = append(summaries, d.Summary())
	}
	require.ElementsMatch(t, []string{"Missing config child", "Config child does not match type"}, summaries)
}

func TestFlatSMTAttributes_DerivedFromReleasedMappings(t *testing.T) {
	names := FlatSMTAttributes(map[string]string{
		"transforms_value_to_key_fields_include_list": "transforms.ValueToKey.fields.include.list",
		"predicates_is_topic_to_enrich_pattern":       "predicates.IsTopicToEnrich.pattern",
		"database_hostname":                           "database.hostname.user.defined",
		"insert_static_key_field_1":                   "transforms.InsertStaticKey1.static.field",
	})
	require.Equal(t, []string{"insert_static_key_field_1", "predicates_is_topic_to_enrich_pattern", "transforms_value_to_key_fields_include_list"}, names)
}
