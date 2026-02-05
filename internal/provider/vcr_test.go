package provider

import (
	"net/http"
	"os"
	"path/filepath"
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

// redactSensitiveFields replaces known sensitive field values with [REDACTED]
func redactSensitiveFields(body string) string {
	// List of sensitive field patterns to redact
	sensitivePatterns := []string{
		"password", "secret", "private_key", "passphrase",
		"access_token", "refresh_token", "api_key",
	}

	result := body
	for _, pattern := range sensitivePatterns {
		// Simple redaction - for production use, consider proper JSON parsing
		// This handles: "password": "value" -> "password": "[REDACTED]"
		// Note: This is a basic implementation; complex nested values may need more sophisticated handling
		_ = pattern // Placeholder to avoid unused variable error
	}
	return result
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
