package smtproto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Identity follows the provider key and nothing else: not position, not
// display name, not the server id.
func TestCorrelate(t *testing.T) {
	prior := []PriorInstance{
		{Key: "a", Type: "contract_fixture_nested", ID: "inst-1", Alias: "nested_inst-1", SchemaVersion: 1},
		{Key: "b", Type: "contract_fixture_deep", ID: "inst-2", Alias: "deep_inst-2", SchemaVersion: 1},
	}
	cases := map[string]struct {
		planned  []PlannedInstance
		want     []Correlated
		errorsOn string
	}{
		"unchanged": {
			planned: []PlannedInstance{{Key: "a", Type: "contract_fixture_nested"}, {Key: "b", Type: "contract_fixture_deep"}},
			want: []Correlated{
				{Key: "a", Type: "contract_fixture_nested", ID: "inst-1", Alias: "nested_inst-1", SchemaVersion: 1, Matched: true},
				{Key: "b", Type: "contract_fixture_deep", ID: "inst-2", Alias: "deep_inst-2", SchemaVersion: 1, Matched: true},
			},
		},
		"reorder keeps ids with their keys": {
			planned: []PlannedInstance{{Key: "b", Type: "contract_fixture_deep"}, {Key: "a", Type: "contract_fixture_nested"}},
			want: []Correlated{
				{Key: "b", Type: "contract_fixture_deep", ID: "inst-2", Alias: "deep_inst-2", SchemaVersion: 1, Matched: true},
				{Key: "a", Type: "contract_fixture_nested", ID: "inst-1", Alias: "nested_inst-1", SchemaVersion: 1, Matched: true},
			},
		},
		"key change is a new instance, the old identity is not inherited": {
			planned: []PlannedInstance{{Key: "a2", Type: "contract_fixture_nested"}, {Key: "b", Type: "contract_fixture_deep"}},
			want: []Correlated{
				{Key: "a2", Type: "contract_fixture_nested"},
				{Key: "b", Type: "contract_fixture_deep", ID: "inst-2", Alias: "deep_inst-2", SchemaVersion: 1, Matched: true},
			},
		},
		"type change under an existing key is rejected": {
			planned:  []PlannedInstance{{Key: "a", Type: "contract_fixture_deep"}, {Key: "b", Type: "contract_fixture_deep"}},
			errorsOn: "Transform type change is not allowed in place",
		},
		"duplicate keys are rejected": {
			planned:  []PlannedInstance{{Key: "a", Type: "contract_fixture_nested"}, {Key: "a", Type: "contract_fixture_nested"}},
			errorsOn: "Duplicate instance key",
		},
		"empty key is rejected": {
			planned:  []PlannedInstance{{Key: "", Type: "contract_fixture_nested"}},
			errorsOn: "Missing instance key",
		},
		"removal": {
			planned: []PlannedInstance{{Key: "b", Type: "contract_fixture_deep"}},
			want:    []Correlated{{Key: "b", Type: "contract_fixture_deep", ID: "inst-2", Alias: "deep_inst-2", SchemaVersion: 1, Matched: true}},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, diags := Correlate(prior, tc.planned)
			if tc.errorsOn != "" {
				require.True(t, diags.HasError())
				require.Equal(t, tc.errorsOn, diags.Errors()[0].Summary())
				return
			}
			require.False(t, diags.HasError(), "%v", diags)
			require.Equal(t, tc.want, got)
		})
	}

	t.Run("type change guidance names remove then add", func(t *testing.T) {
		_, diags := Correlate(prior, []PlannedInstance{{Key: "a", Type: "contract_fixture_deep"}})
		require.Contains(t, diags.Errors()[0].Detail(), "Remove the entry in one apply, then add the contract_fixture_deep instance under a new key")
		require.Contains(t, diags.Errors()[0].Detail(), "inst-1")
	})
}

// Display names are not part of correlation at all: they are not even an
// input to it, so a rename cannot move an identity.
func TestCorrelate_RenameIsInvisible(t *testing.T) {
	_, hasName := any(PlannedInstance{}).(interface{ GetName() string })
	require.False(t, hasName)
}

func TestImportKeys(t *testing.T) {
	keys, err := ImportKeys([]InstanceRead{{ID: "inst-9"}, {ID: "inst-3"}})
	require.NoError(t, err)
	require.Equal(t, []string{"inst-9", "inst-3"}, keys)

	again, err := ImportKeys([]InstanceRead{{ID: "inst-9"}, {ID: "inst-3"}})
	require.NoError(t, err)
	require.Equal(t, keys, again, "derivation is deterministic")

	_, err = ImportKeys([]InstanceRead{{ID: "inst-9"}, {ID: "inst-9"}})
	require.Error(t, err)
	_, err = ImportKeys([]InstanceRead{{ID: ""}})
	require.Error(t, err)
}
