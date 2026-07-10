package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// deprecationTargetPattern extracts the replacement attribute from the
// "Use 'x' instead." convention every deprecated alias in this provider uses.
var deprecationTargetPattern = regexp.MustCompile(`Use '([^']+)' instead`)

// TestDeprecatedAliasTargetsExist asserts that every deprecated attribute points
// at an attribute that is actually in the schema.
//
// A backend field can be dropped from configuration.latest.json, which removes
// the new attribute from the generated schema while the hand-maintained alias in
// <connector>_generated.go keeps referring to it. Nothing else catches that: the
// alias only fails at plan time, when a v2 user sets it and ConflictsWith tries
// to resolve a path expression that matches nothing.
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
			for attrName, attr := range schemaResp.Schema.Attributes {
				match := deprecationTargetPattern.FindStringSubmatch(attr.GetDeprecationMessage())
				if match == nil {
					continue
				}
				target := match[1]
				_, exists := schemaResp.Schema.Attributes[target]
				assert.Truef(t, exists,
					"deprecated attribute %q directs users to %q, which is not in the schema; "+
						"either the alias is stale or the replacement was dropped",
					attrName, target)
			}
		})
	}
}
