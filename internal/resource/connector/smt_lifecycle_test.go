package connector_test

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/smt"
)

const addr = "streamkap_source_postgresql.t"

const smtTerraformMatrixVersionEnv = "SMT_TERRAFORM_MATRIX_VERSION"

type smtTerraformMatrixVersionCheck struct{}

type terraform1105OnlyCheck struct{}

func (smtTerraformMatrixVersionCheck) CheckTerraformVersion(_ context.Context, req tfversion.CheckTerraformVersionRequest, resp *tfversion.CheckTerraformVersionResponse) {
	expected := os.Getenv(smtTerraformMatrixVersionEnv)
	if expected == "" {
		return
	}
	actual := req.TerraformVersion.Core().String()
	if actual != expected {
		resp.Error = fmt.Errorf("SMT Terraform matrix expected CLI %s, selected %s", expected, actual)
		return
	}
	fmt.Fprintf(os.Stderr, "SMT Terraform matrix selected CLI %s (expected %s)\n", actual, expected)
}

func smtTerraformVersionChecks(checks ...tfversion.TerraformVersionCheck) []tfversion.TerraformVersionCheck {
	return append([]tfversion.TerraformVersionCheck{smtTerraformMatrixVersionCheck{}}, checks...)
}

func (terraform1105OnlyCheck) CheckTerraformVersion(_ context.Context, req tfversion.CheckTerraformVersionRequest, resp *tfversion.CheckTerraformVersionResponse) {
	if actual := req.TerraformVersion.Core().String(); actual != "1.10.5" {
		resp.Skip = fmt.Sprintf("Terraform CLI version %s is not 1.10.5: skipping compatibility test", req.TerraformVersion)
	}
}

// The released required attributes; everything else on the flat surface is
// left to its defaults exactly as a v3 user would.
const flatRequired = `
  name                = "chain"
  database_hostname   = "db.invalid"
  database_user       = "u"
  database_password   = "p"
  database_dbname     = "d"
  schema_include_list = "public"
  table_include_list  = "public.t"
`

func hcl(body string) string {
	return "resource \"streamkap_source_postgresql\" \"t\" {\n" + flatRequired + body + "\n}\n"
}

const chainAB = `
  smt_chain = [
    {
      key  = "a"
      type = "contract_fixture_nested"
      name = "East"
      contract_fixture_nested = {
        routes = [
          { key = "row-east", endpoint = "example.invalid" },
          { key = "row~west/1", endpoint = "other.invalid", label = "west" },
        ]
      }
    },
    {
      key  = "b"
      type = "contract_fixture_deep"
      name = "Deep"
      contract_fixture_deep = {
        services = [{ key = "svc-a", endpoints = [{ id = "ep/1" }] }]
      }
    },
  ]`

const chainBA = `
  smt_chain = [
    {
      key  = "b"
      type = "contract_fixture_deep"
      name = "Deep"
      contract_fixture_deep = {
        services = [{ key = "svc-a", endpoints = [{ id = "ep/1" }] }]
      }
    },
    {
      key  = "a"
      type = "contract_fixture_nested"
      name = "East renamed"
      contract_fixture_nested = {
        routes = [
          { key = "row-east", endpoint = "example.invalid" },
          { key = "row~west/1", endpoint = "other.invalid", label = "west" },
        ]
      }
    },
  ]`

func secrets(aVersion int, aEast, aWest string, bVersion int, bEp string) string {
	return fmt.Sprintf(`
  smt_secrets = [
    { key = "a", version = %d, values = [
      { pointer = "/routes/row-east/token", value = %q },
      { pointer = "/routes/row~0west~11/token", value = %q },
    ] },
    { key = "b", version = %d, values = [
      { pointer = "/services/svc-a/endpoints/ep~11/secret", value = %q },
    ] },
  ]`, aVersion, aEast, aWest, bVersion, bEp)
}

func attrOf(s *terraform.State, name string) string {
	return attrAt(s, addr, name)
}

func attrAt(s *terraform.State, address, name string) string {
	return s.RootModule().Resources[address].Primary.Attributes[name]
}

