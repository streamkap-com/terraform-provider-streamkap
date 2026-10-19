package smt

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

const declarationOrderedSecretCatalog = `{
  "contract_version": 1,
  "producer": "test",
  "digest": "sha256:test",
  "types": [{
    "type_id": "declaration_order",
    "schema_version": 1,
    "codec_version": 1,
    "publishable": false,
    "authorable": false,
    "role": "transform",
    "data_schema": {
      "type": "object",
      "properties": {
        "z_secret": {"type": "string", "writeOnly": true},
        "a_secret": {"type": "string", "writeOnly": true}
      },
      "additionalProperties": false
    },
    "ui_schema": {"type": "Category", "elements": []},
    "consumer_requirements": {
      "version": 1,
      "schema_keywords": ["additionalProperties", "properties", "type", "writeOnly"],
      "schema_types": ["object", "string"],
      "schema_formats": [],
      "ui_elements": ["Category"],
      "ui_controls": [],
      "secret_addressing": ["json-pointer-v1"]
    },
    "secret_paths": ["/z_secret", "/a_secret"],
    "row_keys": {},
    "examples": {"read_projection": {}}
  }]
}`

func TestParseCorpus_PreservesProducerSecretPathOrder(t *testing.T) {
	corpus, err := ParseCorpus([]byte(declarationOrderedSecretCatalog))
	require.NoError(t, err)
	require.Equal(t, []string{"/z_secret", "/a_secret"}, corpus.Types[0].SecretPaths)
}

func TestParseCorpus_ValidatesNestedSecretMetadata(t *testing.T) {
	raw, err := os.ReadFile(corpusPath)
	require.NoError(t, err)

	corpus, err := ParseCorpus(raw)
	require.NoError(t, err)
	deep := currentSpec(t, NewCatalog(corpus), "transform", "contract_fixture_deep")
	require.Equal(t, []string{"/services/*/auth/token", "/services/*/endpoints/*/secret"}, deep.SecretPaths)
	require.Equal(t, map[string]string{"/services": "key", "/services/*/endpoints": "id"}, deep.RowKeys)
}

func TestParseCorpus_RejectsInvalidSecretMetadata(t *testing.T) {
	tests := map[string]struct {
		mutate func(map[string]any)
		want   []string
	}{
		"missing secret path": {
			mutate: func(spec map[string]any) {
				spec["secret_paths"] = []any{"/services/*/auth/token"}
			},
			want: []string{"contract_fixture_deep", "secret_paths", "/services/*/endpoints/*/secret"},
		},
		"unknown secret path": {
			mutate: func(spec map[string]any) {
				spec["secret_paths"] = append(spec["secret_paths"].([]any), "/services/*/missing")
			},
			want: []string{"contract_fixture_deep", "secret_paths", "/services/*/missing"},
		},
		"null secret paths": {
			mutate: func(spec map[string]any) {
				spec["secret_paths"] = nil
				spec["consumer_requirements"].(map[string]any)["secret_addressing"] = []any{"keyed-rows-v1"}
			},
			want: []string{"contract_fixture_deep", "secret_paths", "array"},
		},
		"duplicate secret path": {
			mutate: func(spec map[string]any) {
				paths := spec["secret_paths"].([]any)
				spec["secret_paths"] = append(paths, paths[0])
			},
			want: []string{"contract_fixture_deep", "secret_paths", "duplicate", "/services/*/auth/token"},
		},
		"null row keys": {
			mutate: func(spec map[string]any) {
				spec["row_keys"] = nil
				spec["consumer_requirements"].(map[string]any)["secret_addressing"] = []any{"json-pointer-v1"}
			},
			want: []string{"contract_fixture_deep", "row_keys", "object"},
		},
		"missing nested row key": {
			mutate: func(spec map[string]any) {
				delete(spec["row_keys"].(map[string]any), "/services/*/endpoints")
			},
			want: []string{"contract_fixture_deep", "row_keys", "/services/*/endpoints", "id"},
		},
		"unknown row key path": {
			mutate: func(spec map[string]any) {
				spec["row_keys"].(map[string]any)["/services/*/missing"] = "id"
			},
			want: []string{"contract_fixture_deep", "row_keys", "/services/*/missing"},
		},
		"row key names an unknown property": {
			mutate: func(spec map[string]any) {
				spec["row_keys"].(map[string]any)["/services"] = "missing"
			},
			want: []string{"contract_fixture_deep", "row_keys", "/services", "missing"},
		},
		"row key is optional": {
			mutate: func(spec map[string]any) {
				services := objectListItems(spec, "services")
				services["required"] = []any{}
			},
			want: []string{"contract_fixture_deep", "row_keys", "/services", "key", "required"},
		},
		"row key is not a string": {
			mutate: func(spec map[string]any) {
				services := objectListItems(spec, "services")
				key := services["properties"].(map[string]any)["key"].(map[string]any)
				key["type"] = "integer"
				key["minimum"] = minSafeInteger
				key["maximum"] = maxSafeInteger
				requirements := spec["consumer_requirements"].(map[string]any)
				keywords := append(requirements["schema_keywords"].([]any), "maximum", "minimum")
				sort.Slice(keywords, func(i, j int) bool { return keywords[i].(string) < keywords[j].(string) })
				requirements["schema_keywords"] = keywords
				types := append(requirements["schema_types"].([]any), "integer")
				sort.Slice(types, func(i, j int) bool { return types[i].(string) < types[j].(string) })
				requirements["schema_types"] = types
			},
			want: []string{"contract_fixture_deep", "row_keys", "/services", "key", "string"},
		},
		"row key is secret": {
			mutate: func(spec map[string]any) {
				services := objectListItems(spec, "services")
				services["properties"].(map[string]any)["key"].(map[string]any)["writeOnly"] = true
				spec["secret_paths"] = []any{
					"/services/*/auth/token",
					"/services/*/endpoints/*/secret",
					"/services/*/key",
				}
			},
			want: []string{"contract_fixture_deep", "row_keys", "/services", "key", "writeOnly"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			raw := mutateCorpusType(t, "contract_fixture_deep", tc.mutate)
			_, err := ParseCorpus(raw)
			require.Error(t, err)
			for _, want := range tc.want {
				require.ErrorContains(t, err, want)
			}
		})
	}
}

func mutateCorpusType(t *testing.T, typeID string, mutate func(map[string]any)) []byte {
	t.Helper()
	raw, err := os.ReadFile(corpusPath)
	require.NoError(t, err)
	var corpus map[string]any
	require.NoError(t, json.Unmarshal(raw, &corpus))
	for _, rawSpec := range corpus["types"].([]any) {
		spec := rawSpec.(map[string]any)
		if spec["type_id"] == typeID {
			mutate(spec)
			encoded, err := json.Marshal(corpus)
			require.NoError(t, err)
			return encoded
		}
	}
	t.Fatalf("missing corpus type %q", typeID)
	return nil
}

func objectListItems(spec map[string]any, property string) map[string]any {
	dataSchema := spec["data_schema"].(map[string]any)
	list := dataSchema["properties"].(map[string]any)[property].(map[string]any)
	return list["items"].(map[string]any)
}
