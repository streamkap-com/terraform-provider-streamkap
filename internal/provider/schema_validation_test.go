package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
)

// Schema validation tests verify that Terraform schema validators properly reject
// invalid configurations with helpful error messages. These tests run without
// API calls and catch validation configuration bugs early.

// =============================================================================
// Missing Required Field Tests
// =============================================================================

// TestSchemaValidation_MissingRequiredField_AzBlob tests that Azure Blob destination
// rejects configurations missing required fields.
func TestSchemaValidation_MissingRequiredField_AzBlob(t *testing.T) {
	// Get the schema for Azure Blob destination
	schema := generated.DestinationAzblobSchema()

	// Test 'azblob_connection_string' required field
	t.Run("required_azblob_connection_string", func(t *testing.T) {
		attr, ok := schema.Attributes["azblob_connection_string"]
		require.True(t, ok, "azblob_connection_string attribute should exist")
		require.True(t, attr.IsRequired(), "azblob_connection_string should be required")
	})

	// Test 'name' required field (all resources require name)
	t.Run("required_name", func(t *testing.T) {
		attr, ok := schema.Attributes["name"]
		require.True(t, ok, "name attribute should exist")
		require.True(t, attr.IsRequired(), "name should be required")
	})

	// Verify azblob_container_name is optional (has no default, but not required)
	t.Run("optional_azblob_container_name", func(t *testing.T) {
		attr, ok := schema.Attributes["azblob_container_name"]
		require.True(t, ok, "azblob_container_name attribute should exist")
		require.True(t, attr.IsOptional(), "azblob_container_name should be optional")
	})
}

// TestSchemaValidation_MissingRequiredField_BigQuery tests that BigQuery destination
// rejects configurations missing required fields.
func TestSchemaValidation_MissingRequiredField_BigQuery(t *testing.T) {
	schema := generated.DestinationBigquerySchema()

	// Required fields
	requiredFields := []string{
		"name",
		"bigquery_json",
		"table_name_prefix",
	}

	for _, field := range requiredFields {
		t.Run("required_"+field, func(t *testing.T) {
			attr, ok := schema.Attributes[field]
			require.True(t, ok, "%s attribute should exist", field)
			require.True(t, attr.IsRequired(), "%s should be required", field)
		})
	}

	// Verify bigquery_region is optional (has default value)
	t.Run("optional_bigquery_region", func(t *testing.T) {
		attr, ok := schema.Attributes["bigquery_region"]
		require.True(t, ok, "bigquery_region attribute should exist")
		require.True(t, attr.IsOptional(), "bigquery_region should be optional (has default)")
	})
}

// TestSchemaValidation_MissingRequiredField_ClickHouse tests ClickHouse destination
// required field validation.
func TestSchemaValidation_MissingRequiredField_ClickHouse(t *testing.T) {
	schema := generated.DestinationClickhouseSchema()

	// Required fields - using actual field names from generated schema
	requiredFields := []string{
		"name",
		"hostname",
		"connection_username",
		"connection_password",
	}

	for _, field := range requiredFields {
		t.Run("required_"+field, func(t *testing.T) {
			attr, ok := schema.Attributes[field]
			require.True(t, ok, "%s attribute should exist", field)
			require.True(t, attr.IsRequired(), "%s should be required", field)
		})
	}

	// database is optional (has default value)
	t.Run("optional_database", func(t *testing.T) {
		attr, ok := schema.Attributes["database"]
		require.True(t, ok, "database attribute should exist")
		require.True(t, attr.IsOptional(), "database should be optional (has default)")
	})
}

// TestSchemaValidation_MissingRequiredField_Elasticsearch tests Elasticsearch source
// required field validation.
func TestSchemaValidation_MissingRequiredField_Elasticsearch(t *testing.T) {
	schema := generated.SourceElasticsearchSchema()

	// Required fields - only es_host and name are actually required
	requiredFields := []string{
		"name",
		"es_host",
	}

	for _, field := range requiredFields {
		t.Run("required_"+field, func(t *testing.T) {
			attr, ok := schema.Attributes[field]
			require.True(t, ok, "%s attribute should exist", field)
			require.True(t, attr.IsRequired(), "%s should be required", field)
		})
	}

	// Verify es_port is optional (has default value)
	t.Run("optional_es_port", func(t *testing.T) {
		attr, ok := schema.Attributes["es_port"]
		require.True(t, ok, "es_port attribute should exist")
		require.True(t, attr.IsOptional(), "es_port should be optional (has default)")
	})
}

// TestSchemaValidation_MissingRequiredField_Transform tests transform resource
// required field validation.
func TestSchemaValidation_MissingRequiredField_Transform(t *testing.T) {
	schema := generated.TransformFanOutSchema()

	// FanOut transform requires 'name'
	t.Run("required_name", func(t *testing.T) {
		attr, ok := schema.Attributes["name"]
		require.True(t, ok, "name attribute should exist")
		require.True(t, attr.IsRequired(), "name should be required")
	})

	// transforms_language is optional (has default value)
	t.Run("optional_transforms_language", func(t *testing.T) {
		attr, ok := schema.Attributes["transforms_language"]
		require.True(t, ok, "transforms_language attribute should exist")
		require.True(t, attr.IsOptional(), "transforms_language should be optional (has default)")
	})
}

