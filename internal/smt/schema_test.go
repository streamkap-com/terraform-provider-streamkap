package smt

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

const corpusPath = "testdata/contract_corpus.json"

func loadCorpus(t *testing.T) (*Corpus, Catalog) {
	t.Helper()
	c, err := LoadCorpus(corpusPath)
	require.NoError(t, err)
	require.Len(t, c.Types, 2, "the corpus copy carries both fixture types")
	return c, NewCatalog(c)
}

// The builder must produce exactly the tree the corpus declares, with the
// writeOnly leaves gone and nothing Dynamic anywhere.
func TestBuildConfigChild_CorpusShapes(t *testing.T) {
	_, cat := loadCorpus(t)

	for _, id := range cat.TypeIDs() {
		spec := cat[id]
		t.Run(id, func(t *testing.T) {
			child, err := BuildConfigChild(spec)
			require.NoError(t, err)
			require.ElementsMatch(t, spec.SecretPaths, child.SecretTemplates,
				"stripped writeOnly leaves must be exactly the corpus secret_paths")

			paths := map[string]string{}
			DescribePaths(child.Attribute, "/", paths)
			for p, typ := range paths {
				require.NotContains(t, typ, "Dynamic", "path %s became dynamic", p)
			}
			for _, tmpl := range spec.SecretPaths {
				require.NotContains(t, paths, tmpl, "secret leaf must not be a config attribute")
			}

			// Validate the shape the way the framework will at GetProviderSchema.
			diags := schema.Schema{Attributes: map[string]schema.Attribute{id: child.Attribute}}.ValidateImplementation(context.Background())
			require.False(t, diags.HasError(), "%v", diags)
		})
	}

	nested, err := BuildConfigChild(cat["contract_fixture_nested"])
	require.NoError(t, err)
	paths := map[string]string{}
	DescribePaths(nested.Attribute, "/", paths)
	require.Equal(t, map[string]string{
		"/":                   nested.Attribute.GetType().String(),
		"/enabled":            "basetypes.BoolType",
		"/fallback":           "basetypes.StringType",
		"/labels":             "types.ListType[basetypes.StringType]",
		"/max_records":        "basetypes.Int64Type",
		"/output":             `types.ObjectType["label":basetypes.StringType, "prefix":basetypes.StringType, "retry_limit":basetypes.Int64Type]`,
		"/output/label":       "basetypes.StringType",
		"/output/prefix":      "basetypes.StringType",
		"/output/retry_limit": "basetypes.Int64Type",
		"/retries":            "basetypes.Int64Type",
		"/routes":             `types.ListType[types.ObjectType["endpoint":basetypes.StringType, "key":basetypes.StringType, "label":basetypes.StringType]]`,
		"/routes/endpoint":    "basetypes.StringType",
		"/routes/key":         "basetypes.StringType",
		"/routes/label":       "basetypes.StringType",
	}, paths)

	attrs := nested.Attribute.Attributes
	fallback := attrs["fallback"].(schema.StringAttribute)
	require.True(t, fallback.Optional && !fallback.Computed && fallback.Default == nil, "nullable scalar is plain Optional")
	retries := attrs["retries"].(schema.Int64Attribute)
	require.True(t, retries.Optional && retries.Computed && retries.Default != nil, "defaulted scalar is Optional+Computed+Default")
	output := attrs["output"].(schema.SingleNestedAttribute)
	require.NotNil(t, output.Default, "an object whose children all default gets an object default")
	routes := attrs["routes"].(schema.ListNestedAttribute)
	require.NotNil(t, routes.Default, "an optional object list defaults to the empty list")
	require.True(t, routes.NestedObject.Attributes["key"].IsRequired())
	require.True(t, routes.NestedObject.Attributes["endpoint"].IsRequired())

	deep, err := BuildConfigChild(cat["contract_fixture_deep"])
	require.NoError(t, err)
	paths = map[string]string{}
	DescribePaths(deep.Attribute, "/", paths)
	require.Contains(t, paths, "/services/endpoints/url")
	require.Contains(t, paths, "/services/auth/username")
	require.NotContains(t, paths, "/services/auth/token")
	require.NotContains(t, paths, "/services/endpoints/secret")
}

