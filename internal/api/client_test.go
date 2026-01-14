// internal/api/client_test.go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a configured test client
func newTestClient(baseURL string) StreamkapAPI {
	client := NewClient(&Config{BaseURL: baseURL})
	client.SetToken(&Token{AccessToken: "test-token"})
	return client
}

// TestGetSource_Success tests successful retrieval of a source
func TestGetSource_Success(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	expectedSource := Source{
		ID:        "source-123",
		Name:      "test-postgresql",
		Connector: "postgresql",
		Config: map[string]any{
			"hostname": "db.example.com",
			"port":     "5432",
			"database": "testdb",
		},
	}

	// Mock the API response
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/source-123?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			// Verify authorization header is set
			assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
			assert.Equal(t, "application/json", req.Header.Get("Accept"))

			response := GetSourceResponse{
				Total:    1,
				PageSize: 10,
				Page:     1,
				Result:   []Source{expectedSource},
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()
	source, err := client.GetSource(ctx, "source-123")

	require.NoError(t, err)
	require.NotNil(t, source)
	assert.Equal(t, expectedSource.ID, source.ID)
	assert.Equal(t, expectedSource.Name, source.Name)
	assert.Equal(t, expectedSource.Connector, source.Connector)
	assert.Equal(t, expectedSource.Config["hostname"], source.Config["hostname"])

	// Verify the request was made
	assert.Equal(t, 1, httpmock.GetTotalCallCount())
}

// TestGetSource_NotFound tests retrieval of a non-existent source
func TestGetSource_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	// Mock empty result (source not found)
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/non-existent-id?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			response := GetSourceResponse{
				Total:    0,
				PageSize: 10,
				Page:     1,
				Result:   []Source{},
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()
	source, err := client.GetSource(ctx, "non-existent-id")

	require.NoError(t, err)
	assert.Nil(t, source, "Expected nil source when not found")

	assert.Equal(t, 1, httpmock.GetTotalCallCount())
}

// TestGetSource_APIError tests handling of API errors
func TestGetSource_APIError(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	// Mock API error response
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/source-123?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := APIErrorResponse{
				Detail: "Internal server error",
			}
			return httpmock.NewJsonResponse(http.StatusInternalServerError, errResponse)
		},
	)

	ctx := context.Background()
	source, err := client.GetSource(ctx, "source-123")

	require.Error(t, err)
	assert.Nil(t, source)
	assert.Contains(t, err.Error(), "Internal server error")
}

// TestCreateSource_Success tests successful creation of a source
func TestCreateSource_Success(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	inputSource := Source{
		Name:      "new-postgresql",
		Connector: "postgresql",
		Config: map[string]any{
			"hostname": "db.example.com",
			"port":     "5432",
			"database": "newdb",
		},
	}

	// Mock the API response
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/sources?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			// Verify authorization header
			assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

			// Verify the request body contains created_from
			var reqBody map[string]any
			err := json.NewDecoder(req.Body).Decode(&reqBody)
			require.NoError(t, err)
			assert.Equal(t, "terraform", reqBody["created_from"])
			assert.Equal(t, "new-postgresql", reqBody["name"])
			assert.Equal(t, "postgresql", reqBody["connector"])

			// Return created source with ID
			createdSource := Source{
				ID:        "created-source-456",
				Name:      inputSource.Name,
				Connector: inputSource.Connector,
				Config:    inputSource.Config,
			}
			return httpmock.NewJsonResponse(http.StatusCreated, createdSource)
		},
	)

	ctx := context.Background()
	source, err := client.CreateSource(ctx, inputSource)

	require.NoError(t, err)
	require.NotNil(t, source)
	assert.Equal(t, "created-source-456", source.ID)
	assert.Equal(t, inputSource.Name, source.Name)
	assert.Equal(t, inputSource.Connector, source.Connector)

	assert.Equal(t, 1, httpmock.GetTotalCallCount())
}

// TestCreateSource_ValidationError tests handling of validation errors
func TestCreateSource_ValidationError(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	inputSource := Source{
		Name:      "", // Invalid: empty name
		Connector: "postgresql",
		Config:    map[string]any{},
	}

	// Mock validation error response
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/sources?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := APIErrorResponse{
				Detail: "Validation error: name is required",
			}
			return httpmock.NewJsonResponse(http.StatusBadRequest, errResponse)
		},
	)

	ctx := context.Background()
	source, err := client.CreateSource(ctx, inputSource)

	require.Error(t, err)
	assert.Nil(t, source)
	assert.Contains(t, err.Error(), "Validation error: name is required")
}

// TestCreateSource_Unauthorized tests handling of authentication errors
func TestCreateSource_Unauthorized(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	// Create client without token
	client := NewClient(&Config{BaseURL: baseURL})

	inputSource := Source{
		Name:      "test-source",
		Connector: "postgresql",
		Config:    map[string]any{},
	}

	// Mock unauthorized response
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/sources?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			// Verify no authorization header when token is not set
			assert.Empty(t, req.Header.Get("Authorization"))

			errResponse := APIErrorResponse{
				Detail: "Unauthorized: invalid or missing token",
			}
			return httpmock.NewJsonResponse(http.StatusUnauthorized, errResponse)
		},
	)

	ctx := context.Background()
	source, err := client.CreateSource(ctx, inputSource)

	require.Error(t, err)
	assert.Nil(t, source)
	assert.Contains(t, err.Error(), "Unauthorized")
}

