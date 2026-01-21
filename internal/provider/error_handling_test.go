// internal/provider/error_handling_test.go
package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// newTestAPIClient creates a test API client for error handling tests
func newTestAPIClient(baseURL string) api.StreamkapAPI {
	client := api.NewClient(&api.Config{BaseURL: baseURL})
	client.SetToken(&api.Token{AccessToken: "test-token"})
	return client
}

// =============================================================================
// 401 Unauthorized Error Tests
// =============================================================================

// TestAPIError401_MissingToken tests that requests without a token receive 401 Unauthorized
func TestAPIError401_MissingToken(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	// Create client WITHOUT setting a token
	client := api.NewClient(&api.Config{BaseURL: baseURL})

	// Mock 401 response for unauthorized requests
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/test-source-id?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			// Verify no authorization header when token is not set
			assert.Empty(t, req.Header.Get("Authorization"))

			errResponse := api.APIErrorResponse{
				Detail: "Unauthorized: invalid or missing token",
			}
			return httpmock.NewJsonResponse(http.StatusUnauthorized, errResponse)
		},
	)

	ctx := context.Background()
	source, err := client.GetSource(ctx, "test-source-id")

	require.Error(t, err, "Expected error for 401 Unauthorized")
	assert.Nil(t, source, "Source should be nil when unauthorized")
	assert.Contains(t, err.Error(), "Unauthorized", "Error message should contain 'Unauthorized'")
}

// TestAPIError401_ExpiredToken tests that requests with expired token receive 401
func TestAPIError401_ExpiredToken(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	// Create client with an "expired" token
	client := api.NewClient(&api.Config{BaseURL: baseURL})
	client.SetToken(&api.Token{AccessToken: "expired-token"})

	// Mock 401 response for expired token
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/destinations/test-dest-id?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			// Token is set but API rejects it
			assert.Equal(t, "Bearer expired-token", req.Header.Get("Authorization"))

			errResponse := api.APIErrorResponse{
				Detail: "Token has expired. Please re-authenticate.",
			}
			return httpmock.NewJsonResponse(http.StatusUnauthorized, errResponse)
		},
	)

	ctx := context.Background()
	destination, err := client.GetDestination(ctx, "test-dest-id")

	require.Error(t, err, "Expected error for expired token")
	assert.Nil(t, destination, "Destination should be nil when unauthorized")
	assert.Contains(t, err.Error(), "expired", "Error message should indicate token expiration")
}

// TestAPIError401_InvalidCredentials tests authentication failure with invalid client credentials
func TestAPIError401_InvalidCredentials(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := api.NewClient(&api.Config{BaseURL: baseURL})

	// Mock 401 response for invalid credentials
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/auth/access-token",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Invalid client credentials",
			}
			return httpmock.NewJsonResponse(http.StatusUnauthorized, errResponse)
		},
	)

	token, err := client.GetAccessToken("invalid-client-id", "invalid-secret")

	require.Error(t, err, "Expected error for invalid credentials")
	assert.Nil(t, token, "Token should be nil when credentials are invalid")
	assert.Contains(t, err.Error(), "Invalid client credentials", "Error message should indicate credential failure")
}

// TestAPIError401_CreateSource tests 401 on create operations
func TestAPIError401_CreateSource(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := api.NewClient(&api.Config{BaseURL: baseURL})
	// No token set

	source := api.Source{
		Name:      "test-source",
		Connector: "postgresql",
		Config:    map[string]any{"hostname": "localhost"},
	}

	// Mock 401 response
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/sources?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Authentication required",
			}
			return httpmock.NewJsonResponse(http.StatusUnauthorized, errResponse)
		},
	)

	ctx := context.Background()
	result, err := client.CreateSource(ctx, source)

	require.Error(t, err, "Expected error for 401 on create")
	assert.Nil(t, result, "Result should be nil when unauthorized")
	assert.Contains(t, err.Error(), "Authentication required")
}

// TestAPIError401_UpdateDestination tests 401 on update operations
func TestAPIError401_UpdateDestination(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := api.NewClient(&api.Config{BaseURL: baseURL})
	client.SetToken(&api.Token{AccessToken: "revoked-token"})

	destination := api.Destination{
		ID:        "dest-123",
		Name:      "updated-destination",
		Connector: "snowflake",
		Config:    map[string]any{},
	}

	// Mock 401 response for revoked token
	httpmock.RegisterResponder(
		http.MethodPut,
		baseURL+"/destinations/dest-123?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Token has been revoked",
			}
			return httpmock.NewJsonResponse(http.StatusUnauthorized, errResponse)
		},
	)

	ctx := context.Background()
	result, err := client.UpdateDestination(ctx, "dest-123", destination)

	require.Error(t, err, "Expected error for 401 on update")
	assert.Nil(t, result, "Result should be nil when unauthorized")
	assert.Contains(t, err.Error(), "revoked")
}

