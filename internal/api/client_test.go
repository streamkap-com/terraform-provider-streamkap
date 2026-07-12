// internal/api/client_test.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

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
		baseURL+"/sources?secret_returned=true&wait=false",
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
		baseURL+"/sources?secret_returned=true&wait=false",
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
		baseURL+"/sources?secret_returned=true&wait=false",
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
		baseURL+"/sources/source-to-delete?secret_returned=true&wait=false",
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

// TestDeleteSource_NotFound — deleting a source that is already gone succeeds.
// Terraform's desired end state after a destroy is absence; erroring because
// someone removed the resource in the UI first only forces `terraform state rm`.
func TestDeleteSource_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	// Mock not found response
	httpmock.RegisterResponder(
		http.MethodDelete,
		baseURL+"/sources/non-existent?secret_returned=true&wait=false",
		func(req *http.Request) (*http.Response, error) {
			errResponse := APIErrorResponse{
				Detail: "Source not found",
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()
	err := client.DeleteSource(ctx, "non-existent")

	require.NoError(t, err, "a 404 on delete means the record is already gone")
	assert.Equal(t, 1, httpmock.GetTotalCallCount(), "a 404 must not be retried")
}

// TestDelete_NotFoundIsIdempotent covers the remaining delete paths that share
// the helper: out-of-band deletion must not break `terraform destroy`.
func TestDelete_NotFoundIsIdempotent(t *testing.T) {
	baseURL := "https://api.test.streamkap.com"

	tests := []struct {
		name   string
		url    string
		status int
		detail string
		call   func(StreamkapAPI) error
	}{
		{
			name:   "destination",
			url:    baseURL + "/destinations/gone?secret_returned=true&wait=false",
			status: http.StatusNotFound,
			detail: "Destination 'gone' does not exist",
			call:   func(c StreamkapAPI) error { return c.DeleteDestination(context.Background(), "gone") },
		},
		{
			name:   "pipeline",
			url:    baseURL + "/pipelines/gone?secret_returned=true&wait=false",
			status: http.StatusNotFound,
			detail: "Pipeline not found",
			call:   func(c StreamkapAPI) error { return c.DeletePipeline(context.Background(), "gone") },
		},
		{
			name:   "transform",
			url:    baseURL + "/transforms?id=gone",
			status: http.StatusNotFound,
			detail: "Transform not found",
			call:   func(c StreamkapAPI) error { return c.DeleteTransform(context.Background(), "gone") },
		},
		{
			name:   "tag",
			url:    baseURL + "/tags/gone",
			status: http.StatusNotFound,
			detail: "Tag not found",
			call:   func(c StreamkapAPI) error { return c.DeleteTag(context.Background(), "gone") },
		},
		{
			name:   "kafka user",
			url:    baseURL + "/kafka-access/kafka-users/gone",
			status: http.StatusNotFound,
			detail: "Kafka user not found",
			call:   func(c StreamkapAPI) error { return c.DeleteKafkaUser(context.Background(), "gone") },
		},
		{
			name:   "client credential",
			url:    baseURL + "/auth/client-credentials/gone",
			status: http.StatusNotFound,
			detail: "Client credential not found",
			call:   func(c StreamkapAPI) error { return c.DeleteClientCredential(context.Background(), "gone") },
		},
		{
			// Some endpoints answer 400 rather than 404 for a missing record.
			name:   "destination reported gone with a 400",
			url:    baseURL + "/destinations/gone?secret_returned=true&wait=false",
			status: http.StatusBadRequest,
			detail: "Destination with id 'gone' does not exist",
			call:   func(c StreamkapAPI) error { return c.DeleteDestination(context.Background(), "gone") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			client := newTestClient(baseURL)
			httpmock.RegisterResponder(http.MethodDelete, tt.url,
				func(req *http.Request) (*http.Response, error) {
					return httpmock.NewJsonResponse(tt.status, APIErrorResponse{Detail: tt.detail})
				},
			)

			require.NoError(t, tt.call(client))
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
		})
	}
}

// TestDelete_RealErrorStillFails guards the idempotency helper against
// swallowing genuine failures.
func TestDelete_RealErrorStillFails(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	httpmock.RegisterResponder(
		http.MethodDelete,
		baseURL+"/sources/in-use?secret_returned=true&wait=false",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusConflict, APIErrorResponse{
				Detail: "Source is referenced by pipeline 'p-1'",
			})
		},
	)

	err := client.DeleteSource(context.Background(), "in-use")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "referenced by pipeline")
}

