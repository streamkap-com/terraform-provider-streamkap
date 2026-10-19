package transform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

func updateMapFilterRequestWithRevision(t *testing.T, client api.StreamkapAPI, oldImplementation, newImplementation, revision string) (*BaseTransformResource, resource.UpdateRequest, resource.UpdateResponse) {
	return updateMapFilterRequestWithNullableRevision(t, client, oldImplementation, newImplementation, &revision)
}

func updateMapFilterRequestWithNullableRevision(t *testing.T, client api.StreamkapAPI, oldImplementation, newImplementation string, revision *string) (*BaseTransformResource, resource.UpdateRequest, resource.UpdateResponse) {
	t.Helper()
	ctx := context.Background()
	r := &BaseTransformResource{client: client, config: &MapFilterConfig{}}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	require.Contains(t, objType.AttributeTypes, "edit_revision", "every transform schema must retain its opaque edit revision")
	values := func(implementation string, editRevision *string) map[string]tftypes.Value {
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
		result["edit_revision"] = tftypes.NewValue(tftypes.String, editRevision)
		result["deploy"] = tftypes.NewValue(tftypes.Bool, false)
		return result
	}

	plan := tftypes.NewValue(objType, values(newImplementation, revision))
	state := tftypes.NewValue(objType, values(oldImplementation, revision))
	return r, resource.UpdateRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: plan},
		Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: plan},
		State:  tfsdk.State{Schema: schemaResp.Schema, Raw: state},
	}, resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
}

func TestUpdateLegacyNoopChainsNullRevisionToFirstImplementationChange(t *testing.T) {
	oldImplementation := `{"language":"JAVASCRIPT","value_transform":"return value"}`
	newImplementation := `{"language":"JAVASCRIPT","value_transform":"return {...value, updated: true}"}`
	implementationRevision := "33333333333333333333333333333333"
	genericPuts := 0
	implementationPuts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPut && req.URL.Path == "/transforms":
			genericPuts++
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			require.Contains(t, body, "expected_revision")
			require.Nil(t, body["expected_revision"])
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":             "transform-1",
				"name":           "atomic-transform",
				"transform_type": "map_filter",
				"config":         body["config"],
				"tags":           []string{},
				"edit_revision":  nil,
			}))
		case req.Method == http.MethodPut && req.URL.Path == "/transforms/transform-1/implementation_details":
			implementationPuts++
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			require.Contains(t, body, "expected_revision")
			require.Nil(t, body["expected_revision"])
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"transform_id":   "transform-1",
				"version_id":     "version-7",
				"implementation": body["implementation"],
				"edit_revision":  implementationRevision,
			}))
		case req.Method == http.MethodGet && req.URL.Path == "/transforms/transform-1/job_status":
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"status": "UNKNOWN"}))
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	}))
	defer server.Close()

	r, req, resp := updateMapFilterRequestWithNullableRevision(t, api.NewClient(&api.Config{BaseURL: server.URL}), oldImplementation, newImplementation, nil)
	r.Update(context.Background(), req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "update diagnostics: %v", resp.Diagnostics)
	require.Equal(t, 1, genericPuts)
	require.Equal(t, 1, implementationPuts)
	var stateRevision types.String
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("edit_revision"), &stateRevision).HasError())
	require.Equal(t, implementationRevision, stateRevision.ValueString())
}

