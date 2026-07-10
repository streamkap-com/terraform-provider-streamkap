package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deprecationTargetPattern extracts the replacement attribute from the
// "Use 'x' instead." convention every deprecated alias in this provider uses.
var deprecationTargetPattern = regexp.MustCompile(`Use '([^']+)' instead`)

// configWithOnlyAttrSet builds a tfsdk.Config for s in which every attribute is
// null except attrName, which carries value. ConflictsWith short-circuits on a
// null config value, so the alias must be non-null for its path expression to
// be resolved against the schema.
func configWithOnlyAttrSet(ctx context.Context, s rschema.Schema, attrName string, value tftypes.Value) tfsdk.Config {
	objType := s.Type().(attr.TypeWithAttributeTypes)
	raw := map[string]tftypes.Value{}
	for name, attrType := range objType.AttributeTypes() {
		raw[name] = tftypes.NewValue(attrType.TerraformType(ctx), nil)
	}
	raw[attrName] = value

	return tfsdk.Config{
		Schema: s,
		Raw:    tftypes.NewValue(s.Type().TerraformType(ctx).(tftypes.Object), raw),
	}
}

// runAttributeValidators invokes every validator attached to attrName with a
// non-null value, returning whatever diagnostics they emit.
func runAttributeValidators(ctx context.Context, s rschema.Schema, attrName string, a rschema.Attribute) diag.Diagnostics {
	var diags diag.Diagnostics
	p := path.Root(attrName)

	switch typed := a.(type) {
	case rschema.StringAttribute:
		cfg := configWithOnlyAttrSet(ctx, s, attrName, tftypes.NewValue(tftypes.String, "sentinel"))
		for _, v := range typed.Validators {
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, validator.StringRequest{
				Path: p, PathExpression: p.Expression(), Config: cfg,
				ConfigValue: types.StringValue("sentinel"),
			}, resp)
			diags.Append(resp.Diagnostics...)
		}
	case rschema.Int64Attribute:
		cfg := configWithOnlyAttrSet(ctx, s, attrName, tftypes.NewValue(tftypes.Number, 1))
		for _, v := range typed.Validators {
			resp := &validator.Int64Response{}
			v.ValidateInt64(ctx, validator.Int64Request{
				Path: p, PathExpression: p.Expression(), Config: cfg,
				ConfigValue: types.Int64Value(1),
			}, resp)
			diags.Append(resp.Diagnostics...)
		}
	case rschema.BoolAttribute:
		cfg := configWithOnlyAttrSet(ctx, s, attrName, tftypes.NewValue(tftypes.Bool, true))
		for _, v := range typed.Validators {
			resp := &validator.BoolResponse{}
			v.ValidateBool(ctx, validator.BoolRequest{
				Path: p, PathExpression: p.Expression(), Config: cfg,
				ConfigValue: types.BoolValue(true),
			}, resp)
			diags.Append(resp.Diagnostics...)
		}
	}

	return diags
}

// TestDeprecatedAliasTargetsExist asserts that every deprecated attribute points
// at an attribute that is actually in the schema, and that its ConflictsWith
// path expression resolves.
//
// A backend field can be dropped from configuration.latest.json, which removes
// the new attribute from the generated schema while the hand-maintained alias in
// <connector>_generated.go keeps referring to it. Nothing else catches that: the
// alias only fails at plan time, when a v2 user sets it and ConflictsWith tries
// to resolve a path expression that matches nothing.
//
// The DeprecationMessage and the ConflictsWith target are separate literals for
// the snowflake and kafkadirect aliases, so both are checked: the message tells
// users where to go, the validator enforces it, and they can disagree.
func TestDeprecatedAliasTargetsExist(t *testing.T) {
	ctx := context.Background()
	p := &streamkapProvider{}

	for _, factory := range p.Resources(ctx) {
		res := factory()

		metaResp := &resource.MetadataResponse{}
		res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "streamkap"}, metaResp)

		schemaResp := &resource.SchemaResponse{}
		res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
		require.False(t, schemaResp.Diagnostics.HasError(), "schema should not have errors")

		t.Run(metaResp.TypeName, func(t *testing.T) {
			for attrName, attribute := range schemaResp.Schema.Attributes {
				if attribute.GetDeprecationMessage() == "" {
					continue
				}

				// The message must name an attribute users can actually migrate to.
				if match := deprecationTargetPattern.FindStringSubmatch(attribute.GetDeprecationMessage()); match != nil {
					_, exists := schemaResp.Schema.Attributes[match[1]]
					assert.Truef(t, exists,
						"deprecated attribute %q directs users to %q, which is not in the schema; "+
							"either the alias is stale or the replacement was dropped",
						attrName, match[1])
				}

				// ConflictsWith must resolve against the schema. A path expression
				// that matches nothing errors at plan time, not at schema build.
				for _, d := range runAttributeValidators(ctx, schemaResp.Schema, attrName, attribute) {
					assert.NotContainsf(t, d.Summary(), "Invalid Path Expression",
						"deprecated attribute %q has a validator whose path expression does not "+
							"resolve against the schema: %s: %s", attrName, d.Summary(), d.Detail())
				}
			}
		})
	}
}
