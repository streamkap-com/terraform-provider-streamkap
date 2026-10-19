package smt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

func TestStateFromOpaqueRetentionBoundary(t *testing.T) {
	for _, test := range []struct {
		name string
		rows []api.SMTInstanceRead
		want string
	}{
		{
			name: "opaque",
			rows: []api.SMTInstanceRead{{Kind: "opaque", ID: "retained", Alias: "legacy", Reason: "migration_only", Config: map[string]any{"private": "hidden"}}},
			want: "Transform instance cannot be represented",
		},
		{
			name: "mixed",
			rows: []api.SMTInstanceRead{{Kind: "typed", ID: "typed"}, {Kind: "opaque", ID: "retained", Alias: "legacy", Reason: "migration_only"}},
			want: "Transform instance cannot be represented",
		},
		{
			name: "unknown kind",
			rows: []api.SMTInstanceRead{{Kind: "future-private-kind", ID: "private-id", Alias: "private-alias", Config: map[string]any{"private": "hidden"}, Predicate: []byte(`{"private":"hidden"}`), Selector: []byte(`{"private":"hidden"}`)}},
			want: "Transform kind cannot be represented",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			read := &api.SMTChainRead{Desired: api.SMTChainSnapshot{ChainRevision: "captured", Transforms: test.rows}}
			// No catalog or writer is available: unsupported state must refuse before conversion.
			require.NotPanics(t, func() {
				list, revision, diags := (&Surface{}).stateFrom(context.Background(), read, nil, nil)
				require.True(t, diags.HasError())
				require.Empty(t, list.Elements())
				require.True(t, revision.IsNull())
				require.Contains(t, diags.Errors()[0].Summary(), test.want)
				for _, diagnostic := range diags.Errors() {
					require.NotContains(t, diagnostic.Detail(), "future-private-kind")
					require.NotContains(t, diagnostic.Detail(), "hidden")
					require.NotContains(t, diagnostic.Detail(), "private-id")
					require.NotContains(t, diagnostic.Detail(), "private-alias")
				}
			})
		})
	}
}