// destAddr and destHCL are the Snowflake destination counterparts of addr and
// hcl: a destination writes its chain through PUT .../smt-chain, a source
// embeds it in the source request.
const destAddr = "streamkap_destination_snowflake.t"

func destHCL(body string) string {
	return `
resource "streamkap_destination_snowflake" "t" {
  name                    = "chain"
  snowflake_url_name      = "acct.snowflakecomputing.invalid"
  snowflake_user_name     = "u"
  snowflake_private_key   = "k"
  snowflake_database_name = "D"
  snowflake_schema_name   = "S"
` + body + "\n}\n"
}

func secretsNullInPlanAndState() ([]plancheck.PlanCheck, []statecheck.StateCheck) {
	var plans []plancheck.PlanCheck
	var states []statecheck.StateCheck
	for i := 0; i < 2; i++ {
		p := tfjsonpath.New(smt.AttrSecrets).AtSliceIndex(i).AtMapKey("values")
		plans = append(plans, plancheck.ExpectKnownValue(addr, p, knownvalue.Null()))
		states = append(states, statecheck.ExpectKnownValue(addr, p, knownvalue.Null()))
	}
	return plans, states
}

// Real terraform drives the prototype end to end against the in-memory
// backend: chain create, correlation across reorder and rename, key change,
// write-only secret rotation, and import.
func TestSMT_ChainLifecycle(t *testing.T) {
	factories, fake := newFixture(t)
	ids := map[string]string{}
	planChecks, stateChecks := secretsNullInPlanAndState()
	opsBefore := 0

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   smtTerraformVersionChecks(tfversion.SkipBelow(tfversion.Version1_11_0)),
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config:            hcl(chainAB + secrets(1, "east-1", "west-1", 1, "ep-1")),
				ConfigPlanChecks:  resource.ConfigPlanChecks{PreApply: planChecks},
				ConfigStateChecks: stateChecks,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "smt_chain.#", "2"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.key", "a"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.name", "East"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.enabled", "true"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.schema_version", "1"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.contract_fixture_nested.retries", "0"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.contract_fixture_nested.enabled", "false"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.contract_fixture_nested.output.prefix", ""),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.contract_fixture_nested.labels.#", "0"),
					resource.TestCheckNoResourceAttr(addr, "smt_chain.0.contract_fixture_nested.fallback"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.contract_fixture_nested.routes.1.key", "row~west/1"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.contract_fixture_nested.routes.0.label", ""),
					resource.TestCheckNoResourceAttr(addr, "smt_chain.0.contract_fixture_deep.%"),
					resource.TestCheckResourceAttr(addr, "smt_chain.1.key", "b"),
					resource.TestCheckResourceAttr(addr, "smt_chain.1.contract_fixture_deep.services.0.auth.username", ""),
					resource.TestCheckResourceAttr(addr, "smt_chain.1.contract_fixture_deep.services.0.endpoints.0.id", "ep/1"),
					resource.TestCheckResourceAttr(addr, "smt_secrets.0.version", "1"),
					resource.TestCheckNoResourceAttr(addr, "smt_secrets.0.values.#"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.secret_version", "1"),
					resource.TestCheckResourceAttr(addr, "smt_chain.1.secret_version", "1"),
					func(s *terraform.State) error {
						ids["a"], ids["b"] = attrOf(s, "smt_chain.0.id"), attrOf(s, "smt_chain.1.id")
						ids["alias-a"] = attrOf(s, "smt_chain.0.alias")
						if ids["a"] == "" || ids["b"] == "" || ids["a"] == ids["b"] {
							return fmt.Errorf("server ids not recorded: %v", ids)
						}
						ops := fake.opsSince(0)
						if len(ops) != 3 {
							return fmt.Errorf("want 3 initial secret ops, got %v", ops)
						}
						want := map[string]string{
							ids["a"] + "|/routes/row-east/token":                 "east-1",
							ids["a"] + "|/routes/row~0west~11/token":             "west-1",
							ids["b"] + "|/services/svc-a/endpoints/ep~11/secret": "ep-1",
						}
						for _, op := range ops {
							if want[op.InstanceID+"|"+op.Pointer] != op.Value {
								return fmt.Errorf("unexpected op %+v", op)
							}
						}
						opsBefore = 3
						return nil
					},
				),
			},
			{
				// Same configuration, values still written: no operation.
				Config: hcl(chainAB + secrets(1, "east-1", "west-1", 1, "ep-1")),
				Check: func(s *terraform.State) error {
					if got := fake.opsSince(opsBefore); len(got) != 0 {
						return fmt.Errorf("unchanged version must send nothing, got %v", got)
					}
					return nil
				},
			},
			{
				// Edited value, same version: undetectable by design.
				Config: hcl(chainAB + secrets(1, "east-EDITED", "west-1", 1, "ep-1")),
				Check: func(s *terraform.State) error {
					if got := fake.opsSince(opsBefore); len(got) != 0 {
						return fmt.Errorf("an edit without a version bump must not rotate, got %v", got)
					}
					return nil
				},
			},
			{
				// Reorder, rename a, rotate a to version 2.
				Config: hcl(chainBA + secrets(2, "east-2", "west-2", 1, "ep-1")),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "smt_chain.0.key", "b"),
					resource.TestCheckResourceAttr(addr, "smt_chain.1.key", "a"),
					resource.TestCheckResourceAttr(addr, "smt_chain.1.name", "East renamed"),
					resource.TestCheckResourceAttr(addr, "smt_chain.1.secret_version", "2"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.secret_version", "1"),
					func(s *terraform.State) error {
						if attrOf(s, "smt_chain.1.id") != ids["a"] || attrOf(s, "smt_chain.0.id") != ids["b"] {
							return fmt.Errorf("ids did not follow their keys: a=%s b=%s", attrOf(s, "smt_chain.1.id"), attrOf(s, "smt_chain.0.id"))
						}
						if attrOf(s, "smt_chain.1.alias") != ids["alias-a"] {
							return fmt.Errorf("alias did not follow its key")
						}
						ops := fake.opsSince(opsBefore)
						if len(ops) != 2 || ops[0].InstanceID != ids["a"] || ops[1].InstanceID != ids["a"] || ops[0].Value != "east-2" || ops[1].Value != "west-2" {
							return fmt.Errorf("want exactly a's two rotated rows, got %v", ops)
						}
						opsBefore += 2
						return nil
					},
				),
			},
			{
				// Key change: a2 is a new instance, a's identity and secrets are not inherited.
				Config: hcl(strings.ReplaceAll(chainBA, `key  = "a"`, `key  = "a2"`) + strings.ReplaceAll(secrets(1, "east-3", "west-3", 1, "ep-1"), `key = "a"`, `key = "a2"`)),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "smt_chain.1.key", "a2"),
					resource.TestCheckResourceAttr(addr, "smt_chain.1.secret_version", "1"),
					func(s *terraform.State) error {
						if attrOf(s, "smt_chain.1.id") == ids["a"] {
							return fmt.Errorf("a changed key inherited the old server id")
						}
						if attrOf(s, "smt_chain.0.id") != ids["b"] {
							return fmt.Errorf("untouched key lost its id")
						}
						if len(fake.instances(attrOf(s, "id"))) != 2 {
							return fmt.Errorf("server chain should hold two instances")
						}
						ops := fake.opsSince(opsBefore)
						if len(ops) != 2 || ops[0].InstanceID != attrOf(s, "smt_chain.1.id") {
							return fmt.Errorf("new key must rotate under its own new id, got %v", ops)
						}
						opsBefore += 2
						return nil
					},
				),
			},
			{
				ResourceName:      addr,
				ImportState:       true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) { return attrOf(s, "id"), nil },
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("want one imported state, got %d", len(states))
					}
					a := states[0].Attributes
					if a["smt_chain.#"] != "2" {
						return fmt.Errorf("imported chain has %q entries", a["smt_chain.#"])
					}
					for i := 0; i < 2; i++ {
						id, key := a[fmt.Sprintf("smt_chain.%d.id", i)], a[fmt.Sprintf("smt_chain.%d.key", i)]
						if id == "" || key != id {
							return fmt.Errorf("entry %d: key %q must be derived from id %q", i, key, id)
						}
					}
					for name := range a {
						if strings.HasPrefix(name, "transforms_") || strings.HasPrefix(name, "predicates_") || strings.HasPrefix(name, "insert_static_") {
							return fmt.Errorf("import populated the flat surface too: %s", name)
						}
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

// Managed-slot overrides are destination-scoped on the backend, so the
// lifecycle runs on the destination.
func TestSMT_ManagedOverridesLifecycleIsIndependentFromUserChain(t *testing.T) {
	factories, fake := newFixture(t)
	checkBackend := func(want map[string]bool, transformCount int) resource.TestCheckFunc {
		return func(*terraform.State) error {
			ids := fake.sourceIDs()
			if len(ids) != 1 {
				return fmt.Errorf("got %d connector ids, want 1", len(ids))
			}
			if got := fake.managedOverrides(ids[0]); !reflect.DeepEqual(got, want) {
				return fmt.Errorf("managed overrides = %#v, want %#v", got, want)
			}
			if got := len(fake.instances(ids[0])); got != transformCount {
				return fmt.Errorf("user transform count = %d, want %d", got, transformCount)
			}
			return nil
		}
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   smtTerraformVersionChecks(tfversion.SkipBelow(tfversion.Version1_11_0)),
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: destHCL(`
  smt_managed_overrides = { "user_defined:MaskField" = false }
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(destAddr, "smt_managed_overrides.user_defined:MaskField", "false"),
					resource.TestCheckNoResourceAttr(destAddr, "smt_chain.#"),
					checkBackend(map[string]bool{"user_defined:MaskField": false}, 0),
				),
			},
			{
				Config: destHCL(chainAB),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(destAddr, "smt_chain.#", "2"),
					checkBackend(map[string]bool{"user_defined:MaskField": false}, 2),
				),
			},
			{
				Config: destHCL(chainAB + `
  smt_managed_overrides = {}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(destAddr, "smt_managed_overrides.%", "0"),
					checkBackend(map[string]bool{}, 2),
				),
			},
		},
	})
}

