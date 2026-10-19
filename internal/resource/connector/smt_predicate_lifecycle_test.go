package connector_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func predicateGraphConfig(copyOnWrite, keepOnlyB bool) string {
	predicates := `{ key = "p", type = "topic_match_secret", topic_match_secret = { pattern = "^orders" } }`
	secrets := `{ key = "p", version = 1, values = [{ pointer = "/token", value = "p-token" }] }`
	if copyOnWrite && !keepOnlyB {
		predicates += `, { key = "q", type = "topic_match_secret", topic_match_secret = { pattern = "^payments" } }`
		secrets += `, { key = "q", version = 1, values = [{ pointer = "/token", value = "q-token" }] }`
	}
	transform := func(key, predicate string) string {
		return fmt.Sprintf(`{
      key = %q
      type = "contract_fixture_nested"
      name = %q
      predicate = { key = %q }
      contract_fixture_nested = { routes = [{ key = "row", endpoint = "example.invalid" }] }
    }`, key, key, predicate)
	}
	transforms := transform("b", "p")
	if !keepOnlyB {
		predicate := "p"
		if copyOnWrite {
			predicate = "q"
		}
		transforms = transform("a", predicate) + "," + transforms
	}
	return hcl(fmt.Sprintf(`
  smt_predicates = [%s]
  smt_predicate_secrets = [%s]
  smt_chain = [%s]
`, predicates, secrets, transforms))
}

func emptyPredicateGraphConfig() string {
	return hcl(`
  smt_predicates = []
  smt_predicate_secrets = []
  smt_chain = []
`)
}

func TestSMTPredicateGraphRunsThroughBaseConnectorLifecycle(t *testing.T) {
	factories, fake := newPredicateFixture(t)
	var predicateP string
	var predicateQ string
	secretIs := func(id *string, want string, present bool) resource.TestCheckFunc {
		return func(*terraform.State) error {
			fake.mu.Lock()
			defer fake.mu.Unlock()
			value, exists := fake.secrets[*id]["/token"]
			if exists != present || present && value != want {
				return fmt.Errorf("predicate secret storage does not match the expected isolated lifecycle")
			}
			return nil
		}
	}
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   smtTerraformVersionChecks(tfversion.SkipBelow(tfversion.Version1_11_0)),
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: predicateGraphConfig(false, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "smt_predicates.#", "1"),
					resource.TestCheckResourceAttr(addr, "smt_predicates.0.key", "p"),
					resource.TestCheckResourceAttr(addr, "smt_predicates.0.secret_version", "1"),
					resource.TestCheckResourceAttr(addr, "smt_predicates.0.configured_secret_paths.#", "1"),
					resource.TestCheckResourceAttr(addr, "smt_predicates.0.configured_secret_paths.0", "/token"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.predicate.key", "p"),
					resource.TestCheckResourceAttr(addr, "smt_chain.1.predicate.key", "p"),
					func(state *terraform.State) error {
						predicateP = attrOf(state, "smt_predicates.0.id")
						if predicateP == "" {
							return fmt.Errorf("predicate P has no persisted id")
						}
						return nil
					},
					secretIs(&predicateP, "p-token", true),
				),
			},
			{
				Config: predicateGraphConfig(true, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "smt_predicates.#", "2"),
					resource.TestCheckResourceAttr(addr, "smt_predicates.0.secret_version", "1"),
					resource.TestCheckResourceAttr(addr, "smt_predicates.1.secret_version", "1"),
					resource.TestCheckResourceAttr(addr, "smt_predicates.1.configured_secret_paths.#", "1"),
					func(state *terraform.State) error {
						if got := attrOf(state, "smt_predicates.0.id"); got != predicateP {
							return fmt.Errorf("predicate P identity changed: got %q want %q", got, predicateP)
						}
						predicateQ = attrOf(state, "smt_predicates.1.id")
						if predicateQ == "" || predicateQ == predicateP {
							return fmt.Errorf("predicate Q has no fresh persisted identity")
						}
						return nil
					},
					secretIs(&predicateP, "p-token", true),
					secretIs(&predicateQ, "q-token", true),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.predicate.key", "q"),
					resource.TestCheckResourceAttr(addr, "smt_chain.1.predicate.key", "p"),
				),
			},
			{
				Config: predicateGraphConfig(true, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "smt_predicates.#", "1"),
					func(state *terraform.State) error {
						if got := attrOf(state, "smt_predicates.0.id"); got != predicateP {
							return fmt.Errorf("predicate P identity changed after cleanup: got %q want %q", got, predicateP)
						}
						return nil
					},
					resource.TestCheckResourceAttr(addr, "smt_chain.#", "1"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.predicate.key", "p"),
					secretIs(&predicateP, "p-token", true),
					secretIs(&predicateQ, "", false),
				),
			},
			{
				Config: emptyPredicateGraphConfig(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "smt_predicates.#", "0"),
					resource.TestCheckResourceAttr(addr, "smt_chain.#", "0"),
					secretIs(&predicateP, "", false),
				),
			},
		},
	})
	if fake.connectorWrites != 4 {
		t.Fatalf("want one connector write per graph apply, got %d", fake.connectorWrites)
	}
}
