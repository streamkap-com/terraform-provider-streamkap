package connector_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/destination"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/smt"
)

const typeName = "streamkap_source_postgresql"

// smtAttributes are the graph attributes the chain adds on top of the
// released v3 schema.
var smtAttributes = map[string]bool{smt.AttrChain: true, smt.AttrSecrets: true, smt.AttrPredicates: true, smt.AttrPredicateSecrets: true, smt.AttrManagedOverrides: true, smt.AttrRevision: true}

// releasedState encodes a state as the released v3 provider stores it: the
// released attribute set with every Default-bearing attribute holding its
// resolved default (a real v3 state always does, the flat SMT attributes
// included), the configured values on top, everything else null, and no
// chain attributes at all. It returns the `attributes` object of a state
// file, the attribute names that hold a value, and the released attributes.
func releasedState(t *testing.T, factory func() resource.Resource, cfg connector.ConnectorConfig, configured map[string]any) ([]byte, []string, map[string]schema.Attribute) {
	t.Helper()
	ctx := context.Background()
	current := &resource.SchemaResponse{}
	factory().Schema(ctx, resource.SchemaRequest{}, current)
	require.False(t, current.Diagnostics.HasError())

	released := map[string]schema.Attribute{}
	for name, a := range current.Schema.Attributes {
		if !smtAttributes[name] {
			released[name] = a
		}
	}
	// The released snapshot is the attribute set of record; the integrated
	// schema must be exactly that set plus the graph attributes.
	snapshotRaw, err := os.ReadFile(filepath.Join("..", "..", "provider", "testdata", "schemas", cfg.GetResourceName()+"_v1.json"))
	require.NoError(t, err)
	var snapshot struct {
		Attributes map[string]json.RawMessage `json:"attributes"`
	}
	require.NoError(t, json.Unmarshal(snapshotRaw, &snapshot))
	var snapshotNames, releasedNames []string
	for name := range snapshot.Attributes {
		if !smtAttributes[name] {
			snapshotNames = append(snapshotNames, name)
		}
	}
	for name := range released {
		releasedNames = append(releasedNames, name)
	}
	sort.Strings(snapshotNames)
	sort.Strings(releasedNames)
	require.Equal(t, snapshotNames, releasedNames, "the integrated schema must be the released attribute set plus the chain attributes")

	// Defaults come from the generated schema: the base strips the Default of
	// a canonical attribute that has a deprecated alias and plans it in
	// ModifyPlan instead, so the runtime schema under-reports what a v3
	// state holds.
	// A deprecated alias holds its canonical attribute's default for the same
	// reason.
	generatedSchema := cfg.GetSchema()
	mappings := cfg.GetFieldMappings()
	defaultOf := func(name string) (attr.Value, bool) {
		if v, ok := smt.DefaultValue(generatedSchema.Attributes[name]); ok {
			return v, true
		}
		if generatedSchema.Attributes[name].GetDeprecationMessage() == "" {
			return nil, false
		}
		for canonical, apiField := range mappings {
			if canonical != name && apiField == mappings[name] {
				return smt.DefaultValue(generatedSchema.Attributes[canonical])
			}
		}
		return nil, false
	}
	values := map[string]any{"timeouts": nil}
	var populated []string
	for name := range released {
		values[name] = nil
		if v, ok := defaultOf(name); ok {
			wire, err := smt.ToWire(v, path.Root(name))
			require.NoError(t, err)
			values[name] = wire
			populated = append(populated, name)
		}
	}
	for name, v := range configured {
		_, known := released[name]
		require.True(t, known, "%s is not a released attribute", name)
		if values[name] == nil {
			populated = append(populated, name)
		}
		values[name] = v
	}
	sort.Strings(populated)
	raw, err := json.Marshal(values)
	require.NoError(t, err)
	return raw, populated, released
}

