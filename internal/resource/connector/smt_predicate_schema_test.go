package connector_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/smt"
)

func TestGeneratedSMTCatalogFeedsTransformAndPredicateAuthoring(t *testing.T) {
	attrs := map[string]schema.Attribute{}
	smt.NewSurface(generated.SMTChain("destinations"), nil).AddAttributes(attrs)

	transforms := attrs[smt.AttrChain].(schema.ListNestedAttribute)
	predicates, ok := attrs["smt_predicates"].(schema.ListNestedAttribute)
	require.True(t, ok, "the generated catalog must reach the predicate schema before role filtering")
	if !ok {
		return
	}
	require.Contains(t, transforms.NestedObject.Attributes, "regex_router")
	require.NotContains(t, transforms.NestedObject.Attributes, "topic_name_matches")
	require.Contains(t, predicates.NestedObject.Attributes, "topic_name_matches")
	require.NotContains(t, predicates.NestedObject.Attributes, "regex_router")
}
