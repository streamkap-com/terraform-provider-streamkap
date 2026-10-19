package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const corpusCopy = "../../internal/smt/testdata/contract_corpus.json"

// hostCatalog publishes one transform per host and a predicate on both, the
// shape of the backend catalog: the destination and source chains are built
// from different type sets.
const hostCatalog = `{
  "contract_version": 1,
  "producer": "test",
  "digest": "sha256:host",
  "types": [
    {"type_id": "drop_fields", "schema_version": 1, "codec_version": 1, "publishable": true, "authorable": true, "role": "transform", "hosts": ["destinations"],
     "data_schema": {"type": "object", "properties": {}, "additionalProperties": false},
     "consumer_requirements": {"version": 1, "schema_keywords": ["additionalProperties", "properties", "type"], "schema_types": ["object"], "schema_formats": [], "ui_elements": [], "ui_controls": [], "secret_addressing": []},
     "secret_paths": [], "row_keys": {}, "examples": {"read_projection": {}}},
    {"type_id": "filter", "schema_version": 1, "codec_version": 1, "publishable": true, "authorable": true, "role": "transform", "hosts": ["sources"],
     "data_schema": {"type": "object", "properties": {}, "additionalProperties": false},
     "consumer_requirements": {"version": 1, "schema_keywords": ["additionalProperties", "properties", "type"], "schema_types": ["object"], "schema_formats": [], "ui_elements": [], "ui_controls": [], "secret_addressing": []},
     "secret_paths": [], "row_keys": {}, "examples": {"read_projection": {}}},
    {"type_id": "topic_name_matches", "schema_version": 1, "codec_version": 1, "publishable": true, "authorable": true, "role": "predicate", "hosts": ["destinations", "sources"],
     "data_schema": {"type": "object", "properties": {}, "additionalProperties": false},
     "consumer_requirements": {"version": 1, "schema_keywords": ["additionalProperties", "properties", "type"], "schema_types": ["object"], "schema_formats": [], "ui_elements": [], "ui_controls": [], "secret_addressing": []},
     "secret_paths": [], "row_keys": {}, "examples": {"read_projection": {}}}
  ]
}`

var chainFieldTags = []string{`tfsdk:"smt_chain"`, `tfsdk:"smt_secrets"`, `tfsdk:"smt_predicates"`, `tfsdk:"smt_predicate_secrets"`, `tfsdk:"smt_managed_overrides"`, `tfsdk:"smt_chain_revision"`}

// apiSourcePluginConfig is a plugin config the backend marks as served by the
// API-source runner.
const apiSourcePluginConfig = `{"display_name":"HubSpot","api_source":true,"config":[{"name":"database.hostname.user.defined","user_defined":true,"required":true,"display_name":"Host","value":{"control":"string"}}]}`

func generateWithCatalog(t *testing.T, entityDir string, catalog string, connectors ...string) string {
	t.Helper()
	plugins := map[string]string{}
	for _, c := range connectors {
		plugins[c] = validPluginConfig
		if c == "hubspot" {
			plugins[c] = apiSourcePluginConfig
		}
	}
	backend := fixtureBackend(t, entityDir, validCommonConfig, plugins)
	out := t.TempDir()
	if err := runGenerate(backend, out, entityDir, "", smtOptions{CatalogPath: writeCommandCatalog(t, catalog), BackendRevision: "test"}); err != nil {
		t.Fatalf("runGenerate %s: %v", entityDir, err)
	}
	return out
}