// A state written by the released provider decodes against the integrated
// schema with the three chain attributes null, refreshes without populating
// the chain, and plans no change to any attribute when the configuration is
// the one that produced it.
//
// This is the framework in isolation: ProposedNewState is built here the
// way core builds it (config where set, prior state for computed attributes
// otherwise). The same convergence under the real CLI is
// TestSMT_FlatSurfaceUnchanged.
func TestSMT_ReleasedStateDecodesAndConverges(t *testing.T) {
	ctx := context.Background()
	factories, fake := newFixture(t)
	srv, err := factories["streamkap"]()
	require.NoError(t, err)

	// What the practitioner wrote: the required attributes, two flat SMT
	// values that have no default, and two that override a default.
	configured := map[string]any{
		"name":                "legacy",
		"database_hostname":   "db.invalid",
		"database_user":       "u",
		"database_password":   "p",
		"database_dbname":     "d",
		"schema_include_list": "public",
		"table_include_list":  "public.t",
		"transforms_value_to_key_fields_include_list":           "id",
		"insert_static_key_field_1":                             "tenant",
		"transforms_oversized_records_max_field_size_bytes":     2097152,
		"transforms_oversized_records_oversized_field_behavior": "NULLIFY",
	}
	created, err := fake.CreateSource(ctx, api.Source{Name: "legacy", Connector: "postgresql", Config: map[string]any{}})
	require.NoError(t, err)
	// The released Create fills both the deprecated alias and its canonical
	// attribute from the API echo, so a real v3 state holds both.
	computedOnly := map[string]any{"id": created.ID, "connector": "postgresql", "connector_status": "Active", "kc_cluster_id": "", "transforms_insert_static_key1_static_field": "tenant"}
	stored := map[string]any{}
	for k, v := range configured {
		stored[k] = v
	}
	for k, v := range computedOnly {
		stored[k] = v
	}
	factory := func() resource.Resource { return fixtureResource(t) }
	raw, populated, releasedAttrs := releasedState(t, factory, &source.PostgreSQLConfig{}, stored)
	require.Greater(t, len(populated), 30, "the fixture must carry the resolved defaults: %v", populated)
	require.Contains(t, populated, "transforms_source_regex_support_regex_replacement")
	require.Contains(t, populated, "transforms_oversized_records_replace_null_with_default")
	assertReleasedStateConverges(t, srv, fake, typeName, fixtureMappings(t), created.ID, raw, populated, releasedAttrs, configured)
}

// The released-state check on the real generated resources of each family
// that gained the chain: their generated model now carries the chain fields,
// and computed attributes without a value (MongoDB's derived
// mongodb_connection_hostname, S3's) must stay null rather than plan unknown.
func TestSMT_ReleasedStateConvergesPerFamily(t *testing.T) {
	for _, cfg := range []connector.ConnectorConfig{&source.MongoDBConfig{}, &source.MySQLConfig{}, &destination.S3Config{}, &destination.ClickHouseConfig{}} {
		t.Run(cfg.GetResourceName(), func(t *testing.T) {
			ctx := context.Background()
			fake := newFakeAPI(generated.SMTChain(string(cfg.GetConnectorType()) + "s").Catalog)
			p := &testProvider{client: fake, configs: []connector.ConnectorConfig{cfg}}
			srv := providerserver.NewProtocol6(p)()
			current := &resource.SchemaResponse{}
			connector.NewBaseConnectorResource(cfg).Schema(ctx, resource.SchemaRequest{}, current)
			require.False(t, current.Diagnostics.HasError())
			configured := map[string]any{}
			for name, a := range current.Schema.Attributes {
				if a.IsRequired() {
					_, isString := a.(schema.StringAttribute)
					require.True(t, isString, "required attribute %s needs a placeholder", name)
					configured[name] = "x"
				}
			}
			configured["name"] = "legacy"
			created, err := fake.CreateSource(ctx, api.Source{Name: "legacy", Connector: cfg.GetConnectorCode(), Config: map[string]any{}})
			require.NoError(t, err)
			if cfg.GetConnectorType() == connector.ConnectorTypeDestination {
				fake.kinds[created.ID] = "destinations"
			}
			stored := maps.Clone(configured)
			for k, v := range map[string]any{"id": created.ID, "connector": cfg.GetConnectorCode(), "connector_status": "Active", "kc_cluster_id": ""} {
				stored[k] = v
			}
			factory := func() resource.Resource { return connector.NewBaseConnectorResource(cfg) }
			raw, populated, releasedAttrs := releasedState(t, factory, cfg, stored)
			assertReleasedStateConverges(t, srv, fake, "streamkap_"+cfg.GetResourceName(), cfg.GetFieldMappings(), created.ID, raw, populated, releasedAttrs, configured)
		})
	}
}

