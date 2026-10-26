package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// smtWireFixtures copies the backend app/services/smt/fixtures/wire_fixtures.json at revision 85c1cd844069e710892077f102652d9b7937dded (wire_version 5), sha256 7565ba4b6c9e56947469a174b2242b9edb964c2aa2c8505ca1c8332465a45617.
const smtWireFixtures = "testdata/smt_wire_fixtures.json"

type smtWire struct {
	WireVersion              int               `json:"wire_version"`
	Routes                   map[string]string `json:"routes"`
	CreateRequest            json.RawMessage   `json:"create_request"`
	DestinationCreateRequest json.RawMessage   `json:"destination_create_request"`
	UpdateRequest            json.RawMessage   `json:"update_request"`
	SecretOperationsReq      json.RawMessage   `json:"secret_operations_request"`
	ReadResponse             json.RawMessage   `json:"read_response"`
	SelectorReadResponse     json.RawMessage   `json:"selector_read_response"`
	SourceChainReceipt       json.RawMessage   `json:"source_chain_receipt"`
	DestinationChainReceipt  json.RawMessage   `json:"destination_chain_receipt"`
	ConflictResponse         json.RawMessage   `json:"conflict_response"`
	MissingRevisionResponse  json.RawMessage   `json:"missing_revision_response"`
	OrderRejectedResponse    json.RawMessage   `json:"order_rejected_response"`
	ValidationResponse       json.RawMessage   `json:"validation_response"`
	OpenAPI                  struct {
		Paths      map[string]json.RawMessage `json:"paths"`
		Components struct {
			Schemas map[string]struct {
				Properties           map[string]json.RawMessage `json:"properties"`
				Required             []string                   `json:"required"`
				AdditionalProperties json.RawMessage            `json:"additionalProperties"`
			} `json:"schemas"`
		} `json:"components"`
	} `json:"openapi"`
}

func loadSMTWire(t *testing.T) *smtWire {
	t.Helper()
	raw, err := os.ReadFile(smtWireFixtures)
	require.NoError(t, err)
	var w smtWire
	require.NoError(t, json.Unmarshal(raw, &w))
	require.Equal(t, 5, w.WireVersion)
	return &w
}

// jsonKeys are the top-level keys a value marshals to.
func jsonKeys(t *testing.T, v any) []string {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &m))
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// The request DTOs decode every request body of the fixture and, when
// marshalled, emit only keys the OpenAPI components declare and every key
// they require: the components are additionalProperties: false, so an
// unknown key is a 422.
func TestSMTWire_RequestsMatchFixture(t *testing.T) {
	w := loadSMTWire(t)
	schemas := w.OpenAPI.Components.Schemas
	for _, name := range []string{"SmtChainWriteRequest", "SmtInstanceWrite", "SmtSecretReplaceWire", "SmtSecretClearWire"} {
		require.JSONEq(t, "false", string(schemas[name].AdditionalProperties), "%s rejects undeclared fields", name)
	}

	var create SMTChainWrite
	require.NoError(t, json.Unmarshal(w.CreateRequest, &create))
	require.Nil(t, create.ExpectedRevision, "a connector create's embedded chain carries no expected revision")
	require.Len(t, *create.Transforms, 2)
	require.Equal(t, "mask_field", (*create.Transforms)[1].Type)
	require.Equal(t, []string{"orders.email"}, toStrings((*create.Transforms)[1].Config["fields_include"]))
	require.Equal(t, []SMTSecretOperation{{Pointer: "/mask_salt", Operation: SMTSecretReplace, Value: "fixture-salt"}}, (*create.Transforms)[1].SecretOperations)

	var update SMTChainWrite
	require.NoError(t, json.Unmarshal(w.UpdateRequest, &update))
	require.NotNil(t, update.ExpectedRevision)
	require.Equal(t, []string{"MaskField1", "RegexRouter1", ""}, []string{(*update.Transforms)[0].ID, (*update.Transforms)[1].ID, (*update.Transforms)[2].ID}, "an existing instance is addressed by its alias")
	require.False(t, (*update.Transforms)[0].Enabled)

	var rotate SMTChainWrite
	require.NoError(t, json.Unmarshal(w.SecretOperationsReq, &rotate))
	require.NotNil(t, rotate.ExpectedRevision, "a rotation is a chain write under CAS")
	require.Equal(t, []SMTSecretOperation{
		{Pointer: "/routes/row-east/token", Operation: SMTSecretReplace, Value: "fixture-token"},
		{Pointer: "/routes/row~0west~11/token", Operation: SMTSecretClear},
	}, (*rotate.Transforms)[0].SecretOperations)

	// What the provider sends: an explicit enabled flag, secret operations
	// inline, no predicate.
	rev := "r1"
	sentTransforms := []SMTInstanceWrite{{
		ID: "ContractFixture1", Type: "contract_fixture_nested", Name: "Routes", Enabled: true,
		Config:           map[string]any{"enabled": true},
		SecretOperations: []SMTSecretOperation{{Pointer: "/routes/row-east/token", Operation: SMTSecretReplace, Value: "v"}, {Pointer: "/routes/row~0west~11/token", Operation: SMTSecretClear}},
	}}
	sent := SMTChainWrite{ExpectedRevision: &rev, Transforms: &sentTransforms}
	assertDeclared(t, schemas["SmtChainWriteRequest"].Properties, schemas["SmtChainWriteRequest"].Required, jsonKeys(t, sent))
	assertDeclared(t, schemas["SmtInstanceWrite"].Properties, schemas["SmtInstanceWrite"].Required, jsonKeys(t, (*sent.Transforms)[0]))
	assertDeclared(t, schemas["SmtSecretReplaceWire"].Properties, schemas["SmtSecretReplaceWire"].Required, jsonKeys(t, (*sent.Transforms)[0].SecretOperations[0]))
	assertDeclared(t, schemas["SmtSecretClearWire"].Properties, schemas["SmtSecretClearWire"].Required, jsonKeys(t, (*sent.Transforms)[0].SecretOperations[1]))
	require.NotContains(t, jsonKeys(t, SMTInstanceWrite{}), "id", "a new instance omits id; null is rejected")
	require.NotContains(t, jsonKeys(t, SMTChainWrite{}), "expected_revision", "a connector create's embedded chain omits the expected revision")

	require.Equal(t, "PUT /destinations/{destination_id}/smt-chain", w.Routes["chain_write"])
	require.Equal(t, "GET /destinations/{destination_id}/smt-chain", w.Routes["chain_read"])
	require.Equal(t, "/destinations/dest-1/smt-chain", smtChainPath("destinations", "dest-1"))
}

