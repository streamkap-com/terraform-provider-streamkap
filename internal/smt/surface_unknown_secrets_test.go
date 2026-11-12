package smt

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// A write-only secret can come from a resource created in the same apply, so
// any part of smt_secrets may be unknown at plan. The plan must defer what it
// cannot decide instead of reading unknown as null, zero or "".
func TestSurfaceModifyPlanDefersUnknownSecretInputs(t *testing.T) {
	ctx := context.Background()
	chain, err := BuildChain(NewCatalog(evolutionCorpus(t, evolutionSecretType(1, "token"))).ForHost("destinations").Authorable())
	require.NoError(t, err)
	surface, resourceSchema := testSurface(chain)
	read := &api.SMTChainRead{Desired: api.SMTChainSnapshot{
		ChainRevision: "r1",
		Transforms: []api.SMTInstanceRead{{
			Kind: "typed", ID: "instance-1", Alias: "Route1", Type: "route",
			SchemaVersion: 1, CodecVersion: 1, Name: "old", Enabled: true,
			Config: map[string]any{"value": "retained"},
		}},
	}}
	prior, _, readDiags := surface.stateFrom(ctx, read, []string{"old"}, nil)
	require.False(t, readDiags.HasError(), "%v", readDiags)

	entryType := resourceSchema.Attributes[AttrSecrets].GetType().(types.ListType).ElemType.(types.ObjectType)
	valuesType := entryType.AttributeTypes()[attrValues].(types.ListType)
	rowType := valuesType.ElemType.(types.ObjectType)
	row := func(pointer, value attr.Value, clear attr.Value) attr.Value {
		return types.ObjectValueMust(rowType.AttributeTypes(), map[string]attr.Value{attrPointer: pointer, attrValue: value, attrClear: clear})
	}
	entry := func(key, version, values attr.Value) attr.Value {
		return types.ObjectValueMust(entryType.AttributeTypes(), map[string]attr.Value{attrKey: key, attrVersion: version, attrValues: values})
	}
	rows := func(r ...attr.Value) attr.Value { return types.ListValueMust(rowType, r) }
	one := func(e attr.Value) types.List { return types.ListValueMust(entryType, []attr.Value{e}) }
	pointer, old, v1 := types.StringValue("/token"), types.StringValue("old"), types.Int64Value(1)

	cases := map[string]struct {
		secrets types.List
		// version is the secret_version the plan records for "old"; nil means unknown.
		version *int64
	}{
		"unknown value":   {secrets: one(entry(old, v1, rows(row(pointer, types.StringUnknown(), types.BoolNull())))), version: ptr(1)},
		"unknown clear":   {secrets: one(entry(old, v1, rows(row(pointer, types.StringNull(), types.BoolUnknown())))), version: ptr(1)},
		"unknown pointer": {secrets: one(entry(old, v1, rows(row(types.StringUnknown(), types.StringValue("s"), types.BoolNull())))), version: ptr(1)},
		"unknown values":  {secrets: one(entry(old, v1, types.ListUnknown(rowType))), version: ptr(1)},
		"unknown version": {secrets: one(entry(old, types.Int64Unknown(), rows(row(pointer, types.StringValue("s"), types.BoolNull()))))},
		"unknown key":     {secrets: one(entry(types.StringUnknown(), v1, rows(row(pointer, types.StringValue("s"), types.BoolNull()))))},
		"unknown entry":   {secrets: one(types.ObjectUnknown(entryType.AttributeTypes()))},
		"unknown list":    {secrets: types.ListUnknown(entryType)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			predicates := types.ListValueMust(chain.PredicateEntryType, nil)
			cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{
				AttrChain: prior, AttrSecrets: tc.secrets, AttrPredicates: predicates, AttrRevision: types.StringNull(),
			})
			state := tfsdk.State{Schema: resourceSchema, Raw: testSurfaceRaw(t, resourceSchema, map[string]attr.Value{
				AttrChain: prior, AttrPredicates: predicates, AttrRevision: types.StringValue("r1"),
			})}
			plan := tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}

			diags := surface.ModifyPlan(ctx, cfg, state, &plan)
			require.False(t, diags.HasError(), "%v", diags)

			var planned types.List
			require.False(t, plan.GetAttribute(ctx, path.Root(AttrChain), &planned).HasError())
			got := planned.Elements()[0].(types.Object).Attributes()[attrSecretVersion].(types.Int64)
			if tc.version == nil {
				require.True(t, got.IsUnknown(), "secret_version must be unknown, got %s", got)
			} else {
				require.Equal(t, types.Int64Value(*tc.version), got)
			}
			var revision types.String
			require.False(t, plan.GetAttribute(ctx, path.Root(AttrRevision), &revision).HasError())
			require.NotEqual(t, types.StringValue("r1"), revision, "a possible rotation must not pin the prior revision")
		})
	}

	t.Run("known ambiguous row is still rejected", func(t *testing.T) {
		secrets := one(entry(old, v1, rows(row(pointer, types.StringValue("s"), types.BoolValue(true)))))
		predicates := types.ListValueMust(chain.PredicateEntryType, nil)
		cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{
			AttrChain: prior, AttrSecrets: secrets, AttrPredicates: predicates, AttrRevision: types.StringNull(),
		})
		state := tfsdk.State{Schema: resourceSchema, Raw: testSurfaceRaw(t, resourceSchema, map[string]attr.Value{
			AttrChain: prior, AttrPredicates: predicates, AttrRevision: types.StringValue("r1"),
		})}
		plan := tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}
		diags := surface.ModifyPlan(ctx, cfg, state, &plan)
		require.True(t, diags.HasError())
		require.Equal(t, "Ambiguous secret operation", diags.Errors()[0].Summary())
	})
}