// A source chain accepts no managed-slot override, so a configured override
// fails validation before any request; an empty map is accepted.
func TestSMT_SourceRefusesManagedOverrides(t *testing.T) {
	factories, fake := newFixture(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: hcl(chainAB + `
  smt_managed_overrides = { "user_defined:MaskField" = false }
`),
				ExpectError: regexp.MustCompile(`Managed overrides apply to destinations only`),
			},
			{
				PreConfig: func() {
					if got := fake.writeCount(); got != 0 {
						t.Fatalf("a refused override reached the API: %d writes", got)
					}
				},
				Config: hcl(chainAB + `
  smt_managed_overrides = {}
`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "smt_chain.#", "2"),
					resource.TestCheckResourceAttr(addr, "smt_managed_overrides.%", "0"),
				),
			},
		},
	})
}

// Every rejected rotation and the type change fail at plan time: the
// backend sees no write.
func TestSMT_RejectionsHappenBeforeAnyWrite(t *testing.T) {
	factories, fake := newFixture(t)
	writesAfterCreate := 0

	rejected := func(config, want string) resource.TestStep {
		return resource.TestStep{
			Config:      hcl(config),
			ExpectError: regexp.MustCompile(want),
		}
	}
	noNewWrites := func(s *terraform.State) error {
		if got := fake.writeCount(); got != writesAfterCreate {
			return fmt.Errorf("a rejected plan reached the backend: %d writes, expected %d", got, writesAfterCreate)
		}
		return nil
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   smtTerraformVersionChecks(tfversion.SkipBelow(tfversion.Version1_11_0)),
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: hcl(chainAB + secrets(2, "e", "w", 2, "p")),
				Check: func(s *terraform.State) error {
					writesAfterCreate = fake.writeCount()
					return nil
				},
			},
			rejected(chainAB+secrets(1, "e", "w", 2, "p"), "Rotation version decreased"),
			rejected(chainAB+`
  smt_secrets = [{ key = "a", version = 3 }, { key = "b", version = 2 }]`, "Rotation without operations"),
			rejected(chainAB+`
  smt_secrets = [{ key = "ghost", version = 1, values = [{ pointer = "/routes/row-east/token", value = "x" }] }]`, "Secret entry for unknown instance"),
			rejected(chainAB+`
  smt_secrets = [{ key = "a", version = 3, values = [{ pointer = "/routes/1/token", value = "x" }] }]`, "Secret pointer names a missing row"),
			rejected(chainAB+`
  smt_secrets = [{ key = "a", version = 3, values = [{ pointer = "/routes/row-east/token", value = "x" }] }, { key = "b", version = 0 }]`, "must be at least 1"),
			rejected(strings.Replace(chainAB, `type = "contract_fixture_deep"`, `type = "contract_fixture_nested"`, 1), "Transform type change is not allowed in place"),
			rejected(chainAB+`
  transforms_value_to_key_fields_include_list = "id"`, "Flat and nested transform attributes cannot be mixed"),
			{
				Config: hcl(chainAB + secrets(2, "e", "w", 2, "p")),
				Check:  noNewWrites,
			},
		},
	})
}