// Every unsupported construct must fail with its path and name. The inputs
// are deliberately outside the corpus: they are the shapes the catalog must
// never admit.
func TestBuildConfigChild_RejectsUnsupportedConstructs(t *testing.T) {
	cases := map[string]struct {
		schema    string
		path      string
		construct string
	}{
		"polymorphic anyOf": {
			schema:    `{"type":"object","properties":{"v":{"anyOf":[{"type":"string"},{"type":"integer"}]}}}`,
			path:      "/v",
			construct: "anyOf",
		},
		"polymorphic type list": {
			schema:    `{"type":"object","properties":{"v":{"type":["string","integer"]}}}`,
			path:      "/v",
			construct: "type list",
		},
		"oneOf union": {
			schema:    `{"type":"object","properties":{"v":{"oneOf":[{"type":"string"},{"type":"object","properties":{"a":{"type":"string"}}}]}}}`,
			path:      "/v",
			construct: "oneOf",
		},
		"heterogeneous tuple prefixItems": {
			schema:    `{"type":"object","properties":{"v":{"type":"array","prefixItems":[{"type":"string"},{"type":"integer"}]}}}`,
			path:      "/v",
			construct: "prefixItems",
		},
		"heterogeneous tuple items array": {
			schema:    `{"type":"object","properties":{"v":{"type":"array","items":[{"type":"string"},{"type":"integer"}]}}}`,
			path:      "/v",
			construct: "items[]",
		},
		"recursive ref": {
			schema:    `{"type":"object","properties":{"child":{"type":"object","properties":{"next":{"$ref":"#"}}}}}`,
			path:      "/child/next",
			construct: "$ref",
		},
		"nested inside a list row": {
			schema:    `{"type":"object","properties":{"rows":{"type":"array","items":{"type":"object","properties":{"v":{"anyOf":[{"type":"string"},{"type":"boolean"}]}}}}}}`,
			path:      "/rows/*/v",
			construct: "anyOf",
		},
		"bare secret list": {
			schema:    `{"type":"object","properties":{"tokens":{"type":"array","items":{"type":"string","writeOnly":true}}}}`,
			path:      "/tokens/*",
			construct: "writeOnly list element",
		},
		"map inside a config child": {
			schema:    `{"type":"object","properties":{"m":{"type":"object","additionalProperties":{"type":"string"}}}}`,
			path:      "/m",
			construct: "object without properties",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var node JSONSchema
			require.NoError(t, json.Unmarshal([]byte(tc.schema), &node))
			_, err := BuildConfigChild(&TypeSpec{TypeID: "t", DataSchema: &node})
			require.Error(t, err)
			var uerr *UnsupportedConstructError
			require.ErrorAs(t, err, &uerr)
			require.Equal(t, tc.path, uerr.Path)
			require.Equal(t, tc.construct, uerr.Construct)
			require.NotContains(t, strings.ToLower(err.Error()), "dynamic")
		})
	}
}

// Homogeneous maps are a separate gate: built on request, never admitted into
// a config child by the corpus walk.
func TestBuildMapAttribute_Gate(t *testing.T) {
	var scalarMap JSONSchema
	require.NoError(t, json.Unmarshal([]byte(`{"type":"object","additionalProperties":{"type":"string"}}`), &scalarMap))
	a, err := BuildMapAttribute(&scalarMap, "/m")
	require.NoError(t, err)
	require.Equal(t, types.MapType{ElemType: types.StringType}, a.GetType())

	var objectMap JSONSchema
	require.NoError(t, json.Unmarshal([]byte(`{"type":"object","additionalProperties":{"type":"object","properties":{"url":{"type":"string"}}}}`), &objectMap))
	a, err = BuildMapAttribute(&objectMap, "/m")
	require.NoError(t, err)
	_, isNested := a.(schema.MapNestedAttribute)
	require.True(t, isNested)

	var open JSONSchema
	require.NoError(t, json.Unmarshal([]byte(`{"type":"object","additionalProperties":true}`), &open))
	_, err = BuildMapAttribute(&open, "/m")
	require.Error(t, err, "an untyped map has no element type to carry")
}