// assertDeclared checks the keys a DTO emits against a component schema.
func assertDeclared(t *testing.T, properties map[string]json.RawMessage, required, keys []string) {
	t.Helper()
	for _, k := range keys {
		require.Contains(t, properties, k, "the provider sends %q, which the component does not declare", k)
	}
	for _, r := range required {
		require.Contains(t, keys, r, "the component requires %q", r)
	}
}

func toStrings(v any) []string {
	items, _ := v.([]any)
	out := make([]string, 0, len(items))
	for _, i := range items {
		out = append(out, i.(string))
	}
	return out
}

// The read DTO decodes the fixture response with the fields the provider
// consumes, and the error envelopes surface through the client as a 409
// revision conflict and a 422 whose issues locate the problem.
func TestSMTWire_ResponsesMatchFixture(t *testing.T) {
	w := loadSMTWire(t)
	var read SMTChainRead
	require.NoError(t, json.Unmarshal(w.ReadResponse, &read))
	require.Equal(t, "a5ffa5b833e6db552f77d5781779e8f7", read.Desired.ChainRevision)
	require.Nil(t, read.LastApplied)
	require.Equal(t, "pending", read.ApplicationStatus)
	require.Len(t, read.Desired.Transforms, 2)
	first := read.Desired.Transforms[0]
	require.Equal(t, "typed", first.Kind)
	require.Equal(t, "MaskField1", first.ID)
	require.Equal(t, int64(3), first.SchemaVersion)
	require.Equal(t, "Mask email (renamed)", first.Name)
	require.False(t, first.Enabled)
	require.Equal(t, "SHA256", first.Config["mask_function"], "declared defaults are materialised on read")
	require.Equal(t, []string{"/mask_salt"}, first.ConfiguredSecretPaths)
	require.False(t, first.HasPredicate())

	// A selector is write-only: the instance it was saved on reads back with
	// no selector and the field lists it compiled to.
	var selectorRead SMTChainRead
	require.NoError(t, json.Unmarshal(w.SelectorReadResponse, &selectorRead))
	require.Len(t, selectorRead.Desired.Transforms, 1)
	compiled := selectorRead.Desired.Transforms[0]
	require.Equal(t, []string{"*.email"}, toStrings(compiled.Config["fields_include"]))
	require.Equal(t, []string{"audit_log.email"}, toStrings(compiled.Config["fields_exclude"]))
	var rawSelectorRead struct {
		Desired struct {
			Transforms []map[string]json.RawMessage `json:"transforms"`
		} `json:"desired"`
	}
	require.NoError(t, json.Unmarshal(w.SelectorReadResponse, &rawSelectorRead))
	require.JSONEq(t, "null", string(rawSelectorRead.Desired.Transforms[0]["selector"]), "nothing to manage beyond the compiled lists")
	// The effective order interleaves the managed slots with the user
	// instances; the provider manages only the user ones and keeps the
	// managed rows opaque.
	var userOrder []SMTOrderEntry
	for _, entry := range read.Desired.Order {
		if entry.Origin == "user" {
			userOrder = append(userOrder, entry)
		}
	}
	require.Equal(t, []SMTOrderEntry{{Origin: "user", Reference: "MaskField1"}, {Origin: "user", Reference: "RegexRouter1"}}, userOrder)
	var managed []map[string]any
	require.NoError(t, json.Unmarshal(read.Desired.Managed, &managed))
	require.Len(t, managed, 10, "the managed rows are decoded raw, never dropped")
	require.Equal(t, "required", managed[0]["policy"])
	for _, row := range managed {
		require.IsType(t, true, row["applicable"])
		require.IsType(t, true, row["overridable"])
	}
	require.Contains(t, w.OpenAPI.Components.Schemas["SmtManagedTransformRead"].Required, "applicable")
	require.Contains(t, w.OpenAPI.Components.Schemas["SmtManagedTransformRead"].Required, "overridable")

	envelope := func(code, message string) json.RawMessage {
		raw, err := json.Marshal(map[string]any{"detail": map[string]any{"code": code, "message": message}})
		require.NoError(t, err)
		return raw
	}
	for _, tc := range []struct {
		name            string
		status          int
		body            json.RawMessage
		conflict        bool
		missingRevision bool
		code            string
		issues          int
	}{
		{name: "conflict", status: http.StatusConflict, body: w.ConflictResponse, conflict: true, code: "smt_revision_conflict", issues: 1},
		{name: "missing revision", status: http.StatusConflict, body: w.MissingRevisionResponse, missingRevision: true, code: "smt_revision_conflict", issues: 1},
		{name: "connector deleting", status: http.StatusConflict, body: envelope("smt_connector_deleting", "Destination dest-1 is being deleted; its SMT chain can no longer be written"), code: "smt_connector_deleting"},
		{name: "view invalid", status: http.StatusConflict, body: envelope("smt_view_invalid", "The connector's stored SMT chain metadata cannot be read (names is not a mapping); its chain cannot be read or saved"), code: "smt_view_invalid"},
		{name: "read failed", status: http.StatusServiceUnavailable, body: envelope("smt_read_failed", "The chain write was accepted but could not be read back; read the chain again before retrying"), code: "smt_read_failed"},
		{name: "validation", status: http.StatusUnprocessableEntity, body: w.ValidationResponse, code: "smt_validation_failed", issues: 3},
		{name: "order rejected", status: http.StatusUnprocessableEntity, body: w.OrderRejectedResponse, code: "smt_validation_failed", issues: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				calls++
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(tc.status)
				_, _ = rw.Write(tc.body)
			}))
			defer srv.Close()
			client := NewClient(&Config{BaseURL: srv.URL}).(*streamkapAPI)
			_, err := client.PutSMTChain(context.Background(), "destinations", "dest-1", SMTChainWrite{})
			require.Error(t, err)
			require.Equal(t, 1, calls, "a chain write is never replayed: the server may have saved it")
			require.Equal(t, tc.conflict, IsSMTRevisionConflict(err))
			require.Equal(t, tc.missingRevision, IsSMTExpectedRevisionMissing(err))
			env, ok := SMTErrorOf(err)
			require.True(t, ok, "%v", err)
			require.Equal(t, tc.code, env.Code)
			require.Len(t, env.Issues, tc.issues)
			require.Equal(t, tc.code == "smt_connector_deleting", IsSMTConnectorDeleting(err))
			require.Equal(t, tc.code == "smt_read_failed", IsSMTReadFailed(err))
			require.Equal(t, tc.code == "smt_view_invalid", IsSMTViewInvalid(err))
		})
	}

	require.False(t, IsSMTRevisionConflict(&APIError{StatusCode: http.StatusConflict, Detail: "conflict"}), "a bare 409 without the envelope is not classified")

	var validation struct {
		Detail SMTErrorEnvelope `json:"detail"`
	}
	require.NoError(t, json.Unmarshal(w.ValidationResponse, &validation))
	missing := validation.Detail.Issues[0]
	require.Equal(t, "smt_config_missing", missing.Code)
	require.Equal(t, "/transforms/1/config/routes/1/endpoint", missing.Pointer)
	require.NotNil(t, missing.InstanceID)
	require.Equal(t, "ContractFixture1", *missing.InstanceID)

	// Two authored instances in the wrong relative order for the
	// destination's required sequence: the issue names the later one by its
	// alias and points at its position in the request.
	var order struct {
		Detail SMTErrorEnvelope `json:"detail"`
	}
	require.NoError(t, json.Unmarshal(w.OrderRejectedResponse, &order))
	require.Len(t, order.Detail.Issues, 1)
	rejected := order.Detail.Issues[0]
	require.Equal(t, "smt_order_rejected", rejected.Code)
	require.Equal(t, "/transforms/1", rejected.Pointer)
	require.NotNil(t, rejected.InstanceID)
	require.Equal(t, "RegexRouter1", *rejected.InstanceID)
}