// Terraform 1.10 can use the provider while the write-only input is absent,
// but the framework must reject a configured value before any API write.
func TestTerraform1105ConfiguredSMTSecretsRefused(t *testing.T) {
	factories, fake := newFixture(t)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks: smtTerraformVersionChecks(
			terraform1105OnlyCheck{},
		),
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: hcl(chainAB + secrets(1, "east", "west", 1, "endpoint")),
				ExpectError: regexp.MustCompile(
					`WriteOnly Attribute Not Allowed[\s\S]*Write-only attributes are only supported in Terraform\s+1\.11 and later\.`,
				),
			},
		},
	})

	require.Zero(t, fake.writeCount(), "an unsupported write-only value reached the API")
}

// A released v3 flat configuration plans and converges with the chain
// attributes present in the schema and never touched: the flat keys still
// reach the API, the chain endpoints are never called, and a refresh that
// changes a flat attribute on the server is still visible as drift.
func TestSMT_FlatSurfaceUnchanged(t *testing.T) {
	factories, fake := newFixture(t)
	flat := `
  transforms_value_to_key_fields_include_list = "id"
  transforms_oversized_records_max_field_size_bytes = 2097152
  insert_static_key_field_1 = "tenant"`

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   smtTerraformVersionChecks(),
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: hcl(flat),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "transforms_value_to_key_fields_include_list", "id"),
					resource.TestCheckResourceAttr(addr, "transforms_oversized_records_max_field_size_bytes", "2097152"),
					resource.TestCheckResourceAttr(addr, "insert_static_key_field_1", "tenant"),
					resource.TestCheckNoResourceAttr(addr, "smt_chain.#"),
					resource.TestCheckNoResourceAttr(addr, "smt_secrets.#"),
					resource.TestCheckNoResourceAttr(addr, "smt_chain_revision"),
					func(s *terraform.State) error {
						sent := fake.flatKeysSent(attrOf(s, "id"))
						for _, key := range []string{"transforms.ValueToKey.fields.include.list", "transforms.OversizedRecords.max.field.size.bytes", "transforms.InsertStaticKey1.static.field", "transforms.SourceRegexSupport.regex.replacement"} {
							if !sent[key] {
								return fmt.Errorf("flat key %s did not reach the API on the released surface: %v", key, sent)
							}
						}
						if fake.chainReadCount() != 0 {
							return fmt.Errorf("the chain endpoint was read for a connector on the released surface")
						}
						return nil
					},
				),
			},
			{
				Config:   hcl(flat),
				PlanOnly: true,
			},
			{
				// A flat attribute edited on the server is drift, exactly as
				// released: the chain-aware Read leaves the flat surface alone.
				PreConfig: func() { fake.setFlat("transforms.ValueToKey.fields.include.list", "edited-in-ui") },
				Config:    hcl(flat),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
				}},
			},
		},
	})
}

