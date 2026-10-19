package smt

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

func TestSurfaceValidateConfigDefersUnknownInstanceIdentity(t *testing.T) {
	ctx := context.Background()
	chain := buildChain(t)
	surface, resourceSchema := testSurface(chain)

	resolved := objectWith(t, chain.EntryType, map[string]attr.Value{
		attrKey:  types.StringValue("users"),
		attrType: types.StringValue("contract_fixture_nested"),
	})
	cases := map[string]types.List{
		"unknown key": types.ListValueMust(chain.EntryType, []attr.Value{
			objectWith(t, chain.EntryType, map[string]attr.Value{
				attrKey:  types.StringUnknown(),
				attrType: types.StringValue("contract_fixture_nested"),
			}),
		}),
		"unknown type": types.ListValueMust(chain.EntryType, []attr.Value{
			objectWith(t, chain.EntryType, map[string]attr.Value{
				attrKey:  types.StringValue("users"),
				attrType: types.StringUnknown(),
			}),
		}),
		"unknown object": types.ListValueMust(chain.EntryType, []attr.Value{
			types.ObjectUnknown(chain.EntryType.AttributeTypes()),
		}),
		"unknown sibling defers known invalid identity": types.ListValueMust(chain.EntryType, []attr.Value{
			objectWith(t, chain.EntryType, map[string]attr.Value{
				attrKey:  types.StringValue(""),
				attrType: types.StringValue("contract_fixture_nested"),
			}),
			objectWith(t, chain.EntryType, map[string]attr.Value{
				attrKey:  types.StringUnknown(),
				attrType: types.StringValue("contract_fixture_nested"),
			}),
		}),
		"resolved identity": types.ListValueMust(chain.EntryType, []attr.Value{resolved}),
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := testSurfaceConfig(t, resourceSchema, value)
			diags := surface.ValidateConfig(ctx, cfg)
			require.False(t, diags.HasError(), "%v", diags)
		})
	}

	t.Run("flat attributes still conflict while identity is unknown", func(t *testing.T) {
		const flat = "legacy_transform"
		surface := NewSurface(chain, map[string]string{flat: "transforms.Legacy.type"})
		attrs := map[string]schema.Attribute{flat: schema.StringAttribute{Optional: true}}
		surface.AddAttributes(attrs)
		resourceSchema := schema.Schema{Attributes: attrs}
		unknown := types.ListValueMust(chain.EntryType, []attr.Value{
			objectWith(t, chain.EntryType, map[string]attr.Value{
				attrKey:  types.StringUnknown(),
				attrType: types.StringValue("contract_fixture_nested"),
			}),
		})
		cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{
			AttrChain: unknown,
			flat:      types.StringValue("configured"),
		})
		diags := surface.ValidateConfig(ctx, cfg)
		require.True(t, diags.HasError())
		require.Equal(t, "Flat and nested transform attributes cannot be mixed", diags.Errors()[0].Summary())
	})
}

func TestSurfaceValidateConfigStillRejectsKnownInvalidKeys(t *testing.T) {
	ctx := context.Background()
	chain := buildChain(t)
	surface, resourceSchema := testSurface(chain)

	cases := map[string]struct {
		entries []attr.Value
		want    string
	}{
		"null object": {
			entries: []attr.Value{types.ObjectNull(chain.EntryType.AttributeTypes())},
			want:    "Missing instance key",
		},
		"empty": {
			entries: []attr.Value{objectWith(t, chain.EntryType, map[string]attr.Value{
				attrKey:  types.StringValue(""),
				attrType: types.StringValue("contract_fixture_nested"),
			})},
			want: "Missing instance key",
		},
		"duplicate": {
			entries: []attr.Value{
				objectWith(t, chain.EntryType, map[string]attr.Value{attrKey: types.StringValue("users"), attrType: types.StringValue("contract_fixture_nested")}),
				objectWith(t, chain.EntryType, map[string]attr.Value{attrKey: types.StringValue("users"), attrType: types.StringValue("contract_fixture_nested")}),
			},
			want: "Duplicate instance key",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := testSurfaceConfig(t, resourceSchema, types.ListValueMust(chain.EntryType, tc.entries))
			diags := surface.ValidateConfig(ctx, cfg)
			require.True(t, diags.HasError())
			require.Equal(t, tc.want, diags.Errors()[0].Summary())
		})
	}
}

