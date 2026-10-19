package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdoptionAuditClientUsesReadOnlyDestinationRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/destinations/dest-1/smt-chain-adoption", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"available","expected_revision":"legacy-r1","managed_aliases":["InsertField"],"transforms":[{"alias":"MaskField1","type":"mask_field","schema_version":1,"codec_version":1,"name":"MaskField1","config":{"fields_include":["orders.email"]},"configured_secret_paths":["/mask_salt"]}]}`))
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL}).(*streamkapAPI)
	audit, err := client.GetSMTChainAdoption(context.Background(), "destinations", "dest-1")
	require.NoError(t, err)
	require.Equal(t, SMTAdoptionAvailable, audit.Status)
	require.Equal(t, "legacy-r1", audit.ExpectedRevision)
	require.Equal(t, []string{"InsertField"}, audit.ManagedAliases)
	require.Equal(t, []string{"/mask_salt"}, audit.Transforms[0].ConfiguredSecretPaths)
	require.NotContains(t, audit.Transforms[0].Config, "mask_salt")
	require.True(t, IsSMTAdoptionRequired(&APIError{StatusCode: http.StatusConflict, DetailJSON: []byte(`{"code":"smt_adoption_required","message":"adopt first","issues":[]}`)}))
}
