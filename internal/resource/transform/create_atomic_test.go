package transform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

func createMapFilterRequest(t *testing.T, client api.StreamkapAPI, implementation *string) (*BaseTransformResource, resource.CreateRequest, resource.CreateResponse) {
	t.Helper()
	ctx := context.Background()
	r := &BaseTransformResource{client: client, config: &MapFilterConfig{}}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, attrType := range objType.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, nil)
	}
	values["name"] = tftypes.NewValue(tftypes.String, "atomic-transform")
	values["transforms_language"] = tftypes.NewValue(tftypes.String, "JavaScript")
	values["transforms_input_topic_pattern"] = tftypes.NewValue(tftypes.String, "source\\.orders")
	values["transforms_output_topic_pattern"] = tftypes.NewValue(tftypes.String, "computed-$0")
	values["transforms_input_serialization_format"] = tftypes.NewValue(tftypes.String, "Any")
	values["transforms_output_serialization_format"] = tftypes.NewValue(tftypes.String, "Any")
	values["transforms_input_job_parallelism"] = tftypes.NewValue(tftypes.Number, 1)
	if implementation == nil {
		values["implementation_json"] = tftypes.NewValue(tftypes.String, nil)
	} else {
		values["implementation_json"] = tftypes.NewValue(tftypes.String, *implementation)
	}
	values["deploy"] = tftypes.NewValue(tftypes.Bool, false)

	raw := tftypes.NewValue(objType, values)
	return r, resource.CreateRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw},
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
	}, resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
}

func updateMapFilterRequest(t *testing.T, client api.StreamkapAPI, oldImplementation, newImplementation string) (*BaseTransformResource, resource.UpdateRequest, resource.UpdateResponse) {
	t.Helper()
	ctx := context.Background()
	r := &BaseTransformResource{client: client, config: &MapFilterConfig{}}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	values := func(implementation string) map[string]tftypes.Value {
		result := make(map[string]tftypes.Value, len(objType.AttributeTypes))
		for name, attrType := range objType.AttributeTypes {
			result[name] = tftypes.NewValue(attrType, nil)
		}
		result["id"] = tftypes.NewValue(tftypes.String, "transform-1")
		result["name"] = tftypes.NewValue(tftypes.String, "atomic-transform")
		result["transform_type"] = tftypes.NewValue(tftypes.String, "map_filter")
		result["transforms_language"] = tftypes.NewValue(tftypes.String, "JavaScript")
		result["transforms_input_topic_pattern"] = tftypes.NewValue(tftypes.String, "source\\.orders")
		result["transforms_output_topic_pattern"] = tftypes.NewValue(tftypes.String, "computed-$0")
		result["transforms_input_serialization_format"] = tftypes.NewValue(tftypes.String, "Any")
		result["transforms_output_serialization_format"] = tftypes.NewValue(tftypes.String, "Any")
		result["transforms_input_job_parallelism"] = tftypes.NewValue(tftypes.Number, 1)
		result["implementation_json"] = tftypes.NewValue(tftypes.String, implementation)
		result["deploy"] = tftypes.NewValue(tftypes.Bool, false)
		return result
	}

	plan := tftypes.NewValue(objType, values(newImplementation))
	state := tftypes.NewValue(objType, values(oldImplementation))
	return r, resource.UpdateRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: plan},
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: plan},
		State:  tfsdk.State{Schema: schemaResp.Schema, Raw: state},
	}, resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
}

func TestCreateWithImplementationUsesOneAtomicPost(t *testing.T) {
	implementation := `{"language":"JAVASCRIPT","value_transform":"function transform(value) { return value; }"}`
	postCalls := 0
	putCalls := 0
	var posted map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/transforms":
			postCalls++
			require.NoError(t, json.NewDecoder(req.Body).Decode(&posted))
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":             "transform-1",
				"name":           "atomic-transform",
				"transform_type": "map_filter",
				"edit_revision":  "0123456789abcdef0123456789abcdef",
				"config":         posted["config"],
				"implementation": posted["implementation"],
				"tags":           []string{},
			}))
		case req.Method == http.MethodPut:
			putCalls++
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{}))
		case req.Method == http.MethodGet && req.URL.Path == "/transforms/transform-1/job_status":
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"status": "UNKNOWN"}))
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	}))
	defer server.Close()

	r, req, resp := createMapFilterRequest(t, api.NewClient(&api.Config{BaseURL: server.URL}), &implementation)
	r.Create(context.Background(), req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "create diagnostics: %v", resp.Diagnostics)
	require.Equal(t, 1, postCalls)
	require.Zero(t, putCalls)
	require.Equal(t, map[string]any{
		"language":        "JAVASCRIPT",
		"value_transform": "function transform(value) { return value; }",
	}, posted["implementation"])
}