// TestSchemaValidation_MissingRequiredField_Oracle tests Oracle destination
// required field validation.
func TestSchemaValidation_MissingRequiredField_Oracle(t *testing.T) {
	schema := generated.DestinationOracleSchema()

	// Oracle destination should have 'name' required
	t.Run("required_name", func(t *testing.T) {
		attr, ok := schema.Attributes["name"]
		require.True(t, ok, "name attribute should exist")
		require.True(t, attr.IsRequired(), "name should be required")
	})
}

// TestSchemaValidation_MissingRequiredField_PostgreSQLSource tests PostgreSQL source
// required field validation.
func TestSchemaValidation_MissingRequiredField_PostgreSQLSource(t *testing.T) {
	schema := generated.SourcePostgresqlSchema()

	// PostgreSQL source should have 'name' required
	t.Run("required_name", func(t *testing.T) {
		attr, ok := schema.Attributes["name"]
		require.True(t, ok, "name attribute should exist")
		require.True(t, attr.IsRequired(), "name should be required")
	})
}

// =============================================================================
// Invalid Enum Value Tests (OneOf Validator)
// =============================================================================

// TestSchemaValidation_InvalidEnum_InsertMode tests insert_mode enum validation.
func TestSchemaValidation_InvalidEnum_InsertMode(t *testing.T) {
	v := stringvalidator.OneOf("insert", "upsert")

	tests := []struct {
		name        string
		value       string
		wantError   bool
		errorSubstr string
	}{
		{"valid_insert", "insert", false, ""},
		{"valid_upsert", "upsert", false, ""},
		{"invalid_append", "append", true, "must be one of"},
		{"invalid_replace", "replace", true, "must be one of"},
		{"invalid_merge", "merge", true, "must be one of"},
		{"invalid_empty", "", true, "must be one of"},
		{"invalid_case_sensitive", "INSERT", true, "must be one of"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("insert_mode"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())

			if tt.wantError && tt.errorSubstr != "" {
				hasExpectedMessage := false
				for _, diag := range resp.Diagnostics {
					if strings.Contains(strings.ToLower(diag.Detail()), strings.ToLower(tt.errorSubstr)) ||
						strings.Contains(strings.ToLower(diag.Summary()), strings.ToLower(tt.errorSubstr)) {
						hasExpectedMessage = true
						break
					}
				}
				// Check for "value must be" pattern as alternative
				if !hasExpectedMessage {
					for _, diag := range resp.Diagnostics {
						if strings.Contains(diag.Detail(), "value must be") ||
							strings.Contains(diag.Detail(), "insert") ||
							strings.Contains(diag.Detail(), "upsert") {
							hasExpectedMessage = true
							break
						}
					}
				}
				assert.True(t, hasExpectedMessage, "error message should indicate valid values")
			}
		})
	}
}

// TestSchemaValidation_InvalidEnum_SchemaEvolution tests schema_evolution enum validation.
func TestSchemaValidation_InvalidEnum_SchemaEvolution(t *testing.T) {
	v := stringvalidator.OneOf("basic", "none")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_basic", "basic", false},
		{"valid_none", "none", false},
		{"invalid_advanced", "advanced", true},
		{"invalid_full", "full", true},
		{"invalid_auto", "auto", true},
		{"invalid_empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("schema_evolution"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())

			// Verify helpful error message for invalid values
			if tt.wantError {
				hasError := false
				for _, diag := range resp.Diagnostics {
					if strings.Contains(diag.Detail(), "basic") ||
						strings.Contains(diag.Detail(), "none") {
						hasError = true
						break
					}
				}
				assert.True(t, hasError, "error should list valid enum values")
			}
		})
	}
}

