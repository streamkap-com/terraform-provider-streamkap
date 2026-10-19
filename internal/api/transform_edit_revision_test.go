package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateTransformCarriesAndReturnsEditRevision(t *testing.T) {
	wantExpected := "11111111111111111111111111111111"
	wantFresh := "22222222222222222222222222222222"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests++
		require.Equal(t, http.MethodPut, req.Method)
		require.Equal(t, "/transforms", req.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.Equal(t, wantExpected, body["expected_revision"])
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":             "transform-1",
			"name":           "edited",
			"transform_type": "map_filter",
			"config":         map[string]any{"transforms.name": "edited"},
			"tags":           []string{},
			"edit_revision":  wantFresh,
		}))
	}))
	defer server.Close()

	payload := UpdateTransformRequest{
		Transform:        "map_filter",
		Config:           map[string]any{"transforms.name": "edited"},
		Tags:             []string{},
		ExpectedRevision: &wantExpected,
	}

	got, err := NewClient(&Config{BaseURL: server.URL}).UpdateTransform(context.Background(), "transform-1", payload)
	require.NoError(t, err)
	require.Equal(t, 1, requests)
	require.Equal(t, wantFresh, *got.EditRevision)
}

func TestUpdateTransformIncludesNullExpectedRevisionForLegacyState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.Contains(t, body, "expected_revision")
		require.Nil(t, body["expected_revision"])
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"id":             "transform-1",
			"name":           "legacy",
			"transform_type": "map_filter",
			"config":         map[string]any{},
			"tags":           []string{},
			"edit_revision":  "11111111111111111111111111111111",
		}))
	}))
	defer server.Close()

	payload := UpdateTransformRequest{Transform: "map_filter", Config: map[string]any{}, Tags: []string{}, ExpectedRevision: nil}

	_, err := NewClient(&Config{BaseURL: server.URL}).UpdateTransform(context.Background(), "transform-1", payload)
	require.NoError(t, err)
}

func TestUpdateTransformAcceptsNullRevisionOnlyForLegacyNoop(t *testing.T) {
	legacyRevision := "0123456789abcdef0123456789abcdef"
	tests := []struct {
		name             string
		expectedRevision *string
		wantError        bool
	}{
		{name: "legacy null authority", expectedRevision: nil},
		{name: "versioned authority", expectedRevision: &legacyRevision, wantError: true},
	}
	for _, tc := range tests {
		t.Run("configuration "+tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				require.Contains(t, body, "expected_revision")
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"id":             "transform-1",
					"name":           "legacy",
					"transform_type": "map_filter",
					"config":         map[string]any{},
					"tags":           []string{},
					"edit_revision":  nil,
				}))
			}))
			defer server.Close()

			_, err := NewClient(&Config{BaseURL: server.URL}).UpdateTransform(context.Background(), "transform-1", UpdateTransformRequest{
				Transform:        "map_filter",
				Config:           map[string]any{},
				Tags:             []string{},
				ExpectedRevision: tc.expectedRevision,
			})
			if tc.wantError {
				require.ErrorContains(t, err, "invalid edit revision")
			} else {
				require.NoError(t, err)
			}
		})
	}

	for _, tc := range tests {
		t.Run("implementation "+tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				require.Contains(t, body, "expected_revision")
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"transform_id":   "transform-1",
					"version_id":     "version-7",
					"implementation": body["implementation"],
					"edit_revision":  nil,
				}))
			}))
			defer server.Close()

			_, err := NewClient(&Config{BaseURL: server.URL}).UpdateTransformImplementationDetails(context.Background(), "transform-1", UpdateTransformImplementationRequest{
				ExpectedRevision: tc.expectedRevision,
				Implementation:   map[string]any{"language": "JAVASCRIPT", "value_transform": "return value"},
			})
			if tc.wantError {
				require.ErrorContains(t, err, "invalid edit revision")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdateTransformImplementationCarriesAndReturnsEditRevision(t *testing.T) {
	wantExpected := "22222222222222222222222222222222"
	wantFresh := "33333333333333333333333333333333"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, http.MethodPut, req.Method)
		require.Equal(t, "/transforms/transform-1/implementation_details", req.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.Equal(t, wantExpected, body["expected_revision"])
		require.NotContains(t, body, "edit_revision")
		require.NotContains(t, body, "version_id")
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"transform_id":   "transform-1",
			"version_id":     "version-7",
			"implementation": body["implementation"],
			"edit_revision":  wantFresh,
		}))
	}))
	defer server.Close()

	details := UpdateTransformImplementationRequest{
		ExpectedRevision: &wantExpected,
		Implementation:   map[string]any{"language": "JAVASCRIPT", "value_transform": "return value"},
	}

	got, err := NewClient(&Config{BaseURL: server.URL}).UpdateTransformImplementationDetails(context.Background(), "transform-1", details)
	require.NoError(t, err)
	require.Equal(t, "version-7", got.VersionID)
	require.Equal(t, wantFresh, *got.EditRevision)
}

