package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

// A source chain rides in the source create and update bodies: the request
// keys are the ones CreateSourceReq and UpdateSourceReq declare, the chain
// key is absent unless a chain is written, and the response receipt decodes
// with exactly the keys SourceSmtChainReceipt requires.
func TestSourceSMTChainWire_MatchesFixture(t *testing.T) {
	w := loadSMTWire(t)
	require.Equal(t, "POST /sources", w.Routes["source_create"])
	require.Equal(t, "PUT /sources/{source_id}", w.Routes["source_update"])
	require.Equal(t, "GET /sources/{source_id}/smt-chain", w.Routes["source_chain_read"])
	require.NotContains(t, w.Routes, "source_chain_write", "sources have no chain route; the chain rides in the source body")
	require.Equal(t, "/sources/src-1/smt-chain", smtChainPath("sources", "src-1"))

	schemas := w.OpenAPI.Components.Schemas
	rev := "r1"
	transforms := []SMTInstanceWrite{{Type: "filter", Name: "Drop", Enabled: true, Config: map[string]any{}}}
	predicates := []SMTPredicateWrite{}
	withChain := Source{Name: "pg", Connector: "postgresql", Config: map[string]any{}, Tags: nil,
		SMTChain: &SMTChainWrite{ExpectedRevision: &rev, Transforms: &transforms, Predicates: &predicates}}
	createKeys := append(jsonKeys(t, withChain), "created_from")
	assertDeclared(t, schemas["CreateSourceReq"].Properties, schemas["CreateSourceReq"].Required, createKeys)
	assertDeclared(t, schemas["UpdateSourceReq"].Properties, schemas["UpdateSourceReq"].Required, jsonKeys(t, withChain))
	require.Contains(t, jsonKeys(t, withChain), "smt_chain")
	require.NotContains(t, jsonKeys(t, Source{Name: "pg", Connector: "postgresql", Config: map[string]any{}}), "smt_chain",
		"an omitted chain keeps the stored one; the backend refuses null")

	var resp Source
	require.NoError(t, json.Unmarshal([]byte(`{"id":"src-1","name":"pg","connector":"postgresql","config":{},"smt_chain_receipt":{"incarnation":"inc","chain_revision":"rev-2","generation":3}}`), &resp))
	require.Equal(t, &SourceSMTChainReceipt{Incarnation: "inc", ChainRevision: "rev-2", Generation: 3}, resp.SMTChainReceipt)
	assertDeclared(t, schemas["SourceConnector"].Properties, nil, []string{"smt_chain_receipt"})
	receipt := schemas["SourceSmtChainReceipt"]
	require.JSONEq(t, "false", string(receipt.AdditionalProperties))
	assertDeclared(t, receipt.Properties, receipt.Required, jsonKeys(t, *resp.SMTChainReceipt))
}

// A source write that carries a chain is sent once even when the gateway
// fails, and its debug log never carries the chain, whose secret operations
// hold plaintext. A write without a chain keeps the released retry.
func TestSourceWrite_EmbeddedChainIsNotRetriedOrLogged(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	var bodies [][]byte
	failing := func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		bodies = append(bodies, body)
		return httpmock.NewStringResponse(http.StatusBadGateway, `{"detail":"gateway"}`), nil
	}
	httpmock.RegisterResponder(http.MethodPost, baseURL+"/sources?secret_returned=true&wait=false", failing)
	httpmock.RegisterResponder(http.MethodPut, baseURL+"/sources/src-1?secret_returned=true&wait=false", failing)

	transforms := []SMTInstanceWrite{{Type: "filter", Name: "Drop", Enabled: true, Config: map[string]any{},
		SecretOperations: []SMTSecretOperation{{Pointer: "/token", Operation: SMTSecretReplace, Value: "plaintext-secret"}}}}
	chained := Source{Name: "pg", Connector: "postgresql", Config: map[string]any{"database.hostname.user.defined": "db"},
		SMTChain: &SMTChainWrite{Transforms: &transforms}}

	var logs bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &logs)

	_, err := client.CreateSource(ctx, chained)
	require.Error(t, err)
	require.Len(t, bodies, 1, "a create carrying a chain must not be replayed")
	require.Contains(t, string(bodies[0]), `"smt_chain"`)

	_, err = client.UpdateSource(ctx, "src-1", chained)
	require.Error(t, err)
	require.Len(t, bodies, 2, "an update carrying a chain must not be replayed")

	require.NotContains(t, logs.String(), "plaintext-secret")
	require.NotContains(t, logs.String(), "smt_chain\\\":")
	require.Contains(t, logs.String(), "SMT chain: true (omitted from the log)")

	bodies = nil
	_, err = client.UpdateSource(ctx, "src-1", Source{Name: "pg", Connector: "postgresql", Config: map[string]any{}})
	require.Error(t, err)
	require.Greater(t, len(bodies), 1, "a write without a chain keeps the released retry")
	require.NotContains(t, string(bodies[0]), "smt_chain")
}
