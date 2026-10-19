package smt_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/smt"
)

const hostCatalog = `{
  "contract_version": 1,
  "producer": "test",
  "digest": "sha256:test",
  "types": [
    {
      "type_id": "filter",
      "schema_version": 1,
      "codec_version": 1,
      "publishable": true,
    "authorable": true,
      "role": "transform",
      "hosts": ["sources"],
      "data_schema": {"type": "object", "properties": {}, "additionalProperties": false},
	  "consumer_requirements": {"version": 1, "schema_keywords": ["additionalProperties", "properties", "type"], "schema_types": ["object"], "schema_formats": [], "ui_elements": [], "ui_controls": [], "secret_addressing": []},
      "secret_paths": [],
      "row_keys": {},
      "examples": {"read_projection": {}}
    },
    {
      "type_id": "regex_router",
      "schema_version": 1,
      "codec_version": 1,
      "publishable": true,
    "authorable": true,
      "role": "transform",
      "hosts": ["destinations"],
      "data_schema": {"type": "object", "properties": {}, "additionalProperties": false},
	  "consumer_requirements": {"version": 1, "schema_keywords": ["additionalProperties", "properties", "type"], "schema_types": ["object"], "schema_formats": [], "ui_elements": [], "ui_controls": [], "secret_addressing": []},
      "secret_paths": [],
      "row_keys": {},
      "examples": {"read_projection": {}}
    },
    {
      "type_id": "topic_name_matches",
      "schema_version": 1,
      "codec_version": 1,
      "publishable": true,
    "authorable": true,
      "role": "predicate",
      "hosts": ["destinations", "sources"],
      "data_schema": {"type": "object", "properties": {}, "additionalProperties": false},
	  "consumer_requirements": {"version": 1, "schema_keywords": ["additionalProperties", "properties", "type"], "schema_types": ["object"], "schema_formats": [], "ui_elements": [], "ui_controls": [], "secret_addressing": []},
      "secret_paths": [],
      "row_keys": {},
      "examples": {"read_projection": {}}
    },
    {
      "type_id": "unscoped",
      "schema_version": 1,
      "codec_version": 1,
      "publishable": true,
    "authorable": true,
      "role": "transform",
      "data_schema": {"type": "object", "properties": {}, "additionalProperties": false},
	  "consumer_requirements": {"version": 1, "schema_keywords": ["additionalProperties", "properties", "type"], "schema_types": ["object"], "schema_formats": [], "ui_elements": [], "ui_controls": [], "secret_addressing": []},
      "secret_paths": [],
      "row_keys": {},
      "examples": {"read_projection": {}}
    }
  ]
}`

func TestCatalogForHostFiltersTransformsAndPredicatesWithoutFallback(t *testing.T) {
	corpus, err := smt.ParseCorpus([]byte(hostCatalog))
	require.NoError(t, err)
	catalog := smt.NewCatalog(corpus)

	destination := catalog.ForHost("destinations")
	require.Equal(t, []string{"regex_router"}, destination.Publishable().TypeIDs())
	require.Equal(t, []string{"regex_router", "topic_name_matches"}, destination.Authorable().TypeIDs())

	source := catalog.ForHost("sources")
	require.Equal(t, []string{"filter"}, source.Publishable().TypeIDs())
	require.Equal(t, []string{"filter", "topic_name_matches"}, source.Authorable().TypeIDs())

	require.Zero(t, catalog.ForHost("unsupported").Len())
}

// Each host's runtime chain is built from its own catalog types: Filter only
// on sources, the destination transforms only on destinations, and the shared
// predicate on both.
func TestGeneratedRuntimeChainsFollowTheirHost(t *testing.T) {
	corpus, err := smt.LoadCorpus(filepath.Join("..", "generated", "smt_catalog.json"))
	require.NoError(t, err)
	require.Equal(t, []string{"sources"}, corpusType(t, corpus, "filter").Hosts)
	require.ElementsMatch(t, []string{"destinations", "sources"}, corpusType(t, corpus, "topic_name_matches").Hosts)

	destination := generated.SMTChain("destinations")
	require.NotContains(t, destination.Catalog.TypeIDs(), "filter")
	require.Contains(t, destination.Catalog.TypeIDs(), "regex_router")
	require.Contains(t, destination.PredicateCatalog.TypeIDs(), "topic_name_matches")

	source := generated.SMTChain("sources")
	require.Equal(t, []string{"filter"}, source.Catalog.TypeIDs())
	require.Equal(t, []string{"topic_name_matches"}, source.PredicateCatalog.TypeIDs())
	require.Empty(t, source.Children["filter"].Attribute.Attributes, "Filter has no settings; its predicate decides")

	require.Panics(t, func() { generated.SMTChain("transforms") }, "only a generated model names a host, so an unknown one is a broken build")
}

func corpusType(t *testing.T, corpus *smt.Corpus, typeID string) *smt.TypeSpec {
	t.Helper()
	for i := range corpus.Types {
		if corpus.Types[i].TypeID == typeID {
			return &corpus.Types[i]
		}
	}
	t.Fatalf("catalog type %q is missing", typeID)
	return nil
}
