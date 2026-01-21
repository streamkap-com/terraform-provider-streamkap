package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/destination"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
)

// Smoke tests verify that schemas compile correctly and model conversion
// works without runtime errors, even for connectors where we lack test credentials.
// These tests provide confidence in the generated code without needing API access.

// smokeTestCase defines a test case for schema and model smoke testing.
type smokeTestCase struct {
	name            string
	resourceFactory func() resource.Resource
	modelFactory    func() any
	fieldMappings   map[string]string
}

// runSchemaSmoke verifies that a resource schema compiles without errors.
func runSchemaSmoke(t *testing.T, tc smokeTestCase) {
	t.Helper()

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res := tc.resourceFactory()

	// Verify schema request doesn't panic or produce errors
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), "schema should not have errors: %v", schemaResp.Diagnostics)

	// Verify schema has attributes
	require.NotNil(t, schemaResp.Schema.Attributes, "schema should have attributes")
	require.Greater(t, len(schemaResp.Schema.Attributes), 0, "schema should have at least one attribute")

	// Verify required computed fields exist
	assertComputedFieldExists(t, schemaResp.Schema, "id")
	assertComputedFieldExists(t, schemaResp.Schema, "connector")

	// Verify name field is required
	assertRequiredFieldExists(t, schemaResp.Schema, "name")

	t.Logf("Schema smoke test passed for %s: %d attributes", tc.name, len(schemaResp.Schema.Attributes))
}

// runModelSmoke verifies that model instantiation and reflection work correctly.
func runModelSmoke(t *testing.T, tc smokeTestCase) {
	t.Helper()

	// Verify model can be instantiated
	model := tc.modelFactory()
	require.NotNil(t, model, "model factory should return non-nil instance")

	// Verify model is a pointer to a struct
	modelType := reflect.TypeOf(model)
	require.Equal(t, reflect.Ptr, modelType.Kind(), "model should be a pointer")
	require.Equal(t, reflect.Struct, modelType.Elem().Kind(), "model should point to a struct")

	// Verify model has tfsdk tags
	structType := modelType.Elem()
	tagCount := 0
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if tag := field.Tag.Get("tfsdk"); tag != "" {
			tagCount++
		}
	}
	require.Greater(t, tagCount, 0, "model should have tfsdk tags")

	t.Logf("Model smoke test passed for %s: %d fields with tfsdk tags", tc.name, tagCount)
}

// runFieldMappingsSmoke verifies field mappings are valid and non-empty.
func runFieldMappingsSmoke(t *testing.T, tc smokeTestCase) {
	t.Helper()

	require.NotNil(t, tc.fieldMappings, "field mappings should not be nil")
	require.Greater(t, len(tc.fieldMappings), 0, "field mappings should have entries")

	// Verify all mappings have non-empty values
	for tfField, apiField := range tc.fieldMappings {
		assert.NotEmpty(t, tfField, "terraform field name should not be empty")
		assert.NotEmpty(t, apiField, "api field name should not be empty")
	}

	t.Logf("Field mappings smoke test passed for %s: %d mappings", tc.name, len(tc.fieldMappings))
}

// Helper functions for schema assertions
func assertComputedFieldExists(t *testing.T, s schema.Schema, fieldName string) {
	t.Helper()
	attr, exists := s.Attributes[fieldName]
	assert.True(t, exists, "expected computed field %q to exist", fieldName)
	if exists {
		assert.True(t, attr.IsComputed(), "field %q should be computed", fieldName)
	}
}

func assertRequiredFieldExists(t *testing.T, s schema.Schema, fieldName string) {
	t.Helper()
	attr, exists := s.Attributes[fieldName]
	assert.True(t, exists, "expected required field %q to exist", fieldName)
	if exists {
		assert.True(t, attr.IsRequired(), "field %q should be required", fieldName)
	}
}

// TestSmokeSourceOracle verifies Oracle source schema compiles and model works.
func TestSmokeSourceOracle(t *testing.T) {
	tc := smokeTestCase{
		name:            "source_oracle",
		resourceFactory: source.NewOracleResource,
		modelFactory: func() any {
			return &generated.SourceOracleModel{}
		},
		fieldMappings: generated.SourceOracleFieldMappings,
	}

	t.Run("Schema", func(t *testing.T) {
		runSchemaSmoke(t, tc)
	})

	t.Run("Model", func(t *testing.T) {
		runModelSmoke(t, tc)
	})

	t.Run("FieldMappings", func(t *testing.T) {
		runFieldMappingsSmoke(t, tc)
	})
}

