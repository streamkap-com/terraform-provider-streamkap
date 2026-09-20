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

// An opted-in connector gets the three chain fields in its model and nothing
// else changes; a connector that is not listed is untouched; the catalog copy
// is byte-identical and its provenance names the digest and the revision.
func TestRunGenerate_SMTChainOptIn(t *testing.T) {
	backend := fixtureBackend(t, "sources", validCommonConfig, map[string]string{
		"postgresql": validPluginConfig,
		"mysql":      validPluginConfig,
	})
	out := t.TempDir()
	if err := runGenerate(backend, out, "sources", "", testSMTOptions()); err != nil {
		t.Fatalf("runGenerate: %v", err)
	}

	postgres, err := os.ReadFile(filepath.Join(out, "source_postgresql.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`tfsdk:"smt_chain"`, `tfsdk:"smt_secrets"`, `tfsdk:"smt_chain_revision"`} {
		if !strings.Contains(string(postgres), want) {
			t.Errorf("opted-in postgresql model lacks %s", want)
		}
	}
	if strings.Contains(string(postgres), "smt.") {
		t.Error("the connector file must not build the chain schema itself; the base resource adds the attributes")
	}
	mysql, err := os.ReadFile(filepath.Join(out, "source_mysql.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mysql), "smt_chain") {
		t.Error("a connector not listed under smt_chain_connectors must not carry the chain fields")
	}

	copied, err := os.ReadFile(filepath.Join(out, smtCatalogJSONName))
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(corpusCopy)
	if err != nil {
		t.Fatal(err)
	}
	if string(copied) != string(original) {
		t.Error("the embedded catalog copy must be byte-identical to the artifact")
	}
	sum := sha256.Sum256(original)
	catalogGo, err := os.ReadFile(filepath.Join(out, smtCatalogGoName))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"//\tartifact sha256:   " + hex.EncodeToString(sum[:]),
		"//\tenvelope digest:   sha256:50ba3ac5ef381a480cb608a5f5b2c00bacc5c868948337335078ef6d36ca04f7",
		"//\tbackend revision:  test",
		"//\tpublishable types: 0 of 2",
		`const SMTCatalogSHA256 = "` + hex.EncodeToString(sum[:]) + `"`,
		"//go:embed smt_catalog.json",
		"smt.BuildChain(smt.NewCatalog(corpus).Publishable())",
	} {
		if !strings.Contains(string(catalogGo), want) {
			t.Errorf("catalog provenance lacks %q", want)
		}
	}
}

// Without the pinned artifact an opted-in connector refuses to generate, and
// a connector with no SMT surface cannot opt in.
func TestRunGenerate_SMTChainRequiresCatalog(t *testing.T) {
	backend := fixtureBackend(t, "sources", validCommonConfig, map[string]string{
		"postgresql": validPluginConfig,
	})
	err := runGenerate(backend, t.TempDir(), "sources", "postgresql", smtOptions{})
	if err == nil || !strings.Contains(err.Error(), "--smt-catalog") {
		t.Fatalf("an opted-in connector must fail without the catalog; got %v", err)
	}

	err = runGenerate(backend, t.TempDir(), "sources", "postgresql", smtOptions{CatalogPath: corpusCopy})
	if err == nil || !strings.Contains(err.Error(), "--backend-revision") {
		t.Fatalf("the catalog must be attributed to a backend revision; got %v", err)
	}

	g := NewGeneratorWithOverrides(t.TempDir(), "source", &OverrideConfig{SMTChainConnectors: []SMTChainConnector{{Connector: "kafkadirect", EntityType: "sources"}}})
	g.smtCatalog = true
	_, err = g.prepareTemplateData(&ConnectorConfig{DisplayName: "Kafka Direct"}, "kafkadirect")
	if err == nil || !strings.Contains(err.Error(), "no SMT chain surface") {
		t.Fatalf("kafkadirect must not opt in; got %v", err)
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
    "type_id": "union_type", "schema_version": 1, "codec_version": 1, "publishable": true,
    "data_schema": {"type": "object", "properties": {"mode": {"oneOf": [{"type": "string"}, {"type": "integer"}]}}},
    "secret_paths": [], "row_keys": {}
  }]
}`), 0644); err != nil {
		t.Fatal(err)
	}
	backend := fixtureBackend(t, "sources", validCommonConfig, map[string]string{"postgresql": validPluginConfig})
	out := t.TempDir()
	err := runGenerate(backend, out, "sources", "postgresql", smtOptions{CatalogPath: catalog, BackendRevision: "test"})
	if err == nil {
		t.Fatal("a polymorphic union must fail generation")
	}
	for _, want := range []string{"union_type", "oneOf", "/mode"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error must name the type and path; got %v", err)
		}
	}
	if _, statErr := os.Stat(filepath.Join(out, "source_postgresql.go")); !os.IsNotExist(statErr) {
		t.Error("nothing may be written when the catalog is rejected")
	}
	if _, statErr := os.Stat(filepath.Join(out, smtCatalogGoName)); !os.IsNotExist(statErr) {
		t.Error("the rejected catalog must not be embedded")
	}
}