// TestDelete_EmptyBodyIsSuccess — /kafka-access and /auth/client-credentials
// answer a successful delete with 200 and no body. Decoding it reported io.EOF
// and turned every successful delete into a failure.
func TestDelete_EmptyBodyIsSuccess(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	httpmock.RegisterResponder(
		http.MethodDelete,
		baseURL+"/kafka-access/kafka-users/svc-user",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		},
	)
	httpmock.RegisterResponder(
		http.MethodDelete,
		baseURL+"/auth/client-credentials/cred-1",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(http.StatusOK, ""), nil
		},
	)

	ctx := context.Background()
	require.NoError(t, client.DeleteKafkaUser(ctx, "svc-user"))
	require.NoError(t, client.DeleteClientCredential(ctx, "cred-1"))
}

// TestCreateSource_AlreadyExists_NoAdoption — a name collision must fail with
// recovery guidance, never adopt the existing record. Adopting hands the new
// live state entry the deposed entry's backend id under
// `lifecycle { create_before_destroy = true }`, and destroying the deposed entry
// then deletes the live source. Pipelines and transforms already refuse this.
func TestCreateSource_AlreadyExists_NoAdoption(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/sources?secret_returned=true&wait=false",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusUnprocessableEntity, APIErrorResponse{
				Detail: "A source with name 'existing-source' already exists",
			})
		},
	)

	source, err := client.CreateSource(context.Background(), Source{
		Name:      "existing-source",
		Connector: "postgresql",
		Config:    map[string]any{},
	})

	require.Error(t, err, "a duplicate name must surface, never auto-adopt")
	assert.Nil(t, source)
	assert.Contains(t, err.Error(), "auto-adoption is unsafe")
	assert.Contains(t, err.Error(), "terraform import streamkap_source_")
	assert.Equal(t, 1, httpmock.GetTotalCallCount(), "must not call the list endpoint to adopt")
}

// TestCreateDestination_AlreadyExists_NoAdoption — see the source variant.
func TestCreateDestination_AlreadyExists_NoAdoption(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/destinations?secret_returned=true&wait=false",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusUnprocessableEntity, APIErrorResponse{
				Detail: "A destination with name 'existing-destination' already exists",
			})
		},
	)

	destination, err := client.CreateDestination(context.Background(), Destination{
		Name:      "existing-destination",
		Connector: "snowflake",
		Config:    map[string]any{},
	})

	require.Error(t, err)
	assert.Nil(t, destination)
	assert.Contains(t, err.Error(), "auto-adoption is unsafe")
	assert.Contains(t, err.Error(), "terraform import streamkap_destination_")
	assert.Equal(t, 1, httpmock.GetTotalCallCount(), "must not call the list endpoint to adopt")
}

// TestAPIError_CarriesStatusCode — the status must survive as data, not prose:
// retry, delete-idempotency and token renewal all key off it.
func TestAPIError_CarriesStatusCode(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/source-123?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			resp, err := httpmock.NewJsonResponse(http.StatusTooManyRequests, APIErrorResponse{
				Detail: "rate limit exceeded",
			})
			if err != nil {
				return nil, err
			}
			resp.Header.Set("X-Request-Id", "req-123")
			return resp, nil
		},
	)

	_, err := client.GetSource(context.Background(), "source-123")

	require.Error(t, err)
	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr), "doRequest must return an *APIError")
	assert.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	assert.Equal(t, "rate limit exceeded", apiErr.Detail)
	assert.Equal(t, "req-123", apiErr.RequestID)
	assert.Contains(t, err.Error(), "rate limit exceeded")
	assert.Contains(t, err.Error(), "req-123")
	assert.True(t, IsRetryableError(err), "a 429 is retryable whatever its detail says")
}

