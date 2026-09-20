package smtproto

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
)

const protoTypeName = "streamkap_smtproto_source_postgresql"

// releasedState encodes a state exactly as the released v3 provider stores
// it: every attribute of its own schema, most of them null, and no chain
// attributes at all. This is the `attributes` object of a state file.
func releasedState(t *testing.T, set map[string]any) []byte {
	t.Helper()
	ctx := context.Background()
	released := &resource.SchemaResponse{}
	source.NewPostgreSQLResource().Schema(ctx, resource.SchemaRequest{}, released)
	require.False(t, released.Diagnostics.HasError())
	objType := released.Schema.Type().TerraformType(ctx).(tftypes.Object)
	require.NotContains(t, objType.AttributeTypes, AttrChain, "the released schema has no chain")

	values := map[string]any{}
	for name := range objType.AttributeTypes {
		values[name] = nil
	}
	for name, v := range set {
		_, known := objType.AttributeTypes[name]
		require.True(t, known, "%s is not a released attribute", name)
		values[name] = v
	}
	raw, err := json.Marshal(values)
	require.NoError(t, err)
	return raw
}

// A state written by the released provider decodes against the prototype
// schema with both chain attributes null, refreshes without populating the
// chain, and plans no change.
func TestReleasedStateDecodesAndConverges(t *testing.T) {
	ctx := context.Background()
	res, fake := newProto(t)
	connID, _, err := fake.Create(ctx, nil)
	require.NoError(t, err)

	raw := releasedState(t, map[string]any{
		"id":                  connID,
		"name":                "legacy",
		"connector":           "postgresql",
		"connector_status":    "Active",
		"database_hostname":   "db.invalid",
		"database_user":       "u",
		"database_password":   "p",
		"database_dbname":     "d",
		"schema_include_list": "public",
		"table_include_list":  "public.t",
		"transforms_value_to_key_fields_include_list":       "id",
		"transforms_oversized_records_max_field_size_bytes": 2097152,
		"insert_static_key_field_1":                         "tenant",
	})

	srv, err := providerFactories(res)["streamkap"]()
	require.NoError(t, err)
	schemaResp, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	require.NoError(t, err)
	require.Empty(t, schemaResp.Diagnostics)
	protoType := schemaResp.ResourceSchemas[protoTypeName].ValueType()

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
	var s string
	require.NoError(t, attrs["transforms_value_to_key_fields_include_list"].As(&s))
	require.Equal(t, "id", s)
	require.NoError(t, attrs["insert_static_key_field_1"].As(&s))
	require.Equal(t, "tenant", s)

	read, err := srv.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: protoTypeName, CurrentState: upgraded.UpgradedState})
	require.NoError(t, err)
	require.Empty(t, read.Diagnostics)
	after, err := read.NewState.Unmarshal(protoType)
	require.NoError(t, err)
	require.True(t, state.Equal(after), "refresh must leave a released state untouched, chain included")

	// Config is the state minus the computed attributes; core would propose
	// the prior state for an unchanged configuration. The synthetic state
	// holds null where a real one holds the released Defaults, so the plan
	// fills those in; what must not move is the chain and the flat values
	// the state does carry. Convergence of a real flat apply is shown by
	// TestResource_FlatSurfaceUnchanged under terraform itself.
	configAttrs := map[string]tftypes.Value{}
	for name, v := range attrs {
		configAttrs[name] = v
	}
	for _, computedOnly := range []string{"id", "connector", "connector_status"} {
		configAttrs[computedOnly] = tftypes.NewValue(tftypes.String, nil)
	}
	config, err := tfprotov6.NewDynamicValue(protoType, tftypes.NewValue(protoType, configAttrs))
	require.NoError(t, err)
	plan, err := srv.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         protoTypeName,
		PriorState:       read.NewState,
		ProposedNewState: read.NewState,
		Config:           &config,
	})
	require.NoError(t, err)
	require.Empty(t, plan.Diagnostics)
	planned, err := plan.PlannedState.Unmarshal(protoType)
	require.NoError(t, err)
	var plannedAttrs map[string]tftypes.Value
	require.NoError(t, planned.As(&plannedAttrs))
	require.True(t, plannedAttrs[AttrChain].IsNull(), "plan invented a chain: %s", plannedAttrs[AttrChain])
	require.True(t, plannedAttrs[AttrSecrets].IsNull())
	for _, name := range []string{"transforms_value_to_key_fields_include_list", "transforms_oversized_records_max_field_size_bytes", "insert_static_key_field_1", "name", "id"} {
		require.True(t, attrs[name].Equal(plannedAttrs[name]), "%s moved: %s -> %s", name, attrs[name], plannedAttrs[name])
	}
}
