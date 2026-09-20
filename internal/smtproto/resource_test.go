package smtproto

import (
	"context"
	"fmt"
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
)

const addr = "streamkap_smtproto_source_postgresql.t"

// The released required attributes; everything else on the flat surface is
// left to its defaults exactly as a v3 user would.
const flatRequired = `
  name                = "smtproto"
  database_hostname   = "db.invalid"
  database_user       = "u"
  database_password   = "p"
  database_dbname     = "d"
  schema_include_list = "public"
  table_include_list  = "public.t"
`

func hcl(body string) string {
	return "resource \"streamkap_smtproto_source_postgresql\" \"t\" {\n" + flatRequired + body + "\n}\n"
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

func newProto(t *testing.T) (*Resource, *fakeBackend) {
	t.Helper()
	_, cat := loadCorpus(t)
	fake := newFakeBackend(cat)
	res, err := NewResource(fake, cat)
	require.NoError(t, err)
	return res, fake
}

func attrOf(s *terraform.State, name string) string {
	return s.RootModule().Resources[addr].Primary.Attributes[name]
}

func secretsNullInPlanAndState() ([]plancheck.PlanCheck, []statecheck.StateCheck) {
	var plans []plancheck.PlanCheck
	var states []statecheck.StateCheck
	for i := 0; i < 2; i++ {
		p := tfjsonpath.New(AttrSecrets).AtSliceIndex(i).AtMapKey(attrValues)
		plans = append(plans, plancheck.ExpectKnownValue(addr, p, knownvalue.Null()))
		states = append(states, statecheck.ExpectKnownValue(addr, p, knownvalue.Null()))
	}
	return plans, states
}

// Real terraform drives the prototype end to end against the in-memory
// backend: chain create, correlation across reorder and rename, key change,
// write-only secret rotation, and import.
func TestResource_ChainLifecycle(t *testing.T) {
	res, fake := newProto(t)
	ids := map[string]string{}
	planChecks, stateChecks := secretsNullInPlanAndState()
	opsBefore := 0

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		ProtoV6ProviderFactories: providerFactories(res),
		Steps: []resource.TestStep{
			{
				Config:            hcl(chainAB + secrets(1, "east-1", "west-1", 1, "ep-1")),
				ConfigPlanChecks:  resource.ConfigPlanChecks{PreApply: planChecks},
				ConfigStateChecks: stateChecks,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "smt_chain.#", "2"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.key", "a"),
					resource.TestCheckResourceAttr(addr, "smt_chain.0.name", "East"),
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
						fake.mu.Lock()
						defer fake.mu.Unlock()
						if len(fake.connectors[attrOf(s, "id")]) != 2 {
							return fmt.Errorf("server chain should hold two instances")
						}
						ops := fake.ops[opsBefore:]
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

// Every rejected rotation and the type change fail at plan time: the
// backend sees no write.
func TestResource_RejectionsHappenBeforeAnyWrite(t *testing.T) {
	res, fake := newProto(t)
	writesAfterCreate := 0

	rejected := func(config, want string) resource.TestStep {
		return resource.TestStep{
			Config:      hcl(config),
			ExpectError: regexp.MustCompile(want),
		}
	}
	noNewWrites := func(s *terraform.State) error {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		if fake.writes != writesAfterCreate {
			return fmt.Errorf("a rejected plan reached the backend: %d writes, expected %d", fake.writes, writesAfterCreate)
		}
		return nil
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		ProtoV6ProviderFactories: providerFactories(res),
		Steps: []resource.TestStep{
			{
				Config: hcl(chainAB + secrets(2, "e", "w", 2, "p")),
				Check: func(s *terraform.State) error {
					fake.mu.Lock()
					defer fake.mu.Unlock()
					writesAfterCreate = fake.writes
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

// A released v3 flat configuration plans and converges with the chain
// attributes present in the schema and never touched.
func TestResource_FlatSurfaceUnchanged(t *testing.T) {
	res, _ := newProto(t)
	flat := `
  transforms_value_to_key_fields_include_list = "id"
  transforms_oversized_records_max_field_size_bytes = 2097152
  insert_static_key_field_1 = "tenant"`

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		ProtoV6ProviderFactories: providerFactories(res),
		Steps: []resource.TestStep{
			{
				Config: hcl(flat),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "transforms_value_to_key_fields_include_list", "id"),
					resource.TestCheckResourceAttr(addr, "transforms_oversized_records_max_field_size_bytes", "2097152"),
					resource.TestCheckResourceAttr(addr, "insert_static_key_field_1", "tenant"),
					resource.TestCheckNoResourceAttr(addr, "smt_chain.#"),
					resource.TestCheckNoResourceAttr(addr, "smt_secrets.#"),
				),
			},
			{
				Config:   hcl(flat),
				PlanOnly: true,
			},
		},
	})
}

// After import the chain is adopted under keys derived from the server ids.
// HCL that uses those keys keeps every instance in place: the plan pins the
// imported ids, the apply rotates nothing, and the plan after it is empty.
func TestResource_ImportAdoption(t *testing.T) {
	res, fake := newProto(t)
	// A connector created outside Terraform, the way the UI would: the fake
	// mints conn-1 with instances inst-2 and inst-3, and holds a secret.
	east := "East"
	_, reads, err := fake.Create(context.Background(), []InstanceWrite{
		{Type: "contract_fixture_nested", Name: &east, Config: fake.cat["contract_fixture_nested"].Examples.ReadProjection},
		{Type: "contract_fixture_deep", Config: map[string]any{"services": []any{map[string]any{
			"key": "svc-a", "auth": map[string]any{"username": ""},
			"endpoints": []any{map[string]any{"id": "ep/1", "url": ""}},
		}}}},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"inst-2", "inst-3"}, []string{reads[0].ID, reads[1].ID})
	require.NoError(t, fake.ApplySecrets(context.Background(), "conn-1", []ResolvedSecretOp{{InstanceID: "inst-2", Pointer: "/routes/row-east/token", Value: "e"}}))

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
      contract_fixture_deep = {
        services = [{ key = "svc-a", endpoints = [{ id = "ep/1" }] }]
      }
    },
  ]`
	chainPath := tfjsonpath.New(AttrChain)

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		ProtoV6ProviderFactories: providerFactories(res),
		Steps: []resource.TestStep{
			{
				Config:             hcl(adopted),
				ResourceName:       addr,
				ImportState:        true,
				ImportStateId:      "conn-1",
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					a := states[0].Attributes
					if a["smt_chain.0.key"] != "inst-2" || a["smt_chain.1.key"] != "inst-3" {
						return fmt.Errorf("derived keys: %q %q", a["smt_chain.0.key"], a["smt_chain.1.key"])
					}
					return nil
				},
			},
			{
				// The flat required attributes are null after import because
				// the prototype does not read them back, so this is an Update;
				// the chain itself must plan unchanged.
				Config: hcl(adopted),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					plancheck.ExpectKnownValue(addr, chainPath.AtSliceIndex(0).AtMapKey(attrID), knownvalue.StringExact("inst-2")),
					plancheck.ExpectKnownValue(addr, chainPath.AtSliceIndex(1).AtMapKey(attrID), knownvalue.StringExact("inst-3")),
				}},
				Check: func(s *terraform.State) error {
					if got := fake.opsSince(1); len(got) != 0 {
						return fmt.Errorf("adoption must not rotate secrets, got %v", got)
					}
					fake.mu.Lock()
					defer fake.mu.Unlock()
					if fake.secrets["inst-2"]["/routes/row-east/token"] != "e" {
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
func TestResource_RotationVersionFollowsTheKey(t *testing.T) {
	res, fake := newProto(t)
	aOnly := func(version int, value string) string {
		return fmt.Sprintf(`
  smt_secrets = [{ key = "a", version = %d, values = [{ pointer = "/routes/row-east/token", value = %q }] }]`, version, value)
	}

	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		ProtoV6ProviderFactories: providerFactories(res),
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
