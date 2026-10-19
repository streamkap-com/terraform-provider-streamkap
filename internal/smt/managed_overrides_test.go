package smt

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

type managedOverrideClient struct {
	read   *api.SMTChainRead
	err    error
	writes []api.SMTChainWrite
}

func (c *managedOverrideClient) GetSMTChain(context.Context, string, string) (*api.SMTChainRead, error) {
	return c.read, c.err
}

func (c *managedOverrideClient) GetSMTChainAdoption(context.Context, string, string) (*api.SMTChainAdoptionRead, error) {
	return nil, nil
}

func (c *managedOverrideClient) PutSMTChain(_ context.Context, _, _ string, write api.SMTChainWrite) (*api.SMTChainRead, error) {
	c.writes = append(c.writes, write)
	return c.read, nil
}

func overrideMap(values map[string]bool) types.Map {
	elements := make(map[string]attr.Value, len(values))
	for key, value := range values {
		elements[key] = types.BoolValue(value)
	}
	return types.MapValueMust(types.BoolType, elements)
}

func TestManagedOverridesAreIndependentFromUserChainOwnership(t *testing.T) {
	ctx := context.Background()
	chain := buildChain(t)
	surface, resourceSchema := testSurface(chain)
	emptyChain := types.ListValueMust(chain.EntryType, nil)
	emptyPredicates := types.ListValueMust(chain.PredicateEntryType, nil)
	priorOverrides := overrideMap(map[string]bool{"user_defined:MaskField": true})

	t.Run("omission preserves overrides while replacing user chain", func(t *testing.T) {
		cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: emptyChain, AttrPredicates: emptyPredicates})
		write, diags := surface.PlanWrite(ctx, tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}, emptyChain, cfg, emptyPredicates, priorOverrides)
		require.False(t, diags.HasError(), "%v", diags)
		require.True(t, write.ChainConfigured)
		require.Nil(t, write.ManagedOverrides)
		require.True(t, write.Unchanged)
	})

	t.Run("empty map resets overrides without taking user chain ownership", func(t *testing.T) {
		emptyOverrides := overrideMap(map[string]bool{})
		cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: types.ListNull(chain.EntryType), AttrManagedOverrides: emptyOverrides})
		write, diags := surface.PlanWrite(ctx, tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}, types.ListNull(chain.EntryType), cfg, types.ListNull(chain.PredicateEntryType), priorOverrides)
		require.False(t, diags.HasError(), "%v", diags)
		require.True(t, write.Configured)
		require.False(t, write.ChainConfigured)
		require.NotNil(t, write.ManagedOverrides)
		require.Empty(t, *write.ManagedOverrides)
		require.False(t, write.Unchanged)
	})
}

func TestManagedOverrideOnlyApplyOmitsGraphAndReadsExactState(t *testing.T) {
	ctx := context.Background()
	chain := buildChain(t)
	surface, resourceSchema := testSurface(chain)
	configured := overrideMap(map[string]bool{"user_defined:MaskField": false})
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: types.ListNull(chain.EntryType), AttrManagedOverrides: configured})
	write, diags := surface.PlanWrite(ctx, tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}, types.ListNull(chain.EntryType), cfg, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
	require.False(t, diags.HasError(), "%v", diags)

	client := &managedOverrideClient{read: &api.SMTChainRead{Desired: api.SMTChainSnapshot{
		ChainRevision:    "r2",
		ManagedOverrides: map[string]bool{"user_defined:MaskField": false},
		Transforms: []api.SMTInstanceRead{{
			Kind: "opaque", ID: "unmanaged", Alias: "Unmanaged1", Reason: "outside provider catalog",
		}},
	}}}
	state, applyDiags := surface.ApplyGraph(ctx, client, "destinations", "destination-1", write, types.StringValue("r1"))
	require.False(t, applyDiags.HasError(), "%v", applyDiags)
	require.Len(t, client.writes, 1)
	require.Nil(t, client.writes[0].Transforms)
	require.Nil(t, client.writes[0].Predicates)
	require.Equal(t, &map[string]bool{"user_defined:MaskField": false}, client.writes[0].ManagedOverrides)
	require.True(t, state.Transforms.IsNull(), "override-only ownership must not adopt the user chain")
	require.True(t, state.Predicates.IsNull())
	require.Equal(t, configured, state.ManagedOverrides)
	require.Equal(t, "r2", state.Revision.ValueString())
}

func TestManagedOverrideValidationIssuesUseOverrideAttribute(t *testing.T) {
	chain := buildChain(t)
	surface := NewSurface(chain, nil)
	write := &Write{}
	require.Equal(t, path.Root(AttrManagedOverrides), surface.issuePath("/managed_overrides/user_defined:Missing", write))
}

func TestManagedOverrideOnlyRefreshErrorPreservesOwnedStateAndPath(t *testing.T) {
	ctx := context.Background()
	chain := buildChain(t)
	surface := NewSurface(chain, nil)
	priorOverrides := overrideMap(map[string]bool{"user_defined:MaskField": false})

	state, diags := surface.RefreshGraph(
		ctx,
		&managedOverrideClient{err: errors.New("backend unavailable")},
		"destinations",
		"destination-1",
		types.ListNull(chain.EntryType),
		types.ListNull(chain.PredicateEntryType),
		priorOverrides,
	)

	require.True(t, diags.HasError())
	require.Equal(t, priorOverrides, state.ManagedOverrides)
	require.True(t, state.Transforms.IsNull())
	require.True(t, state.Predicates.IsNull())
	diagnosticWithPath, ok := diags.Errors()[0].(diag.DiagnosticWithPath)
	require.True(t, ok)
	require.True(t, path.Root(AttrManagedOverrides).Equal(diagnosticWithPath.Path()))
}

func TestManagedOverridesRejectUnknownBooleanBeforeHTTP(t *testing.T) {
	ctx := context.Background()
	chain := buildChain(t)
	surface, resourceSchema := testSurface(chain)
	unknown := types.MapValueMust(types.BoolType, map[string]attr.Value{"user_defined:MaskField": types.BoolUnknown()})
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: types.ListNull(chain.EntryType), AttrManagedOverrides: unknown})
	_, diags := surface.PlanWrite(ctx, tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}, types.ListNull(chain.EntryType), cfg, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
	require.True(t, diags.HasError())
	require.Equal(t, "Managed override is unknown", diags.Errors()[0].Summary())
}
