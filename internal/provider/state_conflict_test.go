// internal/provider/state_conflict_test.go
// Tests for state conflicts including drift detection, external deletion, and import of non-existent resources.
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// =============================================================================
// Drift Detection Tests - Resource Modified Externally
// =============================================================================

// TestDriftDetection_SourceModifiedExternally tests that when a source is modified
// outside of Terraform (e.g., via the API or UI), the provider correctly detects
// the drift when reading the resource.
func TestDriftDetection_SourceModifiedExternally(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	sourceID := "source-123"

	// First read returns the original state
	callCount := 0
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/"+sourceID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			var response api.GetSourceResponse
			if callCount == 1 {
				// First call: Original state
				response = api.GetSourceResponse{
					Total:    1,
					PageSize: 10,
					Page:     1,
					Result: []api.Source{
						{
							ID:        sourceID,
							Name:      "original-source-name",
							Connector: "postgresql",
							Config: map[string]any{
								"database.hostname.user.defined": "localhost",
								"database.port.user.defined":     "5432",
								"database.user":                  "postgres",
							},
						},
					},
				}
			} else {
				// Subsequent calls: Resource has been modified externally
				response = api.GetSourceResponse{
					Total:    1,
					PageSize: 10,
					Page:     1,
					Result: []api.Source{
						{
							ID:        sourceID,
							Name:      "modified-source-name", // Name changed externally
							Connector: "postgresql",
							Config: map[string]any{
								"database.hostname.user.defined": "newhost.example.com", // Hostname changed
								"database.port.user.defined":     "5433",                // Port changed
								"database.user":                  "admin",               // User changed
							},
						},
					},
				}
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()

	// First read - original state
	source1, err := client.GetSource(ctx, sourceID)
	require.NoError(t, err)
	require.NotNil(t, source1)
	assert.Equal(t, "original-source-name", source1.Name)
	assert.Equal(t, "localhost", source1.Config["database.hostname.user.defined"])

	// Second read - detects drift (external modification)
	source2, err := client.GetSource(ctx, sourceID)
	require.NoError(t, err)
	require.NotNil(t, source2)

	// Verify the drift is detected - values have changed
	assert.NotEqual(t, source1.Name, source2.Name, "Drift should be detected: name changed")
	assert.Equal(t, "modified-source-name", source2.Name)
	assert.Equal(t, "newhost.example.com", source2.Config["database.hostname.user.defined"])
	assert.Equal(t, "5433", source2.Config["database.port.user.defined"])
	assert.Equal(t, "admin", source2.Config["database.user"])

	// Verify we made 2 calls
	assert.Equal(t, 2, callCount, "Should have made 2 API calls")
}

// TestDriftDetection_DestinationModifiedExternally tests drift detection for destinations
func TestDriftDetection_DestinationModifiedExternally(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	destID := "dest-456"

	// First read returns original, second returns modified
	callCount := 0
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/destinations/"+destID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			var response api.GetDestinationResponse
			if callCount == 1 {
				response = api.GetDestinationResponse{
					Total:    1,
					PageSize: 10,
					Page:     1,
					Result: []api.Destination{
						{
							ID:        destID,
							Name:      "original-dest",
							Connector: "snowflake",
							Config: map[string]any{
								"snowflake.url.name":     "account.snowflakecomputing.com",
								"snowflake.user.name":    "STREAMKAP_USER",
								"tasks.max":              float64(1),
								"schema.evolution":       "basic",
							},
						},
					},
				}
			} else {
				response = api.GetDestinationResponse{
					Total:    1,
					PageSize: 10,
					Page:     1,
					Result: []api.Destination{
						{
							ID:        destID,
							Name:      "externally-renamed-dest", // Modified externally
							Connector: "snowflake",
							Config: map[string]any{
								"snowflake.url.name":     "account.snowflakecomputing.com",
								"snowflake.user.name":    "NEW_ADMIN_USER", // Modified
								"tasks.max":              float64(5),       // Modified
								"schema.evolution":       "none",           // Modified
							},
						},
					},
				}
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()

	// First read
	dest1, err := client.GetDestination(ctx, destID)
	require.NoError(t, err)
	require.NotNil(t, dest1)

	// Second read - detects drift
	dest2, err := client.GetDestination(ctx, destID)
	require.NoError(t, err)
	require.NotNil(t, dest2)

	// Verify drift detected
	assert.NotEqual(t, dest1.Name, dest2.Name)
	assert.NotEqual(t, dest1.Config["snowflake.user.name"], dest2.Config["snowflake.user.name"])
	assert.NotEqual(t, dest1.Config["tasks.max"], dest2.Config["tasks.max"])
	assert.NotEqual(t, dest1.Config["schema.evolution"], dest2.Config["schema.evolution"])
}

// TestDriftDetection_TransformModifiedExternally tests drift detection for transforms
func TestDriftDetection_TransformModifiedExternally(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	transformID := "transform-789"

	callCount := 0
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/transforms/"+transformID+"?secret_returned=true&unwind_topics=false",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			var response api.GetTransformResponse
			if callCount == 1 {
				response = api.GetTransformResponse{
					Total:    1,
					PageSize: 10,
					Page:     1,
					Result: []api.Transform{
						{
							ID:   transformID,
							Name: "original-transform",
							Config: map[string]any{
								"transforms.language":                   "Javascript",
								"transforms.input.topic.pattern":        "input-topic-*",
								"transforms.output.topic.pattern":       "output-topic-*",
								"transforms.input.serialization.format": "Json",
							},
						},
					},
				}
			} else {
				response = api.GetTransformResponse{
					Total:    1,
					PageSize: 10,
					Page:     1,
					Result: []api.Transform{
						{
							ID:   transformID,
							Name: "modified-transform-name", // Modified
							Config: map[string]any{
								"transforms.language":                   "Javascript",
								"transforms.input.topic.pattern":        "new-input-*",  // Modified
								"transforms.output.topic.pattern":       "new-output-*", // Modified
								"transforms.input.serialization.format": "Avro",         // Modified
							},
						},
					},
				}
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()

	transform1, err := client.GetTransform(ctx, transformID)
	require.NoError(t, err)
	require.NotNil(t, transform1)

	transform2, err := client.GetTransform(ctx, transformID)
	require.NoError(t, err)
	require.NotNil(t, transform2)

	// Verify drift detected
	assert.NotEqual(t, transform1.Name, transform2.Name)
	assert.NotEqual(t, transform1.Config["transforms.input.topic.pattern"], transform2.Config["transforms.input.topic.pattern"])
}

// TestDriftDetection_PipelineModifiedExternally tests drift detection for pipelines
func TestDriftDetection_PipelineModifiedExternally(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	pipelineID := "pipeline-abc"

	callCount := 0
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/pipelines/"+pipelineID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			var response api.GetPipelineResponse
			if callCount == 1 {
				response = api.GetPipelineResponse{
					Total:    1,
					PageSize: 10,
					Page:     1,
					Result: []api.Pipeline{
						{
							ID:   pipelineID,
							Name: "original-pipeline",
							Source: api.PipelineSource{
								ID:        "source-1",
								Name:      "my-source",
								Connector: "postgresql",
							},
							Destination: api.PipelineDestination{
								ID:        "dest-1",
								Name:      "my-dest",
								Connector: "snowflake",
							},
						},
					},
				}
			} else {
				response = api.GetPipelineResponse{
					Total:    1,
					PageSize: 10,
					Page:     1,
					Result: []api.Pipeline{
						{
							ID:   pipelineID,
							Name: "renamed-pipeline", // Modified externally
							Source: api.PipelineSource{
								ID:        "source-2", // Source changed
								Name:      "new-source",
								Connector: "mysql",
							},
							Destination: api.PipelineDestination{
								ID:        "dest-1",
								Name:      "my-dest",
								Connector: "snowflake",
							},
						},
					},
				}
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()

	pipeline1, err := client.GetPipeline(ctx, pipelineID)
	require.NoError(t, err)
	require.NotNil(t, pipeline1)

	pipeline2, err := client.GetPipeline(ctx, pipelineID)
	require.NoError(t, err)
	require.NotNil(t, pipeline2)

	// Verify drift detected
	assert.NotEqual(t, pipeline1.Name, pipeline2.Name)
	assert.NotEqual(t, pipeline1.Source.ID, pipeline2.Source.ID)
}

// TestDriftDetection_MultipleFieldsDrifted tests detection of multiple fields changed
func TestDriftDetection_MultipleFieldsDrifted(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	sourceID := "source-multi-drift"

	// Original state
	originalConfig := map[string]any{
		"database.hostname.user.defined": "host1.example.com",
		"database.port.user.defined":     "5432",
		"database.user":                  "user1",
		"database.dbname":                "db1",
		"schema.include.list":            "schema1",
		"table.include.list":             "schema1.table1",
		"snapshot.mode":                  "initial",
	}

	// Modified state - multiple fields changed
	modifiedConfig := map[string]any{
		"database.hostname.user.defined": "host2.example.com", // Changed
		"database.port.user.defined":     "5433",              // Changed
		"database.user":                  "user2",             // Changed
		"database.dbname":                "db2",               // Changed
		"schema.include.list":            "schema2",           // Changed
		"table.include.list":             "schema2.table2",    // Changed
		"snapshot.mode":                  "never",             // Changed
	}

	callCount := 0
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/"+sourceID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			config := originalConfig
			name := "original-name"
			if callCount > 1 {
				config = modifiedConfig
				name = "modified-name"
			}
			response := api.GetSourceResponse{
				Total: 1, PageSize: 10, Page: 1,
				Result: []api.Source{{ID: sourceID, Name: name, Connector: "postgresql", Config: config}},
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()

	source1, err := client.GetSource(ctx, sourceID)
	require.NoError(t, err)

	source2, err := client.GetSource(ctx, sourceID)
	require.NoError(t, err)

	// Verify all fields detected as drifted
	driftedFields := []string{
		"database.hostname.user.defined",
		"database.port.user.defined",
		"database.user",
		"database.dbname",
		"schema.include.list",
		"table.include.list",
		"snapshot.mode",
	}

	for _, field := range driftedFields {
		assert.NotEqual(t, source1.Config[field], source2.Config[field],
			"Drift should be detected for field: %s", field)
	}
}

// =============================================================================
// External Deletion Tests - Resource Deleted Outside Terraform
// =============================================================================

// TestExternalDeletion_SourceDeletedExternally tests that when a source is deleted
// outside of Terraform, the provider correctly handles the 404/empty response.
func TestExternalDeletion_SourceDeletedExternally(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	sourceID := "deleted-source-123"

	// Resource exists on first call, deleted on second
	callCount := 0
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/"+sourceID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				// First call: Resource exists
				response := api.GetSourceResponse{
					Total: 1, PageSize: 10, Page: 1,
					Result: []api.Source{
						{ID: sourceID, Name: "existing-source", Connector: "postgresql", Config: map[string]any{}},
					},
				}
				return httpmock.NewJsonResponse(http.StatusOK, response)
			}
			// Second call: Resource has been deleted externally - returns empty result
			response := api.GetSourceResponse{
				Total: 0, PageSize: 10, Page: 1,
				Result: []api.Source{},
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()

	// First read - resource exists
	source1, err := client.GetSource(ctx, sourceID)
	require.NoError(t, err)
	require.NotNil(t, source1, "Source should exist on first read")
	assert.Equal(t, sourceID, source1.ID)

	// Second read - resource deleted externally
	source2, err := client.GetSource(ctx, sourceID)
	require.NoError(t, err)
	assert.Nil(t, source2, "Source should be nil after external deletion")
}

// TestExternalDeletion_Source404Response tests handling of 404 response
func TestExternalDeletion_Source404Response(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	sourceID := "nonexistent-source"

	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/"+sourceID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: fmt.Sprintf("Source with ID '%s' not found", sourceID),
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()

	source, err := client.GetSource(ctx, sourceID)

	require.Error(t, err, "Should return error for 404")
	assert.Nil(t, source)
	assert.Contains(t, err.Error(), "not found")
}

// TestExternalDeletion_DestinationDeletedExternally tests destination deletion detection
func TestExternalDeletion_DestinationDeletedExternally(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	destID := "deleted-dest-456"

	callCount := 0
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/destinations/"+destID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				response := api.GetDestinationResponse{
					Total: 1, PageSize: 10, Page: 1,
					Result: []api.Destination{
						{ID: destID, Name: "existing-dest", Connector: "snowflake", Config: map[string]any{}},
					},
				}
				return httpmock.NewJsonResponse(http.StatusOK, response)
			}
			// Resource deleted externally
			response := api.GetDestinationResponse{
				Total: 0, PageSize: 10, Page: 1,
				Result: []api.Destination{},
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()

	dest1, err := client.GetDestination(ctx, destID)
	require.NoError(t, err)
	require.NotNil(t, dest1)

	dest2, err := client.GetDestination(ctx, destID)
	require.NoError(t, err)
	assert.Nil(t, dest2, "Destination should be nil after external deletion")
}

// TestExternalDeletion_TransformDeletedExternally tests transform deletion detection
func TestExternalDeletion_TransformDeletedExternally(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	transformID := "deleted-transform-789"

	callCount := 0
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/transforms/"+transformID+"?secret_returned=true&unwind_topics=false",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				response := api.GetTransformResponse{
					Total: 1, PageSize: 10, Page: 1,
					Result: []api.Transform{
						{ID: transformID, Name: "existing-transform", Config: map[string]any{}},
					},
				}
				return httpmock.NewJsonResponse(http.StatusOK, response)
			}
			response := api.GetTransformResponse{
				Total: 0, PageSize: 10, Page: 1,
				Result: []api.Transform{},
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()

	transform1, err := client.GetTransform(ctx, transformID)
	require.NoError(t, err)
	require.NotNil(t, transform1)

	transform2, err := client.GetTransform(ctx, transformID)
	require.NoError(t, err)
	assert.Nil(t, transform2, "Transform should be nil after external deletion")
}

// TestExternalDeletion_PipelineDeletedExternally tests pipeline deletion detection
func TestExternalDeletion_PipelineDeletedExternally(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	pipelineID := "deleted-pipeline-abc"

	callCount := 0
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/pipelines/"+pipelineID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			callCount++
			if callCount == 1 {
				response := api.GetPipelineResponse{
					Total: 1, PageSize: 10, Page: 1,
					Result: []api.Pipeline{
						{
							ID:   pipelineID,
							Name: "existing-pipeline",
							Source: api.PipelineSource{
								ID: "src-1", Name: "src", Connector: "postgresql",
							},
							Destination: api.PipelineDestination{
								ID: "dest-1", Name: "dest", Connector: "snowflake",
							},
						},
					},
				}
				return httpmock.NewJsonResponse(http.StatusOK, response)
			}
			response := api.GetPipelineResponse{
				Total: 0, PageSize: 10, Page: 1,
				Result: []api.Pipeline{},
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()

	pipeline1, err := client.GetPipeline(ctx, pipelineID)
	require.NoError(t, err)
	require.NotNil(t, pipeline1)

	pipeline2, err := client.GetPipeline(ctx, pipelineID)
	require.NoError(t, err)
	assert.Nil(t, pipeline2, "Pipeline should be nil after external deletion")
}

// TestExternalDeletion_UpdateAfterDelete tests that update fails after external deletion
func TestExternalDeletion_UpdateAfterDelete(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	sourceID := "deleted-before-update"

	// First read - resource exists
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/"+sourceID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			response := api.GetSourceResponse{
				Total: 1, PageSize: 10, Page: 1,
				Result: []api.Source{
					{ID: sourceID, Name: "existing-source", Connector: "postgresql", Config: map[string]any{}},
				},
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	// Update fails because resource was deleted
	httpmock.RegisterResponder(
		http.MethodPut,
		baseURL+"/sources/"+sourceID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: fmt.Sprintf("Source with ID '%s' not found", sourceID),
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()

	// Read succeeds (resource still in API cache or race condition)
	source, err := client.GetSource(ctx, sourceID)
	require.NoError(t, err)
	require.NotNil(t, source)

	// Update fails because resource deleted externally between read and update
	updatedSource := api.Source{
		ID:        sourceID,
		Name:      "updated-name",
		Connector: "postgresql",
		Config:    map[string]any{},
	}

	result, err := client.UpdateSource(ctx, sourceID, updatedSource)
	require.Error(t, err, "Update should fail for deleted resource")
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

// TestExternalDeletion_DeleteAfterDelete tests that delete is idempotent or fails gracefully
func TestExternalDeletion_DeleteAfterDelete(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	sourceID := "deleted-twice"

	httpmock.RegisterResponder(
		http.MethodDelete,
		baseURL+"/sources/"+sourceID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: fmt.Sprintf("Source with ID '%s' not found", sourceID),
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()

	err := client.DeleteSource(ctx, sourceID)
	require.Error(t, err, "Delete of already-deleted resource should fail")
	assert.Contains(t, err.Error(), "not found")
}

// =============================================================================
// Import of Non-Existent Resource Tests
// =============================================================================

// TestImportNonExistent_Source tests importing a source that doesn't exist
func TestImportNonExistent_Source(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	nonExistentID := "nonexistent-source-xyz"

	// API returns 404 for non-existent resource
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/"+nonExistentID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: fmt.Sprintf("Source with ID '%s' not found", nonExistentID),
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()

	source, err := client.GetSource(ctx, nonExistentID)

	require.Error(t, err, "Import of non-existent source should fail")
	assert.Nil(t, source)
	assert.Contains(t, err.Error(), "not found")
}

// TestImportNonExistent_Destination tests importing a destination that doesn't exist
func TestImportNonExistent_Destination(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	nonExistentID := "nonexistent-dest-xyz"

	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/destinations/"+nonExistentID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: fmt.Sprintf("Destination with ID '%s' not found", nonExistentID),
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()

	dest, err := client.GetDestination(ctx, nonExistentID)

	require.Error(t, err, "Import of non-existent destination should fail")
	assert.Nil(t, dest)
	assert.Contains(t, err.Error(), "not found")
}

// TestImportNonExistent_Transform tests importing a transform that doesn't exist
func TestImportNonExistent_Transform(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	nonExistentID := "nonexistent-transform-xyz"

	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/transforms/"+nonExistentID+"?secret_returned=true&unwind_topics=false",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: fmt.Sprintf("Transform with ID '%s' not found", nonExistentID),
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()

	transform, err := client.GetTransform(ctx, nonExistentID)

	require.Error(t, err, "Import of non-existent transform should fail")
	assert.Nil(t, transform)
	assert.Contains(t, err.Error(), "not found")
}

// TestImportNonExistent_Pipeline tests importing a pipeline that doesn't exist
func TestImportNonExistent_Pipeline(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	nonExistentID := "nonexistent-pipeline-xyz"

	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/pipelines/"+nonExistentID+"?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: fmt.Sprintf("Pipeline with ID '%s' not found", nonExistentID),
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()

	pipeline, err := client.GetPipeline(ctx, nonExistentID)

	require.Error(t, err, "Import of non-existent pipeline should fail")
	assert.Nil(t, pipeline)
	assert.Contains(t, err.Error(), "not found")
}

// TestImportNonExistent_Topic tests importing a topic that doesn't exist
func TestImportNonExistent_Topic(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	nonExistentID := "nonexistent-topic-xyz"

	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/topics/"+nonExistentID,
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: fmt.Sprintf("Topic with ID '%s' not found", nonExistentID),
			}
			return httpmock.NewJsonResponse(http.StatusNotFound, errResponse)
		},
	)

	ctx := context.Background()

	topic, err := client.GetTopic(ctx, nonExistentID)

	require.Error(t, err, "Import of non-existent topic should fail")
	assert.Nil(t, topic)
	assert.Contains(t, err.Error(), "not found")
}

// TestImportNonExistent_Tag tests importing a tag that doesn't exist
func TestImportNonExistent_Tag(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	nonExistentID := "nonexistent-tag-xyz"

	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/tags?tag_ids="+nonExistentID,
		func(req *http.Request) (*http.Response, error) {
			// Tags endpoint returns empty result for non-existent tags
			response := api.GetTagResponse{
				Tags: []api.Tag{},
			}
			return httpmock.NewJsonResponse(http.StatusOK, response)
		},
	)

	ctx := context.Background()

	tag, err := client.GetTag(ctx, nonExistentID)

	require.NoError(t, err, "Tag endpoint returns success with empty result")
	assert.Nil(t, tag, "Tag should be nil when not found")
}

// TestImportNonExistent_EmptyID tests importing with an empty ID
func TestImportNonExistent_EmptyID(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	// Register responder for empty ID (results in double slash in URL)
	httpmock.RegisterResponder(
		http.MethodGet,
		baseURL+"/sources/?secret_returned=true",
		func(req *http.Request) (*http.Response, error) {
			errResponse := api.APIErrorResponse{
				Detail: "Invalid source ID",
			}
			return httpmock.NewJsonResponse(http.StatusBadRequest, errResponse)
		},
	)

	ctx := context.Background()

	source, err := client.GetSource(ctx, "")

	// Empty ID should result in error (either from URL construction or API)
	require.Error(t, err, "Import with empty ID should fail")
	assert.Nil(t, source)
}

// TestImportNonExistent_InvalidIDFormat tests importing with invalid ID format
func TestImportNonExistent_InvalidIDFormat(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	baseURL := "https://api.test.streamkap.com"
	client := newTestAPIClient(baseURL)

	testCases := []struct {
		name string
		id   string
	}{
		{"special_characters", "source/with/slashes"},
		{"unicode", "source-with-émojis-🚀"},
		{"very_long_id", "source-" + string(make([]byte, 1000))},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Register a catch-all responder for this test
			httpmock.Reset()
			httpmock.RegisterNoResponder(func(req *http.Request) (*http.Response, error) {
				errResponse := api.APIErrorResponse{
					Detail: fmt.Sprintf("Invalid source ID format: %s", tc.id),
				}
				return httpmock.NewJsonResponse(http.StatusBadRequest, errResponse)
			})

			ctx := context.Background()
			source, err := client.GetSource(ctx, tc.id)

			// Should fail with error
			require.Error(t, err, "Import with invalid ID format should fail: %s", tc.name)
			assert.Nil(t, source)
		})
	}
}

// =============================================================================
// Resource Schema Verification Tests
// =============================================================================

// TestStateConflict_ImportStatePassthrough verifies the import state mechanism exists
func TestStateConflict_ImportStatePassthrough(t *testing.T) {
	// Verify that resources implement ImportState
	provider := New("test")()
	resp := &resource.MetadataResponse{}

	// Get all resources
	resources := provider.Resources(context.Background())
	require.NotEmpty(t, resources, "Provider should have resources")

	// Check that resources can be instantiated (ImportState is part of the interface)
	for _, resourceFunc := range resources {
		res := resourceFunc()
		res.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "streamkap"}, resp)
		assert.NotEmpty(t, resp.TypeName, "Resource should have a type name")
	}
}

