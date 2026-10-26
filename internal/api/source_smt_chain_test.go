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
// the accepted revision under a key SourceSmtChainReceipt declares and
// requires.
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

	assertDeclared(t, schemas["SourceConnector"].Properties, nil, []string{"smt_chain_receipt"})
	receipt := schemas["SourceSmtChainReceipt"]
	require.JSONEq(t, "false", string(receipt.AdditionalProperties))
	require.Equal(t, []string{"chain_revision"}, receipt.Required)
}

// A source update that carries a chain sends the revision it read and gets
// back the fixture's receipt, which the client surfaces as the accepted
// chain revision.
func TestUpdateSource_ReturnsFixtureReceipt(t *testing.T) {
	w := loadSMTWire(t)
	var want SMTChainReceipt
	require.NoError(t, json.Unmarshal(w.SourceChainReceipt, &want))
	require.NotEmpty(t, want.ChainRevision)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	baseURL := "https://api.test.streamkap.com"
	var sent map[string]json.RawMessage
	httpmock.RegisterResponder(http.MethodPut, baseURL+"/sources/src-1?secret_returned=true&wait=false", func(req *http.Request) (*http.Response, error) {
		require.NoError(t, json.NewDecoder(req.Body).Decode(&sent))
		return httpmock.NewStringResponse(http.StatusOK, `{"id":"src-1","name":"pg","connector":"postgresql","config":{},"smt_chain_receipt":`+string(w.SourceChainReceipt)+`}`), nil
	})

	rev := "r1"
	transforms := []SMTInstanceWrite{{ID: "Filter1", Type: "filter", Name: "Drop", Enabled: true, Config: map[string]any{}}}
	resp, err := newTestClient(baseURL).UpdateSource(context.Background(), "src-1", Source{Name: "pg", Connector: "postgresql", Config: map[string]any{},
		SMTChain: &SMTChainWrite{ExpectedRevision: &rev, Transforms: &transforms}})
	require.NoError(t, err)
	require.Equal(t, &want, resp.SMTChainReceipt)

	var chain map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(sent["smt_chain"], &chain))
	require.JSONEq(t, `"r1"`, string(chain["expected_revision"]), "a source update's chain carries the revision it read")
}

// A connector write that carries a chain is sent once even when the gateway
// fails, and its debug log never carries the chain, whose secret operations
// hold plaintext. A write without a chain keeps the released retry.
func TestConnectorWrite_EmbeddedChainIsNotRetriedOrLogged(t *testing.T) {
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
	httpmock.RegisterResponder(http.MethodPost, baseURL+"/destinations?secret_returned=true&wait=false", failing)
	httpmock.RegisterResponder(http.MethodPut, baseURL+"/destinations/dest-1?secret_returned=true&wait=false", failing)

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

	_, err = client.CreateDestination(ctx, Destination{Name: "pg", Connector: "postgresql", Config: map[string]any{}, SMTChain: chained.SMTChain})
	require.Error(t, err)
	require.Len(t, bodies, 3, "a destination create carrying a chain must not be replayed")
	require.Contains(t, string(bodies[2]), `"smt_chain"`)

	_, err = client.UpdateDestination(ctx, "dest-1", Destination{Name: "pg", Connector: "postgresql", Config: map[string]any{}, SMTChain: chained.SMTChain})
	require.Error(t, err)
	require.Len(t, bodies, 4, "a destination update carrying a chain must not be replayed")
	require.Contains(t, string(bodies[3]), `"smt_chain"`)

	require.NotContains(t, logs.String(), "plaintext-secret")
	require.NotContains(t, logs.String(), "smt_chain\\\":")
	require.Contains(t, logs.String(), "SMT chain: true (omitted from the log)")

	bodies = nil
	_, err = client.UpdateSource(ctx, "src-1", Source{Name: "pg", Connector: "postgresql", Config: map[string]any{}})
	require.Error(t, err)
	require.Greater(t, len(bodies), 1, "a write without a chain keeps the released retry")
	require.NotContains(t, string(bodies[0]), "smt_chain")
}
