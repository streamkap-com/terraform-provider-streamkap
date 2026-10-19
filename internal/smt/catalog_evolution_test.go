package smt

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

func evolutionType(typeID, role string, schemaVersion, codecVersion int, publishable bool, hosts, propertyName, propertyType string) string {
	propertySchema := fmt.Sprintf(`{"type": %q}`, propertyType)
	schemaKeywords := `"additionalProperties", "properties", "required", "type"`
	schemaTypes := fmt.Sprintf(`"object", %q`, propertyType)
	if propertyType == "integer" {
		propertySchema = `{"type": "integer", "minimum": -9007199254740991, "maximum": 9007199254740991}`
		schemaKeywords = `"additionalProperties", "maximum", "minimum", "properties", "required", "type"`
		schemaTypes = `"integer", "object"`
	}
	return fmt.Sprintf(`{
  "type_id": %q,
  "schema_version": %d,
  "codec_version": %d,
  "publishable": %t,
  "authorable": %t,
  "role": %q,
  "hosts": [%q],
  "data_schema": {
    "type": "object",
    "additionalProperties": false,
    "properties": {%q: %s},
    "required": [%q]
  },
  "consumer_requirements": {
    "version": 1,
    "schema_keywords": [%s],
    "schema_types": [%s],
    "schema_formats": [],
    "ui_elements": [],
    "ui_controls": [],
    "secret_addressing": []
  },
  "secret_paths": [],
  "row_keys": {},
  "examples": {"read_projection": {%q: %s}}
}`,
		typeID, schemaVersion, codecVersion, publishable, publishable, role, hosts,
		propertyName, propertySchema, propertyName, schemaKeywords, schemaTypes, propertyName, evolutionExample(propertyType),
	)
}

func evolutionExample(propertyType string) string {
	if propertyType == "integer" {
		return "1"
	}
	return `"value"`
}

func evolutionCorpus(t *testing.T, declarations ...string) *Corpus {
	t.Helper()
	raw := catalogArtifact(joinDeclarations(declarations))
	corpus, err := ParseCorpus(raw)
	require.NoError(t, err)
	return corpus
}

func joinDeclarations(declarations []string) string {
	joined := ""
	for i, declaration := range declarations {
		if i > 0 {
			joined += ","
		}
		joined += declaration
	}
	return joined
}

func TestCatalogEvolutionSelectsCurrentIndependentOfDeclarationOrder(t *testing.T) {
	v2 := evolutionType("route", "transform", 2, 1, true, "destinations", "count", "integer")
	v1 := evolutionType("route", "transform", 1, 1, true, "destinations", "value", "string")

	for name, declarations := range map[string][]string{
		"newest first": {v2, v1},
		"oldest first": {v1, v2},
	} {
		t.Run(name, func(t *testing.T) {
			chain, err := BuildChain(NewCatalog(evolutionCorpus(t, declarations...)).ForHost("destinations").Authorable())
			require.NoError(t, err)
			child, ok := chain.Children["route"]
			require.True(t, ok)
			if ok {
				require.Contains(t, child.Attribute.Attributes, "count")
				require.NotContains(t, child.Attribute.Attributes, "value")
			}
		})
	}
}

func TestCatalogEvolutionDoesNotFallBackAfterCurrentHostFiltering(t *testing.T) {
	currentSource := evolutionType("route", "transform", 2, 1, true, "sources", "count", "integer")
	historicalDestination := evolutionType("route", "transform", 1, 1, true, "destinations", "value", "string")

	chain, err := BuildChain(NewCatalog(evolutionCorpus(t, currentSource, historicalDestination)).ForHost("destinations").Authorable())
	require.NoError(t, err)
	require.NotContains(t, chain.Children, "route", "an inapplicable current release must not make an older release authorable")
}

func TestCatalogEvolutionIgnoresNewerUnpublishableReleaseForAuthoring(t *testing.T) {
	published := evolutionType("route", "transform", 1, 1, true, "destinations", "value", "string")
	unpublishable := evolutionType("route", "transform", 2, 1, false, "destinations", "count", "integer")

	chain, err := BuildChain(NewCatalog(evolutionCorpus(t, published, unpublishable)).ForHost("destinations").Authorable())
	require.NoError(t, err)
	child, ok := chain.Children["route"]
	require.True(t, ok)
	if ok {
		require.Contains(t, child.Attribute.Attributes, "value")
		require.NotContains(t, child.Attribute.Attributes, "count")
	}
}