func TestUpdateChainsEditRevisionAcrossConfigurationAndImplementation(t *testing.T) {
	oldImplementation := `{"language":"JAVASCRIPT","value_transform":"return value"}`
	newImplementation := `{"language":"JAVASCRIPT","value_transform":"return {...value, updated: true}"}`
	initialRevision := "11111111111111111111111111111111"
	configurationRevision := "22222222222222222222222222222222"
	implementationRevision := "33333333333333333333333333333333"
	genericPuts := 0
	implementationPuts := 0
	transformGets := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPut && req.URL.Path == "/transforms":
			genericPuts++
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			require.Equal(t, initialRevision, body["expected_revision"])
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":             "transform-1",
				"name":           "atomic-transform",
				"transform_type": "map_filter",
				"config":         body["config"],
				"implementation": map[string]any{"language": "JAVASCRIPT", "value_transform": "return value"},
				"tags":           []string{},
				"edit_revision":  configurationRevision,
			}))
		case req.Method == http.MethodPut && req.URL.Path == "/transforms/transform-1/implementation_details":
			implementationPuts++
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			require.Equal(t, configurationRevision, body["expected_revision"])
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"transform_id":   "transform-1",
				"version_id":     "version-7",
				"implementation": body["implementation"],
				"edit_revision":  implementationRevision,
			}))
		case req.Method == http.MethodGet && req.URL.Path == "/transforms/transform-1":
			transformGets++
			t.Fatalf("update must not refresh and bless a newer edit revision")
		case req.Method == http.MethodGet && req.URL.Path == "/transforms/transform-1/job_status":
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"status": "UNKNOWN"}))
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	}))
	defer server.Close()

	r, req, resp := updateMapFilterRequestWithRevision(t, api.NewClient(&api.Config{BaseURL: server.URL}), oldImplementation, newImplementation, initialRevision)
	r.Update(context.Background(), req, &resp)

	require.False(t, resp.Diagnostics.HasError(), "update diagnostics: %v", resp.Diagnostics)
	require.Equal(t, 1, genericPuts)
	require.Equal(t, 1, implementationPuts)
	require.Zero(t, transformGets)
	var stateRevision types.String
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("edit_revision"), &stateRevision).HasError())
	require.Equal(t, implementationRevision, stateRevision.ValueString())
}

func TestUpdateStaleEditRevisionStopsBeforeImplementation(t *testing.T) {
	oldImplementation := `{"language":"JAVASCRIPT","value_transform":"return value"}`
	newImplementation := `{"language":"JAVASCRIPT","value_transform":"return {...value, updated: true}"}`
	initialRevision := "11111111111111111111111111111111"
	genericPuts := 0
	implementationPuts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method == http.MethodPut && req.URL.Path == "/transforms":
			genericPuts++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"detail": "transform edit revision is stale"}))
		case req.Method == http.MethodPut && req.URL.Path == "/transforms/transform-1/implementation_details":
			implementationPuts++
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	}))
	defer server.Close()

	r, req, resp := updateMapFilterRequestWithRevision(t, api.NewClient(&api.Config{BaseURL: server.URL}), oldImplementation, newImplementation, initialRevision)
	r.Update(context.Background(), req, &resp)

	require.True(t, resp.Diagnostics.HasError())
	require.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "transform edit revision is stale")
	require.Equal(t, 1, genericPuts, "a stale edit must not be retried")
	require.Zero(t, implementationPuts)
}

func TestUpdateImplementationFailurePreservesPriorImplementationInPartialState(t *testing.T) {
	oldImplementation := `{"language":"JAVASCRIPT","value_transform":"return value"}`
	newImplementation := `{"language":"JAVASCRIPT","value_transform":"return {...value, updated: true}"}`
	initialRevision := "11111111111111111111111111111111"
	configurationRevision := "22222222222222222222222222222222"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/transforms":
			var body map[string]any
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":             "transform-1",
				"name":           "atomic-transform",
				"transform_type": "map_filter",
				"config":         body["config"],
				"tags":           []string{},
				"edit_revision":  configurationRevision,
			}))
		case "/transforms/transform-1/implementation_details":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"detail": "transform edit revision is stale"}))
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	}))
	defer server.Close()

	r, req, resp := updateMapFilterRequestWithRevision(t, api.NewClient(&api.Config{BaseURL: server.URL}), oldImplementation, newImplementation, initialRevision)
	r.Update(context.Background(), req, &resp)

	require.True(t, resp.Diagnostics.HasError())
	var partialImplementation jsontypes.Normalized
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("implementation_json"), &partialImplementation).HasError())
	require.JSONEq(t, oldImplementation, partialImplementation.ValueString())
	var partialRevision types.String
	require.False(t, resp.State.GetAttribute(context.Background(), path.Root("edit_revision"), &partialRevision).HasError())
	require.Equal(t, configurationRevision, partialRevision.ValueString())
}

