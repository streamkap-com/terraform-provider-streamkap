package provider

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

// newVCRClient creates an HTTP client that records/replays HTTP interactions.
// Set UPDATE_CASSETTES=1 to record new cassettes.
//
// Cassette Versioning Strategy:
// - Cassettes are named by feature: source_postgresql_crud, destination_snowflake_crud
// - When API changes, re-record affected cassettes with UPDATE_CASSETTES=1
// - Commit updated cassettes with descriptive message about API change
// - Old cassettes can be kept in git history for reference
//
//nolint:unused // Reserved for integration tests when cassettes are recorded
func newVCRClient(t *testing.T, cassetteName string) (*http.Client, func()) {
	t.Helper()

	cassettePath := filepath.Join("testdata", "cassettes", cassetteName)

	mode := recorder.ModeReplayOnly
	if os.Getenv("UPDATE_CASSETTES") != "" {
		mode = recorder.ModeRecordOnly
		t.Logf("Recording cassette: %s", cassettePath)
	}

	// Hook to redact sensitive data before saving cassettes
	redactHook := func(i *cassette.Interaction) error {
		// Remove auth headers
		delete(i.Request.Headers, "Authorization")
		delete(i.Request.Headers, "X-Api-Key")

		// Redact sensitive fields in request body
		i.Request.Body = redactSensitiveFields(i.Request.Body)

		// Redact sensitive fields in response body
		i.Response.Body = redactSensitiveFields(i.Response.Body)

		return nil
	}

	r, err := recorder.New(
		cassettePath,
		recorder.WithMode(mode),
		recorder.WithHook(redactHook, recorder.BeforeSaveHook),
	)
	if err != nil {
		t.Fatalf("Failed to create recorder: %v", err)
	}

	cleanup := func() {
		if err := r.Stop(); err != nil {
			t.Errorf("Failed to stop recorder: %v", err)
		}
	}

	return r.GetDefaultClient(), cleanup
}

// sensitivePatterns lists field name substrings whose values should be redacted from cassettes.
// Patterns are matched case-insensitively via strings.Contains.
// Keep patterns specific enough to avoid redacting non-sensitive config fields
// (e.g., "auth" alone would match "authorization_type" which is a non-sensitive enum).
var sensitivePatterns = []string{
	"password", "secret", "private_key", "passphrase",
	"access_token", "refresh_token", "api_key",
	"pem", "credential",
	"client_secret", "auth_key", "auth_token",
	"bearer", "connection_string",
}

// redactSensitiveFields parses body as JSON and replaces values of sensitive keys with [REDACTED].
// Falls back to the original body if it is not valid JSON.
func redactSensitiveFields(body string) string {
	if strings.TrimSpace(body) == "" {
		return body
	}

	var data any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return body // not JSON — return unchanged
	}

	redactRecursive(data)

	out, err := json.Marshal(data)
	if err != nil {
		return body
	}
	return string(out)
}

// redactRecursive walks a parsed JSON value in-place and replaces sensitive string values.
func redactRecursive(v any) {
	switch val := v.(type) {
	case map[string]any:
		for key, child := range val {
			if isSensitiveKey(key) {
				val[key] = "[REDACTED]"
			} else {
				redactRecursive(child)
			}
		}
	case []any:
		for _, item := range val {
			redactRecursive(item)
		}
	}
}

// isSensitiveKey returns true if any sensitive pattern is a substring of the key (case-insensitive).
func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	return false
}

// Integration tests using recorded cassettes
// These run on every PR without needing API credentials

func TestIntegration_SourceCRUD(t *testing.T) {
	// Skip until cassettes are recorded
	// To record cassettes: UPDATE_CASSETTES=1 go test -v -run 'TestIntegration_' ./internal/provider/...
	t.Skip("Integration tests will be implemented when cassettes are recorded")

	if os.Getenv("TF_ACC") == "" && os.Getenv("UPDATE_CASSETTES") == "" {
		t.Skip("Set TF_ACC=1 or UPDATE_CASSETTES=1 to run integration tests")
	}

	// Initialize VCR client
	_, cleanup := newVCRClient(t, "source_crud")
	defer cleanup()

	// TODO: Implement actual CRUD test using VCR client
}