func TestCatalogEvolutionKeepsRoleNamespacesDistinct(t *testing.T) {
	predicate := evolutionType("shared", "predicate", 1, 1, true, "destinations", "pattern", "string")
	transform := evolutionType("shared", "transform", 1, 1, true, "destinations", "value", "string")

	chain, err := BuildChain(NewCatalog(evolutionCorpus(t, predicate, transform)).ForHost("destinations").Authorable())
	require.NoError(t, err)
	require.Contains(t, chain.Children, "shared")
	require.Contains(t, chain.PredicateChildren, "shared")
}

func TestCatalogEvolutionRejectsDuplicatePins(t *testing.T) {
	duplicate := evolutionType("route", "transform", 1, 1, true, "destinations", "value", "string")

	_, err := ParseCorpus(catalogArtifact(joinDeclarations([]string{duplicate, duplicate})))
	require.ErrorContains(t, err, "duplicate catalog pin transform/route schema 1 codec 1")
}

func TestCatalogEvolutionRetainedReadRequiresExactPin(t *testing.T) {
	current := evolutionType("route", "transform", 2, 2, true, "destinations", "value", "string")
	chain, err := BuildChain(NewCatalog(evolutionCorpus(t, current)).ForHost("destinations").Authorable())
	require.NoError(t, err)
	surface := NewSurface(chain, nil)
	read := &api.SMTChainRead{Desired: api.SMTChainSnapshot{
		ChainRevision: "r1",
		Transforms: []api.SMTInstanceRead{{
			Kind: "typed", ID: "instance-1", Alias: "Route1", Type: "route",
			SchemaVersion: 1, CodecVersion: 1, Name: "old", Enabled: true,
			Config: map[string]any{"value": "retained"},
		}},
	}}

	state, _, diags := surface.stateFrom(context.Background(), read, []string{"old"}, nil)
	require.True(t, diags.HasError())
	require.True(t, state.IsNull(), "an unresolved pin must not produce partially reinterpreted state")
	require.Equal(t, "Pinned transform artifact cannot be represented", diags.Errors()[0].Summary())
}

func TestCatalogEvolutionCompatibleHistoricalReadRetainsSchemaAndCodecPin(t *testing.T) {
	historical := evolutionType("route", "transform", 1, 3, true, "destinations", "value", "string")
	current := evolutionType("route", "transform", 2, 4, true, "destinations", "value", "string")
	chain, err := BuildChain(NewCatalog(evolutionCorpus(t, historical, current)).ForHost("destinations").Authorable())
	require.NoError(t, err)
	surface := NewSurface(chain, nil)
	read := &api.SMTChainRead{Desired: api.SMTChainSnapshot{
		ChainRevision: "r1",
		Transforms: []api.SMTInstanceRead{{
			Kind: "typed", ID: "instance-1", Alias: "Route1", Type: "route",
			SchemaVersion: 1, CodecVersion: 3, Name: "old", Enabled: true,
			Config: map[string]any{"value": "retained"},
		}},
	}}

	state, _, diags := surface.stateFrom(context.Background(), read, []string{"old"}, nil)
	require.False(t, diags.HasError(), "%v", diags)
	attrs := state.Elements()[0].(types.Object).Attributes()
	require.Equal(t, int64(1), attrs[attrSchemaVersion].(types.Int64).ValueInt64())
	codec, ok := attrs["codec_version"].(types.Int64)
	require.True(t, ok, "retained identity must include codec_version in provider state")
	require.Equal(t, int64(3), codec.ValueInt64())
}