func TestSurfaceModifyPlanCorrelatesResolvedInstanceIdentity(t *testing.T) {
	ctx := context.Background()
	chain := buildChain(t)
	surface, resourceSchema := testSurface(chain)
	resolved := types.ListValueMust(chain.EntryType, []attr.Value{
		objectWith(t, chain.EntryType, map[string]attr.Value{
			attrKey:  types.StringValue("users"),
			attrType: types.StringValue("contract_fixture_nested"),
		}),
	})
	prior := types.ListValueMust(chain.EntryType, []attr.Value{
		objectWith(t, chain.EntryType, map[string]attr.Value{
			attrKey:           types.StringValue("users"),
			attrType:          types.StringValue("contract_fixture_nested"),
			attrID:            types.StringValue("instance-7"),
			attrAlias:         types.StringValue("route_users"),
			attrSchemaVersion: types.Int64Value(4),
			attrCodecVersion:  types.Int64Value(2),
		}),
	})

	state := tfsdk.State{Schema: resourceSchema, Raw: testSurfaceRaw(t, resourceSchema, map[string]attr.Value{
		AttrChain:    prior,
		AttrRevision: types.StringValue("revision-6"),
	})}

	t.Run("resolved identity is correlated", func(t *testing.T) {
		cfg := testSurfaceConfig(t, resourceSchema, resolved)
		plan := tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}
		diags := surface.ModifyPlan(ctx, cfg, state, &plan)
		require.False(t, diags.HasError(), "%v", diags)

		var got types.List
		require.False(t, plan.GetAttribute(ctx, path.Root(AttrChain), &got).HasError())
		attrs := got.Elements()[0].(types.Object).Attributes()
		require.Equal(t, "instance-7", attrs[attrID].(types.String).ValueString())
		require.Equal(t, "route_users", attrs[attrAlias].(types.String).ValueString())
		require.Equal(t, int64(4), attrs[attrSchemaVersion].(types.Int64).ValueInt64())
		require.Equal(t, int64(2), attrs[attrCodecVersion].(types.Int64).ValueInt64())
	})

	t.Run("resolved type change is rejected", func(t *testing.T) {
		changed := types.ListValueMust(chain.EntryType, []attr.Value{
			objectWith(t, chain.EntryType, map[string]attr.Value{
				attrKey:  types.StringValue("users"),
				attrType: types.StringValue("contract_fixture_deep"),
			}),
		})
		cfg := testSurfaceConfig(t, resourceSchema, changed)
		plan := tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}
		diags := surface.ModifyPlan(ctx, cfg, state, &plan)
		require.True(t, diags.HasError())
		require.Equal(t, "Transform type change is not allowed in place", diags.Errors()[0].Summary())
	})

	unknown := map[string]types.List{
		"unknown key": types.ListValueMust(chain.EntryType, []attr.Value{
			objectWith(t, chain.EntryType, map[string]attr.Value{attrKey: types.StringUnknown(), attrType: types.StringValue("contract_fixture_nested")}),
		}),
		"unknown type": types.ListValueMust(chain.EntryType, []attr.Value{
			objectWith(t, chain.EntryType, map[string]attr.Value{attrKey: types.StringValue("users"), attrType: types.StringUnknown()}),
		}),
		"unknown object": types.ListValueMust(chain.EntryType, []attr.Value{
			types.ObjectUnknown(chain.EntryType.AttributeTypes()),
		}),
	}
	for name, planned := range unknown {
		t.Run(name+" is not correlated to prior identity", func(t *testing.T) {
			cfg := testSurfaceConfig(t, resourceSchema, planned)
			plan := tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}
			diags := surface.ModifyPlan(ctx, cfg, state, &plan)
			require.False(t, diags.HasError(), "%v", diags)

			var got types.List
			require.False(t, plan.GetAttribute(ctx, path.Root(AttrChain), &got).HasError())
			require.True(t, planned.Equal(got), "unknown identity must not inherit prior server identity")
		})
	}
}

func testSurface(chain *ChainSchema) (*Surface, schema.Schema) {
	surface := NewSurface(chain, nil)
	attrs := map[string]schema.Attribute{}
	surface.AddAttributes(attrs)
	return surface, schema.Schema{Attributes: attrs}
}

func testSurfaceConfig(t *testing.T, resourceSchema schema.Schema, chain types.List) tfsdk.Config {
	t.Helper()
	return testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{
		AttrChain:    chain,
		AttrRevision: types.StringNull(),
	})
}

func testSurfaceConfigValues(t *testing.T, resourceSchema schema.Schema, values map[string]attr.Value) tfsdk.Config {
	t.Helper()
	return tfsdk.Config{Schema: resourceSchema, Raw: testSurfaceRaw(t, resourceSchema, values)}
}

func testSurfaceRaw(t *testing.T, resourceSchema schema.Schema, values map[string]attr.Value) tftypes.Value {
	t.Helper()
	ctx := context.Background()
	raw := map[string]tftypes.Value{}
	for name, attribute := range resourceSchema.Attributes {
		value := values[name]
		if value == nil && name == AttrPredicates {
			value = types.ListValueMust(attribute.GetType().(types.ListType).ElemType, nil)
		}
		if value == nil {
			value = nullOf(attribute.GetType())
		}
		tfValue, err := value.ToTerraformValue(ctx)
		require.NoError(t, err)
		raw[name] = tfValue
	}
	return tftypes.NewValue(resourceSchema.Type().TerraformType(ctx), raw)
}
