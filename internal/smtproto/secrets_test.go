package smtproto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPointerEscaping(t *testing.T) {
	require.Equal(t, "row~0west~11", EscapeToken("row~west/1"))
	require.Equal(t, "row~west/1", UnescapeToken("row~0west~11"))
	require.Equal(t, "/routes/row-east/token", InstancePointer("/routes/*/token", "row-east"))
	require.Equal(t, "/routes/row~0west~11/token", InstancePointer("/routes/*/token", "row~west/1"))
	require.Equal(t, "/services/svc-a/endpoints/ep~11/secret", InstancePointer("/services/*/endpoints/*/secret", "svc-a", "ep/1"))
}

func TestResolvePointer(t *testing.T) {
	_, cat := loadCorpus(t)
	nested, deep := cat["contract_fixture_nested"], cat["contract_fixture_deep"]

	addr, err := ResolvePointer(nested, "/routes/row~0west~11/token")
	require.NoError(t, err)
	require.Equal(t, "/routes/*/token", addr.Template)
	require.Equal(t, []string{"row~west/1"}, addr.RowKeys)

	addr, err = ResolvePointer(deep, "/services/svc-a/endpoints/ep~11/secret")
	require.NoError(t, err)
	require.Equal(t, []string{"svc-a", "ep/1"}, addr.RowKeys)
	addr, err = ResolvePointer(deep, "/services/svc-b/auth/token")
	require.NoError(t, err)
	require.Equal(t, []string{"svc-b"}, addr.RowKeys)

	for _, bad := range []string{"/routes/1/token", "/routes/row-east/endpoint", "/routes/row-east", "routes/row-east/token", "/output/prefix"} {
		_, err := ResolvePointer(nested, bad)
		if bad == "/routes/1/token" {
			// The runtime index form is syntactically a row key named "1";
			// only the config walk can reject it, which RowExists does below.
			require.NoError(t, err)
			continue
		}
		require.Error(t, err, bad)
	}
}

func TestRowExists_UsesDeclaredRowKeys(t *testing.T) {
	_, cat := loadCorpus(t)
	nested, deep := cat["contract_fixture_nested"], cat["contract_fixture_deep"]

	addr, _ := ResolvePointer(nested, "/routes/row~0west~11/token")
	require.True(t, RowExists(nested, addr, nested.Examples.ReadProjection))
	byIndex, _ := ResolvePointer(nested, "/routes/1/token")
	require.False(t, RowExists(nested, byIndex, nested.Examples.ReadProjection), "index addressing never matches a row")

	addr, _ = ResolvePointer(deep, "/services/svc-a/endpoints/ep~11/secret")
	require.True(t, RowExists(deep, addr, deep.Examples.ReadProjection))
	addr, _ = ResolvePointer(deep, "/services/svc-a/endpoints/ep~02/secret")
	require.True(t, RowExists(deep, addr, deep.Examples.ReadProjection))
	addr, _ = ResolvePointer(deep, "/services/svc-b/endpoints/ep~11/secret")
	require.False(t, RowExists(deep, addr, deep.Examples.ReadProjection), "svc-b has no endpoints")
	addr, _ = ResolvePointer(deep, "/services/svc-b/auth/token")
	require.True(t, RowExists(deep, addr, deep.Examples.ReadProjection))
}

