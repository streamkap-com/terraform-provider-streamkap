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

// smtWireFixtures copies the backend app/services/smt/fixtures/wire_fixtures.json at revision a81369efd10add73a6d133356c4da3a05a633af9 (wire_version 4), sha256 86ce03a7772c50c69e6b302bc32e901ceff89cd22a78f79554950bea53c91802.
const smtWireFixtures = "testdata/smt_wire_fixtures.json"

type smtWire struct {
	WireVersion          int               `json:"wire_version"`
	Routes               map[string]string `json:"routes"`
	CreateRequest        json.RawMessage   `json:"create_request"`
	UpdateRequest        json.RawMessage   `json:"update_request"`
	SecretOperationsReq  json.RawMessage   `json:"secret_operations_request"`
	ReadResponse         json.RawMessage   `json:"read_response"`
	SelectorReadResponse json.RawMessage   `json:"selector_read_response"`
	ConflictResponse     json.RawMessage   `json:"conflict_response"`
	ValidationResponse   json.RawMessage   `json:"validation_response"`
	AdoptionResponse     json.RawMessage   `json:"adoption_response"`
	ManagedOnlyResponse  json.RawMessage   `json:"adoption_managed_only_response"`
	UnavailableResponse  json.RawMessage   `json:"adoption_unavailable_response"`
	OpenAPI              struct {
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
	require.Equal(t, 4, w.WireVersion)
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
	require.Nil(t, create.ExpectedRevision, "a create carries no expected revision")
	require.Len(t, *create.Transforms, 2)
	require.Equal(t, "mask_field", (*create.Transforms)[1].Type)
	require.Equal(t, []string{"orders.email"}, toStrings((*create.Transforms)[1].Config["fields_include"]))
	require.Equal(t, []SMTSecretOperation{{Pointer: "/mask_salt", Operation: SMTSecretReplace, Value: "fixture-salt"}}, (*create.Transforms)[1].SecretOperations)

	var update SMTChainWrite
	require.NoError(t, json.Unmarshal(w.UpdateRequest, &update))
	require.NotNil(t, update.ExpectedRevision)
	require.Equal(t, []string{"instance-b", "instance-a", ""}, []string{(*update.Transforms)[0].ID, (*update.Transforms)[1].ID, (*update.Transforms)[2].ID})
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
		ID: "instance-c", Type: "contract_fixture_nested", Name: "Routes", Enabled: true,
		Config:           map[string]any{"enabled": true},
		SecretOperations: []SMTSecretOperation{{Pointer: "/routes/row-east/token", Operation: SMTSecretReplace, Value: "v"}, {Pointer: "/routes/row~0west~11/token", Operation: SMTSecretClear}},
	}}
	sent := SMTChainWrite{ExpectedRevision: &rev, Transforms: &sentTransforms}
	assertDeclared(t, schemas["SmtChainWriteRequest"].Properties, schemas["SmtChainWriteRequest"].Required, jsonKeys(t, sent))
	assertDeclared(t, schemas["SmtInstanceWrite"].Properties, schemas["SmtInstanceWrite"].Required, jsonKeys(t, (*sent.Transforms)[0]))
	assertDeclared(t, schemas["SmtSecretReplaceWire"].Properties, schemas["SmtSecretReplaceWire"].Required, jsonKeys(t, (*sent.Transforms)[0].SecretOperations[0]))
	assertDeclared(t, schemas["SmtSecretClearWire"].Properties, schemas["SmtSecretClearWire"].Required, jsonKeys(t, (*sent.Transforms)[0].SecretOperations[1]))
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
	for _, fixture := range []struct {
		body   json.RawMessage
		status string
	}{
		{w.AdoptionResponse, SMTAdoptionAvailable},
		{w.ManagedOnlyResponse, SMTAdoptionManagedOnly},
		{w.UnavailableResponse, SMTAdoptionUnavailable},
	} {
		var audit SMTChainAdoptionRead
		require.NoError(t, json.Unmarshal(fixture.body, &audit))
		require.Equal(t, fixture.status, audit.Status)
		if audit.Status == SMTAdoptionAvailable {
			require.NotEmpty(t, audit.ExpectedRevision)
			require.Equal(t, []string{"/mask_salt"}, audit.Transforms[0].ConfiguredSecretPaths)
			require.NotContains(t, audit.Transforms[0].Config, "mask_salt")
		} else {
			require.Empty(t, audit.ExpectedRevision)
		}
	}

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
	require.False(t, first.HasSelector())

	var selectorRead SMTChainRead
	require.NoError(t, json.Unmarshal(w.SelectorReadResponse, &selectorRead))
	require.Len(t, selectorRead.Desired.Transforms, 1)
	require.True(t, selectorRead.Desired.Transforms[0].HasSelector())
	require.Empty(t, selectorRead.Desired.Transforms[0].Config["fields_include"], "desired config remains rule-bearing, not compiled")
	require.NotEmpty(t, selectorRead.LastApplied.Transforms[0].Config["fields_include"], "last_applied records the compiled fields separately")
	// The effective order interleaves the managed slots with the user
	// instances; the provider manages only the user ones and keeps the
	// managed rows opaque.
	var userOrder []SMTOrderEntry
	for _, entry := range read.Desired.Order {
		if entry.Origin == "user" {
			userOrder = append(userOrder, entry)
		}
	}
	require.Equal(t, []SMTOrderEntry{{Origin: "user", Reference: "instance-b"}, {Origin: "user", Reference: "instance-a"}}, userOrder)
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

func TestSMTWire_HasSelector(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want bool
	}{
		{name: "absent", raw: `{}`},
		{name: "null", raw: `{"selector":null}`},
		{name: "configured", raw: `{"selector":{"streams":["orders"]}}`, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var instance SMTInstanceRead
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &instance))
			require.Equal(t, tc.want, instance.HasSelector())
		})
	}
}