func TestCatalogEvolutionRefusesHistoricalPinWithDifferentTerraformShape(t *testing.T) {
	historical := evolutionType("route", "transform", 1, 1, true, "destinations", "value", "string")
	current := evolutionType("route", "transform", 2, 1, true, "destinations", "count", "integer")
	chain, err := BuildChain(NewCatalog(evolutionCorpus(t, historical, current)).ForHost("destinations").Authorable())
	require.NoError(t, err)
	surface := NewSurface(chain, nil)
	read := &api.SMTChainRead{Desired: api.SMTChainSnapshot{Transforms: []api.SMTInstanceRead{{
		Kind: "typed", ID: "instance-1", Alias: "Route1", Type: "route",
		SchemaVersion: 1, CodecVersion: 1, Name: "old", Enabled: true,
		Config: map[string]any{"value": "retained"},
	}}}}

	state, _, diags := surface.stateFrom(context.Background(), read, []string{"old"}, nil)
	require.True(t, diags.HasError())
	require.True(t, state.IsNull())
	require.Equal(t, "Pinned transform artifact cannot be represented", diags.Errors()[0].Summary())
}

func evolutionSemanticType(schemaVersion int, propertySchema, required, schemaKeywords, secretPaths, secretAddressing string) string {
	return fmt.Sprintf(`{
  "type_id": "route",
  "schema_version": %d,
  "codec_version": 1,
  "publishable": true,
  "authorable": true,
  "role": "transform",
  "hosts": ["destinations"],
  "data_schema": {
    "type": "object",
    "additionalProperties": false,
    "properties": %s,
    "required": %s
  },
  "consumer_requirements": {
    "version": 1,
    "schema_keywords": %s,
    "schema_types": ["object", "string"],
    "schema_formats": [],
    "ui_elements": [],
    "ui_controls": [],
    "secret_addressing": %s
  },
  "secret_paths": %s,
  "row_keys": {},
  "examples": {"read_projection": {"value": "retained"}}
}`,
		schemaVersion, propertySchema, required, schemaKeywords, secretAddressing, secretPaths,
	)
}

func TestCatalogEvolutionHistoricalPinRequiresEquivalentWriteSemantics(t *testing.T) {
	const baseKeywords = `["additionalProperties", "properties", "required", "type"]`
	for name, tc := range map[string]struct {
		historicalProperties string
		currentProperties    string
		historicalRequired   string
		currentRequired      string
		keywords             string
		secretPaths          string
		secretAddressing     string
		wantError            bool
	}{
		"requiredness changed": {
			historicalProperties: `{"value":{"type":"string"}}`, currentProperties: `{"value":{"type":"string"}}`,
			historicalRequired: `["value"]`, currentRequired: `[]`, keywords: baseKeywords, secretPaths: `[]`, secretAddressing: `[]`, wantError: true,
		},
		"default changed": {
			historicalProperties: `{"value":{"type":"string","default":"old"}}`, currentProperties: `{"value":{"type":"string","default":"new"}}`,
			historicalRequired: `["value"]`, currentRequired: `["value"]`, keywords: `["additionalProperties", "default", "properties", "required", "type"]`, secretPaths: `[]`, secretAddressing: `[]`, wantError: true,
		},
		"enum changed": {
			historicalProperties: `{"value":{"type":"string","enum":["old"]}}`, currentProperties: `{"value":{"type":"string","enum":["old","new"]}}`,
			historicalRequired: `["value"]`, currentRequired: `["value"]`, keywords: `["additionalProperties", "enum", "properties", "required", "type"]`, secretPaths: `[]`, secretAddressing: `[]`, wantError: true,
		},
		"required secret changed": {
			historicalProperties: `{"value":{"type":"string"},"token":{"type":"string","writeOnly":true}}`, currentProperties: `{"value":{"type":"string"},"token":{"type":"string","writeOnly":true}}`,
			historicalRequired: `["value","token"]`, currentRequired: `["value"]`, keywords: `["additionalProperties", "properties", "required", "type", "writeOnly"]`, secretPaths: `["/token"]`, secretAddressing: `["json-pointer-v1"]`, wantError: true,
		},
		"presentation title changed": {
			historicalProperties: `{"value":{"type":"string","title":"Old label"}}`, currentProperties: `{"value":{"type":"string","title":"New label"}}`,
			historicalRequired: `["value"]`, currentRequired: `["value"]`, keywords: `["additionalProperties", "properties", "required", "title", "type"]`, secretPaths: `[]`, secretAddressing: `[]`, wantError: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			historical := evolutionSemanticType(1, tc.historicalProperties, tc.historicalRequired, tc.keywords, tc.secretPaths, tc.secretAddressing)
			current := evolutionSemanticType(2, tc.currentProperties, tc.currentRequired, tc.keywords, tc.secretPaths, tc.secretAddressing)
			chain, err := BuildChain(NewCatalog(evolutionCorpus(t, historical, current)).ForHost("destinations").Authorable())
			require.NoError(t, err)

			_, diags := chain.pinnedChild("transform", CatalogPin{Role: "transform", TypeID: "route", SchemaVersion: 1, CodecVersion: 1})
			require.Equal(t, tc.wantError, diags.HasError(), "%v", diags)
			if tc.wantError {
				require.Equal(t, "Pinned transform artifact cannot be represented", diags.Errors()[0].Summary())
			}
		})
	}
}

