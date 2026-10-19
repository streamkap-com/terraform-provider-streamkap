package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

func checkSMTImportedAttributes(attributes map[string]string, ids map[string]string, revision string) error {
	if attributes["smt_chain.#"] != "2" {
		return fmt.Errorf("imported chain has %q entries, want 2", attributes["smt_chain.#"])
	}
	for i, key := range []string{"users", "orders"} {
		wantID := ids[key]
		if attributes[fmt.Sprintf("smt_chain.%d.key", i)] != wantID || attributes[fmt.Sprintf("smt_chain.%d.id", i)] != wantID {
			return fmt.Errorf("imported entry %d: key %q id %q, want both %q", i, attributes[fmt.Sprintf("smt_chain.%d.key", i)], attributes[fmt.Sprintf("smt_chain.%d.id", i)], wantID)
		}
		if attributes[fmt.Sprintf("smt_chain.%d.alias", i)] != ids["alias-"+key] {
			return fmt.Errorf("imported entry %d alias %q, want %q", i, attributes[fmt.Sprintf("smt_chain.%d.alias", i)], ids["alias-"+key])
		}
		if attributes[fmt.Sprintf("smt_chain.%d.type", i)] != "regex_router" || attributes[fmt.Sprintf("smt_chain.%d.secret_version", i)] != "0" {
			return fmt.Errorf("imported entry %d lost its type or gained a secret version", i)
		}
	}
	if attributes["smt_chain.0.name"] != "Users" || attributes["smt_chain.0.regex_router.regex"] != "^public\\.users$" || attributes["smt_chain.0.regex_router.replacement"] != "users_routed" {
		return fmt.Errorf("imported users config changed")
	}
	if attributes["smt_chain.1.name"] != "Orders renamed" || attributes["smt_chain.1.regex_router.regex"] != "^public\\.orders$" || attributes["smt_chain.1.regex_router.replacement"] != "orders_routed" {
		return fmt.Errorf("imported orders config changed")
	}
	if attributes["smt_chain_revision"] != revision {
		return fmt.Errorf("imported chain revision %q, want %q", attributes["smt_chain_revision"], revision)
	}
	if _, ok := attributes["smt_secrets.#"]; ok {
		return fmt.Errorf("import invented secret entries")
	}
	return nil
}

