package connector_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/destination"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/smt"
)

const pgDest = "streamkap_destination_postgresql.t"

// newCatalogFixture registers the real destination_postgresql resource, chain
// built from the pinned catalog, over the fake client. Nothing is
// substituted: the types are the ones the released provider admits.
func newCatalogFixture(t *testing.T) (map[string]func() (tfprotov6.ProviderServer, error), *fakeAPI) {
	t.Helper()
	fake := newFakeAPI(generated.SMTChain("destinations").Catalog)
	p := &testProvider{client: fake, configs: []connector.ConnectorConfig{&destination.PostgreSQLConfig{}}}
	return map[string]func() (tfprotov6.ProviderServer, error){
		"streamkap": providerserver.NewProtocol6WithError(p),
	}, fake
}

func pgHCL(body string) string {
	return `
resource "streamkap_destination_postgresql" "t" {
  name                = "chain"
  database_hostname   = "postgres"
  database_port       = 5432
  database_database   = "app"
  connection_username = "postgres"
  connection_password = "postgres"
  table_name_prefix   = "smt_chain_test"
` + body + "\n}\n"
}

func regexRouter(key, name, regex string) string {
	return fmt.Sprintf(`
    { key = %q, type = "regex_router", name = %q, regex_router = { regex = %q } },`, key, name, regex)
}

func maskField(key, name string) string {
	return fmt.Sprintf(`
    { key = %q, type = "mask_field", name = %q, mask_field = { fields_include = ["email"] } },`, key, name)
}

func maskSalt(version int, value string) string {
	return fmt.Sprintf(`
  smt_secrets = [{ key = "m", version = %d, values = [{ pointer = "/mask_salt", value = %q }] }]`, version, value)
}

func pgAttr(s *terraform.State, name string) string {
	return s.RootModule().Resources[pgDest].Primary.Attributes[name]
}

