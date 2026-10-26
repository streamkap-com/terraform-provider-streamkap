package smt

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// A validation issue's runtime pointer lands on the planned entry at that
// position and continues into its typed config child; secret operation
// issues land on smt_secrets; anything else on the chain.
func TestIssuePath(t *testing.T) {
	s := &Surface{}
	w := &Write{correlated: []Correlated{{Key: "a", Type: "contract_fixture_nested"}, {Key: "b", Type: "contract_fixture_deep"}}}
	chain := path.Root(AttrChain)
	for pointer, want := range map[string]path.Path{
		"/transforms/1/config/routes/1/endpoint":    chain.AtListIndex(1).AtName("contract_fixture_deep").AtName("routes").AtListIndex(1).AtName("endpoint"),
		"/transforms/0/config/output/prefix":        chain.AtListIndex(0).AtName("contract_fixture_nested").AtName("output").AtName("prefix"),
		"/transforms/0/config/odd~1key":             chain.AtListIndex(0).AtName("contract_fixture_nested").AtName("odd/key"),
		"/transforms/1/secret_operations/0/pointer": path.Root(AttrSecrets),
		"/transforms/0/name":                        chain.AtListIndex(0).AtName("name"),
		"/transforms/0":                             chain.AtListIndex(0),
		"/transforms/7/config/x":                    chain,
		"/expected_revision":                        chain,
		"":                                          chain,
	} {
		require.True(t, want.Equal(s.issuePath(pointer, w)), "%s: got %s, want %s", pointer, s.issuePath(pointer, w), want)
	}
}

// The read schema still declares a selector on an instance, so a read that
// carries one is refused rather than managed: writing the instance back
// without it would drop the rule. The server compiles selectors away and
// never returns one, so the read here is constructed, not a fixture.
func TestStateFromRefusesSelectorBearingRead(t *testing.T) {
	read := api.SMTChainRead{Desired: api.SMTChainSnapshot{ChainRevision: "r1", Transforms: []api.SMTInstanceRead{
		{Kind: "typed", ID: "MaskField1", Alias: "MaskField1", Type: "mask_field", Selector: json.RawMessage(`{"include":[{"table":"*","column":"email"}],"exclude":[]}`)},
	}}}
	_, _, diags := (&Surface{}).stateFrom(context.Background(), &read, nil, nil)
	require.True(t, diags.HasError())
	require.Contains(t, diags.Errors()[0].Summary(), "Transform selector cannot be represented")
}

func loadWireFixture(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile("../api/testdata/smt_wire_fixtures.json")
	require.NoError(t, err)
	var fixture map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fixture))
	return fixture
}