// assertReleasedStateConverges stores what the released Create sent on the
// server, decodes the released state with the chain attributes null, refreshes
// it unchanged without reading a chain, and plans no change on any attribute.
func assertReleasedStateConverges(t *testing.T, srv tfprotov6.ProviderServer, fake *fakeAPI, typeName string, mappings map[string]string, id string, raw []byte, populated []string, releasedAttrs map[string]schema.Attribute, configured map[string]any) {
	t.Helper()
	ctx := context.Background()
	// The server holds what the released Create sent: every populated
	// attribute under its API key.
	var fixture map[string]any
	require.NoError(t, json.Unmarshal(raw, &fixture))
	wire := map[string]any{}
	for name, apiField := range mappings {
		if v := fixture[name]; v != nil {
			wire[apiField] = v
		}
	}
	fake.replaceConfig(id, wire)

	schemaResp, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	require.NoError(t, err)
	require.Empty(t, schemaResp.Diagnostics)
	objType := schemaResp.ResourceSchemas[typeName].ValueType().(tftypes.Object)

	upgraded, err := srv.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: typeName,
		Version:  0,
		RawState: &tfprotov6.RawState{JSON: raw},
	})
	require.NoError(t, err)
	require.Empty(t, upgraded.Diagnostics)
	state, err := upgraded.UpgradedState.Unmarshal(objType)
	require.NoError(t, err)

	var attrs map[string]tftypes.Value
	require.NoError(t, state.As(&attrs))
	for name := range smtAttributes {
		require.True(t, attrs[name].IsNull(), "released state has no %s: %s", name, attrs[name])
	}
	for _, name := range populated {
		require.False(t, attrs[name].IsNull(), "%s was lost in decode", name)
	}

	_, err = srv.ConfigureProvider(ctx, &tfprotov6.ConfigureProviderRequest{})
	require.NoError(t, err)
	read, err := srv.ReadResource(ctx, &tfprotov6.ReadResourceRequest{TypeName: typeName, CurrentState: upgraded.UpgradedState})
	require.NoError(t, err)
	require.Empty(t, read.Diagnostics)
	after, err := read.NewState.Unmarshal(objType)
	require.NoError(t, err)
	var afterAttrs map[string]tftypes.Value
	require.NoError(t, after.As(&afterAttrs))
	var refreshed []string
	for name := range objType.AttributeTypes {
		if !attrs[name].Equal(afterAttrs[name]) {
			refreshed = append(refreshed, fmt.Sprintf("%s: %s -> %s", name, attrs[name], afterAttrs[name]))
		}
	}
	sort.Strings(refreshed)
	require.Empty(t, refreshed, "refresh must leave a released state untouched, chain included")
	require.Equal(t, 0, fake.chainReadCount(), "a null chain is never read")

	// Core's proposed new state: the configured value where one is set,
	// the prior value for a computed attribute, null otherwise.
	configAttrs := map[string]tftypes.Value{}
	proposedAttrs := map[string]tftypes.Value{}
	for name, typ := range objType.AttributeTypes {
		configAttrs[name] = tftypes.NewValue(typ, nil)
		proposedAttrs[name] = tftypes.NewValue(typ, nil)
		if _, set := configured[name]; set {
			configAttrs[name] = attrs[name]
			proposedAttrs[name] = attrs[name]
			continue
		}
		if a, ok := releasedAttrs[name]; ok && a.IsComputed() {
			proposedAttrs[name] = attrs[name]
		}
		if name == smt.AttrRevision {
			proposedAttrs[name] = attrs[name]
		}
	}
	config, err := tfprotov6.NewDynamicValue(objType, tftypes.NewValue(objType, configAttrs))
	require.NoError(t, err)
	proposed, err := tfprotov6.NewDynamicValue(objType, tftypes.NewValue(objType, proposedAttrs))
	require.NoError(t, err)
	plan, err := srv.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
		TypeName:         typeName,
		PriorState:       read.NewState,
		ProposedNewState: &proposed,
		Config:           &config,
	})
	require.NoError(t, err)
	require.Empty(t, plan.Diagnostics)
	planned, err := plan.PlannedState.Unmarshal(objType)
	require.NoError(t, err)
	var plannedAttrs map[string]tftypes.Value
	require.NoError(t, planned.As(&plannedAttrs))
	for name := range objType.AttributeTypes {
		if !attrs[name].Equal(plannedAttrs[name]) {
			t.Errorf("%s moved: %s -> %s", name, attrs[name], plannedAttrs[name])
		}
	}
}