func TestCreateRejectsInvalidImplementationBeforeRequest(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
	}))
	defer server.Close()

	implementation := `{"language":`
	r, req, resp := createMapFilterRequest(t, api.NewClient(&api.Config{BaseURL: server.URL}), &implementation)
	r.Create(context.Background(), req, &resp)

	require.True(t, resp.Diagnostics.HasError())
	require.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Invalid JSON String Value")
	require.Zero(t, requests)
}

func TestCreateWithoutImplementationKeepsServerDefault(t *testing.T) {
	postCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/transforms":
			postCalls++
			var posted map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&posted))
			require.NotContains(t, posted, "implementation")
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":             "transform-1",
				"name":           "atomic-transform",
				"transform_type": "map_filter",
				"edit_revision":  "0123456789abcdef0123456789abcdef",
				"config":         posted["config"],
				"implementation": map[string]any{"language": "JAVASCRIPT", "value_transform": "return value"},
				"tags":           []string{},
			}))
		case req.Method == http.MethodGet && req.URL.Path == "/transforms/transform-1/job_status":
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"status": "UNKNOWN"}))
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	}))
	defer server.Close()

	r, req, resp := createMapFilterRequest(t, api.NewClient(&api.Config{BaseURL: server.URL}), nil)
	r.Create(context.Background(), req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "create diagnostics: %v", resp.Diagnostics)
	require.Equal(t, 1, postCalls)
}

func TestCreateRejectsNonObjectImplementationBeforeRequest(t *testing.T) {
	for _, implementation := range []string{`[1]`, `null`, `{}`} {
		t.Run(implementation, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				requests++
			}))
			defer server.Close()

			r, req, resp := createMapFilterRequest(t, api.NewClient(&api.Config{BaseURL: server.URL}), &implementation)
			r.Create(context.Background(), req, &resp)

			require.True(t, resp.Diagnostics.HasError())
			require.Contains(t, resp.Diagnostics.Errors()[0].Summary(), "Invalid Implementation JSON")
			require.Zero(t, requests)
		})
	}
}

func TestUpdateKeepsImplementationOutOfGenericPutAndUsesDedicatedEndpoint(t *testing.T) {
	oldImplementation := `{"language":"JAVASCRIPT","value_transform":"return value"}`
	newImplementation := `{"language":"JAVASCRIPT","value_transform":"return {...value, updated: true}"}`
	genericPuts := 0
	implementationPuts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPut && req.URL.Path == "/transforms":
			genericPuts++
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			require.NotContains(t, body, "implementation")
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":             "transform-1",
				"name":           "atomic-transform",
				"transform_type": "map_filter",
				"edit_revision":  "0123456789abcdef0123456789abcdef",
				"config":         body["config"],
				"implementation": map[string]any{"language": "JAVASCRIPT", "value_transform": "return value"},
				"tags":           []string{},
			}))
		case req.Method == http.MethodPut && req.URL.Path == "/transforms/transform-1/implementation_details":
			implementationPuts++
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			require.Equal(t, map[string]any{
				"language":        "JAVASCRIPT",
				"value_transform": "return {...value, updated: true}",
			}, body["implementation"])
			require.Equal(t, "0123456789abcdef0123456789abcdef", body["expected_revision"])
			body["edit_revision"] = "fedcba9876543210fedcba9876543210"
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(body))
		case req.Method == http.MethodGet && req.URL.Path == "/transforms/transform-1/job_status":
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"status": "UNKNOWN"}))
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	}))
	defer server.Close()

	r, req, resp := updateMapFilterRequest(t, api.NewClient(&api.Config{BaseURL: server.URL}), oldImplementation, newImplementation)
	r.Update(context.Background(), req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "update diagnostics: %v", resp.Diagnostics)
	require.Equal(t, 1, genericPuts)
	require.Equal(t, 1, implementationPuts)
}