// TestAPIError401_DeleteTransform tests 401 on delete operations
func TestAPIError401_DeleteTransform(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := api.NewClient(&api.Config{BaseURL: baseURL})
	// No token set

	// Mock 401 response - transforms use DELETE /transforms?id={id} (query param, not path param)
	httpmock.RegisterResponder(
		http.MethodDelete,
		baseURL+"/transforms?id=transform-123",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Unauthorized: Please provide valid credentials",
			}
			return httpmock.NewJsonResponse(http.StatusUnauthorized, errResponse)
		},
	)

	ctx := context.Background()
	err := client.DeleteTransform(ctx, "transform-123")

	require.Error(t, err, "Expected error for 401 on delete")
	assert.Contains(t, err.Error(), "Unauthorized")
}

// =============================================================================
// 404 Not Found Error Tests
// =============================================================================

// TestAPIError404_GetSource tests 404 when reading a non-existent source
func TestAPIError404_GetSource(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	// Mock 404 response for non-existent source
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/non-existent-id?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Source not found: non-existent-id",
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()
	source, err := client.GetSource(ctx, "non-existent-id")

	require.Error(t, err, "Expected error for 404 Not Found")
	assert.Nil(t, source, "Source should be nil when not found")
	assert.Contains(t, err.Error(), "not found", "Error message should indicate resource not found")
}

// TestAPIError404_GetDestination tests 404 when reading a non-existent destination
func TestAPIError404_GetDestination(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	// Mock 404 response
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/destinations/deleted-destination?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Destination with id 'deleted-destination' does not exist",
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()
	destination, err := client.GetDestination(ctx, "deleted-destination")

	require.Error(t, err, "Expected error for 404 Not Found")
	assert.Nil(t, destination, "Destination should be nil when not found")
	assert.Contains(t, err.Error(), "does not exist")
}

// TestAPIError404_GetPipeline tests 404 when reading a non-existent pipeline
func TestAPIError404_GetPipeline(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	// Mock 404 response
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/pipelines/missing-pipeline?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Pipeline not found",
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()
	pipeline, err := client.GetPipeline(ctx, "missing-pipeline")

	require.Error(t, err, "Expected error for 404 Not Found")
	assert.Nil(t, pipeline, "Pipeline should be nil when not found")
	assert.Contains(t, err.Error(), "not found")
}

// TestAPIError404_GetTransform tests 404 when reading a non-existent transform
func TestAPIError404_GetTransform(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	// Mock 404 response - transforms use ?unwind_topics=false query param
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/transforms/unknown-transform?secret_returned=true&unwind_topics=false",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Transform not found: unknown-transform",
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()
	transform, err := client.GetTransform(ctx, "unknown-transform")

	require.Error(t, err, "Expected error for 404 Not Found")
	assert.Nil(t, transform, "Transform should be nil when not found")
	assert.Contains(t, err.Error(), "not found")
}

