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

// smtWireFixtures is a byte-identical copy of the backend's
// app/services/smt/fixtures/wire_fixtures.json at revision
// 3bd029995a1e23931db0859fbb22f491abaa38c6 (wire_version 1; the file was
// last changed in d435615bd9a1ad8b266cf1b4a9aa2b85369194e6), sha256
// 3d90dce763b44ae1841e35ee63bada7efd433c1e12e2ec73ee8061f9c8f2f442.
const smtWireFixtures = "testdata/smt_wire_fixtures.json"

type smtWire struct {
	WireVersion         int               `json:"wire_version"`
	Routes              map[string]string `json:"routes"`
	CreateRequest       json.RawMessage   `json:"create_request"`
	UpdateRequest       json.RawMessage   `json:"update_request"`
	SecretOperationsReq json.RawMessage   `json:"secret_operations_request"`
	ReadResponse        json.RawMessage   `json:"read_response"`
	ConflictResponse    json.RawMessage   `json:"conflict_response"`
	ValidationResponse  json.RawMessage   `json:"validation_response"`
	OpenAPI             struct {
		Paths      map[string]json.RawMessage `json:"paths"`
		Components struct {
			Schemas map[string]struct {
				Properties           map[string]json.RawMessage `json:"properties"`
				Required             []string                   `json:"required"`
				AdditionalProperties *bool                      `json:"additionalProperties"`
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
	require.Equal(t, 1, w.WireVersion)
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

	var create SMTChainWrite
	require.NoError(t, json.Unmarshal(w.CreateRequest, &create))
	require.Nil(t, create.ExpectedRevision, "a create carries no expected revision")
	require.Len(t, create.Transforms, 2)
	require.Equal(t, "mask_field", create.Transforms[1].Type)
	require.Equal(t, []string{"orders.email"}, toStrings(create.Transforms[1].Config["fields_include"]))
	require.Equal(t, []SMTSecretOperation{{Pointer: "/mask_salt", Operation: SMTSecretReplace, Value: "fixture-salt"}}, create.Transforms[1].SecretOperations)

	var update SMTChainWrite
	require.NoError(t, json.Unmarshal(w.UpdateRequest, &update))
	require.NotNil(t, update.ExpectedRevision)
	require.Equal(t, []string{"instance-b", "instance-a", ""}, []string{update.Transforms[0].ID, update.Transforms[1].ID, update.Transforms[2].ID})
	require.False(t, update.Transforms[0].Enabled)

	var rotate SMTChainWrite
	require.NoError(t, json.Unmarshal(w.SecretOperationsReq, &rotate))
	require.NotNil(t, rotate.ExpectedRevision, "a rotation is a chain write under CAS")
	require.Equal(t, []SMTSecretOperation{
		{Pointer: "/routes/row-east/token", Operation: SMTSecretReplace, Value: "fixture-token"},
		{Pointer: "/routes/row~0west~11/token", Operation: SMTSecretClear},
	}, rotate.Transforms[0].SecretOperations)

	// What the provider sends: an explicit enabled flag, secret operations
	// inline, no predicate.
	rev := "r1"
	sent := SMTChainWrite{ExpectedRevision: &rev, Transforms: []SMTInstanceWrite{{
		ID: "instance-c", Type: "contract_fixture_nested", Name: "Routes", Enabled: true,
		Config:           map[string]any{"enabled": true},
		SecretOperations: []SMTSecretOperation{{Pointer: "/routes/row-east/token", Operation: SMTSecretReplace, Value: "v"}, {Pointer: "/routes/row~0west~11/token", Operation: SMTSecretClear}},
	}}}
	assertDeclared(t, schemas["SmtChainWriteRequest"].Properties, schemas["SmtChainWriteRequest"].Required, jsonKeys(t, sent))
	assertDeclared(t, schemas["SmtInstanceWrite"].Properties, schemas["SmtInstanceWrite"].Required, jsonKeys(t, sent.Transforms[0]))
	assertDeclared(t, schemas["SmtSecretReplaceWire"].Properties, schemas["SmtSecretReplaceWire"].Required, jsonKeys(t, sent.Transforms[0].SecretOperations[0]))
	assertDeclared(t, schemas["SmtSecretClearWire"].Properties, schemas["SmtSecretClearWire"].Required, jsonKeys(t, sent.Transforms[0].SecretOperations[1]))
	require.NotContains(t, jsonKeys(t, SMTInstanceWrite{}), "id", "a new instance omits id; null is rejected")
	require.NotContains(t, jsonKeys(t, SMTChainWrite{}), "expected_revision", "a create omits the expected revision")

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
	require.Equal(t, "instance-b", first.ID)
	require.Equal(t, "MaskField1", first.Alias)
	require.Equal(t, int64(1), first.SchemaVersion)
	require.Equal(t, "Mask email (renamed)", first.Name)
	require.False(t, first.Enabled)
	require.Equal(t, "SHA256_TRUNCATE", first.Config["mask_function"], "declared defaults are materialised on read")
	require.Equal(t, []string{"/mask_salt"}, first.ConfiguredSecretPaths)
	require.False(t, first.HasPredicate())
	require.Equal(t, []SMTOrderEntry{{Origin: "user", Reference: "instance-b"}, {Origin: "user", Reference: "instance-a"}}, read.Desired.Order)

	envelope := func(code, message string) json.RawMessage {
		raw, err := json.Marshal(map[string]any{"detail": map[string]any{"code": code, "message": message}})
		require.NoError(t, err)
		return raw
	}
	for _, tc := range []struct {
		name     string
		status   int
		body     json.RawMessage
		conflict bool
		code     string
		issues   int
	}{
		{"conflict", http.StatusConflict, w.ConflictResponse, true, "smt_revision_conflict", 1},
		{"lost create race", http.StatusConflict, envelope("smt_chain_exists", "A chain was created concurrently"), true, "smt_chain_exists", 0},
		{"revision required", http.StatusConflict, envelope("smt_expected_revision_required", "The chain exists; send expected_revision"), false, "smt_expected_revision_required", 0},
		{"connector deleting", http.StatusConflict, envelope("smt_connector_deleting", "Destination dest-1 is being deleted; its SMT chain can no longer be written"), false, "smt_connector_deleting", 0},
		{"validation", http.StatusUnprocessableEntity, w.ValidationResponse, false, "smt_validation_failed", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
				rw.Header().Set("Content-Type", "application/json")
				rw.WriteHeader(tc.status)
				_, _ = rw.Write(tc.body)
			}))
			defer srv.Close()
			client := NewClient(&Config{BaseURL: srv.URL}).(*streamkapAPI)
			_, err := client.PutSMTChain(context.Background(), "destinations", "dest-1", SMTChainWrite{})
			require.Error(t, err)
			require.Equal(t, tc.conflict, IsSMTRevisionConflict(err))
			env, ok := SMTErrorOf(err)
			require.True(t, ok, "%v", err)
			require.Equal(t, tc.code, env.Code)
			require.Len(t, env.Issues, tc.issues)
			require.Equal(t, tc.code == "smt_expected_revision_required", IsSMTRevisionRequired(err))
			require.Equal(t, tc.code == "smt_connector_deleting", IsSMTConnectorDeleting(err))
			require.False(t, IsSMTChainAbsent(err))
		})
	}

	// A wired route answers a structured 404 for a connector without a chain;
	// a plain 404 (unwired kind, feature off) is not that.
	absent := &APIError{StatusCode: http.StatusNotFound, DetailJSON: envelope("smt_chain_absent", "no chain")[len(`{"detail":`) : len(envelope("smt_chain_absent", "no chain"))-1]}
	require.True(t, IsSMTChainAbsent(absent))
	require.False(t, IsSMTChainAbsent(&APIError{StatusCode: http.StatusNotFound, Detail: "Not Found"}))
	require.False(t, IsSMTRevisionConflict(&APIError{StatusCode: http.StatusConflict, Detail: "conflict"}), "a bare 409 without the envelope is not classified")

	var validation struct {
		Detail SMTErrorEnvelope `json:"detail"`
	}
	require.NoError(t, json.Unmarshal(w.ValidationResponse, &validation))
	missing := validation.Detail.Issues[0]
	require.Equal(t, "smt_config_missing", missing.Code)
	require.Equal(t, "/transforms/1/config/routes/1/endpoint", missing.Pointer)
	require.NotNil(t, missing.InstanceID)
	require.Equal(t, "instance-c", *missing.InstanceID)
}
