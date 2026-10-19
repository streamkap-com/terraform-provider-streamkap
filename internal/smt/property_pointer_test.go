package smt

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/require"
)

const escapedPropertyCatalog = `{
  "contract_version": 1,
  "producer": "test",
  "digest": "sha256:test",
  "types": [{
    "type_id": "escaped_properties",
    "schema_version": 1,
    "codec_version": 1,
    "publishable": false,
    "authorable": false,
    "role": "transform",
    "data_schema": {
      "type": "object",
      "properties": {
        "routes/v~1": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "key": {"type": "string"},
              "auth/data~": {
                "type": "object",
                "properties": {
                  "token/name~": {"type": "string", "writeOnly": true}
                },
                "additionalProperties": false
              },
              "endpoints/v~2": {
                "type": "array",
                "items": {
                  "type": "object",
                  "properties": {
                    "id": {"type": "string"},
                    "secret/value~": {"type": "string", "writeOnly": true}
                  },
                  "required": ["id"],
                  "additionalProperties": false
                }
              }
            },
            "required": ["key"],
            "additionalProperties": false
          }
        },
        "routes~1v": {
          "type": "object",
          "properties": {"label": {"type": "string"}},
          "additionalProperties": false
        }
      },
      "additionalProperties": false
    },
    "ui_schema": {"type": "Category", "elements": []},
    "consumer_requirements": {
      "version": 1,
      "schema_keywords": ["additionalProperties", "items", "properties", "required", "type", "writeOnly"],
      "schema_types": ["array", "object", "string"],
      "schema_formats": [],
      "ui_elements": ["Category"],
      "ui_controls": [],
      "secret_addressing": ["json-pointer-v1", "keyed-rows-v1"]
    },
    "secret_paths": [
      "/routes~1v~01/*/auth~1data~0/token~1name~0",
      "/routes~1v~01/*/endpoints~1v~02/*/secret~1value~0"
    ],
    "row_keys": {
      "/routes~1v~01": "key",
      "/routes~1v~01/*/endpoints~1v~02": "id"
    },
    "examples": {"read_projection": {}}
  }]
}`

func escapedPropertySpec(t *testing.T) *TypeSpec {
	t.Helper()
	var corpus Corpus
	require.NoError(t, json.Unmarshal([]byte(escapedPropertyCatalog), &corpus))
	require.Len(t, corpus.Types, 1)
	return &corpus.Types[0]
}

func TestParseCorpus_AcceptsEscapedPropertyPointers(t *testing.T) {
	corpus, err := ParseCorpus([]byte(escapedPropertyCatalog))
	require.NoError(t, err)
	require.Equal(t, []string{
		"/routes~1v~01/*/auth~1data~0/token~1name~0",
		"/routes~1v~01/*/endpoints~1v~02/*/secret~1value~0",
	}, corpus.Types[0].SecretPaths)
	require.Equal(t, map[string]string{
		"/routes~1v~01":                   "key",
		"/routes~1v~01/*/endpoints~1v~02": "id",
	}, corpus.Types[0].RowKeys)
}

func TestBuildConfigChild_EscapesPropertyPointerTokens(t *testing.T) {
	child, err := BuildConfigChild(escapedPropertySpec(t))
	require.NoError(t, err)
	require.Equal(t, []string{
		"/routes~1v~01/*/auth~1data~0/token~1name~0",
		"/routes~1v~01/*/endpoints~1v~02/*/secret~1value~0",
	}, child.SecretTemplates)

	routes := child.Attribute.Attributes["routes/v~1"].(schema.ListNestedAttribute)
	require.Contains(t, child.Attribute.Attributes, "routes~1v", "escaped aliases must remain distinct raw attributes")
	require.Contains(t, routes.NestedObject.Attributes, "auth/data~")
	require.Contains(t, routes.NestedObject.Attributes, "endpoints/v~2")
	auth := routes.NestedObject.Attributes["auth/data~"].(schema.SingleNestedAttribute)
	require.NotContains(t, auth.Attributes, "token/name~")
	endpoints := routes.NestedObject.Attributes["endpoints/v~2"].(schema.ListNestedAttribute)
	require.NotContains(t, endpoints.NestedObject.Attributes, "secret/value~")
}

func TestEscapedPropertyPointers_ResolveAgainstRawConfigFields(t *testing.T) {
	spec := escapedPropertySpec(t)
	config := map[string]any{
		"routes/v~1": []any{
			map[string]any{
				"key":        "row~west/1",
				"auth/data~": map[string]any{},
				"endpoints/v~2": []any{
					map[string]any{"id": "ep~2/1"},
				},
			},
		},
		"routes~1v": map[string]any{"label": "distinct sibling"},
	}
	specBefore, err := json.Marshal(spec)
	require.NoError(t, err)
	configBefore, err := json.Marshal(config)
	require.NoError(t, err)

	authPointer := "/routes~1v~01/row~0west~11/auth~1data~0/token~1name~0"
	auth, err := ResolvePointer(spec, authPointer)
	require.NoError(t, err)
	require.Equal(t, authPointer, auth.Pointer)
	require.Equal(t, "/routes~1v~01/*/auth~1data~0/token~1name~0", auth.Template)
	require.Equal(t, []string{"row~west/1"}, auth.RowKeys)
	require.True(t, RowExists(spec, auth, config))

	endpointPointer := "/routes~1v~01/row~0west~11/endpoints~1v~02/ep~02~11/secret~1value~0"
	endpoint, err := ResolvePointer(spec, endpointPointer)
	require.NoError(t, err)
	require.Equal(t, endpointPointer, endpoint.Pointer)
	require.Equal(t, []string{"row~west/1", "ep~2/1"}, endpoint.RowKeys)
	require.True(t, RowExists(spec, endpoint, config))

	for _, ambiguous := range []string{
		"/routes/v~01/row~0west~11/auth~1data~0/token~1name~0",
		"/routes~01v/row~0west~11/auth~1data~0/token~1name~0",
		"/routes~2v~01/row~0west~11/auth~1data~0/token~1name~0",
		"/routes~1v~01/row~west~11/auth~1data~0/token~1name~0",
		"/routes~1v~01/row~0west~11/endpoints~1v~02/ep~2~11/secret~1value~0",
	} {
		address, err := ResolvePointer(spec, ambiguous)
		require.Error(t, err, ambiguous)
		require.Nil(t, address, ambiguous)
	}
	specAfter, err := json.Marshal(spec)
	require.NoError(t, err)
	configAfter, err := json.Marshal(config)
	require.NoError(t, err)
	require.Equal(t, specBefore, specAfter, "pointer resolution must not mutate the type contract")
	require.Equal(t, configBefore, configAfter, "pointer resolution and row lookup must not mutate config")
}
