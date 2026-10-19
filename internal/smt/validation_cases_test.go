package smt

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/stretchr/testify/require"
)

type corpusValidationCase struct {
	Name   string         `json:"name"`
	Config map[string]any `json:"config"`
	Valid  bool           `json:"valid"`
}

func TestValidationCasesPreserveProviderTypesWithoutClaimingServerValidation(t *testing.T) {
	raw, err := os.ReadFile(corpusPath)
	require.NoError(t, err)
	var artifact struct {
		Types []struct {
			TypeID   string `json:"type_id"`
			Examples struct {
				ValidationCases []corpusValidationCase `json:"validation_cases"`
			} `json:"examples"`
		} `json:"types"`
	}
	require.NoError(t, json.Unmarshal(raw, &artifact))

	var cases []corpusValidationCase
	for _, spec := range artifact.Types {
		if spec.TypeID == "contract_fixture_nested" {
			cases = spec.Examples.ValidationCases
			break
		}
	}
	require.NotEmpty(t, cases, "contract_fixture_nested must publish validation cases")

	corpus, err := ParseCorpus(raw)
	require.NoError(t, err)
	chain, err := BuildChain(NewCatalog(corpus))
	require.NoError(t, err)
	childType := chain.EntryType.AttributeTypes()["contract_fixture_nested"]
	seenNames := map[string]bool{}
	sawValid, sawSemanticInvalid, sawTypeInvalid := false, false, false

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			require.NotEmpty(t, tc.Name)
			require.False(t, seenNames[tc.Name], "validation case names must be unique")
			seenNames[tc.Name] = true

			representable := fixtureConfigIsStructurallyRepresentable(tc.Config)
			value, diags := FromWire(context.Background(), childType, tc.Config, path.Root("contract_fixture_nested"))
			if !representable {
				require.False(t, tc.Valid, "a declared-valid case must use the published provider types")
				require.True(t, diags.HasError(), "mismatched primitives must not be coerced")
				sawTypeInvalid = true
				return
			}

			require.False(t, diags.HasError(), "%v", diags)
			encoded, err := ToWire(value, path.Root("contract_fixture_nested"))
			require.NoError(t, err)
			require.Equal(t, apiRoundTrip(t, tc.Config), apiRoundTrip(t, submittedShape(encoded, tc.Config)))
			if tc.Valid {
				sawValid = true
			} else {
				sawSemanticInvalid = true
			}
		})
	}

	require.True(t, sawValid, "the producer must publish valid normalization cases")
	require.True(t, sawSemanticInvalid, "backend-only bounds and pattern cases must remain representable")
	require.True(t, sawTypeInvalid, "primitive type mismatches must be covered")
}

func fixtureConfigIsStructurallyRepresentable(config map[string]any) bool {
	for name, value := range config {
		switch name {
		case "enabled":
			if _, ok := value.(bool); !ok {
				return false
			}
		case "retries", "max_records":
			if number, ok := value.(float64); !ok || math.Trunc(number) != number || math.Abs(number) > float64(maxJSONSafeInteger) {
				return false
			}
		case "finite_ratio":
			if number, ok := value.(float64); !ok || math.IsNaN(number) || math.IsInf(number, 0) {
				return false
			}
		case "lossless_integer":
			if _, ok := value.(string); !ok {
				return false
			}
		case "fallback":
			if value != nil {
				if _, ok := value.(string); !ok {
					return false
				}
			}
		case "labels":
			items, ok := value.([]any)
			if !ok || !allStrings(items) {
				return false
			}
		case "output":
			object, ok := value.(map[string]any)
			if !ok || !fixtureOutputIsStructurallyRepresentable(object) {
				return false
			}
		case "routes":
			rows, ok := value.([]any)
			if !ok || !fixtureRowsAreStructurallyRepresentable(rows) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func fixtureOutputIsStructurallyRepresentable(output map[string]any) bool {
	for name, value := range output {
		switch name {
		case "prefix", "label":
			if _, ok := value.(string); !ok {
				return false
			}
		case "retry_limit":
			if number, ok := value.(float64); !ok || math.Trunc(number) != number || math.Abs(number) > float64(maxJSONSafeInteger) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func fixtureRowsAreStructurallyRepresentable(rows []any) bool {
	for _, value := range rows {
		row, ok := value.(map[string]any)
		if !ok {
			return false
		}
		for name, field := range row {
			if name != "key" && name != "label" && name != "endpoint" {
				return false
			}
			if _, ok := field.(string); !ok {
				return false
			}
		}
	}
	return true
}

func allStrings(values []any) bool {
	for _, value := range values {
		if _, ok := value.(string); !ok {
			return false
		}
	}
	return true
}

func submittedShape(encoded, submitted any) any {
	switch source := submitted.(type) {
	case map[string]any:
		object, ok := encoded.(map[string]any)
		if !ok {
			return encoded
		}
		out := make(map[string]any, len(source))
		for key, value := range source {
			out[key] = submittedShape(object[key], value)
		}
		return out
	case []any:
		items, ok := encoded.([]any)
		if !ok || len(items) != len(source) {
			return encoded
		}
		out := make([]any, len(source))
		for index, value := range source {
			out[index] = submittedShape(items[index], value)
		}
		return out
	default:
		return encoded
	}
}