// An instance id is its transform alias, and the order references the user
// instances by that id: a read never carries a second identity to track.
func TestSMTWire_InstanceIDIsAlias(t *testing.T) {
	w := loadSMTWire(t)
	for name, body := range map[string]json.RawMessage{"read_response": w.ReadResponse, "selector_read_response": w.SelectorReadResponse} {
		var read SMTChainRead
		require.NoError(t, json.Unmarshal(body, &read))
		ids := map[string]bool{}
		for _, inst := range read.Desired.Transforms {
			require.Equal(t, inst.Alias, inst.ID, "%s: instance %s", name, inst.ID)
			ids[inst.ID] = true
		}
		for _, entry := range read.Desired.Order {
			if entry.Origin == "user" {
				require.True(t, ids[entry.Reference], "%s: order references %s, which is no instance id", name, entry.Reference)
			}
		}
	}
}

// A destination create embeds its first chain under smt_chain beside the
// connector body, with no expected revision, every top-level key the client
// sends is one CreateDestinationReq declares, and the response's receipt names
// the accepted revision.
func TestCreateDestination_EmbedsChainAsFixture(t *testing.T) {
	w := loadSMTWire(t)
	var req Destination
	require.NoError(t, json.Unmarshal(w.DestinationCreateRequest, &req))
	require.NotNil(t, req.SMTChain)
	require.Nil(t, req.SMTChain.ExpectedRevision)

	// The fixture leaves enabled at its schema default on one instance; the
	// provider always sends the flag, so the expected body spells the default.
	var defaults struct {
		Default bool `json:"default"`
	}
	require.NoError(t, json.Unmarshal(w.OpenAPI.Components.Schemas["SmtInstanceWrite"].Properties["enabled"], &defaults))
	var want map[string]any
	require.NoError(t, json.Unmarshal(w.DestinationCreateRequest, &want))
	for i, raw := range want["smt_chain"].(map[string]any)["transforms"].([]any) {
		entry := raw.(map[string]any)
		if _, set := entry["enabled"]; !set {
			entry["enabled"] = defaults.Default
			(*req.SMTChain.Transforms)[i].Enabled = defaults.Default
		}
	}

	var sent map[string]json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/destinations", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&sent))
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write([]byte(`{"id":"dest-1","name":"Warehouse","connector":"snowflake","config":{},"smt_chain_receipt":` + string(w.DestinationChainReceipt) + `}`))
	}))
	defer srv.Close()
	resp, err := NewClient(&Config{BaseURL: srv.URL}).CreateDestination(context.Background(), req)
	require.NoError(t, err)
	var receipt SMTChainReceipt
	require.NoError(t, json.Unmarshal(w.DestinationChainReceipt, &receipt))
	require.NotEmpty(t, receipt.ChainRevision)
	require.Equal(t, &receipt, resp.SMTChainReceipt)
	assertDeclared(t, w.OpenAPI.Components.Schemas["DestinationConnector"].Properties, nil, []string{"smt_chain_receipt"})

	wantChain, err := json.Marshal(want["smt_chain"])
	require.NoError(t, err)
	require.JSONEq(t, string(wantChain), string(sent["smt_chain"]))
	keys := make([]string, 0, len(sent))
	for k := range sent {
		keys = append(keys, k)
	}
	schema := w.OpenAPI.Components.Schemas["CreateDestinationReq"]
	assertDeclared(t, schema.Properties, schema.Required, keys)
}