// After import the chain is adopted under keys derived from the server ids.
// HCL that uses those keys keeps every instance in place: the plan pins the
// imported ids, the apply rotates nothing, and the plan after it is empty.
func TestSMT_ImportAdoption(t *testing.T) {
	factories, fake := newFixture(t)
	// A connector created outside Terraform, the way the UI would: the fake
	// mints conn-1 with instances inst-2 and inst-3, and holds a secret.
	connID := fake.seedChain(t, []api.SMTInstanceWrite{
		{Type: "contract_fixture_nested", Name: "East", Enabled: true, Config: func() map[string]any {
			spec, _ := catalogSpec(fake.cat, "transform", "contract_fixture_nested")
			return spec.Examples.ReadProjection
		}()},
		{Type: "contract_fixture_deep", Name: "Deep", Enabled: true, Config: map[string]any{"services": []any{map[string]any{
			"key": "svc-a", "auth": map[string]any{"username": ""},
			"endpoints": []any{map[string]any{"id": "ep/1", "url": ""}},
		}}}},
	})
	seeded := fake.instances(connID)
	require.Equal(t, []string{"inst-2", "inst-3"}, []string{seeded[0].ID, seeded[1].ID})
	fake.applySecret(t, connID, "inst-2", "/routes/row-east/token", "e")
	fake.setManagedOverrides(connID, map[string]bool{"user_defined:MaskField": false})

	adopted := `
  smt_chain = [
    {
      key  = "inst-2"
      type = "contract_fixture_nested"
      name = "East"
      contract_fixture_nested = {
        routes = [
          { key = "row-east", endpoint = "example.invalid", label = "east" },
          { key = "row~west/1", endpoint = "other.invalid", label = "west" },
        ]
      }
    },
    {
      key  = "inst-3"
      type = "contract_fixture_deep"
      name = "Deep"
      contract_fixture_deep = {
        services = [{ key = "svc-a", endpoints = [{ id = "ep/1" }] }]
      }
    },
  ]`
	chainPath := tfjsonpath.New(smt.AttrChain)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config:             hcl(adopted),
				ResourceName:       addr,
				ImportState:        true,
				ImportStateId:      connID,
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					a := states[0].Attributes
					if a["smt_chain.0.key"] != "inst-2" || a["smt_chain.1.key"] != "inst-3" {
						return fmt.Errorf("derived keys: %q %q", a["smt_chain.0.key"], a["smt_chain.1.key"])
					}
					if a["smt_managed_overrides.user_defined:MaskField"] != "false" {
						return fmt.Errorf("imported overrides: %#v", a["smt_managed_overrides.user_defined:MaskField"])
					}
					return nil
				},
			},
			{
				// The flat SMT attributes are pinned to null on import, so
				// their client-side defaults make this an Update; the chain
				// itself must plan unchanged.
				Config: hcl(adopted),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					plancheck.ExpectKnownValue(addr, chainPath.AtSliceIndex(0).AtMapKey("id"), knownvalue.StringExact("inst-2")),
					plancheck.ExpectKnownValue(addr, chainPath.AtSliceIndex(1).AtMapKey("id"), knownvalue.StringExact("inst-3")),
				}},
				Check: func(s *terraform.State) error {
					if got := fake.opsSince(1); len(got) != 0 {
						return fmt.Errorf("adoption must not rotate secrets, got %v", got)
					}
					if fake.secretValue("inst-2", "/routes/row-east/token") != "e" {
						return fmt.Errorf("server secret was disturbed")
					}
					return nil
				},
			},
			{
				Config:   hcl(adopted),
				PlanOnly: true,
			},
		},
	})
}

