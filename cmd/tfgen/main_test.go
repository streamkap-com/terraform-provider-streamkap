package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureBackend builds a minimal backend tree under a temp dir:
//
//	app/sources/configurations_for_all.json
//	app/sources/plugins/<connector>/configuration.latest.json
//
// Each value of connectorConfigs is written verbatim, so a test can inject
// malformed JSON. Returns the backend root.
func fixtureBackend(t *testing.T, entityDir string, commonJSON string, connectorConfigs map[string]string) string {
	t.Helper()

	root := t.TempDir()
	entityRoot := filepath.Join(root, "app", entityDir)
	if err := os.MkdirAll(entityRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if commonJSON != "" {
		if err := os.WriteFile(filepath.Join(entityRoot, "configurations_for_all.json"), []byte(commonJSON), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for connector, cfg := range connectorConfigs {
		dir := filepath.Join(entityRoot, "plugins", connector)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		// An empty config string means "connector directory exists but has no
		// configuration.latest.json" — the genuine-absence case.
		if cfg == "" {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, "configuration.latest.json"), []byte(cfg), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const validCommonConfig = `{"config":[{"name":"sentinel.common.field","user_defined":true,"display_name":"Sentinel","value":{"control":"string"}}]}`

const validPluginConfig = `{"display_name":"Test","config":[{"name":"database.hostname.user.defined","user_defined":true,"required":true,"display_name":"Host","value":{"control":"string"}}]}`

// TestRunGenerate_MalformedConnectorConfigIsFatal covers GEN-1: a
// configuration.latest.json that exists but does not parse must abort the run.
// Treating it like an absent file drops the connector from the generated set
// while the process still exits 0.
func TestRunGenerate_MalformedConnectorConfigIsFatal(t *testing.T) {
	backend := fixtureBackend(t, "sources", validCommonConfig, map[string]string{
		"postgresql": validPluginConfig,
		"truncated":  `{"display_name":"Truncated","config":[{"name":"a.b",`,
	})

	err := runGenerate(backend, t.TempDir(), "sources", "")
	if err == nil {
		t.Fatal("runGenerate returned nil for a connector whose config exists but fails to parse; the connector would be silently dropped from the generated set")
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Errorf("error must name the connector that failed to parse; got: %v", err)
	}
}

// TestRunGenerate_AbsentConnectorConfigIsSkipped is the GEN-1 counter-case: a
// connector directory with no configuration.latest.json at all is a genuine
// absence and stays a skip, not an error.
func TestRunGenerate_AbsentConnectorConfigIsSkipped(t *testing.T) {
	backend := fixtureBackend(t, "sources", validCommonConfig, map[string]string{
		"postgresql": validPluginConfig,
		"nocfg":      "",
	})

	out := t.TempDir()
	if err := runGenerate(backend, out, "sources", ""); err != nil {
		t.Fatalf("runGenerate should skip a connector with no configuration.latest.json; got error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "source_postgresql.go")); err != nil {
		t.Errorf("the well-formed connector should still be generated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "source_nocfg.go")); !os.IsNotExist(err) {
		t.Error("the connector without a config file should not be generated")
	}
}

// TestRunGenerate_BrokenCommonConfigIsFatal covers GEN-2: the entity-wide
// configurations_for_all.json contributes shared fields (consumer.override.*,
// transforms.*, ...) to every connector in the run. Failing to load it silently
// strips those fields from every generated schema at once.
func TestRunGenerate_BrokenCommonConfigIsFatal(t *testing.T) {
	tests := []struct {
		name   string
		common string
	}{
		{"missing", ""},
		{"malformed", `{"config":[`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := fixtureBackend(t, "sources", tt.common, map[string]string{
				"postgresql": validPluginConfig,
			})

			err := runGenerate(backend, t.TempDir(), "sources", "")
			if err == nil {
				t.Fatal("runGenerate returned nil with an unusable configurations_for_all.json; every connector in the run would silently lose its shared fields")
			}
			if !strings.Contains(err.Error(), "configurations_for_all.json") {
				t.Errorf("error must name the common config file; got: %v", err)
			}
		})
	}
}

// TestRunGenerate_KafkaDirectExemptFromCommonConfig guards the documented
// exemption while GEN-2 makes the common config load fatal: the backend's
// _load_global_configuration() resolves kafkadirect from its plugin config
// alone, so kafkadirect must generate even when no common config is present.
func TestRunGenerate_KafkaDirectExemptFromCommonConfig(t *testing.T) {
	backend := fixtureBackend(t, "sources", "", map[string]string{
		"kafkadirect": validPluginConfig,
	})

	out := t.TempDir()
	if err := runGenerate(backend, out, "sources", "kafkadirect"); err != nil {
		t.Fatalf("kafkadirect must generate without configurations_for_all.json; got error: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(out, "source_kafkadirect.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "sentinel_common_field") {
		t.Error("kafkadirect must not merge common config fields")
	}
}
