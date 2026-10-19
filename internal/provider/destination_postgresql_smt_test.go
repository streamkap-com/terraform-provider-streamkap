package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const smtDest = "streamkap_destination_postgresql.smt"

// The destination the local stack runs end to end; the fields mirror the
// spec's postgresql.json.
func destinationPostgresqlSMTConfig(name, chain string) string {
	return providerConfig + fmt.Sprintf(`
resource "streamkap_destination_postgresql" "smt" {
  name                = %q
  database_hostname   = "postgres"
  database_port       = 5432
  database_database   = "app"
  connection_username = "postgres"
  connection_password = "postgres"
  table_name_prefix   = "smt_chain_test"
  schema_evolution    = "basic"
  insert_mode         = "upsert"
  delete_enabled      = false
  primary_key_mode    = "record_key"
  tasks_max           = 1
%s
}
`, name, chain)
}

func smtRegexRouter(key, name, regex, replacement string) string {
	return fmt.Sprintf(`
    { key = %q, type = "regex_router", name = %q, regex_router = { regex = %q, replacement = %q } },`, key, name, regex, replacement)
}

func smtAttr(s *terraform.State, name string) string {
	return s.RootModule().Resources[smtDest].Primary.Attributes[name]
}

// TestAccDestinationPostgresqlSMTChain runs a two-instance regex_router chain
// against a live backend: create computes the server identities and the
// revision, reorder plus rename keeps the identities with their keys, and an
// empty chain removes every instance and converges on refresh.
func TestAccDestinationPostgresqlSMTChain(t *testing.T) {
	name := acctestName(t, "smt")
	ids := map[string]string{}
	var revision string

	orders := smtRegexRouter("orders", "Orders", "^public\\.orders$", "orders_routed")
	users := smtRegexRouter("users", "Users", "^public\\.users$", "users_routed")
	chain := func(entries ...string) string {
		out := "\n  smt_chain = ["
		for _, e := range entries {
			out += e
		}
		return out + "\n  ]"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			{
				Config: destinationPostgresqlSMTConfig(name, chain(orders, users)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(smtDest, "id"),
					resource.TestCheckResourceAttr(smtDest, "name", name),
					resource.TestCheckResourceAttr(smtDest, "smt_chain.#", "2"),
					resource.TestCheckResourceAttr(smtDest, "smt_chain.0.key", "orders"),
					resource.TestCheckResourceAttr(smtDest, "smt_chain.0.type", "regex_router"),
					resource.TestCheckResourceAttr(smtDest, "smt_chain.0.name", "Orders"),
					resource.TestCheckResourceAttr(smtDest, "smt_chain.0.enabled", "true"),
					resource.TestCheckResourceAttr(smtDest, "smt_chain.0.regex_router.regex", "^public\\.orders$"),
					resource.TestCheckResourceAttr(smtDest, "smt_chain.0.regex_router.replacement", "orders_routed"),
					resource.TestCheckResourceAttr(smtDest, "smt_chain.0.schema_version", "1"),
					resource.TestCheckResourceAttrSet(smtDest, "smt_chain.0.id"),
					resource.TestCheckResourceAttrSet(smtDest, "smt_chain.0.alias"),
					resource.TestCheckResourceAttr(smtDest, "smt_chain.1.key", "users"),
					resource.TestCheckResourceAttrSet(smtDest, "smt_chain.1.id"),
					resource.TestCheckResourceAttrSet(smtDest, "smt_chain.1.alias"),
					resource.TestCheckResourceAttrSet(smtDest, "smt_chain_revision"),
					func(s *terraform.State) error {
						ids["orders"], ids["users"] = smtAttr(s, "smt_chain.0.id"), smtAttr(s, "smt_chain.1.id")
						ids["alias-orders"], ids["alias-users"] = smtAttr(s, "smt_chain.0.alias"), smtAttr(s, "smt_chain.1.alias")
						if ids["orders"] == ids["users"] || ids["alias-orders"] == ids["alias-users"] {
							return fmt.Errorf("server identities are not distinct: %v", ids)
						}
						revision = smtAttr(s, "smt_chain_revision")
						return nil
					},
				),
			},
			{
				Config:   destinationPostgresqlSMTConfig(name, chain(orders, users)),
				PlanOnly: true,
			},
			{
				Config: destinationPostgresqlSMTConfig(name, chain(users, smtRegexRouter("orders", "Orders renamed", "^public\\.orders$", "orders_routed"))),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(smtDest, plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(smtDest, "smt_chain.0.key", "users"),
					resource.TestCheckResourceAttr(smtDest, "smt_chain.1.key", "orders"),
					resource.TestCheckResourceAttr(smtDest, "smt_chain.1.name", "Orders renamed"),
					func(s *terraform.State) error {
						if smtAttr(s, "smt_chain.0.id") != ids["users"] || smtAttr(s, "smt_chain.1.id") != ids["orders"] {
							return fmt.Errorf("ids did not follow their keys: %q %q, want %q %q", smtAttr(s, "smt_chain.0.id"), smtAttr(s, "smt_chain.1.id"), ids["users"], ids["orders"])
						}
						if smtAttr(s, "smt_chain.0.alias") != ids["alias-users"] || smtAttr(s, "smt_chain.1.alias") != ids["alias-orders"] {
							return fmt.Errorf("aliases did not follow their keys: %q %q", smtAttr(s, "smt_chain.0.alias"), smtAttr(s, "smt_chain.1.alias"))
						}
						if smtAttr(s, "smt_chain_revision") == revision {
							return fmt.Errorf("revision did not advance on update")
						}
						return nil
					},
				),
			},
			{
				Config: destinationPostgresqlSMTConfig(name, "\n  smt_chain = []"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(smtDest, "smt_chain.#", "0"),
					resource.TestCheckResourceAttrSet(smtDest, "smt_chain_revision"),
				),
			},
			{
				Config:   destinationPostgresqlSMTConfig(name, "\n  smt_chain = []"),
				PlanOnly: true,
			},
		},
	})
}