func TestCatalogEvolutionPredicateWriteUsesRetainedCompatibilityMetadata(t *testing.T) {
	for name, tc := range map[string]struct {
		mutateHistorical func(*TypeSpec)
		mutateCurrent    func(*TypeSpec)
		negate           bool
		want             string
	}{
		"compatible transform types": {
			mutateHistorical: func(spec *TypeSpec) { spec.CompatibleTransformTypes = nil },
			mutateCurrent:    func(spec *TypeSpec) { spec.CompatibleTransformTypes = []string{"route"} },
			want:             "Predicate is incompatible with transform",
		},
		"negate support": {
			mutateHistorical: func(spec *TypeSpec) { spec.CompatibleTransformTypes, spec.SupportsNegate = []string{"route"}, false },
			mutateCurrent:    func(spec *TypeSpec) { spec.CompatibleTransformTypes, spec.SupportsNegate = []string{"route"}, true },
			negate:           true,
			want:             "Predicate does not support negate",
		},
	} {
		t.Run(name, func(t *testing.T) {
			corpus := evolutionCorpus(t,
				evolutionType("route", "transform", 1, 1, true, "destinations", "replacement", "string"),
				evolutionType("topic_match", "predicate", 1, 1, true, "destinations", "pattern", "string"),
				evolutionType("topic_match", "predicate", 2, 1, true, "destinations", "pattern", "string"),
			)
			for i := range corpus.Types {
				spec := &corpus.Types[i]
				if spec.Role != "predicate" {
					continue
				}
				if spec.SchemaVersion == 1 {
					tc.mutateHistorical(spec)
				} else {
					tc.mutateCurrent(spec)
				}
			}
			chain, err := BuildChain(NewCatalog(corpus).ForHost("destinations").Authorable())
			require.NoError(t, err)
			planned := types.ListValueMust(chain.EntryType, []attr.Value{transformFixtureValue(t, chain, "a", "transform-a", "value", "p", tc.negate)})

			_, diags := chain.ProjectGraph(
				planned,
				[]Correlated{{Key: "a", Type: "route", ID: "transform-a", SchemaVersion: 1, CodecVersion: 1, Matched: true}},
				nil,
				map[string]Correlated{"p": {Key: "p", Type: "topic_match", ID: "predicate-p", SchemaVersion: 1, CodecVersion: 1, Matched: true}},
			)
			require.True(t, diags.HasError())
			require.Equal(t, tc.want, diags.Errors()[0].Summary())
		})
	}
}

func evolutionSecretType(schemaVersion int, secretName string) string {
	return fmt.Sprintf(`{
  "type_id": "route",
  "schema_version": %d,
  "codec_version": 1,
  "publishable": true,
  "authorable": true,
  "role": "transform",
  "hosts": ["destinations"],
  "data_schema": {
    "type": "object",
    "additionalProperties": false,
    "properties": {
      "value": {"type": "string"},
      %q: {"type": "string", "writeOnly": true}
    },
    "required": ["value", %q]
  },
  "consumer_requirements": {
    "version": 1,
    "schema_keywords": ["additionalProperties", "properties", "required", "type", "writeOnly"],
    "schema_types": ["object", "string"],
    "schema_formats": [],
    "ui_elements": [],
    "ui_controls": [],
    "secret_addressing": ["json-pointer-v1"]
  },
  "secret_paths": ["/%s"],
  "row_keys": {},
  "examples": {"read_projection": {"value": "retained"}}
}`,
		schemaVersion, secretName, secretName, secretName,
	)
}