func TestPlanSecretOps(t *testing.T) {
	_, cat := loadCorpus(t)
	chain := map[string]string{"a": "contract_fixture_nested", "d": "contract_fixture_deep"}
	v := func(s string) *string { return &s }
	replaceEast := SecretValue{Pointer: "/routes/row-east/token", Value: v("s1")}
	clearWest := SecretValue{Pointer: "/routes/row~0west~11/token", Clear: true}

	cases := map[string]struct {
		prior    map[string]int64
		entries  []SecretEntry
		wantOps  []SecretOp
		wantNext map[string]int64
		wantErr  string
	}{
		"initial rotation needs a version": {
			entries: []SecretEntry{{Key: "a", Version: 0, Values: []SecretValue{replaceEast}}},
			wantErr: "Invalid rotation version",
		},
		"initial rotation executes once per row": {
			entries: []SecretEntry{{Key: "a", Version: 1, Values: []SecretValue{replaceEast, clearWest}}},
			wantOps: []SecretOp{
				{Key: "a", Pointer: "/routes/row-east/token", Value: "s1"},
				{Key: "a", Pointer: "/routes/row~0west~11/token", Clear: true},
			},
			wantNext: map[string]int64{"a": 1},
		},
		"increased version executes once": {
			prior:    map[string]int64{"a": 1},
			entries:  []SecretEntry{{Key: "a", Version: 2, Values: []SecretValue{replaceEast}}},
			wantOps:  []SecretOp{{Key: "a", Pointer: "/routes/row-east/token", Value: "s1"}},
			wantNext: map[string]int64{"a": 2},
		},
		"unchanged version with values still configured sends nothing": {
			prior:    map[string]int64{"a": 2},
			entries:  []SecretEntry{{Key: "a", Version: 2, Values: []SecretValue{replaceEast}}},
			wantNext: map[string]int64{"a": 2},
		},
		"unchanged version without values is fine": {
			prior:    map[string]int64{"a": 2},
			entries:  []SecretEntry{{Key: "a", Version: 2}},
			wantNext: map[string]int64{"a": 2},
		},
		"edited value without a version bump is not detected": {
			prior:    map[string]int64{"a": 2},
			entries:  []SecretEntry{{Key: "a", Version: 2, Values: []SecretValue{{Pointer: "/routes/row-east/token", Value: v("edited")}}}},
			wantNext: map[string]int64{"a": 2},
		},
		"decreased version is rejected": {
			prior:   map[string]int64{"a": 3},
			entries: []SecretEntry{{Key: "a", Version: 2, Values: []SecretValue{replaceEast}}},
			wantErr: "Rotation version decreased",
		},
		"removed then re-added entry cannot restart below the chain's version": {
			// The chain entry still says 5 although no smt_secrets entry was
			// in state at all, so this is a decrease, not a fresh rotation.
			prior:   map[string]int64{"a": 5},
			entries: []SecretEntry{{Key: "a", Version: 1, Values: []SecretValue{replaceEast}}},
			wantErr: "Rotation version decreased",
		},
		"removed entry leaves the chain's version alone": {
			prior:    map[string]int64{"a": 5, "d": 2},
			entries:  nil,
			wantNext: map[string]int64{"a": 5, "d": 2},
		},
		"increased version with no operation is rejected": {
			prior:   map[string]int64{"a": 1},
			entries: []SecretEntry{{Key: "a", Version: 2}},
			wantErr: "Rotation without operations",
		},
		"unknown key is rejected": {
			entries: []SecretEntry{{Key: "ghost", Version: 1, Values: []SecretValue{replaceEast}}},
			wantErr: "Secret entry for unknown instance",
		},
		"pointer outside the type's secret paths is rejected": {
			entries: []SecretEntry{{Key: "a", Version: 1, Values: []SecretValue{{Pointer: "/routes/row-east/endpoint", Value: v("x")}}}},
			wantErr: "Invalid secret pointer",
		},
		"pointer of another type is rejected": {
			entries: []SecretEntry{{Key: "a", Version: 1, Values: []SecretValue{{Pointer: "/services/svc-a/auth/token", Value: v("x")}}}},
			wantErr: "Invalid secret pointer",
		},
		"value and clear together are ambiguous": {
			entries: []SecretEntry{{Key: "a", Version: 1, Values: []SecretValue{{Pointer: "/routes/row-east/token", Value: v("x"), Clear: true}}}},
			wantErr: "Ambiguous secret operation",
		},
		"neither value nor clear is ambiguous": {
			entries: []SecretEntry{{Key: "a", Version: 1, Values: []SecretValue{{Pointer: "/routes/row-east/token"}}}},
			wantErr: "Ambiguous secret operation",
		},
		"two-level keyed pointer": {
			entries:  []SecretEntry{{Key: "d", Version: 1, Values: []SecretValue{{Pointer: "/services/svc-a/endpoints/ep~11/secret", Value: v("x")}}}},
			wantOps:  []SecretOp{{Key: "d", Pointer: "/services/svc-a/endpoints/ep~11/secret", Value: "x"}},
			wantNext: map[string]int64{"d": 1},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			ops, next, diags := PlanSecretOps(tc.prior, tc.entries, chain, cat)
			if tc.wantErr != "" {
				require.True(t, diags.HasError())
				require.Equal(t, tc.wantErr, diags.Errors()[0].Summary())
				return
			}
			require.False(t, diags.HasError(), "%v", diags)
			require.Equal(t, tc.wantOps, ops)
			if tc.wantNext == nil {
				tc.wantNext = map[string]int64{}
			}
			require.Equal(t, tc.wantNext, next)
		})
	}
}
