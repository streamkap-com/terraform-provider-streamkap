package smt

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/require"
)

func predicateSurfaceFixture(t *testing.T) *Surface {
	t.Helper()
	corpus, err := ParseCorpus([]byte(`{
  "contract_version": 1,
  "producer": "test",
  "digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  "types": [
    {
      "type_id": "route",
      "schema_version": 1,
      "codec_version": 1,
      "publishable": true,
    "authorable": true,
      "role": "transform",
	  "compatible_transform_types": [],
	  "supports_negate": false,
      "data_schema": {"type": "object", "additionalProperties": false, "properties": {"replacement": {"type": "string"}}, "required": ["replacement"]},
	  "consumer_requirements": {"version": 1, "schema_keywords": ["additionalProperties", "properties", "required", "type"], "schema_types": ["object", "string"], "schema_formats": [], "ui_elements": [], "ui_controls": [], "secret_addressing": []},
      "secret_paths": [],
      "row_keys": {}
    },
    {
      "type_id": "topic_match",
      "schema_version": 1,
      "codec_version": 1,
      "publishable": true,
    "authorable": true,
      "role": "predicate",
	  "compatible_transform_types": ["route"],
	  "supports_negate": true,
      "data_schema": {"type": "object", "additionalProperties": false, "properties": {"pattern": {"type": "string"}}, "required": ["pattern"]},
	  "consumer_requirements": {"version": 1, "schema_keywords": ["additionalProperties", "properties", "required", "type"], "schema_types": ["object", "string"], "schema_formats": [], "ui_elements": [], "ui_controls": [], "secret_addressing": []},
      "secret_paths": [],
      "row_keys": {}
    },
    {
      "type_id": "topic_match_secret",
      "schema_version": 1,
      "codec_version": 1,
      "publishable": true,
    "authorable": true,
      "role": "predicate",
	  "compatible_transform_types": ["route"],
	  "supports_negate": true,
      "data_schema": {"type": "object", "additionalProperties": false, "properties": {"pattern": {"type": "string"}, "token": {"type": "string", "writeOnly": true}}, "required": ["pattern", "token"]},
	  "consumer_requirements": {"version": 1, "schema_keywords": ["additionalProperties", "properties", "required", "type", "writeOnly"], "schema_types": ["object", "string"], "schema_formats": [], "ui_elements": [], "ui_controls": [], "secret_addressing": ["json-pointer-v1"]},
      "secret_paths": ["/token"],
      "row_keys": {}
    }
  ]
}`))
	require.NoError(t, err)
	chain, err := BuildChain(NewCatalog(corpus))
	require.NoError(t, err)
	return NewSurface(chain, nil)
}

func TestPredicateSurfaceHasSeparateAuthoringAndSecretNamespaces(t *testing.T) {
	s := predicateSurfaceFixture(t)
	attrs := map[string]schema.Attribute{}
	s.AddAttributes(attrs)

	require.Contains(t, attrs, "smt_predicates")
	require.Contains(t, attrs, "smt_predicate_secrets")
	require.Contains(t, attrs, AttrChain)
	require.Contains(t, attrs, AttrSecrets)

	chain := attrs[AttrChain].(schema.ListNestedAttribute)
	predicates := attrs["smt_predicates"].(schema.ListNestedAttribute)
	require.Contains(t, chain.NestedObject.Attributes, "route")
	require.NotContains(t, chain.NestedObject.Attributes, "topic_match", "predicate types are not transform choices")
	require.Contains(t, predicates.NestedObject.Attributes, "topic_match")
	require.NotContains(t, predicates.NestedObject.Attributes, "route", "transform types are not predicate choices")
}

func TestPredicateIssuePathWithoutPlannedAuthorityFallsBackSafely(t *testing.T) {
	s := predicateSurfaceFixture(t)
	w := &Write{}

	require.True(t,
		path.Root("smt_predicates").Equal(s.issuePath("/predicates/0/config/pattern", w)),
	)
	require.True(t,
		path.Root("smt_predicates").Equal(s.issuePath("/predicates/0/secret_operations/0/pointer", w)),
	)
}

func TestPredicateIssuePathUsesPlannedPredicateAuthority(t *testing.T) {
	s := predicateSurfaceFixture(t)
	w := &Write{predicateCorrelated: []Correlated{{Key: "p", Type: "topic_match"}}}
	require.True(t, path.Root(AttrPredicates).AtListIndex(0).AtName("topic_match").AtName("pattern").Equal(s.issuePath("/predicates/0/config/pattern", w)))
	require.True(t, path.Root(AttrPredicateSecrets).Equal(s.issuePath("/predicates/0/secret_operations/0/pointer", w)))
}