// TestAPIError_NonJSONBody keeps the status and a body snippet in the message —
// a bare "invalid character '<'" decoder error is a debugging dead end.
func TestAPIError_NonJSONBody(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/source-123?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(http.StatusBadGateway, "<html><body>502 Bad Gateway</body></html>"), nil
		},
	)

	_, err := client.GetSource(context.Background(), "source-123")

	require.Error(t, err)
	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	assert.Equal(t, http.StatusBadGateway, apiErr.StatusCode)
	assert.Contains(t, err.Error(), "502 Bad Gateway")
	assert.NotContains(t, err.Error(), "invalid character")
}

// TestTokenRenewal_On401RetriesWithFreshToken — the OAuth token used to be
// fetched once in provider.Configure and never renewed, so any apply outliving
// the token TTL failed every remaining resource with an opaque 401.
func TestTokenRenewal_On401RetriesWithFreshToken(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := NewClient(&Config{BaseURL: baseURL})

	authCalls := 0
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/auth/access-token",
		func(req *http.Request) (*http.Response, error) {
			authCalls++
			return httpmock.NewJsonResponse(http.StatusOK, Token{
				AccessToken: fmt.Sprintf("token-%d", authCalls),
				ExpiresIn:   3600,
			})
		},
	)

	// The provider's Configure flow: exchange credentials, then hand the token
	// to the client. The credentials stay behind so it can renew on its own.
	token, err := client.GetAccessToken("client-id", "client-secret")
	require.NoError(t, err)
	client.SetToken(token)
	require.Equal(t, 1, authCalls)

	var seenBearers []string
	var seenBodies []string
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/sources?secret_returned=true&wait=false",
		func(req *http.Request) (*http.Response, error) {
			seenBearers = append(seenBearers, req.Header.Get("Authorization"))
			body, readErr := io.ReadAll(req.Body)
			require.NoError(t, readErr)
			seenBodies = append(seenBodies, string(body))

			if req.Header.Get("Authorization") != "Bearer token-2" {
				return httpmock.NewJsonResponse(http.StatusUnauthorized, APIErrorResponse{
					Detail: "Token has expired",
				})
			}
			return httpmock.NewJsonResponse(http.StatusCreated, Source{
				ID:        "source-1",
				Name:      "renewed",
				Connector: "postgresql",
			})
		},
	)

	source, err := client.CreateSource(context.Background(), Source{
		Name:      "renewed",
		Connector: "postgresql",
		Config:    map[string]any{"hostname": "db.example.com"},
	})

	require.NoError(t, err, "a 401 must trigger one renewal and a replay, not fail the apply")
	require.NotNil(t, source)
	assert.Equal(t, "source-1", source.ID)
	assert.Equal(t, 2, authCalls, "exactly one renewal")
	assert.Equal(t, []string{"Bearer token-1", "Bearer token-2"}, seenBearers)
	require.Len(t, seenBodies, 2)
	assert.Equal(t, seenBodies[0], seenBodies[1], "the replayed request must carry the original body")
	assert.Contains(t, seenBodies[1], "db.example.com")
}

// TestTokenRenewal_ProactiveBeforeExpiry renews inside the skew window without
// waiting for a 401.
func TestTokenRenewal_ProactiveBeforeExpiry(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := NewClient(&Config{BaseURL: baseURL})

	authCalls := 0
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/auth/access-token",
		func(req *http.Request) (*http.Response, error) {
			authCalls++
			return httpmock.NewJsonResponse(http.StatusOK, Token{
				AccessToken: fmt.Sprintf("token-%d", authCalls),
				ExpiresIn:   3600,
			})
		},
	)

	_, err := client.GetAccessToken("client-id", "client-secret")
	require.NoError(t, err)
	// A token whose remaining lifetime is inside tokenRenewSkew: the next
	// request renews before sending rather than racing the expiry.
	client.SetToken(&Token{AccessToken: "about-to-expire", ExpiresIn: 5})

	var seenBearer string
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/source-1?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			seenBearer = req.Header.Get("Authorization")
			return httpmock.NewJsonResponse(http.StatusOK, GetSourceResponse{
				Total:  1,
				Result: []Source{{ID: "source-1"}},
			})
		},
	)

	_, err = client.GetSource(context.Background(), "source-1")
	require.NoError(t, err)
	assert.Equal(t, 2, authCalls, "the expiring token must be renewed before the request")
	assert.Equal(t, "Bearer token-2", seenBearer)
}

