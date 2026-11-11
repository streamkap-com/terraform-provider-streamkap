package transform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// releasedRequest is one request the provider sent: method, path with query,
// and the decoded JSON body (nil when there was none).
type releasedRequest struct {
	Target string
	Body   map[string]any
}

// releasedBackend answers like a backend without transform edit revisions: no
// response carries `edit_revision`, and the generic PUT stores whatever
// implementation the body holds. It records every request in order.
type releasedBackend struct {
	t              *testing.T
	mu             sync.Mutex
	requests       []releasedRequest
	implementation map[string]any
}

func newReleasedBackend(t *testing.T, implementation map[string]any) (*releasedBackend, *httptest.Server) {
	b := &releasedBackend{t: t, implementation: implementation}
	server := httptest.NewServer(http.HandlerFunc(b.serve))
	t.Cleanup(server.Close)
	return b, server
}

func (b *releasedBackend) serve(w http.ResponseWriter, req *http.Request) {
	var body map[string]any
	if req.Body != nil && req.ContentLength != 0 {
		require.NoError(b.t, json.NewDecoder(req.Body).Decode(&body))
	}
	b.mu.Lock()
	b.requests = append(b.requests, releasedRequest{Target: req.Method + " " + req.URL.RequestURI(), Body: body})
	b.mu.Unlock()

	entity := func(config any) map[string]any {
		return map[string]any{
			"id":             "transform-1",
			"name":           "atomic-transform",
			"transform_type": "map_filter",
			"config":         config,
			"implementation": b.implementation,
			"tags":           []string{},
		}
	}
	w.Header().Set("Content-Type", "application/json")
	switch {
	case req.Method == http.MethodPost && req.URL.Path == "/transforms":
		b.implementation = map[string]any{}
		require.NoError(b.t, json.NewEncoder(w).Encode(entity(body["config"])))
	case req.Method == http.MethodGet && req.URL.Path == "/transforms/transform-1":
		require.NoError(b.t, json.NewEncoder(w).Encode(map[string]any{"result": []any{entity(map[string]any{})}}))
	case req.Method == http.MethodPut && req.URL.Path == "/transforms":
		implementation, _ := body["implementation"].(map[string]any)
		b.implementation = implementation
		require.NoError(b.t, json.NewEncoder(w).Encode(entity(body["config"])))
	case req.Method == http.MethodPut && req.URL.Path == "/transforms/transform-1/implementation_details":
		if implementation, ok := body["implementation"].(map[string]any); ok {
			b.implementation = implementation
		}
		require.NoError(b.t, json.NewEncoder(w).Encode(map[string]any{
			"transform_id":   "transform-1",
			"version_id":     "2",
			"implementation": b.implementation,
		}))
	case req.Method == http.MethodGet && req.URL.Path == "/transforms/transform-1/job_status":
		require.NoError(b.t, json.NewEncoder(w).Encode(map[string]any{"status": "UNKNOWN"}))
	default:
		b.t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
	}
}

func (b *releasedBackend) targets() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	targets := make([]string, 0, len(b.requests))
	for _, request := range b.requests {
		targets = append(targets, request.Target)
	}
	return targets
}

func (b *releasedBackend) body(t *testing.T, target string) map[string]any {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, request := range b.requests {
		if request.Target == target {
			return request.Body
		}
	}
	t.Fatalf("no %s request was sent; sent %v", target, b.requests)
	return nil
}