func TestUpdateRejectsMalformedObservedRevisionBeforeRequest(t *testing.T) {
	oldImplementation := `{"language":"JAVASCRIPT","value_transform":"return value"}`
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests++
	}))
	defer server.Close()

	r, req, resp := updateMapFilterRequestWithRevision(t, api.NewClient(&api.Config{BaseURL: server.URL}), oldImplementation, oldImplementation, "NOT-A-REVISION")
	r.Update(context.Background(), req, &resp)

	require.True(t, resp.Diagnostics.HasError())
	require.Contains(t, resp.Diagnostics.Errors()[0].Detail(), "exactly 32 lowercase hexadecimal")
	require.Zero(t, requests)
}

// planTransformUpdate runs the resource's ModifyPlan over a plan in which
// UseStateForUnknown has already copied the prior edit_revision, as the
// framework does before calling it.
func planTransformUpdate(t *testing.T, r *BaseTransformResource, req resource.UpdateRequest) tfsdk.Plan {
	t.Helper()
	modifier, ok := any(r).(resource.ResourceWithModifyPlan)
	require.True(t, ok, "transforms must plan edit_revision themselves: the backend rotates it on every edit")
	resp := resource.ModifyPlanResponse{Plan: req.Plan}
	modifier.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Config: req.Config, Plan: req.Plan, State: req.State}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "plan diagnostics: %v", resp.Diagnostics)
	return resp.Plan
}

func TestPlannedEditRevisionIsConsistentWithAppliedRotation(t *testing.T) {
	oldImplementation := `{"language":"JAVASCRIPT","value_transform":"return value"}`
	newImplementation := `{"language":"JAVASCRIPT","value_transform":"return {...value, updated: true}"}`
	initialRevision := "11111111111111111111111111111111"
	configurationRevision := "22222222222222222222222222222222"
	implementationRevision := "33333333333333333333333333333333"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		switch {
		case req.Method == http.MethodPut && req.URL.Path == "/transforms":
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"id":             "transform-1",
				"name":           "atomic-transform",
				"transform_type": "map_filter",
				"config":         body["config"],
				"tags":           []string{},
				"edit_revision":  configurationRevision,
			}))
		case req.Method == http.MethodPut && req.URL.Path == "/transforms/transform-1/implementation_details":
			require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"transform_id":   "transform-1",
				"version_id":     "version-7",
				"implementation": body["implementation"],
				"edit_revision":  implementationRevision,
			}))
		case req.Method == http.MethodGet && req.URL.Path == "/transforms/transform-1/job_status":
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"status": "UNKNOWN"}))
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}
	}))
	defer server.Close()

	r, req, resp := updateMapFilterRequestWithRevision(t, api.NewClient(&api.Config{BaseURL: server.URL}), oldImplementation, newImplementation, initialRevision)
	req.Plan = planTransformUpdate(t, r, req)
	r.Update(context.Background(), req, &resp)
	require.False(t, resp.Diagnostics.HasError(), "update diagnostics: %v", resp.Diagnostics)

	ctx := context.Background()
	var planned, applied types.String
	require.False(t, req.Plan.GetAttribute(ctx, path.Root("edit_revision"), &planned).HasError())
	require.False(t, resp.State.GetAttribute(ctx, path.Root("edit_revision"), &applied).HasError())
	require.Equal(t, implementationRevision, applied.ValueString())
	// Terraform core rejects an apply whose result differs from a known planned value.
	require.True(t, planned.IsUnknown() || planned.Equal(applied), "planned edit_revision %s, applied %s", planned, applied)
}

func TestNoopPlanKeepsKnownEditRevision(t *testing.T) {
	implementation := `{"language":"JAVASCRIPT","value_transform":"return value"}`
	revision := "11111111111111111111111111111111"
	r, req, _ := updateMapFilterRequestWithRevision(t, nil, implementation, implementation, revision)

	plan := planTransformUpdate(t, r, req)

	var planned types.String
	require.False(t, plan.GetAttribute(context.Background(), path.Root("edit_revision"), &planned).HasError())
	require.Equal(t, revision, planned.ValueString(), "a no-op plan must not show edit_revision as changing")
	require.True(t, plan.Raw.Equal(req.State.Raw), "a no-op plan must equal prior state")
}