// TestSmokeDestinationBigQuery verifies BigQuery destination schema compiles and model works.
func TestSmokeDestinationBigQuery(t *testing.T) {
	tc := smokeTestCase{
		name:            "destination_bigquery",
		resourceFactory: destination.NewBigQueryResource,
		modelFactory: func() any {
			return &generated.DestinationBigqueryModel{}
		},
		fieldMappings: generated.DestinationBigqueryFieldMappings,
	}

	t.Run("Schema", func(t *testing.T) {
		runSchemaSmoke(t, tc)
	})

	t.Run("Model", func(t *testing.T) {
		runModelSmoke(t, tc)
	})

	t.Run("FieldMappings", func(t *testing.T) {
		runFieldMappingsSmoke(t, tc)
	})
}

// TestSmokeDestinationRedshift verifies Redshift destination schema compiles and model works.
func TestSmokeDestinationRedshift(t *testing.T) {
	tc := smokeTestCase{
		name:            "destination_redshift",
		resourceFactory: destination.NewRedshiftResource,
		modelFactory: func() any {
			return &generated.DestinationRedshiftModel{}
		},
		fieldMappings: generated.DestinationRedshiftFieldMappings,
	}

	t.Run("Schema", func(t *testing.T) {
		runSchemaSmoke(t, tc)
	})

	t.Run("Model", func(t *testing.T) {
		runModelSmoke(t, tc)
	})

	t.Run("FieldMappings", func(t *testing.T) {
		runFieldMappingsSmoke(t, tc)
	})
}

// TestSmokeDestinationStarburst verifies Starburst destination schema compiles and model works.
func TestSmokeDestinationStarburst(t *testing.T) {
	tc := smokeTestCase{
		name:            "destination_starburst",
		resourceFactory: destination.NewStarburstResource,
		modelFactory: func() any {
			return &generated.DestinationStarburstModel{}
		},
		fieldMappings: generated.DestinationStarburstFieldMappings,
	}

	t.Run("Schema", func(t *testing.T) {
		runSchemaSmoke(t, tc)
	})

	t.Run("Model", func(t *testing.T) {
		runModelSmoke(t, tc)
	})

	t.Run("FieldMappings", func(t *testing.T) {
		runFieldMappingsSmoke(t, tc)
	})
}

// TestSmokeDestinationMotherduck verifies Motherduck destination schema compiles and model works.
func TestSmokeDestinationMotherduck(t *testing.T) {
	tc := smokeTestCase{
		name:            "destination_motherduck",
		resourceFactory: destination.NewMotherduckResource,
		modelFactory: func() any {
			return &generated.DestinationMotherduckModel{}
		},
		fieldMappings: generated.DestinationMotherduckFieldMappings,
	}

	t.Run("Schema", func(t *testing.T) {
		runSchemaSmoke(t, tc)
	})

	t.Run("Model", func(t *testing.T) {
		runModelSmoke(t, tc)
	})

	t.Run("FieldMappings", func(t *testing.T) {
		runFieldMappingsSmoke(t, tc)
	})
}

// TestSmokeAllConnectorSchemas provides a comprehensive smoke test for all connectors.
// This ensures all registered resources have valid schemas that compile.
func TestSmokeAllConnectorSchemas(t *testing.T) {
	// Get all resources from provider
	p := &streamkapProvider{}
	resources := p.Resources(context.Background())

	require.Greater(t, len(resources), 0, "provider should have registered resources")

	for _, resFactory := range resources {
		res := resFactory()

		// Get resource metadata to identify the resource
		metaResp := &resource.MetadataResponse{}
		res.Metadata(context.Background(), resource.MetadataRequest{
			ProviderTypeName: "streamkap",
		}, metaResp)

		t.Run(metaResp.TypeName, func(t *testing.T) {
			ctx := context.Background()
			schemaResp := &resource.SchemaResponse{}

			// Verify schema request doesn't panic or produce errors
			res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
			require.False(t, schemaResp.Diagnostics.HasError(),
				"schema should not have errors for %s: %v", metaResp.TypeName, schemaResp.Diagnostics)

			// Verify schema has attributes
			require.NotNil(t, schemaResp.Schema.Attributes,
				"schema should have attributes for %s", metaResp.TypeName)

			t.Logf("Schema smoke test passed: %d attributes", len(schemaResp.Schema.Attributes))
		})
	}

	t.Logf("All %d connector schemas validated", len(resources))
}