// A selector is write-only: a rule saved elsewhere reads back as the field
// lists it compiled to, which is what the fixture's last_applied records, and
// with no selector. Read keeps those lists as the instance's plain mask_field
// config under the pinned catalog's current release instead of refusing the
// instance, and the instance id is its alias.
func TestStateFromKeepsCompiledSelectorLists(t *testing.T) {
	corpus, err := LoadCorpus("../generated/smt_catalog.json")
	require.NoError(t, err)
	chain, err := BuildChain(NewCatalog(corpus).ForHost("destinations").Authorable())
	require.NoError(t, err)

	var read api.SMTChainRead
	require.NoError(t, json.Unmarshal(loadWireFixture(t)["selector_read_response"], &read))
	require.NotNil(t, read.LastApplied)
	read.Desired = *read.LastApplied
	require.False(t, read.Desired.Transforms[0].HasSelector())

	list, revision, diags := NewSurface(chain, nil).stateFrom(context.Background(), &read, []string{"MaskField1"}, nil)
	require.False(t, diags.HasError(), "%v", diags)
	require.Equal(t, read.Desired.ChainRevision, revision.ValueString())
	require.Len(t, list.Elements(), 1)
	entry := list.Elements()[0].(types.Object).Attributes()
	require.Equal(t, types.StringValue("MaskField1"), entry[attrID])
	require.Equal(t, types.StringValue("MaskField1"), entry[attrAlias])
	require.Equal(t, types.Int64Value(3), entry[attrSchemaVersion])
	mask := entry["mask_field"].(types.Object).Attributes()
	require.Equal(t, types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*.email")}), mask["fields_include"])
	require.Equal(t, types.ListValueMust(types.StringType, []attr.Value{types.StringValue("audit_log.email")}), mask["fields_exclude"])
}

// A refused chain write is named by what the server reports: a write without
// its revision is a provider defect, not drift; an unreadable chain leaves the
// outcome unknown; a stale revision is drift; and an order refusal lands on
// the chain entry the issue points at.
func TestWriteErrorNamesFixtureRefusals(t *testing.T) {
	fixture := loadWireFixture(t)
	refusal := func(status int, body json.RawMessage) error {
		var envelope struct {
			Detail json.RawMessage `json:"detail"`
		}
		require.NoError(t, json.Unmarshal(body, &envelope))
		return &api.APIError{StatusCode: status, Detail: "refused", DetailJSON: envelope.Detail}
	}
	readFailed := json.RawMessage(`{"detail":{"code":"smt_read_failed","message":"The chain write was accepted but could not be read back; read the chain again before retrying","issues":[]}}`)
	w := &Write{Configured: true, ChainConfigured: true, correlated: []Correlated{{Key: "mask", Type: "mask_field"}, {Key: "route", Type: "regex_router"}}}
	chain := path.Root(AttrChain)
	for _, tc := range []struct {
		name    string
		err     error
		summary string
		at      path.Path
		detail  string
	}{
		{"missing revision", refusal(409, fixture["missing_revision_response"]), "Transform chain sent without its revision", chain, "report this issue"},
		{"read failed", refusal(503, readFailed), "Transform chain outcome unconfirmed", chain, "may or may not have been saved"},
		{"stale revision", refusal(409, fixture["conflict_response"]), "Transform chain changed outside Terraform", chain, "no longer current"},
		{"order rejected", refusal(422, fixture["order_rejected_response"]), "Transform chain rejected: The destination was not saved; fix the SMT chain issues and send it again", chain.AtListIndex(1), "smt_order_rejected"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := (&Surface{}).WriteError(tc.err, types.StringValue("r1"), w)
			require.Len(t, diags, 1)
			d := diags[0]
			require.Equal(t, tc.summary, d.Summary())
			require.Contains(t, d.Detail(), tc.detail)
			withPath, ok := d.(diag.DiagnosticWithPath)
			require.True(t, ok)
			require.True(t, tc.at.Equal(withPath.Path()), "got %s, want %s", withPath.Path(), tc.at)
		})
	}
}

// Before a write to an existing connector the chain is read: the write takes
// the current revision, an edit made since the revision in state was read is
// refused, and a first write is refused when the connector already holds user
// transforms, which a chain replacement would delete. An override-only write
// replaces no transforms, so it may proceed over them. None of the refusals
// writes anything.
func TestExpectedRevisionReadsBeforeWriting(t *testing.T) {
	ctx := context.Background()
	s := &Surface{}
	chainWrite := &Write{Configured: true, ChainConfigured: true}
	overrideWrite := &Write{Configured: true, ManagedOverrides: &map[string]bool{}}
	empty := &api.SMTChainRead{Desired: api.SMTChainSnapshot{ChainRevision: "r2"}}
	held := &api.SMTChainRead{Desired: api.SMTChainSnapshot{ChainRevision: "r2", Transforms: []api.SMTInstanceRead{{Kind: "typed", ID: "MaskField1"}}}}
	for _, tc := range []struct {
		name  string
		read  *api.SMTChainRead
		write *Write
		prior types.String
		error string
	}{
		{name: "current revision", read: held, write: chainWrite, prior: types.StringValue("r2")},
		{name: "first write to an empty chain", read: empty, write: chainWrite, prior: types.StringNull()},
		{name: "first override write over user transforms", read: held, write: overrideWrite, prior: types.StringNull()},
		{name: "edited since the plan", read: held, write: chainWrite, prior: types.StringValue("r1"), error: "Transform chain changed outside Terraform"},
		{name: "first write over user transforms", read: held, write: chainWrite, prior: types.StringNull(), error: "Transform chain already exists outside Terraform"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &managedOverrideClient{read: tc.read}
			revision, diags := s.ExpectedRevision(ctx, client, "destinations", "destination-1", tc.write, tc.prior)
			require.Empty(t, client.writes)
			if tc.error != "" {
				require.True(t, diags.HasError())
				require.Equal(t, tc.error, diags.Errors()[0].Summary())
				return
			}
			require.False(t, diags.HasError(), "%v", diags)
			require.Equal(t, types.StringValue("r2"), revision)
		})
	}

	unchanged := &Write{Configured: true, ChainConfigured: true, Unchanged: true}
	revision, diags := s.ExpectedRevision(ctx, &managedOverrideClient{err: errors.New("no read expected")}, "destinations", "destination-1", unchanged, types.StringValue("r1"))
	require.False(t, diags.HasError(), "a write that sends nothing reads nothing: %v", diags)
	require.Equal(t, types.StringValue("r1"), revision)
}