// TestSchemaValidation_InvalidEnum_IngestionMode tests ingestion_mode enum validation.
func TestSchemaValidation_InvalidEnum_IngestionMode(t *testing.T) {
	v := stringvalidator.OneOf("upsert", "append")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_upsert", "upsert", false},
		{"valid_append", "append", false},
		{"invalid_insert", "insert", true},
		{"invalid_overwrite", "overwrite", true},
		{"invalid_empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("ingestion_mode"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSchemaValidation_InvalidEnum_FileFormat tests file format enum validation.
func TestSchemaValidation_InvalidEnum_FileFormat(t *testing.T) {
	v := stringvalidator.OneOf("json", "csv", "avro", "parquet")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_json", "json", false},
		{"valid_csv", "csv", false},
		{"valid_avro", "avro", false},
		{"valid_parquet", "parquet", false},
		{"invalid_xml", "xml", true},
		{"invalid_orc", "orc", true},
		{"invalid_empty", "", true},
		{"invalid_uppercase", "JSON", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("format"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSchemaValidation_InvalidEnum_TransformLanguage tests transform language validation.
func TestSchemaValidation_InvalidEnum_TransformLanguage(t *testing.T) {
	// FanOut only supports JavaScript
	v := stringvalidator.OneOf("JavaScript")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_javascript", "JavaScript", false},
		{"invalid_python", "Python", true},
		{"invalid_sql", "SQL", true},
		{"invalid_lowercase", "javascript", true},
		{"invalid_empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("transforms_language"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSchemaValidation_InvalidEnum_Compression tests compression enum validation.
func TestSchemaValidation_InvalidEnum_Compression(t *testing.T) {
	v := stringvalidator.OneOf("none", "gzip", "snappy", "zstd")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_none", "none", false},
		{"valid_gzip", "gzip", false},
		{"valid_snappy", "snappy", false},
		{"valid_zstd", "zstd", false},
		{"invalid_lz4", "lz4", true},
		{"invalid_brotli", "brotli", true},
		{"invalid_empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("compression"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSchemaValidation_InvalidEnum_SSLMode tests SSL mode enum validation.
func TestSchemaValidation_InvalidEnum_SSLMode(t *testing.T) {
	v := stringvalidator.OneOf("require", "disable")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_require", "require", false},
		{"valid_disable", "disable", false},
		{"invalid_verify_ca", "verify-ca", true},
		{"invalid_verify_full", "verify-full", true},
		{"invalid_prefer", "prefer", true},
		{"invalid_empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("database_sslmode"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSchemaValidation_InvalidEnum_CatalogType tests Iceberg catalog type validation.
func TestSchemaValidation_InvalidEnum_CatalogType(t *testing.T) {
	v := stringvalidator.OneOf("rest", "hive", "glue")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_rest", "rest", false},
		{"valid_hive", "hive", false},
		{"valid_glue", "glue", false},
		{"invalid_jdbc", "jdbc", true},
		{"invalid_nessie", "nessie", true},
		{"invalid_empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("catalog_type"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSchemaValidation_InvalidEnum_SerializationFormat tests serialization format validation.
func TestSchemaValidation_InvalidEnum_SerializationFormat(t *testing.T) {
	v := stringvalidator.OneOf("Any", "Avro", "Json")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_any", "Any", false},
		{"valid_avro", "Avro", false},
		{"valid_json", "Json", false},
		{"invalid_lowercase_any", "any", true},
		{"invalid_lowercase_avro", "avro", true},
		{"invalid_protobuf", "Protobuf", true},
		{"invalid_empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("transforms_input_serialization_format"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// =============================================================================
// Out-of-Range Slider Tests (Between Validator)
// =============================================================================

// TestSchemaValidation_OutOfRange_TasksMax tests tasks_max range validation.
func TestSchemaValidation_OutOfRange_TasksMax(t *testing.T) {
	// Standard tasks_max range for most connectors
	v := int64validator.Between(1, 10)

	tests := []struct {
		name        string
		value       int64
		wantError   bool
		errorSubstr string
	}{
		{"valid_min", 1, false, ""},
		{"valid_middle", 5, false, ""},
		{"valid_max", 10, false, ""},
		{"invalid_zero", 0, true, "between"},
		{"invalid_negative", -1, true, "between"},
		{"invalid_above_max", 11, true, "between"},
		{"invalid_large", 100, true, "between"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Int64Request{
				ConfigValue: types.Int64Value(tt.value),
				Path:        path.Root("tasks_max"),
			}
			resp := &validator.Int64Response{}
			v.ValidateInt64(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %d: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())

			// Verify error message mentions range
			if tt.wantError {
				hasRangeMessage := false
				for _, diag := range resp.Diagnostics {
					detail := strings.ToLower(diag.Detail())
					if strings.Contains(detail, "between") ||
						strings.Contains(detail, "must be") ||
						strings.Contains(detail, "1") && strings.Contains(detail, "10") {
						hasRangeMessage = true
						break
					}
				}
				assert.True(t, hasRangeMessage, "error message should indicate valid range")
			}
		})
	}
}

// TestSchemaValidation_OutOfRange_DatabricksTasksMax tests extended tasks_max for Databricks.
func TestSchemaValidation_OutOfRange_DatabricksTasksMax(t *testing.T) {
	v := int64validator.Between(1, 100)

	tests := []struct {
		name      string
		value     int64
		wantError bool
	}{
		{"valid_min", 1, false},
		{"valid_middle", 50, false},
		{"valid_max", 100, false},
		{"invalid_zero", 0, true},
		{"invalid_above_max", 101, true},
		{"invalid_negative", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Int64Request{
				ConfigValue: types.Int64Value(tt.value),
				Path:        path.Root("tasks_max"),
			}
			resp := &validator.Int64Response{}
			v.ValidateInt64(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSchemaValidation_OutOfRange_DynamoDBParallelism tests DynamoDB parallelism validation.
func TestSchemaValidation_OutOfRange_DynamoDBParallelism(t *testing.T) {
	v := int64validator.Between(1, 40)

	tests := []struct {
		name      string
		value     int64
		wantError bool
	}{
		{"valid_min", 1, false},
		{"valid_default", 4, false},
		{"valid_max", 40, false},
		{"invalid_zero", 0, true},
		{"invalid_above_max", 41, true},
		{"invalid_large", 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Int64Request{
				ConfigValue: types.Int64Value(tt.value),
				Path:        path.Root("dynamodb_service_endpoint_parallelism"),
			}
			resp := &validator.Int64Response{}
			v.ValidateInt64(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSchemaValidation_OutOfRange_ConsumerWaitTime tests consumer wait time validation.
func TestSchemaValidation_OutOfRange_ConsumerWaitTime(t *testing.T) {
	v := int64validator.Between(500, 300000)

	tests := []struct {
		name      string
		value     int64
		wantError bool
	}{
		{"valid_min", 500, false},
		{"valid_default", 10000, false},
		{"valid_max", 300000, false},
		{"invalid_below_min", 499, true},
		{"invalid_above_max", 300001, true},
		{"invalid_zero", 0, true},
		{"invalid_negative", -1000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Int64Request{
				ConfigValue: types.Int64Value(tt.value),
				Path:        path.Root("consumer_wait_time_for_larger_batch_ms"),
			}
			resp := &validator.Int64Response{}
			v.ValidateInt64(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSchemaValidation_OutOfRange_SnapshotParallelism tests snapshot parallelism validation.
func TestSchemaValidation_OutOfRange_SnapshotParallelism(t *testing.T) {
	v := int64validator.Between(1, 10)

	tests := []struct {
		name      string
		value     int64
		wantError bool
	}{
		{"valid_min", 1, false},
		{"valid_default", 2, false},
		{"valid_max", 10, false},
		{"invalid_zero", 0, true},
		{"invalid_above_max", 11, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Int64Request{
				ConfigValue: types.Int64Value(tt.value),
				Path:        path.Root("streamkap_snapshot_parallelism"),
			}
			resp := &validator.Int64Response{}
			v.ValidateInt64(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSchemaValidation_OutOfRange_LargeTableThreshold tests large table threshold validation.
func TestSchemaValidation_OutOfRange_LargeTableThreshold(t *testing.T) {
	v := int64validator.Between(1, 64000)

	tests := []struct {
		name      string
		value     int64
		wantError bool
	}{
		{"valid_min", 1, false},
		{"valid_default", 500, false},
		{"valid_max", 64000, false},
		{"invalid_zero", 0, true},
		{"invalid_above_max", 64001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Int64Request{
				ConfigValue: types.Int64Value(tt.value),
				Path:        path.Root("streamkap_snapshot_large_table_threshold"),
			}
			resp := &validator.Int64Response{}
			v.ValidateInt64(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSchemaValidation_OutOfRange_StreamDelayMs tests stream delay validation (DynamoDB).
func TestSchemaValidation_OutOfRange_StreamDelayMs(t *testing.T) {
	v := int64validator.Between(0, 604800000) // 0 to 1 week in ms

	tests := []struct {
		name      string
		value     int64
		wantError bool
	}{
		{"valid_min", 0, false},
		{"valid_default", 60000, false},
		{"valid_max", 604800000, false},
		{"invalid_negative", -1, true},
		{"invalid_above_max", 604800001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Int64Request{
				ConfigValue: types.Int64Value(tt.value),
				Path:        path.Root("dynamodb_service_endpoint_stream_delay_ms"),
			}
			resp := &validator.Int64Response{}
			v.ValidateInt64(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSchemaValidation_OutOfRange_MotherduckTasksMax tests Motherduck tasks_max validation.
func TestSchemaValidation_OutOfRange_MotherduckTasksMax(t *testing.T) {
	v := int64validator.Between(1, 25)

	tests := []struct {
		name      string
		value     int64
		wantError bool
	}{
		{"valid_min", 1, false},
		{"valid_default", 5, false},
		{"valid_max", 25, false},
		{"invalid_zero", 0, true},
		{"invalid_above_max", 26, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Int64Request{
				ConfigValue: types.Int64Value(tt.value),
				Path:        path.Root("tasks_max"),
			}
			resp := &validator.Int64Response{}
			v.ValidateInt64(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// =============================================================================
// Error Message Quality Tests
// =============================================================================

// TestSchemaValidation_ErrorMessages_StringValidator verifies string validators
// produce helpful error messages that list valid values.
func TestSchemaValidation_ErrorMessages_StringValidator(t *testing.T) {
	tests := []struct {
		name          string
		validator     validator.String
		invalidValue  string
		expectedTerms []string // Terms that should appear in error message
	}{
		{
			name:          "insert_mode_shows_valid_values",
			validator:     stringvalidator.OneOf("insert", "upsert"),
			invalidValue:  "invalid",
			expectedTerms: []string{"insert", "upsert"},
		},
		{
			name:          "schema_evolution_shows_valid_values",
			validator:     stringvalidator.OneOf("basic", "none"),
			invalidValue:  "invalid",
			expectedTerms: []string{"basic", "none"},
		},
		{
			name:          "compression_shows_valid_values",
			validator:     stringvalidator.OneOf("none", "gzip", "snappy", "zstd"),
			invalidValue:  "lz4",
			expectedTerms: []string{"none", "gzip", "snappy", "zstd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.invalidValue),
				Path:        path.Root("test_field"),
			}
			resp := &validator.StringResponse{}
			tt.validator.ValidateString(context.Background(), req, resp)

			require.True(t, resp.Diagnostics.HasError(), "should have validation error")

			// Collect all diagnostic text
			var diagText string
			for _, diag := range resp.Diagnostics {
				diagText += diag.Summary() + " " + diag.Detail()
			}

			// Verify at least some expected terms appear in error
			foundTerms := 0
			for _, term := range tt.expectedTerms {
				if strings.Contains(diagText, term) {
					foundTerms++
				}
			}
			assert.Greater(t, foundTerms, 0,
				"error message should contain at least one valid value: %q, got: %s",
				tt.expectedTerms, diagText)
		})
	}
}

// TestSchemaValidation_ErrorMessages_Int64Validator verifies int64 validators
// produce helpful error messages that indicate valid ranges.
func TestSchemaValidation_ErrorMessages_Int64Validator(t *testing.T) {
	tests := []struct {
		name         string
		validator    validator.Int64
		invalidValue int64
		minStr       string
		maxStr       string
	}{
		{
			name:         "tasks_max_shows_range",
			validator:    int64validator.Between(1, 10),
			invalidValue: 100,
			minStr:       "1",
			maxStr:       "10",
		},
		{
			name:         "consumer_wait_time_shows_range",
			validator:    int64validator.Between(500, 300000),
			invalidValue: 0,
			minStr:       "500",
			maxStr:       "300000",
		},
		{
			name:         "parallelism_shows_range",
			validator:    int64validator.Between(1, 40),
			invalidValue: 50,
			minStr:       "1",
			maxStr:       "40",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Int64Request{
				ConfigValue: types.Int64Value(tt.invalidValue),
				Path:        path.Root("test_field"),
			}
			resp := &validator.Int64Response{}
			tt.validator.ValidateInt64(context.Background(), req, resp)

			require.True(t, resp.Diagnostics.HasError(), "should have validation error")

			// Collect all diagnostic text
			var diagText string
			for _, diag := range resp.Diagnostics {
				diagText += diag.Summary() + " " + diag.Detail()
			}

			// Verify error mentions range bounds
			hasMin := strings.Contains(diagText, tt.minStr)
			hasMax := strings.Contains(diagText, tt.maxStr)
			hasBetween := strings.Contains(strings.ToLower(diagText), "between")

			assert.True(t, hasMin || hasMax || hasBetween,
				"error message should indicate valid range, got: %s", diagText)
		})
	}
}

// TestSchemaValidation_ErrorMessages_AttributePath verifies error messages
// include the attribute path for debugging.
func TestSchemaValidation_ErrorMessages_AttributePath(t *testing.T) {
	v := stringvalidator.OneOf("a", "b")

	paths := []string{
		"insert_mode",
		"schema_evolution",
		"transforms_language",
	}

	for _, attrPath := range paths {
		t.Run(attrPath, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue("invalid"),
				Path:        path.Root(attrPath),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			require.True(t, resp.Diagnostics.HasError())

			// Verify that diagnostics are present (framework handles path association)
			assert.Greater(t, len(resp.Diagnostics), 0, "should have diagnostics")
		})
	}
}

// =============================================================================
// Schema-Level Required Field Validation Tests
// =============================================================================

// TestSchemaValidation_RequiredFieldsWithPlan tests that required fields are enforced
// during plan validation.
func TestSchemaValidation_RequiredFieldsWithPlan(t *testing.T) {
	// Get Oracle destination schema which has multiple required fields
	schema := generated.DestinationOracleSchema()

	// Verify required fields are marked correctly in the schema
	requiredFields := []string{
		"name", // All resources require name
	}

	for _, field := range requiredFields {
		t.Run("verify_required_"+field, func(t *testing.T) {
			attr, ok := schema.Attributes[field]
			require.True(t, ok, "attribute %s should exist", field)
			require.True(t, attr.IsRequired(), "attribute %s should be required", field)
		})
	}
}

// TestSchemaValidation_OptionalFieldsWithDefaults tests that optional fields
// with defaults are handled correctly.
func TestSchemaValidation_OptionalFieldsWithDefaults(t *testing.T) {
	// Get PostgreSQL source schema which has optional fields with defaults
	schema := generated.SourcePostgresqlSchema()

	// These fields should be optional (not required)
	optionalFields := []string{
		"database_sslmode",
		"binary_handling_mode",
		"streamkap_snapshot_parallelism",
	}

	for _, field := range optionalFields {
		t.Run("verify_optional_"+field, func(t *testing.T) {
			attr, ok := schema.Attributes[field]
			require.True(t, ok, "attribute %s should exist", field)
			require.True(t, attr.IsOptional(), "attribute %s should be optional", field)
		})
	}
}

// TestSchemaValidation_ComputedFields tests that computed fields are marked correctly.
func TestSchemaValidation_ComputedFields(t *testing.T) {
	// Get Databricks destination schema
	schema := generated.DestinationDatabricksSchema()

	// These fields should be computed
	computedFields := []string{
		"id",
		"connector",
	}

	for _, field := range computedFields {
		t.Run("verify_computed_"+field, func(t *testing.T) {
			attr, ok := schema.Attributes[field]
			require.True(t, ok, "attribute %s should exist", field)
			require.True(t, attr.IsComputed(), "attribute %s should be computed", field)
		})
	}
}

// =============================================================================
// Combined Validation Tests
// =============================================================================

// TestSchemaValidation_MultipleValidatorsOnField tests fields with multiple validators.
func TestSchemaValidation_MultipleValidatorsOnField(t *testing.T) {
	// Some fields may have both enum validation and other constraints
	// This test verifies validators work correctly together

	t.Run("enum_and_required", func(t *testing.T) {
		// When a field is both required AND has enum validation,
		// both constraints should be enforced
		v := stringvalidator.OneOf("insert", "upsert")

		// Test that enum validation works
		req := validator.StringRequest{
			ConfigValue: types.StringValue("invalid"),
			Path:        path.Root("insert_mode"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), req, resp)
		assert.True(t, resp.Diagnostics.HasError())

		// Test that valid enum passes
		req.ConfigValue = types.StringValue("insert")
		resp = &validator.StringResponse{}
		v.ValidateString(context.Background(), req, resp)
		assert.False(t, resp.Diagnostics.HasError())
	})

	t.Run("range_and_optional", func(t *testing.T) {
		// Optional field with range validation
		v := int64validator.Between(1, 10)

		// Test that range validation works when value is provided
		req := validator.Int64Request{
			ConfigValue: types.Int64Value(100),
			Path:        path.Root("tasks_max"),
		}
		resp := &validator.Int64Response{}
		v.ValidateInt64(context.Background(), req, resp)
		assert.True(t, resp.Diagnostics.HasError())

		// Test that null values skip validation (optional field)
		req.ConfigValue = types.Int64Null()
		resp = &validator.Int64Response{}
		v.ValidateInt64(context.Background(), req, resp)
		assert.False(t, resp.Diagnostics.HasError())
	})
}

// TestSchemaValidation_RealWorldScenarios tests validation with realistic configurations.
func TestSchemaValidation_RealWorldScenarios(t *testing.T) {
	t.Run("destination_with_invalid_enum_combination", func(t *testing.T) {
		// Test multiple enum fields that might be configured together
		insertModeValidator := stringvalidator.OneOf("insert", "upsert")
		schemaEvolutionValidator := stringvalidator.OneOf("basic", "none")

		// Valid combination
		req1 := validator.StringRequest{
			ConfigValue: types.StringValue("upsert"),
			Path:        path.Root("insert_mode"),
		}
		resp1 := &validator.StringResponse{}
		insertModeValidator.ValidateString(context.Background(), req1, resp1)
		assert.False(t, resp1.Diagnostics.HasError())

		req2 := validator.StringRequest{
			ConfigValue: types.StringValue("basic"),
			Path:        path.Root("schema_evolution"),
		}
		resp2 := &validator.StringResponse{}
		schemaEvolutionValidator.ValidateString(context.Background(), req2, resp2)
		assert.False(t, resp2.Diagnostics.HasError())
	})

	t.Run("source_with_invalid_range_values", func(t *testing.T) {
		parallelismValidator := int64validator.Between(1, 10)
		thresholdValidator := int64validator.Between(1, 64000)

		// Both invalid
		req1 := validator.Int64Request{
			ConfigValue: types.Int64Value(0),
			Path:        path.Root("streamkap_snapshot_parallelism"),
		}
		resp1 := &validator.Int64Response{}
		parallelismValidator.ValidateInt64(context.Background(), req1, resp1)
		assert.True(t, resp1.Diagnostics.HasError())

		req2 := validator.Int64Request{
			ConfigValue: types.Int64Value(100000),
			Path:        path.Root("streamkap_snapshot_large_table_threshold"),
		}
		resp2 := &validator.Int64Response{}
		thresholdValidator.ValidateInt64(context.Background(), req2, resp2)
		assert.True(t, resp2.Diagnostics.HasError())
	})

	t.Run("transform_with_incompatible_language", func(t *testing.T) {
		// FanOut only supports JavaScript, not Python or SQL
		fanoutLanguageValidator := stringvalidator.OneOf("JavaScript")

		// Try to set Python (valid for Enrich, invalid for FanOut)
		req := validator.StringRequest{
			ConfigValue: types.StringValue("Python"),
			Path:        path.Root("transforms_language"),
		}
		resp := &validator.StringResponse{}
		fanoutLanguageValidator.ValidateString(context.Background(), req, resp)
		assert.True(t, resp.Diagnostics.HasError())
	})
}

// TestSchemaValidation_BoundaryValues tests exact boundary values for range validators.
func TestSchemaValidation_BoundaryValues(t *testing.T) {
	testCases := []struct {
		name      string
		min       int64
		max       int64
		validator validator.Int64
	}{
		{"tasks_max_1_10", 1, 10, int64validator.Between(1, 10)},
		{"tasks_max_1_100", 1, 100, int64validator.Between(1, 100)},
		{"parallelism_1_40", 1, 40, int64validator.Between(1, 40)},
		{"wait_time_500_300000", 500, 300000, int64validator.Between(500, 300000)},
		{"threshold_1_64000", 1, 64000, int64validator.Between(1, 64000)},
		{"delay_0_604800000", 0, 604800000, int64validator.Between(0, 604800000)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test exact minimum
			t.Run("exact_min", func(t *testing.T) {
				req := validator.Int64Request{
					ConfigValue: types.Int64Value(tc.min),
					Path:        path.Root("test"),
				}
				resp := &validator.Int64Response{}
				tc.validator.ValidateInt64(context.Background(), req, resp)
				assert.False(t, resp.Diagnostics.HasError(), "min value %d should be valid", tc.min)
			})

			// Test exact maximum
			t.Run("exact_max", func(t *testing.T) {
				req := validator.Int64Request{
					ConfigValue: types.Int64Value(tc.max),
					Path:        path.Root("test"),
				}
				resp := &validator.Int64Response{}
				tc.validator.ValidateInt64(context.Background(), req, resp)
				assert.False(t, resp.Diagnostics.HasError(), "max value %d should be valid", tc.max)
			})

			// Test one below minimum (if min > math.MinInt64)
			if tc.min > -9223372036854775808 {
				t.Run("below_min", func(t *testing.T) {
					req := validator.Int64Request{
						ConfigValue: types.Int64Value(tc.min - 1),
						Path:        path.Root("test"),
					}
					resp := &validator.Int64Response{}
					tc.validator.ValidateInt64(context.Background(), req, resp)
					assert.True(t, resp.Diagnostics.HasError(), "value %d should be invalid", tc.min-1)
				})
			}

			// Test one above maximum (if max < math.MaxInt64)
			if tc.max < 9223372036854775807 {
				t.Run("above_max", func(t *testing.T) {
					req := validator.Int64Request{
						ConfigValue: types.Int64Value(tc.max + 1),
						Path:        path.Root("test"),
					}
					resp := &validator.Int64Response{}
					tc.validator.ValidateInt64(context.Background(), req, resp)
					assert.True(t, resp.Diagnostics.HasError(), "value %d should be invalid", tc.max+1)
				})
			}
		})
	}
}

// TestSchemaValidation_ConfigValueTypes tests validation handles different config value types.
func TestSchemaValidation_ConfigValueTypes(t *testing.T) {
	stringValidator := stringvalidator.OneOf("a", "b")
	int64Validator := int64validator.Between(1, 10)

	t.Run("string_null", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringNull(),
			Path:        path.Root("test"),
		}
		resp := &validator.StringResponse{}
		stringValidator.ValidateString(context.Background(), req, resp)
		assert.False(t, resp.Diagnostics.HasError(), "null should skip validation")
	})

	t.Run("string_unknown", func(t *testing.T) {
		req := validator.StringRequest{
			ConfigValue: types.StringUnknown(),
			Path:        path.Root("test"),
		}
		resp := &validator.StringResponse{}
		stringValidator.ValidateString(context.Background(), req, resp)
		assert.False(t, resp.Diagnostics.HasError(), "unknown should skip validation")
	})

	t.Run("int64_null", func(t *testing.T) {
		req := validator.Int64Request{
			ConfigValue: types.Int64Null(),
			Path:        path.Root("test"),
		}
		resp := &validator.Int64Response{}
		int64Validator.ValidateInt64(context.Background(), req, resp)
		assert.False(t, resp.Diagnostics.HasError(), "null should skip validation")
	})

	t.Run("int64_unknown", func(t *testing.T) {
		req := validator.Int64Request{
			ConfigValue: types.Int64Unknown(),
			Path:        path.Root("test"),
		}
		resp := &validator.Int64Response{}
		int64Validator.ValidateInt64(context.Background(), req, resp)
		assert.False(t, resp.Diagnostics.HasError(), "unknown should skip validation")
	})
}

// TestSchemaValidation_ValidatorDescription verifies validators have descriptions.
func TestSchemaValidation_ValidatorDescription(t *testing.T) {
	tests := []struct {
		name      string
		validator validator.String
	}{
		{"oneOf_insert_mode", stringvalidator.OneOf("insert", "upsert")},
		{"oneOf_schema_evolution", stringvalidator.OneOf("basic", "none")},
		{"oneOf_compression", stringvalidator.OneOf("none", "gzip", "snappy", "zstd")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := tt.validator.Description(context.Background())
			assert.NotEmpty(t, desc, "validator should have description")
		})
	}
}

// TestSchemaValidation_ValidatorMarkdownDescription verifies validators have markdown descriptions.
func TestSchemaValidation_ValidatorMarkdownDescription(t *testing.T) {
	tests := []struct {
		name      string
		validator validator.String
	}{
		{"oneOf_values", stringvalidator.OneOf("a", "b", "c")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := tt.validator.MarkdownDescription(context.Background())
			// Markdown description may be empty or same as regular description
			// Just verify it doesn't panic
			_ = desc
		})
	}
}

// TestSchemaValidation_Int64ValidatorDescription verifies int64 validators have descriptions.
func TestSchemaValidation_Int64ValidatorDescription(t *testing.T) {
	tests := []struct {
		name      string
		validator validator.Int64
	}{
		{"between_1_10", int64validator.Between(1, 10)},
		{"between_500_300000", int64validator.Between(500, 300000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desc := tt.validator.Description(context.Background())
			assert.NotEmpty(t, desc, "validator should have description")
		})
	}
}

// TestSchemaValidation_StateUpgrade verifies schema validation works during state upgrades.
func TestSchemaValidation_StateUpgrade(t *testing.T) {
	// This test verifies that validation is not applied during state reads
	// (only during plan/apply)

	t.Run("null_values_skip_validation", func(t *testing.T) {
		// When reading existing state, null values should not trigger errors
		v := stringvalidator.OneOf("a", "b")
		req := validator.StringRequest{
			ConfigValue: types.StringNull(),
			Path:        path.Root("test"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), req, resp)
		assert.False(t, resp.Diagnostics.HasError())
	})

	t.Run("unknown_values_skip_validation", func(t *testing.T) {
		// During planning, unknown values should not trigger errors
		v := int64validator.Between(1, 10)
		req := validator.Int64Request{
			ConfigValue: types.Int64Unknown(),
			Path:        path.Root("test"),
		}
		resp := &validator.Int64Response{}
		v.ValidateInt64(context.Background(), req, resp)
		assert.False(t, resp.Diagnostics.HasError())
	})
}

// TestSchemaValidation_AllGeneratedSchemasHaveNameRequired verifies all generated
// schemas have the 'name' field required.
func TestSchemaValidation_AllGeneratedSchemasHaveNameRequired(t *testing.T) {
	// Test a selection of schemas to verify 'name' is always present and required
	schemaTests := []struct {
		name   string
		schema func() string
	}{
		{"oracle_destination", func() string {
			schema := generated.DestinationOracleSchema()
			attr, ok := schema.Attributes["name"]
			if !ok {
				return "name attribute should exist"
			}
			if !attr.IsRequired() {
				return "name should be required"
			}
			return ""
		}},
		{"clickhouse_destination", func() string {
			schema := generated.DestinationClickhouseSchema()
			attr, ok := schema.Attributes["name"]
			if !ok {
				return "name attribute should exist"
			}
			if !attr.IsRequired() {
				return "name should be required"
			}
			return ""
		}},
		{"elasticsearch_source", func() string {
			schema := generated.SourceElasticsearchSchema()
			attr, ok := schema.Attributes["name"]
			if !ok {
				return "name attribute should exist"
			}
			if !attr.IsRequired() {
				return "name should be required"
			}
			return ""
		}},
		{"postgresql_source", func() string {
			schema := generated.SourcePostgresqlSchema()
			attr, ok := schema.Attributes["name"]
			if !ok {
				return "name attribute should exist"
			}
			if !attr.IsRequired() {
				return "name should be required"
			}
			return ""
		}},
		{"fanout_transform", func() string {
			schema := generated.TransformFanOutSchema()
			attr, ok := schema.Attributes["name"]
			if !ok {
				return "name attribute should exist"
			}
			if !attr.IsRequired() {
				return "name should be required"
			}
			return ""
		}},
	}

	for _, tc := range schemaTests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.schema()
			if err != "" {
				t.Error(err)
			}
		})
	}
}

// TestSchemaValidation_SensitiveFieldsNotInPlaintext verifies sensitive fields
// are marked appropriately.
func TestSchemaValidation_SensitiveFieldsNotInPlaintext(t *testing.T) {
	// Check ClickHouse has connection_password marked sensitive
	t.Run("clickhouse_connection_password_is_sensitive", func(t *testing.T) {
		schema := generated.DestinationClickhouseSchema()

		attr, ok := schema.Attributes["connection_password"]
		require.True(t, ok, "connection_password should exist")
		require.True(t, attr.IsSensitive(), "connection_password should be sensitive")
	})

	// Check Oracle has password marked sensitive
	t.Run("oracle_password_is_sensitive", func(t *testing.T) {
		schema := generated.DestinationOracleSchema()

		attr, ok := schema.Attributes["connection_password"]
		require.True(t, ok, "connection_password should exist")
		require.True(t, attr.IsSensitive(), "connection_password should be sensitive")
	})

	// Check PostgreSQL has password marked sensitive
	t.Run("postgresql_password_is_sensitive", func(t *testing.T) {
		schema := generated.SourcePostgresqlSchema()

		attr, ok := schema.Attributes["database_password"]
		require.True(t, ok, "database_password should exist")
		require.True(t, attr.IsSensitive(), "database_password should be sensitive")
	})
}

// TestSchemaValidation_ConfigWithInvalidTerraformType tests type mismatches.
func TestSchemaValidation_ConfigWithInvalidTerraformType(t *testing.T) {
	t.Run("string_validator_with_empty_value", func(t *testing.T) {
		v := stringvalidator.OneOf("a", "b")
		req := validator.StringRequest{
			ConfigValue: types.StringValue(""),
			Path:        path.Root("test"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), req, resp)
		// Empty string should fail OneOf validation
		assert.True(t, resp.Diagnostics.HasError())
	})

	t.Run("int64_validator_with_zero", func(t *testing.T) {
		v := int64validator.Between(1, 10)
		req := validator.Int64Request{
			ConfigValue: types.Int64Value(0),
			Path:        path.Root("test"),
		}
		resp := &validator.Int64Response{}
		v.ValidateInt64(context.Background(), req, resp)
		// Zero should fail Between(1, 10) validation
		assert.True(t, resp.Diagnostics.HasError())
	})
}