// A new predicate's required secret may be unknown at plan; the plan defers
// the requirement and the apply, which sees it known, refuses anything still
// unresolved instead of sending an empty secret.
func TestNewPredicateUnknownSecretDefersToApply(t *testing.T) {
	ctx := context.Background()
	s := predicateSurfaceFixture(t)
	chain := s.Chain
	predicates := types.ListValueMust(chain.PredicateEntryType, []attr.Value{predicateFixtureValue(t, chain, "q", "topic_match_secret", "payments", "", 0, nil)})
	transforms := types.ListValueMust(chain.EntryType, []attr.Value{transformFixtureValue(t, chain, "a", "", "route", "q", false)})
	entryType := PredicateSecretsAttribute().NestedObject.Type().(types.ObjectType)
	rowType := entryType.AttributeTypes()[attrValues].(types.ListType).ElemType.(types.ObjectType)
	secrets := types.ListValueMust(entryType, []attr.Value{objectWith(t, entryType, map[string]attr.Value{
		attrKey:     types.StringValue("q"),
		attrVersion: types.Int64Value(1),
		attrValues: types.ListValueMust(rowType, []attr.Value{objectWith(t, rowType, map[string]attr.Value{
			attrPointer: types.StringValue("/token"),
			attrValue:   types.StringUnknown(),
		})}),
	})})
	_, resourceSchema := testSurface(chain)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{AttrChain: transforms, AttrPredicates: predicates, AttrPredicateSecrets: secrets, AttrRevision: types.StringNull()})
	create := tfsdk.State{Schema: resourceSchema, Raw: tftypes.NewValue(resourceSchema.Type().TerraformType(ctx), nil)}
	plan := tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}

	diags := s.ModifyPlan(ctx, cfg, create, &plan)
	require.False(t, diags.HasError(), "%v", diags)

	_, diags = s.PlanWrite(ctx, plan, types.ListNull(chain.EntryType), cfg, types.ListNull(chain.PredicateEntryType), types.MapNull(types.BoolType))
	require.True(t, diags.HasError())
	require.Equal(t, "Secret input unknown at apply", diags.Errors()[0].Summary())
}

func ptr(n int64) *int64 { return &n }
