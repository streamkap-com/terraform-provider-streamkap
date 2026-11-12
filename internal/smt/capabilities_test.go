package smt

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

const catalogWithConsumerRequirements = `{
  "contract_version": 1,
  "producer": "test",
  "digest": "sha256:test",
  "types": [{
    "type_id": "configured",
    "schema_version": 1,
    "codec_version": 1,
    "publishable": true,
    "authorable": true,
    "role": "transform",
    "data_schema": {
      "type": "object",
      "properties": {
        "config": {
          "type": "object",
          "properties": {"futureKeyword": {"type": "string", "format": "email"}},
          "required": ["futureKeyword"],
          "additionalProperties": false,
          "default": {"futureKeyword": "preserved"}
        }
      },
      "required": ["config"],
      "additionalProperties": false
    },
    "ui_schema": {
      "type": "Category",
      "elements": [{
        "type": "Group",
        "elements": [{
          "type": "Control",
          "scope": "#/properties/config/properties/futureKeyword",
          "options": {"control": "string"},
          "rule": {
            "effect": "SHOW",
            "condition": {"scope": "#/properties/config", "schema": {"type": "string", "format": "uri"}}
          }
        }]
      }]
    },
    "consumer_requirements": {
      "version": 1,
      "schema_keywords": ["additionalProperties", "default", "format", "properties", "required", "type"],
      "schema_types": ["object", "string"],
      "schema_formats": ["email", "uri"],
      "ui_elements": ["Category", "Control", "Group"],
      "ui_controls": ["string"],
      "secret_addressing": []
    },
    "secret_paths": [],
    "row_keys": {},
    "examples": {"read_projection": {"config": {"futureKeyword": "preserved"}}}
  }]
}`

const secretCatalogWithConsumerRequirements = `{
  "contract_version": 1,
  "producer": "test",
  "digest": "sha256:test",
  "types": [{
    "type_id": "secret_rows",
    "schema_version": 1,
    "codec_version": 1,
    "publishable": true,
    "authorable": true,
    "role": "transform",
    "data_schema": {
      "type": "object",
      "properties": {
        "rows": {
          "type": "array",
          "items": {
            "type": "object",
            "properties": {
              "id": {"type": "string"},
              "token": {"type": "string", "writeOnly": true}
            },
            "required": ["id", "token"],
            "additionalProperties": false
          }
        }
      },
      "required": ["rows"],
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
    "secret_paths": ["/rows/*/token"],
    "row_keys": {"/rows": "id"},
    "examples": {"read_projection": {"rows": [{"id": "one"}]}}
  }]
}`

func TestParseCorpus_ValidatesConsumerRequirementsAgainstStructuralPositions(t *testing.T) {
	corpus, err := ParseCorpus([]byte(catalogWithConsumerRequirements))
	require.NoError(t, err)
	require.Len(t, corpus.Types, 1)
}

