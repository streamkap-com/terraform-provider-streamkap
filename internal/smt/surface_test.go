package smt

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

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

func TestStateFromRefusesSelectorReadFixture(t *testing.T) {
	raw, err := os.ReadFile("../api/testdata/smt_wire_fixtures.json")
	require.NoError(t, err)
	var fixture struct {
		SelectorReadResponse json.RawMessage `json:"selector_read_response"`
	}
	require.NoError(t, json.Unmarshal(raw, &fixture))
	var read api.SMTChainRead
	require.NoError(t, json.Unmarshal(fixture.SelectorReadResponse, &read))

	_, _, diags := (&Surface{}).stateFrom(context.Background(), &read, nil, nil)
	require.True(t, diags.HasError())
	require.Contains(t, diags.Errors()[0].Summary(), "Transform selector cannot be represented")
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