// TestDeleteSource_Success tests successful deletion of a source
func TestDeleteSource_Success(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	// Mock successful delete response
	httpmock.RegisterResponder(
		http.MethodDelete,
		baseURL+"/sources/source-to-delete?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			// Verify authorization header
			assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))

			// Return the deleted source (API returns the deleted resource)
			deletedSource := Source{
				ID:        "source-to-delete",
				Name:      "deleted-source",
				Connector: "postgresql",
				Config:    map[string]any{},
			}
			return httpmock.NewJsonResponse(http.StatusOK, deletedSource)
		},
	)

	ctx := context.Background()
	err := client.DeleteSource(ctx, "source-to-delete")

	require.NoError(t, err)
	assert.Equal(t, 1, httpmock.GetTotalCallCount())
}

// TestDeleteSource_NotFound tests deletion of a non-existent source
func TestDeleteSource_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	// Mock not found response
	httpmock.RegisterResponder(
		http.MethodDelete,
		baseURL+"/sources/non-existent?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := APIErrorResponse{
				Detail: "Source not found",
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()
	err := client.DeleteSource(ctx, "non-existent")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Source not found")
}

// TestUpdateSource_Success tests successful update of a source
func TestUpdateSource_Success(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	updatePayload := Source{
		ID:        "source-123",
		Name:      "updated-postgresql",
		Connector: "postgresql",
		Config: map[string]any{
			"hostname": "new-db.example.com",
			"port":     "5432",
			"database": "updateddb",
		},
	}

	// Mock successful update response
	httpmock.RegisterResponder(
		http.MethodPut,
		baseURL+"/sources/source-123?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			// Verify authorization header
			assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
			assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

			// Return updated source
			return httpmock.NewJsonResponse(http.StatusOK, updatePayload)
		},
	)

	ctx := context.Background()
	source, err := client.UpdateSource(ctx, "source-123", updatePayload)

	require.NoError(t, err)
	require.NotNil(t, source)
	assert.Equal(t, "updated-postgresql", source.Name)
	assert.Equal(t, "new-db.example.com", source.Config["hostname"])

	assert.Equal(t, 1, httpmock.GetTotalCallCount())
}

// TestGetAccessToken_Success tests successful token retrieval
func TestGetAccessToken_Success(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := NewClient(&Config{BaseURL: baseURL})

	expectedToken := Token{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		Expires:      "2025-01-01T00:00:00Z",
		ExpiresIn:    3600,
	}

	// Mock token endpoint
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/auth/access-token",
		func(req *http.Request) (*http.Response, error) {
			// Verify request body
			var reqBody GetAccessTokenRequest
			err := json.NewDecoder(req.Body).Decode(&reqBody)
			require.NoError(t, err)
			assert.Equal(t, "test-client-id", reqBody.ClientID)
			assert.Equal(t, "test-secret", reqBody.Secret)

			return httpmock.NewJsonResponse(http.StatusOK, expectedToken)
		},
	)

	token, err := client.GetAccessToken("test-client-id", "test-secret")

	require.NoError(t, err)
	require.NotNil(t, token)
	assert.Equal(t, expectedToken.AccessToken, token.AccessToken)
	assert.Equal(t, expectedToken.RefreshToken, token.RefreshToken)
	assert.Equal(t, expectedToken.ExpiresIn, token.ExpiresIn)
}

// TestGetAccessToken_InvalidCredentials tests authentication failure
func TestGetAccessToken_InvalidCredentials(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := NewClient(&Config{BaseURL: baseURL})

	// Mock authentication failure
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/auth/access-token",
		func(req *http.Request) (*http.Response, error) {
			errResponse := APIErrorResponse{
				Detail: "Invalid client credentials",
			}
			return httpmock.NewJsonResponse(http.StatusUnauthorized, errResponse)
		},
	)

	token, err := client.GetAccessToken("wrong-client-id", "wrong-secret")

	require.Error(t, err)
	assert.Nil(t, token)
	assert.Contains(t, err.Error(), "Invalid client credentials")
}

// TestSetToken tests the SetToken method
func TestSetToken(t *testing.T) {
	baseURL := "https://api.test.streamkap.com"
	client := NewClient(&Config{BaseURL: baseURL})

	// Initially no token
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Test that token is set correctly
	testToken := &Token{
		AccessToken:  "my-access-token",
		RefreshToken: "my-refresh-token",
		ExpiresIn:    3600,
	}
	client.SetToken(testToken)

	// Verify token is used in requests
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/test-id?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "Bearer my-access-token", req.Header.Get("Authorization"))
			response := GetSourceResponse{
				Total:    0,
				PageSize: 10,
				Page:     1,
				Result:   []Source{},
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()
	_, err := client.GetSource(ctx, "test-id")
	require.NoError(t, err)
}

// TestNewClient tests the client constructor
func TestNewClient(t *testing.T) {
	cfg := &Config{BaseURL: "https://api.streamkap.com"}
	client := NewClient(cfg)

	assert.NotNil(t, client)
	// Interface compliance is verified at compile time via NewClient's return type (StreamkapAPI)
}