func readGenerated(t *testing.T, dir, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// The opt-in is derived, not listed: every source and destination other than
// kafkadirect and an API source carries the chain fields and names its own host, the catalog
// file builds one chain per published host, and the copy is byte-identical
// with provenance that names the digest, the revision and each host's types.
func TestRunGenerate_SMTChainOptInFollowsCatalogHosts(t *testing.T) {
	for _, tc := range []struct {
		entityDir string
		prefix    string
		host      string
	}{
		{"destinations", "destination_", "destinations"},
		{"sources", "source_", "sources"},
	} {
		t.Run(tc.entityDir, func(t *testing.T) {
			out := generateWithCatalog(t, tc.entityDir, hostCatalog, "postgresql", "mysql", "kafkadirect", "hubspot")
			for _, code := range []string{"postgresql", "mysql"} {
				model := readGenerated(t, out, tc.prefix+code+".go")
				for _, want := range chainFieldTags {
					if !strings.Contains(model, want) {
						t.Errorf("%s%s lacks %s", tc.prefix, code, want)
					}
				}
				if want := `SMTChainHost() string { return "` + tc.host + `" }`; !strings.Contains(model, want) {
					t.Errorf("%s%s must name host %q", tc.prefix, code, tc.host)
				}
				if strings.Contains(model, "smt.") {
					t.Error("the connector file must not build the chain schema itself; the base resource adds the attributes")
				}
			}
			for _, code := range []string{"kafkadirect", "hubspot"} {
				if model := readGenerated(t, out, tc.prefix+code+".go"); strings.Contains(model, "smt_chain") || strings.Contains(model, "SMTChainHost") {
					t.Errorf("%s has no SMT surface on the backend and must not carry the chain", code)
				}
			}

			if readGenerated(t, out, smtCatalogJSONName) != hostCatalog {
				t.Error("the embedded catalog copy must be byte-identical to the artifact")
			}
			sum := sha256.Sum256([]byte(hostCatalog))
			catalogGo := readGenerated(t, out, smtCatalogGoName)
			for _, want := range []string{
				"//\tartifact sha256:   " + hex.EncodeToString(sum[:]),
				"//\tenvelope digest:   sha256:host",
				"//\tbackend revision:  test",
				"//\tdestinations authorable types: 2\n//\t  - drop_fields\n//\t  - topic_name_matches",
				"//\tsources authorable types: 2\n//\t  - filter\n//\t  - topic_name_matches",
				`"destinations": sync.OnceValues(func() (*smt.ChainSchema, error) { return buildSMTChain("destinations") }),`,
				`"sources":      sync.OnceValues(func() (*smt.ChainSchema, error) { return buildSMTChain("sources") }),`,
				`const SMTCatalogSHA256 = "` + hex.EncodeToString(sum[:]) + `"`,
				"//go:embed smt_catalog.json",
			} {
				if !strings.Contains(catalogGo, want) {
					t.Errorf("catalog file lacks %q:\n%s", want, catalogGo)
				}
			}
		})
	}
}

// A host the catalog publishes nothing for gets no chain: an unpublished type
// does not count, matching the backend's potential_support, which only lists
// publishable entries.
func TestRunGenerate_SMTChainSkipsHostsWithoutPublishedTypes(t *testing.T) {
	destinationOnly := strings.Replace(hostCatalog, `"publishable": true, "authorable": true, "role": "transform", "hosts": ["sources"]`, `"publishable": false, "authorable": true, "role": "transform", "hosts": ["sources"]`, 1)
	destinationOnly = strings.Replace(destinationOnly, `"hosts": ["destinations", "sources"]`, `"hosts": ["destinations"]`, 1)
	out := generateWithCatalog(t, "sources", destinationOnly, "postgresql")
	if model := readGenerated(t, out, "source_postgresql.go"); strings.Contains(model, "smt_chain") {
		t.Error("a source must not carry the chain when the catalog publishes no source type")
	}
	if catalogGo := readGenerated(t, out, smtCatalogGoName); strings.Contains(catalogGo, `"sources":`) {
		t.Error("the catalog file must not build a chain for an unpublished host")
	}

	out = generateWithCatalog(t, "destinations", destinationOnly, "postgresql")
	if model := readGenerated(t, out, "destination_postgresql.go"); !strings.Contains(model, `tfsdk:"smt_chain"`) {
		t.Error("the destination host still publishes a type")
	}
}

// Sources and destinations derive their opt-in from the catalog, so they
// refuse to generate without it or without the revision it came from;
// transforms have no chain and need neither.
func TestRunGenerate_SMTChainRequiresCatalog(t *testing.T) {
	for _, entityDir := range []string{"destinations", "sources"} {
		backend := fixtureBackend(t, entityDir, validCommonConfig, map[string]string{
			"postgresql": validPluginConfig,
		})
		err := runGenerate(backend, t.TempDir(), entityDir, "postgresql", smtOptions{})
		if err == nil || !strings.Contains(err.Error(), "--smt-catalog") {
			t.Fatalf("%s must fail without the catalog; got %v", entityDir, err)
		}
		err = runGenerate(backend, t.TempDir(), entityDir, "postgresql", smtOptions{CatalogPath: corpusCopy})
		if err == nil || !strings.Contains(err.Error(), "--backend-revision") {
			t.Fatalf("the catalog must be attributed to a backend revision; got %v", err)
		}
	}

	backend := fixtureBackend(t, "transforms", validCommonConfig, map[string]string{"map_filter": validPluginConfig})
	out := t.TempDir()
	if err := runGenerate(backend, out, "transforms", "", smtOptions{}); err != nil {
		t.Fatalf("transforms generate without a catalog: %v", err)
	}
	if model := readGenerated(t, out, "transform_map_filter.go"); strings.Contains(model, "smt_chain") {
		t.Error("a transform has no SMT surface")
	}
}

// An artifact whose data_schema uses a construct the typed representation
// cannot carry fails the run with the type and the schema path, before any
// file is written.
func TestRunGenerate_SMTCatalogRejectsUnsupportedConstruct(t *testing.T) {
	catalog := filepath.Join(t.TempDir(), "catalog.json")
	if err := os.WriteFile(catalog, []byte(`{
  "contract_version": 1,
  "producer": "test",
  "digest": "sha256:0",
  "types": [{
	"type_id": "union_type", "schema_version": 1, "codec_version": 1, "publishable": true, "authorable": false, "role": "transform",
	"data_schema": {"type": "object", "properties": {"mode": {"oneOf": [{"type": "string"}, {"type": "integer"}]}}, "additionalProperties": false},
	"consumer_requirements": {
	  "version": 1,
	  "schema_keywords": ["additionalProperties", "properties", "type"],
	  "schema_types": ["integer", "object", "string"],
	  "schema_formats": [],
	  "ui_elements": [],
	  "ui_controls": [],
	  "secret_addressing": []
	},
    "secret_paths": [], "row_keys": {}
  }]
}`), 0644); err != nil {
		t.Fatal(err)
	}
	backend := fixtureBackend(t, "destinations", validCommonConfig, map[string]string{"postgresql": validPluginConfig})
	out := t.TempDir()
	err := runGenerate(backend, out, "destinations", "postgresql", smtOptions{CatalogPath: catalog, BackendRevision: "test"})
	if err == nil {
		t.Fatal("a polymorphic union must fail generation")
	}
	for _, want := range []string{"union_type", "oneOf", "/mode"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error must name the type and path; got %v", err)
		}
	}
	if _, statErr := os.Stat(filepath.Join(out, "destination_postgresql.go")); !os.IsNotExist(statErr) {
		t.Error("nothing may be written when the catalog is rejected")
	}
	if _, statErr := os.Stat(filepath.Join(out, smtCatalogGoName)); !os.IsNotExist(statErr) {
		t.Error("the rejected catalog must not be embedded")
	}
}