// The last applied rotation version lives on the chain entry, not on the
// smt_secrets entry: removing the secret entry and adding it back cannot
// restart a live instance below the version it already reached.
func TestSMT_RotationVersionFollowsTheKey(t *testing.T) {
	factories, fake := newFixture(t)
	aOnly := func(version int, value string) string {
		return fmt.Sprintf(`
  smt_secrets = [{ key = "a", version = %d, values = [{ pointer = "/routes/row-east/token", value = %q }] }]`, version, value)
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   smtTerraformVersionChecks(tfversion.SkipBelow(tfversion.Version1_11_0)),
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: hcl(chainAB + aOnly(5, "v5")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "smt_chain.0.secret_version", "5"),
					resource.TestCheckResourceAttr(addr, "smt_chain.1.secret_version", "0"),
					func(s *terraform.State) error {
						if got := fake.opsSince(0); len(got) != 1 || got[0].Value != "v5" {
							return fmt.Errorf("want one op at version 5, got %v", got)
						}
						return nil
					},
				),
			},
			{
				// The secret entry goes away; the instance and its version stay.
				Config: hcl(chainAB),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(addr, "smt_secrets.#"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.secret_version", "5"),
					func(s *terraform.State) error {
						if got := fake.opsSince(1); len(got) != 0 {
							return fmt.Errorf("removing the entry must not rotate, got %v", got)
						}
						return nil
					},
				),
			},
			{
				Config:      hcl(chainAB + aOnly(1, "v1")),
				ExpectError: regexp.MustCompile(`Rotation version decreased[\s\S]*below the 5 last applied to "a"`),
			},
			{
				// Re-adding at the recorded version is a no-op, not a rotation.
				Config: hcl(chainAB + aOnly(5, "v5-again")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "smt_chain.0.secret_version", "5"),
					func(s *terraform.State) error {
						if got := fake.opsSince(1); len(got) != 0 {
							return fmt.Errorf("re-adding at the recorded version must not rotate, got %v", got)
						}
						return nil
					},
				),
			},
			{
				Config: hcl(chainAB + aOnly(6, "v6")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "smt_chain.0.secret_version", "6"),
					func(s *terraform.State) error {
						if got := fake.opsSince(1); len(got) != 1 || got[0].Value != "v6" {
							return fmt.Errorf("want exactly one op at version 6, got %v", got)
						}
						return nil
					},
				),
			},
		},
	})
}