// The published types drive the whole lifecycle through the real
// destination_postgresql resource: two regex_router instances plus a
// mask_field whose salt is rotated through smt_secrets, reorder and rename
// keeping the server identities, a stale revision refused and retried, a key
// change minting a new instance, and import deriving keys from the ids.
func TestSMT_CatalogTypesOnDestinationPostgresql(t *testing.T) {
	factories, fake := newCatalogFixture(t)
	ids := map[string]string{}
	var revision string

	orders := regexRouter("orders", "Orders", "^public\\.orders$")
	users := regexRouter("users", "Users", "^public\\.users$")
	mask := maskField("m", "Mask email")
	chain := func(entries ...string) string {
		return "\n  smt_chain = [" + strings.Join(entries, "") + "\n  ]"
	}

	tfresource.UnitTest(t, tfresource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{
				Config: pgHCL(chain(orders, users, mask) + maskSalt(1, "salt-1")),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.#", "3"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.0.key", "orders"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.0.type", "regex_router"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.0.regex_router.regex", "^public\\.orders$"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.0.regex_router.replacement", "$0"),
					tfresource.TestCheckNoResourceAttr(pgDest, "smt_chain.0.mask_field.%"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.0.schema_version", "1"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.0.secret_version", "0"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.2.key", "m"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.2.mask_field.fields_include.0", "email"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.2.mask_field.fields_exclude.#", "0"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.2.mask_field.mask_function", "SHA256_TRUNCATE"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.2.mask_field.mask_char", "*"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.2.mask_field.mask_fixed_value", "***"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.2.mask_field.replace_null_with_default", "true"),
					tfresource.TestCheckNoResourceAttr(pgDest, "smt_chain.2.mask_field.mask_salt"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.2.secret_version", "1"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_secrets.0.version", "1"),
					tfresource.TestCheckNoResourceAttr(pgDest, "smt_secrets.0.values.#"),
					tfresource.TestCheckResourceAttrSet(pgDest, "smt_chain_revision"),
					func(s *terraform.State) error {
						for i, key := range []string{"orders", "users", "m"} {
							ids[key] = pgAttr(s, fmt.Sprintf("smt_chain.%d.id", i))
							ids["alias-"+key] = pgAttr(s, fmt.Sprintf("smt_chain.%d.alias", i))
							if ids[key] == "" || ids["alias-"+key] == "" {
								return fmt.Errorf("entry %d has no server identity", i)
							}
						}
						revision = pgAttr(s, "smt_chain_revision")
						id := pgAttr(s, "id")
						if sent := fake.flatKeysSent(id); len(sent) != 0 {
							return fmt.Errorf("flat SMT keys reached the API with the chain configured: %v", sent)
						}
						stored := fake.instances(id)
						if len(stored) != 3 || stored[2].Config["mask_function"] != "SHA256_TRUNCATE" {
							return fmt.Errorf("server chain not stored with defaults: %+v", stored)
						}
						if _, leaked := stored[2].Config["mask_salt"]; leaked {
							return fmt.Errorf("the secret leaf reached the config")
						}
						ops := fake.opsSince(0)
						if len(ops) != 1 || ops[0].InstanceID != ids["m"] || ops[0].Pointer != "/mask_salt" || ops[0].Value != "salt-1" {
							return fmt.Errorf("want one salt replace on %s, got %v", ids["m"], ops)
						}
						return nil
					},
				),
			},
			{
				Config:   pgHCL(chain(orders, users, mask) + maskSalt(1, "salt-1")),
				PlanOnly: true,
			},
			{
				// Reorder and rename: identities follow the keys, nothing rotates.
				Config: pgHCL(chain(users, regexRouter("orders", "Orders renamed", "^public\\.orders$"), mask) + maskSalt(1, "salt-1")),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(pgDest, plancheck.ResourceActionUpdate),
				}},
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.0.key", "users"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.1.key", "orders"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.1.name", "Orders renamed"),
					func(s *terraform.State) error {
						if pgAttr(s, "smt_chain.0.id") != ids["users"] || pgAttr(s, "smt_chain.1.id") != ids["orders"] || pgAttr(s, "smt_chain.2.id") != ids["m"] {
							return fmt.Errorf("ids did not follow their keys")
						}
						if pgAttr(s, "smt_chain.0.alias") != ids["alias-users"] || pgAttr(s, "smt_chain.1.alias") != ids["alias-orders"] {
							return fmt.Errorf("aliases did not follow their keys")
						}
						if got := fake.opsSince(1); len(got) != 0 {
							return fmt.Errorf("reorder must not rotate, got %v", got)
						}
						if stored := fake.instances(pgAttr(s, "id")); stored[0].ID != ids["users"] {
							return fmt.Errorf("server order does not match state")
						}
						if pgAttr(s, "smt_chain_revision") == revision {
							return fmt.Errorf("revision did not advance")
						}
						revision = pgAttr(s, "smt_chain_revision")
						return nil
					},
				),
			},
			{
				// An edit lands elsewhere between plan and apply.
				PreConfig:   func() { fake.bumpRevisionBeforeNextWrite = true },
				Config:      pgHCL(chain(users, orders, mask) + maskSalt(1, "salt-1")),
				ExpectError: regexp.MustCompile(`Transform chain changed outside Terraform[\s\S]*no longer current`),
			},
			{
				// State kept the prior chain; the refresh picks up the new
				// revision and the retried apply goes through.
				Config: pgHCL(chain(users, orders, mask) + maskSalt(1, "salt-1")),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(pgDest, plancheck.ResourceActionUpdate),
				}},
				Check: func(s *terraform.State) error {
					if pgAttr(s, "smt_chain.1.name") != "Orders" || pgAttr(s, "smt_chain.1.id") != ids["orders"] {
						return fmt.Errorf("retried apply lost the rename or the identity")
					}
					if pgAttr(s, "smt_chain_revision") == revision {
						return fmt.Errorf("revision did not advance after the retried apply")
					}
					return nil
				},
			},
			{
				// Key change: a new server instance; the old one is gone.
				Config: pgHCL(chain(users, regexRouter("orders2", "Orders", "^public\\.orders$"), mask) + maskSalt(1, "salt-1")),
				Check: func(s *terraform.State) error {
					if pgAttr(s, "smt_chain.1.key") != "orders2" || pgAttr(s, "smt_chain.1.id") == ids["orders"] || pgAttr(s, "smt_chain.1.id") == "" {
						return fmt.Errorf("a changed key must mint a new instance, got id %q", pgAttr(s, "smt_chain.1.id"))
					}
					stored := fake.instances(pgAttr(s, "id"))
					for _, inst := range stored {
						if inst.ID == ids["orders"] {
							return fmt.Errorf("the old instance survived the key change")
						}
					}
					if len(stored) != 3 {
						return fmt.Errorf("server chain should hold three instances, got %d", len(stored))
					}
					ids["orders2"] = pgAttr(s, "smt_chain.1.id")
					return nil
				},
			},
			{
				// Salt rotation: one replace under the recorded instance, the
				// chain entry records the version.
				Config: pgHCL(chain(users, regexRouter("orders2", "Orders", "^public\\.orders$"), mask) + maskSalt(2, "salt-2")),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(pgDest, "smt_chain.2.secret_version", "2"),
					tfresource.TestCheckResourceAttr(pgDest, "smt_secrets.0.version", "2"),
					func(s *terraform.State) error {
						ops := fake.opsSince(1)
						if len(ops) != 1 || ops[0].InstanceID != ids["m"] || ops[0].Value != "salt-2" {
							return fmt.Errorf("want exactly one salt replace at version 2, got %v", ops)
						}
						if fake.secretValue(ids["m"], "/mask_salt") != "salt-2" {
							return fmt.Errorf("server salt not rotated")
						}
						return nil
					},
				),
			},
			{
				Config:      pgHCL(chain(users, regexRouter("orders2", "Orders", "^public\\.orders$"), mask) + maskSalt(1, "salt-1")),
				ExpectError: regexp.MustCompile(`Rotation version decreased`),
			},
			{
				Config: pgHCL(chain(users, regexRouter("orders2", "Orders", "^public\\.orders$"), mask) + `
  smt_secrets = [{ key = "m", version = 3, values = [{ pointer = "/salt", value = "x" }] }]`),
				ExpectError: regexp.MustCompile(`Invalid secret pointer`),
			},
			{
				ResourceName:      pgDest,
				ImportState:       true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) { return pgAttr(s, "id"), nil },
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					a := states[0].Attributes
					if a["smt_chain.#"] != "3" {
						return fmt.Errorf("imported chain has %q entries", a["smt_chain.#"])
					}
					for i, want := range []string{ids["users"], ids["orders2"], ids["m"]} {
						if a[fmt.Sprintf("smt_chain.%d.key", i)] != want || a[fmt.Sprintf("smt_chain.%d.id", i)] != want {
							return fmt.Errorf("entry %d: key %q id %q, want both %q", i, a[fmt.Sprintf("smt_chain.%d.key", i)], a[fmt.Sprintf("smt_chain.%d.id", i)], want)
						}
					}
					if a["smt_chain.2.mask_field.fields_include.0"] != "email" || a["smt_chain.2.secret_version"] != "0" {
						return fmt.Errorf("import must adopt the server config and record no rotation: %v", a)
					}
					if _, has := a["smt_secrets.#"]; has {
						return fmt.Errorf("import must not invent secret entries")
					}
					return nil
				},
			},
		},
	})
}