func TestTransformReadsPreserveNullableEditRevision(t *testing.T) {
	wantRevision := "44444444444444444444444444444444"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch req.URL.Path {
		case "/transforms/transform-1":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"result": []map[string]any{{
					"id":             "transform-1",
					"name":           "example",
					"transform_type": "map_filter",
					"config":         map[string]any{},
					"edit_revision":  nil,
				}},
			}))
		case "/transforms/transform-1/implementation_details":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"transform_id":  "transform-1",
				"impl_versions": map[string]any{},
				"edit_revision": wantRevision,
			}))
		default:
			t.Fatalf("unexpected request: %s", req.URL.String())
		}
	}))
	defer server.Close()

	client := NewClient(&Config{BaseURL: server.URL})
	transform, err := client.GetTransform(context.Background(), "transform-1")
	require.NoError(t, err)
	require.Nil(t, transform.EditRevision)

	details, err := client.GetTransformImplementationDetails(context.Background(), "transform-1")
	require.NoError(t, err)
	require.Equal(t, wantRevision, *details.EditRevision)
}

func TestTransformEditRevisionValidation(t *testing.T) {
	valid := "0123456789abcdef0123456789abcdef"
	require.NoError(t, ValidateTransformEditRevision(&valid, false))
	require.NoError(t, ValidateTransformEditRevision(nil, true))

	for _, invalid := range []*string{
		nil,
		ptrString("0123456789abcdef0123456789abcde"),
		ptrString("0123456789abcdef0123456789abcdeg"),
		ptrString("0123456789ABCDEF0123456789ABCDEF"),
	} {
		require.Error(t, ValidateTransformEditRevision(invalid, false))
	}
}

func TestUpdateTransformDoesNotReplayUncertainFailure(t *testing.T) {
	revision := "0123456789abcdef0123456789abcdef"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"detail": "transform commit outcome is unknown"}))
	}))
	defer server.Close()

	_, err := NewClient(&Config{BaseURL: server.URL}).UpdateTransform(context.Background(), "transform-1", UpdateTransformRequest{
		Transform:        "map_filter",
		Config:           map[string]any{},
		Tags:             []string{},
		ExpectedRevision: &revision,
	})

	require.Error(t, err)
	require.Equal(t, 1, requests)
}

func TestUpdateTransformImplementationDoesNotReplayUncertainFailure(t *testing.T) {
	revision := "0123456789abcdef0123456789abcdef"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"detail": "transform commit outcome is unknown"}))
	}))
	defer server.Close()

	_, err := NewClient(&Config{BaseURL: server.URL}).UpdateTransformImplementationDetails(context.Background(), "transform-1", UpdateTransformImplementationRequest{
		ExpectedRevision: &revision,
		Implementation:   map[string]any{"language": "JAVASCRIPT", "value_transform": "return value"},
	})

	require.Error(t, err)
	require.Equal(t, 1, requests)
}

func TestTransformUpdatesRejectMalformedAcceptedRevision(t *testing.T) {
	revision := "0123456789abcdef0123456789abcdef"
	for _, tc := range []struct {
		name string
		call func(StreamkapAPI) error
	}{
		{
			name: "configuration",
			call: func(client StreamkapAPI) error {
				_, err := client.UpdateTransform(context.Background(), "transform-1", UpdateTransformRequest{
					Transform:        "map_filter",
					Config:           map[string]any{},
					Tags:             []string{},
					ExpectedRevision: &revision,
				})
				return err
			},
		},
		{
			name: "implementation",
			call: func(client StreamkapAPI) error {
				_, err := client.UpdateTransformImplementationDetails(context.Background(), "transform-1", UpdateTransformImplementationRequest{
					ExpectedRevision: &revision,
					Implementation:   map[string]any{"language": "JAVASCRIPT", "value_transform": "return value"},
				})
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"id":             "transform-1",
					"transform_id":   "transform-1",
					"name":           "example",
					"transform_type": "map_filter",
					"config":         map[string]any{},
					"tags":           []string{},
					"version_id":     "version-7",
					"edit_revision":  "NOT-A-REVISION",
				}))
			}))
			defer server.Close()

			err := tc.call(NewClient(&Config{BaseURL: server.URL}))

			require.ErrorContains(t, err, "invalid edit revision")
		})
	}
}

func ptrString(value string) *string {
	return &value
}