// With the chain configured the flat projection drops every transforms.* and
// predicates.* key on create and update, and a backend that echoes the chain
// through a legacy flat key never gets that value into state, on apply or on
// refresh: the chain is the only owning surface.
func TestSMT_FlatProjectionStrippedAndPinned(t *testing.T) {
	factories, fake := newFixture(t)
	fake.projectFlat = true
	flatUnchanged := tfresource.ComposeTestCheckFunc(
		tfresource.TestCheckNoResourceAttr(addr, "transforms_value_to_key_fields_include_list"),
		tfresource.TestCheckNoResourceAttr(addr, "insert_static_key_field_1"),
		tfresource.TestCheckResourceAttr(addr, "transforms_source_regex_support_regex_replacement", "_REGEX_"),
		tfresource.TestCheckResourceAttr(addr, "transforms_oversized_records_replace_null_with_default", "true"),
		func(s *terraform.State) error {
			if sent := fake.flatKeysSent(attrOf(s, "id")); len(sent) != 0 {
				return fmt.Errorf("flat SMT keys reached the API with the chain configured: %v", sent)
			}
			return nil
		},
	)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{
				Config: hcl(chainAB),
				Check: tfresource.ComposeTestCheckFunc(
					flatUnchanged,
					tfresource.TestCheckResourceAttr(addr, "smt_chain.#", "2"),
					tfresource.TestCheckResourceAttrSet(addr, "smt_chain_revision"),
				),
			},
			{
				// The refresh sees the projected value and leaves state alone.
				Config:   hcl(chainAB),
				PlanOnly: true,
			},
			{
				// A flat, non-SMT change still round-trips; the projection stays out.
				Config: hcl(chainAB + `
  snapshot_read_only = "No"`),
				Check: flatUnchanged,
			},
			{
				Config: hcl(chainAB + `
  snapshot_read_only = "No"`),
				PlanOnly: true,
			},
		},
	})
}

// chainHosts are the two write paths of the same chain: a destination's
// PUT .../smt-chain, answered with a structured envelope, and a source's
// embedded smt_chain, which the backend's source routes refuse with a plain
// 400 detail. Each carries the diagnostic a refused write produces there.
var chainHosts = []struct {
	name     string
	addr     string
	hcl      func(string) string
	stale    *regexp.Regexp
	rejected *regexp.Regexp
}{
	{
		name:     "destination",
		addr:     destAddr,
		hcl:      destHCL,
		stale:    regexp.MustCompile(`Transform chain changed outside Terraform[\s\S]*no longer current`),
		rejected: regexp.MustCompile(`Transform chain rejected[\s\S]*smt_config_string_pattern_mismatch`),
	},
	{
		name:     "source",
		addr:     addr,
		hcl:      hcl,
		stale:    regexp.MustCompile(`Error updating postgresql source[\s\S]*expected revision is stale`),
		rejected: regexp.MustCompile(`Error updating postgresql source[\s\S]*smt_config_string_pattern_mismatch at[\s\S]*/transforms/1/config/services/0/key`),
	},
}

// An edit made elsewhere after the last refresh is rejected at apply: the
// expected revision is stale, no rebase happens, the prior chain stays in
// state, and the next plan refreshes the revision so the apply goes through.
func TestSMT_StaleRevisionIsRejected(t *testing.T) {
	for _, host := range chainHosts {
		t.Run(host.name, func(t *testing.T) {
			factories, fake := newFixture(t)
			var revision string

			tfresource.UnitTest(t, tfresource.TestCase{
				ProtoV6ProviderFactories: factories,
				Steps: []tfresource.TestStep{
					{
						Config: host.hcl(chainAB),
						Check: func(s *terraform.State) error {
							revision = attrAt(s, host.addr, "smt_chain_revision")
							if revision == "" {
								return fmt.Errorf("revision not recorded")
							}
							return nil
						},
					},
					{
						// The edit lands between plan and apply, after the refresh.
						PreConfig:   func() { fake.bumpRevisionBeforeNextWrite = true },
						Config:      host.hcl(chainBA),
						ExpectError: host.stale,
					},
					{
						// State kept the prior chain, so the reorder is still pending;
						// the refresh picks up the new revision and the apply goes through.
						Config: host.hcl(chainBA),
						ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(host.addr, plancheck.ResourceActionUpdate),
						}},
						Check: tfresource.ComposeTestCheckFunc(
							tfresource.TestCheckResourceAttr(host.addr, "smt_chain.0.key", "b"),
							func(s *terraform.State) error {
								if attrAt(s, host.addr, "smt_chain_revision") == revision {
									return fmt.Errorf("revision did not advance after the retried apply")
								}
								if got := fake.instances(attrAt(s, host.addr, "id")); got[0].ID != attrAt(s, host.addr, "smt_chain.0.id") {
									return fmt.Errorf("server order does not match state after the retried apply")
								}
								return nil
							},
						),
					},
				},
			})
		})
	}
}