// sourceFilterChain is the published source chain: a setting-less Filter
// that drops the records its TopicNameMatches predicate selects.
func sourceFilterChain(name string, negate bool) string {
	return fmt.Sprintf(`
  smt_chain = [
    { key = "drop", type = "filter", name = %q, filter = {}, predicate = { key = "audit", negate = %t } },
  ]
  smt_predicates = [
    { key = "audit", type = "topic_name_matches", topic_name_matches = { pattern = "^public\\.audit_.*$" } },
  ]`, name, negate)
}

// The published source types drive the lifecycle through the real
// source_postgresql resource, whose generated model carries the chain: the
// chain rides in the source request, Filter round-trips as an empty object,
// the re-plan is empty, and a rename plus a negate flip keeps both server
// identities under the last-read revision.
func TestSMT_CatalogTypesOnSourcePostgresql(t *testing.T) {
	// The fake validates predicates too, so it needs the source host's whole
	// catalog, not only the chain's transform types.
	corpus, err := smt.LoadCorpus(filepath.Join("..", "..", "generated", "smt_catalog.json"))
	require.NoError(t, err)
	fake := newFakeAPI(smt.NewCatalog(corpus).ForHost("sources"))
	p := &testProvider{client: fake, configs: []connector.ConnectorConfig{&source.PostgreSQLConfig{}}}
	factories := map[string]func() (tfprotov6.ProviderServer, error){"streamkap": providerserver.NewProtocol6WithError(p)}
	ids := map[string]string{}

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{
				Config: hcl(sourceFilterChain("Drop audit tables", false)),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(addr, "smt_chain.#", "1"),
					tfresource.TestCheckResourceAttr(addr, "smt_chain.0.type", "filter"),
					tfresource.TestCheckResourceAttr(addr, "smt_chain.0.filter.%", "0"),
					tfresource.TestCheckResourceAttr(addr, "smt_chain.0.predicate.key", "audit"),
					tfresource.TestCheckResourceAttr(addr, "smt_predicates.0.topic_name_matches.pattern", "^public\\.audit_.*$"),
					tfresource.TestCheckResourceAttrSet(addr, "smt_chain_revision"),
					func(s *terraform.State) error {
						ids["filter"], ids["predicate"], ids["revision"] = attrOf(s, "smt_chain.0.id"), attrOf(s, "smt_predicates.0.id"), attrOf(s, "smt_chain_revision")
						if ids["filter"] == "" || ids["predicate"] == "" {
							return fmt.Errorf("server identities missing: %v", ids)
						}
						if got := fake.recordedEvents(); !reflect.DeepEqual(got, []string{"connector-create", "chain-embedded"}) {
							return fmt.Errorf("the source chain must ride in the create, got %v", got)
						}
						return nil
					},
				),
			},
			{
				Config:   hcl(sourceFilterChain("Drop audit tables", false)),
				PlanOnly: true,
			},
			{
				Config: hcl(sourceFilterChain("Drop audit rows", true)),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
				}},
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(addr, "smt_chain.0.name", "Drop audit rows"),
					tfresource.TestCheckResourceAttr(addr, "smt_chain.0.predicate.negate", "true"),
					func(s *terraform.State) error {
						if attrOf(s, "smt_chain.0.id") != ids["filter"] || attrOf(s, "smt_predicates.0.id") != ids["predicate"] {
							return fmt.Errorf("the rename and negate flip must keep both server identities")
						}
						sent := fake.lastEmbeddedChain()
						if sent.ExpectedRevision == nil || *sent.ExpectedRevision != ids["revision"] {
							return fmt.Errorf("the update must carry the last-read revision %q", ids["revision"])
						}
						return nil
					},
				),
			},
			{
				Config:   hcl(sourceFilterChain("Drop audit rows", true)),
				PlanOnly: true,
			},
		},
	})
}
