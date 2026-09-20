package smtproto

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
)

const protoTypeName = "streamkap_smtproto_source_postgresql"

// releasedState encodes a state as the released v3 provider stores it: its
// own attribute set with every Default-bearing attribute holding its resolved
// default (a real v3 state always does, the flat SMT attributes included),
// the configured values on top, everything else null, and no chain
// attributes at all. It returns the `attributes` object of a state file, the
// attribute names that hold a value, and the released schema's attributes.
func releasedState(t *testing.T, configured map[string]any) ([]byte, []string, map[string]schema.Attribute) {
	t.Helper()
	ctx := context.Background()
	released := &resource.SchemaResponse{}
	source.NewPostgreSQLResource().Schema(ctx, resource.SchemaRequest{}, released)
	require.False(t, released.Diagnostics.HasError())
	require.NotContains(t, released.Schema.Attributes, AttrChain, "the released schema has no chain")

	values := map[string]any{"timeouts": nil}
	var populated []string
	for name, a := range released.Schema.Attributes {
		values[name] = nil
		if v, ok := defaultValue(a); ok {
			wire, err := ToWire(v, path.Root(name))
			require.NoError(t, err)
			values[name] = wire
			populated = append(populated, name)
		}
	}
	for name, v := range configured {
		_, known := released.Schema.Attributes[name]
		require.True(t, known, "%s is not a released attribute", name)
		if values[name] == nil {
			populated = append(populated, name)
		}
		values[name] = v
	}
	sort.Strings(populated)
	raw, err := json.Marshal(values)
	require.NoError(t, err)
	return raw, populated, released.Schema.Attributes
}

// A state written by the released provider decodes against the prototype
// schema with both chain attributes null, refreshes without populating the
// chain, and plans no change to any attribute when the configuration is the
// one that produced it.
//
// This is the framework in isolation: ProposedNewState is built here the
// way core builds it (config where set, prior state for computed attributes
// otherwise). The same convergence under the real CLI is
// TestResource_FlatSurfaceUnchanged.
func TestReleasedStateDecodesAndConverges(t *testing.T) {
	ctx := context.Background()
	res, fake := newProto(t)
	connID, _, err := fake.Create(ctx, nil)
	require.NoError(t, err)

	// What the practitioner wrote: the required attributes, two flat SMT
	// values that have no default, and two that override a default.
	configured := map[string]any{
		"name":                "legacy",
		"database_hostname":   "db.invalid",
		"database_user":       "u",
		"database_password":   "p",
		"database_dbname":     "d",
		"schema_include_list": "public",
		"table_include_list":  "public.t",
		"transforms_value_to_key_fields_include_list":           "id",
		"insert_static_key_field_1":                             "tenant",
		"transforms_oversized_records_max_field_size_bytes":     2097152,
		"transforms_oversized_records_oversized_field_behavior": "NULLIFY",
	}
	computedOnly := map[string]any{"id": connID, "connector": "postgresql", "connector_status": "Active"}
	stored := map[string]any{}
	for k, v := range configured {
		stored[k] = v
	}
	for k, v := range computedOnly {
		stored[k] = v
	}
	raw, populated, releasedAttrs := releasedState(t, stored)
	require.Greater(t, len(populated), 30, "the fixture must carry the resolved defaults: %v", populated)
	require.Contains(t, populated, "transforms_source_regex_support_regex_replacement")
	require.Contains(t, populated, "transforms_oversized_records_replace_null_with_default")

	srv, err := providerFactories(res)["streamkap"]()
	require.NoError(t, err)
	schemaResp, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	require.NoError(t, err)
	require.Empty(t, schemaResp.Diagnostics)
	protoType := schemaResp.ResourceSchemas[protoTypeName].ValueType().(tftypes.Object)

	upgraded, err := srv.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: protoTypeName,
		Version:  0,
		RawState: &tfprotov6.RawState{JSON: raw},
	})
	require.NoError(t, err)
	require.Empty(t, upgraded.Diagnostics)
	state, err := upgraded.UpgradedState.Unmarshal(protoType)
	require.NoError(t, err)

	var attrs map[string]tftypes.Value
	require.NoError(t, state.As(&attrs))
	require.True(t, attrs[AttrChain].IsNull(), "released state has no chain: %s", attrs[AttrChain])
	require.True(t, attrs[AttrSecrets].IsNull())
	for _, name := range populated {
		require.False(t, attrs[name].IsNull(), "%s was lost in decode", name)
	}
	var s string
	require.NoError(t, attrs["transforms_oversized_records_oversized_field_behavior"].As(&s))
	require.Equal(t, "NULLIFY", s)

	read, err := srv.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: protoTypeName, CurrentState: upgraded.UpgradedState})
	require.NoError(t, err)
	require.Empty(t, read.Diagnostics)
	after, err := read.NewState.Unmarshal(protoType)
	require.NoError(t, err)
	require.True(t, state.Equal(after), "refresh must leave a released state untouched, chain included")

	// Core's proposed new state: the configured value where one is set,
	// the prior value for a computed attribute, null otherwise.
	configAttrs := map[string]tftypes.Value{}
	proposedAttrs := map[string]tftypes.Value{}
	for name, typ := range protoType.AttributeTypes {
		configAttrs[name] = tftypes.NewValue(typ, nil)
		proposedAttrs[name] = tftypes.NewValue(typ, nil)
		if _, set := configured[name]; set {
			configAttrs[name] = attrs[name]
			proposedAttrs[name] = attrs[name]
			continue
		}
		if a, ok := releasedAttrs[name]; ok && a.IsComputed() {
			proposedAttrs[name] = attrs[name]
		}
	}
	config, err := tfprotov6.NewDynamicValue(protoType, tftypes.NewValue(protoType, configAttrs))
	require.NoError(t, err)
	proposed, err := tfprotov6.NewDynamicValue(protoType, tftypes.NewValue(protoType, proposedAttrs))
	require.NoError(t, err)
	plan, err := srv.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         protoTypeName,
		PriorState:       read.NewState,
		ProposedNewState: &proposed,
		Config:           &config,
	})
	require.NoError(t, err)
	require.Empty(t, plan.Diagnostics)
	planned, err := plan.PlannedState.Unmarshal(protoType)
	require.NoError(t, err)
	var plannedAttrs map[string]tftypes.Value
	require.NoError(t, planned.As(&plannedAttrs))
	for name := range protoType.AttributeTypes {
		require.True(t, attrs[name].Equal(plannedAttrs[name]), "%s moved: %s -> %s", name, attrs[name], plannedAttrs[name])
	}
}