// A destination chain write that fails right after the connector was
// created still records the connector, so Terraform taints it instead of
// losing the record and colliding on the name at the next apply.
func TestSMT_FailedChainWriteAfterCreateIsTainted(t *testing.T) {
	factories, fake := newFixture(t)
	fake.failNextChainWrite = true

	tfresource.UnitTest(t, tfresource.TestCase{
		TerraformVersionChecks:   []tfversion.TerraformVersionCheck{tfversion.SkipBelow(tfversion.Version1_11_0)},
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{
				Config:      destHCL(chainAB),
				ExpectError: regexp.MustCompile(`Error writing transform chain`),
			},
			{
				Config: destHCL(chainAB),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(destAddr, plancheck.ResourceActionReplace),
				}},
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(destAddr, "smt_chain.#", "2"),
					func(s *terraform.State) error {
						if n := len(fake.sourceIDs()); n != 1 {
							return fmt.Errorf("replace must leave exactly one connector, got %d", n)
						}
						return nil
					},
				),
			},
		},
	})
}

// A source saves its chain in the same request as the connector, so a
// refused chain creates nothing: no state entry, no server record, and the
// next apply is a plain create.
func TestSMT_FailedEmbeddedChainCreatesNoSource(t *testing.T) {
	factories, fake := newFixture(t)
	fake.failNextChainWrite = true

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{
				Config:      hcl(chainAB),
				ExpectError: regexp.MustCompile(`Error creating postgresql source[\s\S]*chain service unavailable`),
			},
			{
				PreConfig: func() {
					if n := len(fake.sourceIDs()); n != 0 {
						t.Fatalf("a refused embedded chain left %d sources behind", n)
					}
				},
				Config: hcl(chainAB),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(addr, plancheck.ResourceActionCreate),
				}},
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(addr, "smt_chain.#", "2"),
					tfresource.TestCheckResourceAttrSet(addr, "smt_chain_revision"),
					func(s *terraform.State) error {
						if n := len(fake.sourceIDs()); n != 1 {
							return fmt.Errorf("want exactly one source, got %d", n)
						}
						return nil
					},
				),
			},
		},
	})
}

// A source has no chain route: the chain rides in the source create and
// update bodies, an update carries the last-read revision, an unchanged chain
// omits the key so the stored one is kept, and the chain is read back under
// the revision the response accepted.
func TestSMT_SourceChainRidesInTheSourceRequest(t *testing.T) {
	factories, fake := newFixture(t)
	var revision string
	events := func(want ...string) tfresource.TestCheckFunc {
		return func(*terraform.State) error {
			if got := fake.recordedEvents(); !reflect.DeepEqual(got, want) {
				return fmt.Errorf("source requests = %v, want %v", got, want)
			}
			return nil
		}
	}

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{
				Config: hcl(chainAB),
				Check: tfresource.ComposeTestCheckFunc(
					events("connector-create", "chain-embedded"),
					func(s *terraform.State) error {
						if sent := fake.lastEmbeddedChain(); sent == nil || sent.ExpectedRevision != nil || sent.Transforms == nil || len(*sent.Transforms) != 2 {
							return fmt.Errorf("the create must embed the two-instance chain without an expected revision, got %+v", sent)
						}
						revision = attrOf(s, "smt_chain_revision")
						return nil
					},
				),
			},
			{
				PreConfig: fake.resetEvents,
				Config: hcl(chainAB + `
  snapshot_read_only = "No"`),
				Check: tfresource.ComposeTestCheckFunc(
					events("connector-update"),
					tfresource.TestCheckResourceAttrPtr(addr, "smt_chain_revision", &revision),
				),
			},
			{
				PreConfig: fake.resetEvents,
				Config: hcl(chainBA + `
  snapshot_read_only = "No"`),
				Check: tfresource.ComposeTestCheckFunc(
					events("connector-update", "chain-embedded"),
					func(s *terraform.State) error {
						sent := fake.lastEmbeddedChain()
						if sent.ExpectedRevision == nil || *sent.ExpectedRevision != revision {
							return fmt.Errorf("the update must carry the last-read revision %q, got %v", revision, sent.ExpectedRevision)
						}
						if attrOf(s, "smt_chain_revision") == revision {
							return fmt.Errorf("revision did not advance")
						}
						return nil
					},
				),
			},
		},
	})
}

