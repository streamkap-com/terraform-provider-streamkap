package smt

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func catalogArtifact(types string) []byte {
	return []byte(fmt.Sprintf(`{"contract_version":1,"producer":"test","digest":"sha256:test","types":[%s]}`, types))
}

func catalogType(typeID, role string, publishable bool) string {
	return fmt.Sprintf(
		`{"type_id":%q,"schema_version":1,"codec_version":1,"publishable":%t,"authorable":%t,"role":%q,"data_schema":{"type":"object","properties":{"value":{"type":"string"}},"required":["value"],"additionalProperties":false},"consumer_requirements":{"version":1,"schema_keywords":["additionalProperties","properties","required","type"],"schema_types":["object","string"],"schema_formats":[],"ui_elements":[],"ui_controls":[],"secret_addressing":[]},"secret_paths":[],"row_keys":{},"examples":{"read_projection":{"value":"fixture"}}}`,
		typeID,
		publishable,
		publishable,
		role,
	)
}

func TestPublishable_OffersOnlyTransformRolesForTransformAuthoring(t *testing.T) {
	raw := catalogArtifact(fmt.Sprintf(
		"%s,%s,%s",
		catalogType("route", "transform", true),
		catalogType("topic_match", "predicate", true),
		catalogType("fixture", "transform", false),
	))

	corpus, err := ParseCorpus(raw)
	require.NoError(t, err)
	fullCatalog := NewCatalog(corpus)
	require.Equal(t, "predicate", currentSpec(t, fullCatalog, "predicate", "topic_match").Role, "the full parsed artifact retains predicate metadata")

	authorable := fullCatalog.Publishable()
	require.Equal(t, []string{"route"}, authorable.TypeIDs())
	chain, err := BuildChain(authorable)
	require.NoError(t, err)
	require.Contains(t, chain.Children, "route")
	require.NotContains(t, chain.Children, "topic_match")
}

func TestParseCorpus_RejectsMissingOrUnknownRoles(t *testing.T) {
	tests := map[string]string{
		"missing": strings.Replace(catalogType("unsupported", "", true), `,"role":""`, "", 1),
		"unknown": catalogType("unsupported", "condition", true),
	}
	for name, typeJSON := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ParseCorpus(catalogArtifact(typeJSON))
			require.ErrorContains(t, err, "types[0].role")
		})
	}
}