func TestParseCorpus_RejectsInvalidConsumerRequirements(t *testing.T) {
	tests := map[string]struct {
		mutate func(map[string]any)
		want   []string
	}{
		"missing": {
			mutate: func(spec map[string]any) { delete(spec, "consumer_requirements") },
			want:   []string{"configured", "consumer_requirements"},
		},
		"unsupported version": {
			mutate: func(spec map[string]any) { requirements(spec)["version"] = 2 },
			want:   []string{"configured", "consumer_requirements.version", "2"},
		},
		"duplicate capability": {
			mutate: func(spec map[string]any) { requirements(spec)["schema_types"] = []any{"object", "string", "string"} },
			want:   []string{"configured", "schema_types", "duplicate", "string"},
		},
		"unsorted capability": {
			mutate: func(spec map[string]any) { requirements(spec)["ui_elements"] = []any{"Group", "Category", "Control"} },
			want:   []string{"configured", "ui_elements", "sorted"},
		},
		"derived inventory mismatch": {
			mutate: func(spec map[string]any) { requirements(spec)["schema_formats"] = []any{"email"} },
			want:   []string{"configured", "schema_formats", "uri"},
		},
		"unknown requirement member": {
			mutate: func(spec map[string]any) { requirements(spec)["provider_validates"] = true },
			want:   []string{"configured", "consumer_requirements", "provider_validates"},
		},
		"unknown schema keyword": {
			mutate: func(spec map[string]any) {
				config := spec["data_schema"].(map[string]any)["properties"].(map[string]any)["config"].(map[string]any)
				config["properties"].(map[string]any)["futureKeyword"].(map[string]any)["unsupportedKeyword"] = true
			},
			want: []string{"configured", "unsupportedKeyword", "/config/futureKeyword"},
		},
		"open object": {
			mutate: func(spec map[string]any) {
				config := spec["data_schema"].(map[string]any)["properties"].(map[string]any)["config"].(map[string]any)
				config["additionalProperties"] = true
			},
			want: []string{"configured", "additionalProperties", "/config", "false"},
		},
		"root object without closed declaration": {
			mutate: func(spec map[string]any) {
				delete(spec["data_schema"].(map[string]any), "additionalProperties")
			},
			want: []string{"configured", "additionalProperties", "/", "false"},
		},
		"nested object without closed declaration": {
			mutate: func(spec map[string]any) {
				config := spec["data_schema"].(map[string]any)["properties"].(map[string]any)["config"].(map[string]any)
				delete(config, "additionalProperties")
			},
			want: []string{"configured", "additionalProperties", "/config", "false"},
		},
		"read only property": {
			mutate: func(spec map[string]any) {
				config := spec["data_schema"].(map[string]any)["properties"].(map[string]any)["config"].(map[string]any)
				config["properties"].(map[string]any)["futureKeyword"].(map[string]any)["readOnly"] = true
				requirements(spec)["schema_keywords"] = []any{
					"additionalProperties", "default", "format", "properties", "readOnly", "required", "type",
				}
			},
			want: []string{"configured", "readOnly", "/config/futureKeyword"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			catalog := mutateCatalog(t, catalogWithConsumerRequirements, tc.mutate)
			_, err := ParseCorpus(catalog)
			require.Error(t, err)
			for _, want := range tc.want {
				require.ErrorContains(t, err, want)
			}
		})
	}
}

func TestParseCorpus_AllowsPartialSchemaInUIRuleCondition(t *testing.T) {
	catalog := mutateCatalog(t, catalogWithConsumerRequirements, func(spec map[string]any) {
		ui := spec["ui_schema"].(map[string]any)
		group := ui["elements"].([]any)[0].(map[string]any)
		control := group["elements"].([]any)[0].(map[string]any)
		control["rule"].(map[string]any)["condition"].(map[string]any)["schema"] = map[string]any{"const": "enabled"}
		requirements(spec)["schema_keywords"] = []any{
			"additionalProperties", "const", "default", "format", "properties", "required", "type",
		}
		requirements(spec)["schema_formats"] = []any{"email"}
	})

	parsed, err := ParseCorpus(catalog)
	require.NoError(t, err)
	require.Len(t, parsed.Types, 1)
}

func TestParseCorpus_DerivesSecretAddressingRequirements(t *testing.T) {
	_, err := ParseCorpus([]byte(secretCatalogWithConsumerRequirements))
	require.NoError(t, err)

	catalog := mutateCatalog(t, secretCatalogWithConsumerRequirements, func(spec map[string]any) {
		requirements(spec)["secret_addressing"] = []any{"json-pointer-v1"}
	})
	_, err = ParseCorpus(catalog)
	require.ErrorContains(t, err, "secret_rows")
	require.ErrorContains(t, err, "secret_addressing")
	require.ErrorContains(t, err, "keyed-rows-v1")
}

func mutateCatalog(t *testing.T, raw string, mutate func(map[string]any)) []byte {
	t.Helper()
	var catalog map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &catalog))
	spec := catalog["types"].([]any)[0].(map[string]any)
	mutate(spec)
	encoded, err := json.Marshal(catalog)
	require.NoError(t, err)
	return encoded
}

func requirements(spec map[string]any) map[string]any {
	return spec["consumer_requirements"].(map[string]any)
}