// A response that saved a chain but does not name the accepted revision
// cannot be keyed safely: the apply fails and the next plan refreshes.
func TestSMT_SourceChainWithoutReceiptIsUnconfirmed(t *testing.T) {
	factories, fake := newFixture(t)
	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{Config: hcl(chainAB)},
			{
				PreConfig:   func() { fake.dropNextReceipt = true },
				Config:      hcl(chainBA),
				ExpectError: regexp.MustCompile(`Transform chain outcome unconfirmed`),
			},
			{
				// Refresh reads the saved chain; the plan converges.
				Config:           hcl(chainBA),
				ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()}},
			},
		},
	})
}

// The destination base drives the same chain under the destinations kind.
func TestSMT_DestinationChain(t *testing.T) {
	factories, fake := newFixture(t)
	const dest = "streamkap_destination_snowflake.t"
	config := `
resource "streamkap_destination_snowflake" "t" {
  name                    = "chain"
  snowflake_url_name      = "acct.snowflakecomputing.invalid"
  snowflake_user_name     = "u"
  snowflake_private_key   = "k"
  snowflake_database_name = "D"
  snowflake_schema_name   = "S"
` + chainAB + "\n}\n"

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{
				Config: config,
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(dest, "smt_chain.#", "2"),
					tfresource.TestCheckResourceAttr(dest, "smt_chain.1.contract_fixture_deep.services.0.endpoints.0.id", "ep/1"),
					tfresource.TestCheckResourceAttrSet(dest, "smt_chain_revision"),
					func(s *terraform.State) error {
						id := s.RootModule().Resources[dest].Primary.Attributes["id"]
						if sent := fake.flatKeysSent(id); len(sent) != 0 {
							return fmt.Errorf("flat SMT keys reached the API with the chain configured: %v", sent)
						}
						if got := fake.instances(id); len(got) != 2 {
							return fmt.Errorf("destination chain not stored: %v", got)
						}
						return nil
					},
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// An instance the schema cannot represent is refused on refresh instead of
// being dropped by the next write, and a validation issue the plan could not
// catch is reported with its pointer: located on the config attribute for a
// destination, named in the source error for a source.
func TestSMT_UnrepresentableAndRejectedWrites(t *testing.T) {
	for _, host := range chainHosts {
		t.Run(host.name, func(t *testing.T) {
			factories, fake := newFixture(t)

			tfresource.UnitTest(t, tfresource.TestCase{
				ProtoV6ProviderFactories: factories,
				Steps: []tfresource.TestStep{
					{
						Config: host.hcl(chainAB),
					},
					{
						PreConfig: func() {
							fake.rejectNextWrite(api.SMTValidationIssue{Code: "smt_config_string_pattern_mismatch", Message: "String should match pattern", Pointer: "/transforms/1/config/services/0/key"})
						},
						Config:      host.hcl(chainBA),
						ExpectError: host.rejected,
					},
					{
						// The rejected write left the server chain and the state alone.
						Config: host.hcl(chainBA),
						ConfigPlanChecks: tfresource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(host.addr, plancheck.ResourceActionUpdate),
						}},
					},
					{
						PreConfig: func() {
							id := fake.sourceIDs()[0]
							fake.setPredicate(id, fake.instances(id)[0].ID)
						},
						Config:      host.hcl(chainBA),
						PlanOnly:    true,
						ExpectError: regexp.MustCompile(`Transform predicate cannot be represented`),
					},
				},
			})
		})
	}
}