// Re-applying an unchanged destination chain is a no-op on the provider side:
// the refresh after the create reads the chain back (omitted fields at their
// defaults, keyed rows, write-only secret values absent, a managed override)
// and must plan nothing, and an apply that changes only a connector attribute
// updates the connector without a chain write.
func TestSMT_UnchangedDestinationChainSendsNoChainWrite(t *testing.T) {
	factories, fake := newFixture(t)
	config := destHCL(chainAB + secrets(1, "east-1", "west-1", 1, "ep-1") + `
  smt_managed_overrides = { "user_defined:MaskField" = false }
`)
	var revision string
	requests := func(want ...string) resource.TestCheckFunc {
		return func(*terraform.State) error {
			if got := fake.recordedEvents(); !reflect.DeepEqual(got, want) {
				return fmt.Errorf("requests = %v, want %v", got, want)
			}
			if got := fake.opsSince(0); len(got) != 3 {
				return fmt.Errorf("want only the create's 3 secret operations, got %v", got)
			}
			return nil
		}
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   smtTerraformVersionChecks(tfversion.SkipBelow(tfversion.Version1_11_0)),
		ProtoV6ProviderFactories: factories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					requests("connector-create", "chain-put"),
					func(s *terraform.State) error {
						revision = attrAt(s, destAddr, smt.AttrRevision)
						return nil
					},
				),
			},
			{
				PreConfig:        fake.resetEvents,
				Config:           config,
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
				Check: resource.ComposeTestCheckFunc(
					requests(),
					resource.TestCheckResourceAttrPtr(destAddr, smt.AttrRevision, &revision),
				),
			},
			{
				PreConfig: fake.resetEvents,
				Config:    strings.Replace(config, `name                    = "chain"`, `name                    = "chain renamed"`, 1),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(destAddr, plancheck.ResourceActionUpdate),
				}},
				Check: resource.ComposeTestCheckFunc(
					requests("connector-update"),
					resource.TestCheckResourceAttrPtr(destAddr, smt.AttrRevision, &revision),
				),
			},
		},
	})
}