// TestTokenRenewal_SingleFlight — an expired token 401s every in-flight request
// at once. They must share one renewal, not storm the auth endpoint.
func TestTokenRenewal_SingleFlight(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := NewClient(&Config{BaseURL: baseURL})

	var mu sync.Mutex
	authCalls := 0
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/auth/access-token",
		func(req *http.Request) (*http.Response, error) {
			mu.Lock()
			authCalls++
			mu.Unlock()
			// Hold the lock-holder here so the other goroutines pile up on
			// renewMu — without single-flighting they would each re-auth.
			time.Sleep(20 * time.Millisecond)
			return httpmock.NewJsonResponse(http.StatusOK, Token{AccessToken: "fresh-token", ExpiresIn: 3600})
		},
	)

	_, err := client.GetAccessToken("client-id", "client-secret")
	require.NoError(t, err)
	client.SetToken(&Token{AccessToken: "stale-token", ExpiresIn: 3600})

	httpmock.RegisterResponder(
		http.MethodGet,
		`=~^https://api\.test\.streamkap\.com/sources/source-\d+\?secret_returned=true$`,
		func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Authorization") != "Bearer fresh-token" {
				return httpmock.NewJsonResponse(http.StatusUnauthorized, APIErrorResponse{Detail: "Token has expired"})
			}
			return httpmock.NewJsonResponse(http.StatusOK, GetSourceResponse{Total: 0, Result: []Source{}})
		},
	)

	const concurrent = 8
	var wg sync.WaitGroup
	errs := make([]error, concurrent)
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = client.GetSource(context.Background(), fmt.Sprintf("source-%d", i))
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "concurrent request %d", i)
	}
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, authCalls, "the initial exchange plus exactly one renewal shared by all 8 requests")
}

// TestTokenRenewal_NoCredentials_Surfaces401 — a client that was only handed a
// token (never any credentials) cannot renew; the 401 must reach the caller
// untouched rather than being replaced by a renewal failure.
func TestTokenRenewal_NoCredentials_Surfaces401(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestClient(baseURL)

	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/source-1?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusUnauthorized, APIErrorResponse{
				Detail: "Unauthorized: invalid or missing token",
			})
		},
	)

	_, err := client.GetSource(context.Background(), "source-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Unauthorized: invalid or missing token")
	assert.Equal(t, 1, httpmock.GetTotalCallCount(), "no credentials to renew with, so no auth call")
}

// TestTokenRenewal_BadCredentialsDoNotRecurse — a 401 from the auth endpoint
// itself must not trigger another renewal.
func TestTokenRenewal_BadCredentialsDoNotRecurse(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := NewClient(&Config{BaseURL: baseURL})

	authCalls := 0
	httpmock.RegisterResponder(
		http.MethodPost,
		baseURL+"/auth/access-token",
		func(req *http.Request) (*http.Response, error) {
			authCalls++
			if authCalls == 1 {
				return httpmock.NewJsonResponse(http.StatusOK, Token{AccessToken: "token-1", ExpiresIn: 3600})
			}
			return httpmock.NewJsonResponse(http.StatusUnauthorized, APIErrorResponse{Detail: "Invalid client credentials"})
		},
	)

	token, err := client.GetAccessToken("client-id", "revoked-secret")
	require.NoError(t, err)
	client.SetToken(token)

	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/source-1?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			return httpmock.NewJsonResponse(http.StatusUnauthorized, APIErrorResponse{Detail: "Token has expired"})
		},
	)

	_, err = client.GetSource(context.Background(), "source-1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Token has expired", "the original 401 stays the primary error")
	assert.Contains(t, err.Error(), "Invalid client credentials", "the renewal failure is reported alongside it")
	assert.Equal(t, 2, authCalls, "the failed renewal must not recurse")
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
		baseURL+"/sources/source-123?secret_returned=true&wait=false",
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