// TestAPIError404_DeleteSource tests 404 when deleting a non-existent source
func TestAPIError404_DeleteSource(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	// Mock 404 response for delete of non-existent source
	httpmock.RegisterResponder(
		http.MethodDelete,
		baseURL+"/sources/already-deleted?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Source 'already-deleted' not found",
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()
	err := client.DeleteSource(ctx, "already-deleted")

	require.Error(t, err, "Expected error for 404 on delete")
	assert.Contains(t, err.Error(), "not found")
}

// TestAPIError404_UpdateSource tests 404 when updating a non-existent source
func TestAPIError404_UpdateSource(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	source := api.Source{
		ID:        "non-existent-source",
		Name:      "updated-name",
		Connector: "postgresql",
		Config:    map[string]any{},
	}

	// Mock 404 response for update of non-existent source
	httpmock.RegisterResponder(
		http.MethodPut,
		baseURL+"/sources/non-existent-source?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Cannot update: source 'non-existent-source' not found",
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()
	result, err := client.UpdateSource(ctx, "non-existent-source", source)

	require.Error(t, err, "Expected error for 404 on update")
	assert.Nil(t, result, "Result should be nil when resource not found")
	assert.Contains(t, err.Error(), "not found")
}

// TestAPIError404_DeleteDestination tests 404 when deleting a non-existent destination
func TestAPIError404_DeleteDestination(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	// Mock 404 response
	httpmock.RegisterResponder(
		http.MethodDelete,
		baseURL+"/destinations/ghost-dest?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Destination 'ghost-dest' does not exist",
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()
	err := client.DeleteDestination(ctx, "ghost-dest")

	require.Error(t, err, "Expected error for 404 on delete")
	assert.Contains(t, err.Error(), "does not exist")
}

// TestAPIError404_GetTag tests 404 when reading a non-existent tag
func TestAPIError404_GetTag(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	// Mock 404 response - tags use GET /tags?tag_ids={id}
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/tags?tag_ids=missing-tag",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Tag not found: missing-tag",
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()
	tag, err := client.GetTag(ctx, "missing-tag")

	require.Error(t, err, "Expected error for 404 Not Found")
	assert.Nil(t, tag, "Tag should be nil when not found")
	assert.Contains(t, err.Error(), "not found")
}

// TestAPIError404_GetTopic tests 404 when reading a non-existent topic
func TestAPIError404_GetTopic(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	// Mock 404 response - topics use GET /topics/{id} (no secret_returned)
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/topics/non-existent-topic",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Topic 'non-existent-topic' not found",
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()
	topic, err := client.GetTopic(ctx, "non-existent-topic")

	require.Error(t, err, "Expected error for 404 Not Found")
	assert.Nil(t, topic, "Topic should be nil when not found")
	assert.Contains(t, err.Error(), "not found")
}

// =============================================================================
// Error Message Propagation Tests
// =============================================================================

// TestErrorMessagePropagation_DetailField verifies the API error detail field is properly extracted
func TestErrorMessagePropagation_DetailField(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	testCases := []struct {
		name           string
		statusCode     int
		errorDetail    string
		expectedInErr  string
	}{
		{
			name:          "401 with detailed message",
			statusCode:    http.StatusUnauthorized,
			errorDetail:   "Your API token has expired. Please generate a new token from the dashboard.",
			expectedInErr: "token has expired",
		},
		{
			name:          "404 with resource identifier",
			statusCode:    http.StatusNotFound,
			errorDetail:   "Resource with ID 'abc-123' was not found in the current tenant",
			expectedInErr: "abc-123",
		},
		{
			name:          "401 insufficient permissions",
			statusCode:    http.StatusUnauthorized,
			errorDetail:   "Insufficient permissions: requires 'admin' role to perform this operation",
			expectedInErr: "Insufficient permissions",
		},
		{
			name:          "404 with suggested action",
			statusCode:    http.StatusNotFound,
			errorDetail:   "Source 'my-source' not found. It may have been deleted. Check the Streamkap dashboard.",
			expectedInErr: "deleted",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			httpmock.Reset()

			httpmock.RegisterResponder(
				http.MethodGet,
				baseURL+"/sources/test-id?secret_returned=true",
				func(req *http.Request) (*http.Response, error) {
					errResponse := api.APIErrorResponse{
						Detail: tc.errorDetail,
					}
					return httpmock.NewJsonResponse(tc.statusCode, errResponse)
				},
			)

			ctx := context.Background()
			_, err := client.GetSource(ctx, "test-id")

			require.Error(t, err, "Expected error for status %d", tc.statusCode)
			assert.Contains(t, err.Error(), tc.expectedInErr,
				"Error message should contain: %s, got: %s", tc.expectedInErr, err.Error())
		})
	}
}

// TestErrorMessagePropagation_401vs404 verifies different error types have distinct messages
func TestErrorMessagePropagation_401vs404(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"

	// Test 401 error
	client401 := api.NewClient(&api.Config{BaseURL: baseURL})
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/source-1?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Authentication failed: invalid token",
			}
			return httpmock.NewJsonResponse(http.StatusUnauthorized, errResponse)
		},
	)

	ctx := context.Background()
	_, err401 := client401.GetSource(ctx, "source-1")
	require.Error(t, err401)

	httpmock.Reset()

	// Test 404 error
	client404 := newTestAPIClient(baseURL)
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/source-2?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Source not found",
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	_, err404 := client404.GetSource(ctx, "source-2")
	require.Error(t, err404)

	// Verify error messages are distinct
	assert.NotEqual(t, err401.Error(), err404.Error(),
		"401 and 404 errors should have different messages")
	assert.Contains(t, err401.Error(), "Authentication",
		"401 error should mention authentication")
	assert.Contains(t, err404.Error(), "not found",
		"404 error should mention 'not found'")
}