func seedDestinationPostgresqlSMTImport(t *testing.T, name string) (string, map[string]string, string) {
	t.Helper()
	client, err := testAccCheckDestroyClient()
	if err != nil {
		t.Fatalf("create import fixture client: %v", err)
	}
	smtClient, ok := client.(api.SMTChainAPI)
	if !ok {
		t.Fatalf("import fixture client %T does not implement api.SMTChainAPI", client)
	}
	ctx := context.Background()
	destination, err := client.CreateDestination(ctx, api.Destination{
		Name:      name,
		Connector: "postgresql",
		Config: map[string]any{
			"database.hostname.user.defined": "postgres",
			"database.port.user.defined":     "5432",
			"database.database.user.defined": "app",
			"connection.username":            "postgres",
			"connection.password":            "postgres",
			"table.name.prefix":              "smt_chain_test",
			"schema.evolution":               "basic",
			"insert.mode":                    "upsert",
			"delete.enabled":                 false,
			"primary.key.mode.user.defined":  "record_key",
			"tasks.max":                      1,
		},
		Tags: []string{},
	})
	if err != nil {
		t.Fatalf("create import fixture destination: %v", err)
	}
	if destination.ID == "" {
		t.Fatal("created import fixture destination has an empty id")
	}
	t.Cleanup(func() {
		current, getErr := client.GetDestination(context.Background(), destination.ID)
		if api.IsNotFound(getErr) {
			return
		}
		if getErr != nil {
			t.Errorf("read import fixture destination %s for cleanup: %v", destination.ID, getErr)
			return
		}
		if current == nil {
			return
		}
		if deleteErr := client.DeleteDestination(context.Background(), destination.ID); api.IsNotFound(deleteErr) {
			return
		} else if deleteErr != nil {
			t.Errorf("delete import fixture destination %s: %v", destination.ID, deleteErr)
			return
		}
		deadline := time.Now().Add(destroyPollTimeout)
		for {
			current, fetchErr := client.GetDestination(context.Background(), destination.ID)
			if api.IsNotFound(fetchErr) {
				return
			}
			if fetchErr != nil {
				t.Errorf("confirm deletion of import fixture destination %s: %v", destination.ID, fetchErr)
				return
			}
			if current == nil {
				return
			}
			if time.Now().After(deadline) {
				t.Errorf("import fixture destination %s still exists after %s", destination.ID, destroyPollTimeout)
				return
			}
			time.Sleep(destroyPollInterval)
		}
	})

	chain, err := smtClient.PutSMTChain(ctx, "destinations", destination.ID, api.SMTChainWrite{Transforms: &[]api.SMTInstanceWrite{
		{Type: "regex_router", Name: "Users", Enabled: true, Config: map[string]any{"regex": "^public\\.users$", "replacement": "users_routed"}},
		{Type: "regex_router", Name: "Orders renamed", Enabled: true, Config: map[string]any{"regex": "^public\\.orders$", "replacement": "orders_routed"}},
	}})
	if err != nil {
		t.Fatalf("create import fixture SMT chain: %v", err)
	}
	if len(chain.Desired.Transforms) != 2 {
		t.Fatalf("created import fixture has %d transforms, want 2", len(chain.Desired.Transforms))
	}
	if chain.Desired.ChainRevision == "" {
		t.Fatal("created import fixture SMT chain has an empty revision")
	}
	ids := map[string]string{
		"users":        chain.Desired.Transforms[0].ID,
		"orders":       chain.Desired.Transforms[1].ID,
		"alias-users":  chain.Desired.Transforms[0].Alias,
		"alias-orders": chain.Desired.Transforms[1].Alias,
	}
	if ids["users"] == "" || ids["orders"] == "" || ids["users"] == ids["orders"] || ids["alias-users"] == "" || ids["alias-orders"] == "" || ids["alias-users"] == ids["alias-orders"] {
		t.Fatalf("server returned invalid import fixture identities: %v", ids)
	}
	return destination.ID, ids, chain.Desired.ChainRevision
}

func TestAccDestinationPostgresqlSMTChainImport(t *testing.T) {
	name := acctestName(t, "smt-import")
	keys := config.Variables{
		"users_key":  config.StringVariable("uninitialized"),
		"orders_key": config.StringVariable("uninitialized"),
	}
	var destinationID string
	var ids map[string]string
	var revision string

	users := smtRegexRouter("${var.users_key}", "Users", "^public\\.users$", "users_routed")
	orders := smtRegexRouter("${var.orders_key}", "Orders renamed", "^public\\.orders$", "orders_routed")
	chain := "\n  smt_chain = [" + users + orders + "\n  ]"
	importConfig := `
variable "users_key" { type = string }
variable "orders_key" { type = string }
` + destinationPostgresqlSMTConfig(name, chain)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			destinationID, ids, revision = seedDestinationPostgresqlSMTImport(t, name)
			keys["users_key"] = config.StringVariable(ids["users"])
			keys["orders_key"] = config.StringVariable(ids["orders"])
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			{
				ResourceName:       smtDest,
				Config:             importConfig,
				ConfigVariables:    keys,
				ImportState:        true,
				ImportStateIdFunc:  func(*terraform.State) (string, error) { return destinationID, nil },
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("import returned %d states, want 1", len(states))
					}
					return checkSMTImportedAttributes(states[0].Attributes, ids, revision)
				},
			},
			{
				Config:          importConfig,
				ConfigVariables: keys,
				Check: func(s *terraform.State) error {
					if smtAttr(s, "id") != destinationID {
						return fmt.Errorf("imported destination id %q, want %q", smtAttr(s, "id"), destinationID)
					}
					return checkSMTImportedAttributes(s.RootModule().Resources[smtDest].Primary.Attributes, ids, revision)
				},
			},
			{
				Config:          importConfig,
				ConfigVariables: keys,
				PlanOnly:        true,
			},
		},
	})
}