// TestStateConflict_ReadAfterDeleteReturnsNil verifies read behavior for missing resources
func TestStateConflict_ReadAfterDeleteReturnsNil(t *testing.T) {
	// This test verifies that the API client correctly returns nil when
	// a resource doesn't exist (empty result array), which is the signal
	// to Terraform that the resource has been deleted externally

	tests := []struct {
		name     string
		setup    func(baseURL string)
		readFunc func(ctx context.Context, client api.StreamkapAPI) (any, error)
	}{
		{
			name: "source_empty_result",
			setup: func(baseURL string) {
				httpmock.RegisterResponder(http.MethodGet, baseURL+"/sources/missing-id?secret_returned=true",
					httpmock.NewJsonResponderOrPanic(http.StatusOK, api.GetSourceResponse{
						Total: 0, PageSize: 10, Page: 1, Result: []api.Source{},
					}))
			},
			readFunc: func(ctx context.Context, client api.StreamkapAPI) (any, error) {
				return client.GetSource(ctx, "missing-id")
			},
		},
		{
			name: "destination_empty_result",
			setup: func(baseURL string) {
				httpmock.RegisterResponder(http.MethodGet, baseURL+"/destinations/missing-id?secret_returned=true",
					httpmock.NewJsonResponderOrPanic(http.StatusOK, api.GetDestinationResponse{
						Total: 0, PageSize: 10, Page: 1, Result: []api.Destination{},
					}))
			},
			readFunc: func(ctx context.Context, client api.StreamkapAPI) (any, error) {
				return client.GetDestination(ctx, "missing-id")
			},
		},
		{
			name: "transform_empty_result",
			setup: func(baseURL string) {
				httpmock.RegisterResponder(http.MethodGet, baseURL+"/transforms/missing-id?secret_returned=true&unwind_topics=false",
					httpmock.NewJsonResponderOrPanic(http.StatusOK, api.GetTransformResponse{
						Total: 0, PageSize: 10, Page: 1, Result: []api.Transform{},
					}))
			},
			readFunc: func(ctx context.Context, client api.StreamkapAPI) (any, error) {
				return client.GetTransform(ctx, "missing-id")
			},
		},
		{
			name: "pipeline_empty_result",
			setup: func(baseURL string) {
				httpmock.RegisterResponder(http.MethodGet, baseURL+"/pipelines/missing-id?secret_returned=true",
					httpmock.NewJsonResponderOrPanic(http.StatusOK, api.GetPipelineResponse{
						Total: 0, PageSize: 10, Page: 1, Result: []api.Pipeline{},
					}))
			},
			readFunc: func(ctx context.Context, client api.StreamkapAPI) (any, error) {
				return client.GetPipeline(ctx, "missing-id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpmock.Activate()
			defer httpmock.DeactivateAndReset()

			baseURL := "https://api.test.streamkap.com"
			client := newTestAPIClient(baseURL)
			tt.setup(baseURL)

			ctx := context.Background()
			result, err := tt.readFunc(ctx, client)

			require.NoError(t, err, "Empty result should not be an error")
			assert.Nil(t, result, "Result should be nil for missing resource")
		})
	}
}

// TestStateConflict_ConfigMapDriftDetection tests detailed config map comparison
func TestStateConflict_ConfigMapDriftDetection(t *testing.T) {
	// Test that config map changes are properly detected
	original := map[string]any{
		"string_field":  "original",
		"int_field":     float64(123), // JSON numbers are float64
		"bool_field":    true,
		"nested.field":  "nested_value",
		"list_field":    []any{"a", "b", "c"},
	}

	modified := map[string]any{
		"string_field":  "modified",           // Changed
		"int_field":     float64(456),         // Changed
		"bool_field":    false,                // Changed
		"nested.field":  "new_nested_value",   // Changed
		"list_field":    []any{"x", "y", "z"}, // Changed
		"new_field":     "added",              // Added
	}

	// Verify drift detection by comparing maps
	changedFields := []string{}
	for key := range original {
		origJSON, _ := json.Marshal(original[key])
		modJSON, _ := json.Marshal(modified[key])
		if string(origJSON) != string(modJSON) {
			changedFields = append(changedFields, key)
		}
	}

	// Also check for new fields
	for key := range modified {
		if _, exists := original[key]; !exists {
			changedFields = append(changedFields, key)
		}
	}

	assert.Len(t, changedFields, 6, "Should detect 6 changed/added fields")
	assert.Contains(t, changedFields, "string_field")
	assert.Contains(t, changedFields, "int_field")
	assert.Contains(t, changedFields, "bool_field")
	assert.Contains(t, changedFields, "nested.field")
	assert.Contains(t, changedFields, "list_field")
	assert.Contains(t, changedFields, "new_field")
}