func TestRunGenerate_SMTCatalogRejectsCapabilityMismatchBeforeWriting(t *testing.T) {
	catalog := filepath.Join(t.TempDir(), "catalog.json")
	if err := os.WriteFile(catalog, []byte(`{
  "contract_version": 1,
  "producer": "test",
  "digest": "sha256:0",
  "types": [{
    "type_id": "declared_string", "schema_version": 1, "codec_version": 1, "publishable": true, "authorable": false, "role": "transform",
    "data_schema": {"type": "object", "properties": {"enabled": {"type": "boolean"}}, "additionalProperties": false},
    "ui_schema": {"type": "Category", "elements": []},
    "consumer_requirements": {
      "version": 1,
      "schema_keywords": ["additionalProperties", "properties", "type"],
      "schema_types": ["object", "string"],
      "schema_formats": [],
      "ui_elements": ["Category"],
      "ui_controls": [],
      "secret_addressing": []
    },
    "secret_paths": [], "row_keys": {}
  }]
}`), 0644); err != nil {
		t.Fatal(err)
	}
	backend := fixtureBackend(t, "destinations", validCommonConfig, map[string]string{"postgresql": validPluginConfig})
	out := filepath.Join(t.TempDir(), "generated")
	err := runGenerate(backend, out, "destinations", "postgresql", smtOptions{CatalogPath: catalog, BackendRevision: "test"})
	if err == nil {
		t.Fatal("a mismatched capability inventory must fail generation")
	}
	for _, want := range []string{"declared_string", "schema_types", "boolean"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error must name the type and mismatched capability; got %v", err)
		}
	}
	if _, statErr := os.Stat(out); !os.IsNotExist(statErr) {
		t.Fatalf("capability rejection created output: %v", statErr)
	}
}