func TestSMT_SelectorIsRefusedOnRefresh(t *testing.T) {
	factories, fake := newFixture(t)
	config := `
resource "streamkap_destination_snowflake" "t" {
  name                    = "chain"
  snowflake_url_name      = "acct.snowflakecomputing.invalid"
  snowflake_user_name     = "u"
  snowflake_private_key   = "k"
  snowflake_database_name = "D"
  snowflake_schema_name   = "S"
` + chainAB + "\n}\n"

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{Config: config},
			{
				PreConfig: func() {
					id := fake.sourceIDs()[0]
					fake.setSelector(id, fake.instances(id)[0].ID)
				},
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Transform selector cannot be represented`),
			},
		},
	})
}

// A connector without a chain imports as an empty managed chain with no
// revision, and the first chain the HCL adds is created without an expected
// revision. A resource still on the flat surface cannot silently take over a
// chain that exists on the server: the write is refused and the diagnostic
// says to import.
func TestSMT_AdoptionPaths(t *testing.T) {
	factories, fake := newFixture(t)
	ctx := context.Background()
	legacy, err := fake.CreateSource(ctx, api.Source{Name: "chain", Connector: "postgresql", Config: map[string]any{
		"database.hostname.user.defined": "db.invalid", "database.user.user.defined": "u", "database.password.user.defined": "p",
		"database.dbname.user.defined": "d", "schema.include.list.user.defined": "public", "table.include.list.user.defined": "public.t",
	}})
	require.NoError(t, err)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: factories,
		Steps: []tfresource.TestStep{
			{
				Config:             hcl(""),
				ResourceName:       addr,
				ImportState:        true,
				ImportStateId:      legacy.ID,
				ImportStatePersist: true,
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					a := states[0].Attributes
					if a["smt_chain.#"] != "0" {
						return fmt.Errorf("a connector without a chain imports as an empty managed chain, got %q", a["smt_chain.#"])
					}
					if _, has := a["smt_chain_revision"]; has {
						return fmt.Errorf("no revision exists before the chain is created")
					}
					return nil
				},
			},
			{
				PreConfig: fake.resetEvents,
				Config:    hcl(chainAB),
				Check: tfresource.ComposeTestCheckFunc(
					tfresource.TestCheckResourceAttr(addr, "smt_chain.#", "2"),
					tfresource.TestCheckResourceAttrSet(addr, "smt_chain_revision"),
					func(*terraform.State) error {
						if got := fake.recordedEvents(); !reflect.DeepEqual(got, []string{"connector-update", "chain-embedded"}) {
							return fmt.Errorf("the initial source chain must ride in the one source update, got %v", got)
						}
						return nil
					},
				),
			},
		},
	})

	// A second resource stays on the flat surface while the UI creates a chain
	// on its connector.
	factories2, fake2 := newFixture(t)
	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: factories2,
		Steps: []tfresource.TestStep{
			{
				Config: hcl(""),
			},
			{
				PreConfig: func() {
					id := fake2.sourceIDs()[0]
					_, err := fake2.PutSMTChain(ctx, "sources", id, api.SMTChainWrite{Transforms: &[]api.SMTInstanceWrite{
						{Type: "contract_fixture_nested", Name: "From the UI", Enabled: true, Config: func() map[string]any {
							spec, _ := catalogSpec(fake2.cat, "transform", "contract_fixture_nested")
							return spec.Examples.ReadProjection
						}()},
					}})
					require.NoError(t, err)
				},
				Config:      hcl(chainAB),
				ExpectError: regexp.MustCompile(`Transform chain already exists outside Terraform[\s\S]*Import the resource`),
			},
		},
	})
}

func TestProviderInitialDestinationChainIsGuardedBeforeConnectorMutation(t *testing.T) {
	const destinationAddr = "streamkap_destination_snowflake.t"
	destinationConfig := func(name, chain string) string {
		return `
resource "streamkap_destination_snowflake" "t" {
  name                    = "` + name + `"
  snowflake_url_name      = "acct.snowflakecomputing.invalid"
  snowflake_user_name     = "u"
  snowflake_private_key   = "k"
  snowflake_database_name = "D"
  snowflake_schema_name   = "S"
` + chain + "\n}\n"
	}
	t.Run("managed-only chain is created without a connector update", func(t *testing.T) {
		factories, fake := newFixture(t)
		tfresource.UnitTest(t, tfresource.TestCase{
			ProtoV6ProviderFactories: factories,
			Steps: []tfresource.TestStep{
				{Config: destinationConfig("chain", "")},
				{
					PreConfig: fake.resetEvents,
					Config:    destinationConfig("chain", chainAB),
					Check: func(s *terraform.State) error {
						if got := fake.connectorWriteCount(); got != 1 {
							return fmt.Errorf("initial chain caused a connector write: got %d total connector writes, events %v", got, fake.recordedEvents())
						}
						if got := fake.recordedEvents(); !reflect.DeepEqual(got, []string{"adoption-read", "chain-put"}) {
							return fmt.Errorf("initial chain request order = %v", got)
						}
						attributes := s.RootModule().Resources[destinationAddr].Primary.Attributes
						if attributes["smt_chain_revision"] == "" || attributes["smt_chain.0.id"] == "" {
							return fmt.Errorf("accepted chain identities were not recorded in state")
						}
						return nil
					},
				},
				{
					PreConfig: fake.resetEvents,
					Config:    destinationConfig("chain", chainBA),
					Check: func(*terraform.State) error {
						if got := fake.recordedEvents(); !reflect.DeepEqual(got, []string{"connector-update", "chain-put"}) {
							return fmt.Errorf("existing-chain request order = %v", got)
						}
						return nil
					},
				},
			},
		})
	})

	t.Run("legacy edit racing the audit is refused by the chain write", func(t *testing.T) {
		factories, fake := newFixture(t)
		var id string
		tfresource.UnitTest(t, tfresource.TestCase{
			ProtoV6ProviderFactories: factories,
			Steps: []tfresource.TestStep{
				{Config: destinationConfig("chain", "")},
				{
					PreConfig: func() {
						id = fake.sourceIDs()[0]
						fake.resetEvents()
						fake.legacyRaceAfterAdoption[id] = true
					},
					Config:      destinationConfig("chain", chainAB),
					ExpectError: regexp.MustCompile(`Legacy transforms require explicit adoption`),
				},
			},
		})
		require.Equal(t, 1, fake.connectorWriteCount())
		require.Equal(t, []string{"adoption-read", "chain-put"}, fake.recordedEvents())
		require.Empty(t, fake.instances(id))
		var apiError *api.APIError
		require.ErrorAs(t, fake.recordedChainError(), &apiError)
		require.Equal(t, 409, apiError.StatusCode)
		envelope, ok := api.SMTErrorOf(apiError)
		require.True(t, ok)
		require.Equal(t, "smt_adoption_required", envelope.Code)
		require.Equal(t, "/", envelope.Issues[0].Pointer)
	})

	t.Run("legacy and unavailable classifications refuse before writes", func(t *testing.T) {
		for _, tc := range []struct {
			status string
			want   string
		}{{"available", "Legacy transforms require explicit adoption"}, {"unavailable", "fixture adoption is unavailable"}} {
			t.Run(tc.status, func(t *testing.T) {
				factories, fake := newFixture(t)
				tfresource.UnitTest(t, tfresource.TestCase{
					ProtoV6ProviderFactories: factories,
					Steps: []tfresource.TestStep{
						{Config: destinationConfig("chain", "")},
						{PreConfig: func() {
							id := fake.sourceIDs()[0]
							fake.adoptionStatus[id] = tc.status
							fake.resetEvents()
						}, Config: destinationConfig("chain", chainAB), ExpectError: regexp.MustCompile(tc.want)},
					},
				})
				require.Equal(t, 1, fake.connectorWriteCount())
				require.Equal(t, []string{"adoption-read"}, fake.recordedEvents())
			})
		}
	})

	t.Run("ordinary connector changes cannot share the initial chain update", func(t *testing.T) {
		factories, fake := newFixture(t)
		tfresource.UnitTest(t, tfresource.TestCase{
			ProtoV6ProviderFactories: factories,
			Steps: []tfresource.TestStep{
				{Config: destinationConfig("chain", "")},
				{PreConfig: fake.resetEvents, Config: destinationConfig("renamed", chainAB), ExpectError: regexp.MustCompile(`Initial transform chain must be applied separately`)},
			},
		})
		require.Equal(t, 1, fake.connectorWriteCount())
		require.Empty(t, fake.recordedEvents())
	})
}