func bodyKeys(body map[string]any) []string {
	keys := make([]string, 0, len(body))
	for key := range body {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

const (
	releasedCreate         = "POST /transforms?secret_returned=true"
	releasedGet            = "GET /transforms/transform-1?secret_returned=true&unwind_topics=false"
	releasedUpdate         = "PUT /transforms?secret_returned=true"
	releasedImplementation = "PUT /transforms/transform-1/implementation_details"
	releasedJobStatus      = "GET /transforms/transform-1/job_status"
)

// The request sequences and bodies below are what the v3.1.0 provider sends,
// so an upgrade changes nothing on a backend without edit revisions.
func TestReleasedBackendCreateSendsReleasedProviderRequests(t *testing.T) {
	for _, tc := range []struct {
		name               string
		implementation     *string
		wantRequests       []string
		wantImplementation any
	}{
		{
			name:               "implementation",
			implementation:     ptr(`{"language":"JAVASCRIPT","value_transform":"return value"}`),
			wantRequests:       []string{releasedCreate, releasedImplementation, releasedJobStatus},
			wantImplementation: map[string]any{"language": "JAVASCRIPT", "value_transform": "return value"},
		},
		{name: "empty object", implementation: ptr(`{}`), wantRequests: []string{releasedCreate, releasedImplementation, releasedJobStatus}},
		{name: "json null", implementation: ptr(`null`), wantRequests: []string{releasedCreate, releasedImplementation, releasedJobStatus}},
		{name: "omitted", implementation: nil, wantRequests: []string{releasedCreate, releasedJobStatus}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend, server := newReleasedBackend(t, nil)
			r, req, resp := createMapFilterRequest(t, api.NewClient(&api.Config{BaseURL: server.URL}), tc.implementation)

			r.Create(context.Background(), req, &resp)

			require.False(t, resp.Diagnostics.HasError(), "create diagnostics: %v", resp.Diagnostics)
			require.Equal(t, tc.wantRequests, backend.targets())
			require.Equal(t, []string{"config", "created_from", "tags", "transform"}, bodyKeys(backend.body(t, releasedCreate)))
			if slices.Contains(tc.wantRequests, releasedImplementation) {
				implementationBody := backend.body(t, releasedImplementation)
				if tc.wantImplementation == nil {
					require.Equal(t, map[string]any{"transform_id": "transform-1"}, implementationBody)
				} else {
					require.Equal(t, map[string]any{"transform_id": "transform-1", "implementation": tc.wantImplementation}, implementationBody)
				}
			}
			var revision types.String
			require.False(t, resp.State.GetAttribute(context.Background(), path.Root("edit_revision"), &revision).HasError())
			require.True(t, revision.IsNull())
		})
	}
}

func TestReleasedBackendUpdateSendsReleasedProviderRequests(t *testing.T) {
	stored := map[string]any{"language": "JAVASCRIPT", "value_transform": "return value"}
	oldImplementation := `{"language":"JAVASCRIPT","value_transform":"return value"}`
	for _, tc := range []struct {
		name               string
		newImplementation  string
		wantRequests       []string
		wantImplementation map[string]any
	}{
		{
			name:               "configuration only",
			newImplementation:  oldImplementation,
			wantRequests:       []string{releasedGet, releasedUpdate, releasedJobStatus},
			wantImplementation: stored,
		},
		{
			name:               "implementation change",
			newImplementation:  `{"language":"JAVASCRIPT","value_transform":"return {...value, updated: true}"}`,
			wantRequests:       []string{releasedGet, releasedUpdate, releasedImplementation, releasedJobStatus},
			wantImplementation: map[string]any{"language": "JAVASCRIPT", "value_transform": "return {...value, updated: true}"},
		},
		{
			name:               "empty object",
			newImplementation:  `{}`,
			wantRequests:       []string{releasedGet, releasedUpdate, releasedImplementation, releasedJobStatus},
			wantImplementation: stored,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend, server := newReleasedBackend(t, stored)
			r, req, resp := updateMapFilterRequest(t, api.NewClient(&api.Config{BaseURL: server.URL}), oldImplementation, tc.newImplementation)

			r.Update(context.Background(), req, &resp)

			require.False(t, resp.Diagnostics.HasError(), "update diagnostics: %v", resp.Diagnostics)
			require.Equal(t, tc.wantRequests, backend.targets())
			update := backend.body(t, releasedUpdate)
			require.Equal(t, []string{"config", "id", "implementation", "tags", "transform"}, bodyKeys(update))
			require.Equal(t, stored, update["implementation"], "the generic PUT must echo the stored implementation, or the backend blanks it")
			if slices.Contains(tc.wantRequests, releasedImplementation) {
				want := map[string]any{"transform_id": "transform-1"}
				if tc.newImplementation != `{}` {
					want["implementation"] = tc.wantImplementation
				}
				require.Equal(t, want, backend.body(t, releasedImplementation))
			}
			require.Equal(t, tc.wantImplementation, backend.implementation)
			var revision types.String
			require.False(t, resp.State.GetAttribute(context.Background(), path.Root("edit_revision"), &revision).HasError())
			require.True(t, revision.IsNull())
		})
	}
}

func ptr(value string) *string {
	return &value
}
