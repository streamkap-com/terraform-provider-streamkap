package smt

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/stretchr/testify/require"
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