func TestCatalogEvolutionHistoricalSecretUsesRetainedPinMetadata(t *testing.T) {
	historical := evolutionSecretType(1, "legacy_token")
	current := evolutionSecretType(2, "current_token")
	chain, err := BuildChain(NewCatalog(evolutionCorpus(t, historical, current)).ForHost("destinations").Authorable())
	require.NoError(t, err)
	surface, resourceSchema := testSurface(chain)
	read := &api.SMTChainRead{Desired: api.SMTChainSnapshot{
		ChainRevision: "r1",
		Transforms: []api.SMTInstanceRead{{
			Kind: "typed", ID: "instance-1", Alias: "Route1", Type: "route",
			SchemaVersion: 1, CodecVersion: 1, Name: "old", Enabled: true,
			Config: map[string]any{"value": "retained"},
		}},
	}}
	prior, _, readDiags := surface.stateFrom(context.Background(), read, []string{"old"}, nil)
	require.False(t, readDiags.HasError(), "%v", readDiags)

	secretEntryType := resourceSchema.Attributes[AttrSecrets].GetType().(types.ListType).ElemType.(types.ObjectType)
	secretValueListType := secretEntryType.AttributeTypes()[attrValues].(types.ListType)
	secretValueType := secretValueListType.ElemType.(types.ObjectType)
	secretValues := types.ListValueMust(secretValueType, []attr.Value{types.ObjectValueMust(secretValueType.AttributeTypes(), map[string]attr.Value{
		attrPointer: types.StringValue("/legacy_token"),
		attrValue:   types.StringValue("replacement"),
		attrClear:   types.BoolNull(),
	})})
	secrets := types.ListValueMust(secretEntryType, []attr.Value{types.ObjectValueMust(secretEntryType.AttributeTypes(), map[string]attr.Value{
		attrKey:     types.StringValue("old"),
		attrVersion: types.Int64Value(1),
		attrValues:  secretValues,
	})})
	predicates := types.ListValueMust(chain.PredicateEntryType, nil)
	cfg := testSurfaceConfigValues(t, resourceSchema, map[string]attr.Value{
		AttrChain: prior, AttrSecrets: secrets, AttrPredicates: predicates, AttrRevision: types.StringValue("r1"),
	})
	state := tfsdk.State{Schema: resourceSchema, Raw: testSurfaceRaw(t, resourceSchema, map[string]attr.Value{
		AttrChain: prior, AttrPredicates: predicates, AttrRevision: types.StringValue("r1"),
	})}
	plan := tfsdk.Plan{Schema: resourceSchema, Raw: cfg.Raw}

	diags := surface.ModifyPlan(context.Background(), cfg, state, &plan)
	require.False(t, diags.HasError(), "%v", diags)
}

func TestCatalogEvolutionWriteCarriesCatalogPrecondition(t *testing.T) {
	const digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	corpus := evolutionCorpus(t, evolutionType("route", "transform", 1, 1, true, "destinations", "value", "string"))
	corpus.Digest = digest
	chain, err := BuildChain(NewCatalog(corpus).ForHost("destinations").Authorable())
	require.NoError(t, err)
	surface := NewSurface(chain, nil)
	client := &predicateGraphClient{}

	_, diags := surface.ApplyGraph(context.Background(), client, "destinations", "destination-1", &Write{Configured: true, ChainConfigured: true}, types.StringNull())
	require.True(t, diags.HasError(), "the capture client deliberately returns no response")
	require.Len(t, client.writes, 1)
	encoded, err := json.Marshal(client.writes[0])
	require.NoError(t, err)
	var body struct {
		CatalogPrecondition struct {
			ContractVersion int    `json:"contract_version"`
			Digest          string `json:"digest"`
		} `json:"catalog_precondition"`
	}
	require.NoError(t, json.Unmarshal(encoded, &body))
	require.Equal(t, 1, body.CatalogPrecondition.ContractVersion)
	require.Equal(t, digest, body.CatalogPrecondition.Digest)
}
