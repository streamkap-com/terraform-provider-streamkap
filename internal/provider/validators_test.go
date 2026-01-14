package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// Validator tests verify that validators from terraform-plugin-framework-validators
// work correctly for the specific value sets used in Streamkap provider schemas.
// These tests run without API calls and catch validation configuration bugs early.

// TestSSLModeValidator tests the SSL mode validator used in PostgreSQL sources.
func TestSSLModeValidator(t *testing.T) {
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
		{"invalid_mode", "invalid", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("database_sslmode"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestIngestionModeValidator tests the ingestion mode validator used in destinations.
func TestIngestionModeValidator(t *testing.T) {
	v := stringvalidator.OneOf("upsert", "append")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_upsert", "upsert", false},
		{"valid_append", "append", false},
		{"invalid_insert", "insert", true},
		{"invalid_replace", "replace", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("ingestion_mode"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestInsertModeValidator tests the insert mode validator used in some destinations.
func TestInsertModeValidator(t *testing.T) {
	v := stringvalidator.OneOf("insert", "upsert")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_insert", "insert", false},
		{"valid_upsert", "upsert", false},
		{"invalid_append", "append", true},
		{"invalid_replace", "replace", true},
		{"empty_string", "", true},
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
		})
	}
}

// TestSchemaEvolutionValidator tests the schema evolution validator used in destinations.
func TestSchemaEvolutionValidator(t *testing.T) {
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
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("schema_evolution"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestBinaryHandlingModeValidator tests the binary handling mode validator used in sources.
func TestBinaryHandlingModeValidator(t *testing.T) {
	v := stringvalidator.OneOf("bytes", "base64", "base64-url-safe", "hex")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_bytes", "bytes", false},
		{"valid_base64", "base64", false},
		{"valid_base64_url_safe", "base64-url-safe", false},
		{"valid_hex", "hex", false},
		{"invalid_binary", "binary", true},
		{"invalid_raw", "raw", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("binary_handling_mode"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSnapshotReadOnlyValidator tests the snapshot read only validator used in PostgreSQL sources.
func TestSnapshotReadOnlyValidator(t *testing.T) {
	v := stringvalidator.OneOf("Yes", "No")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_yes", "Yes", false},
		{"valid_no", "No", false},
		{"invalid_lowercase_yes", "yes", true},
		{"invalid_lowercase_no", "no", true},
		{"invalid_true", "true", true},
		{"invalid_false", "false", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("snapshot_read_only"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestPartitionModeValidator tests the partition mode validator used in Databricks destination.
func TestPartitionModeValidator(t *testing.T) {
	v := stringvalidator.OneOf("by_topic", "by_partition", "by_topic_and_partition")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_by_topic", "by_topic", false},
		{"valid_by_partition", "by_partition", false},
		{"valid_by_topic_and_partition", "by_topic_and_partition", false},
		{"invalid_none", "none", true},
		{"invalid_auto", "auto", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("partition_mode"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestTransformLanguageValidator tests the transform language validator used in transforms.
func TestTransformLanguageValidator(t *testing.T) {
	v := stringvalidator.OneOf("JavaScript", "Python")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_javascript", "JavaScript", false},
		{"valid_python", "Python", false},
		{"invalid_lowercase_javascript", "javascript", true},
		{"invalid_lowercase_python", "python", true},
		{"invalid_sql", "SQL", true},
		{"invalid_go", "Go", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("transforms_language"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestArrayEncodingValidator tests the array encoding validator used in MongoDB sources.
func TestArrayEncodingValidator(t *testing.T) {
	v := stringvalidator.OneOf("array", "array_string")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_array", "array", false},
		{"valid_array_string", "array_string", false},
		{"invalid_json", "json", true},
		{"invalid_list", "list", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("transforms_unwrap_array_encoding"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestDocumentEncodingValidator tests the document encoding validator used in MongoDB sources.
func TestDocumentEncodingValidator(t *testing.T) {
	v := stringvalidator.OneOf("document", "string")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_document", "document", false},
		{"valid_string", "string", false},
		{"invalid_json", "json", true},
		{"invalid_object", "object", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("transforms_unwrap_document_encoding"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestOutputFormatValidator tests the output format validator used in S3 destinations.
func TestOutputFormatValidator(t *testing.T) {
	v := stringvalidator.OneOf("JSON Lines", "JSON Array", "Parquet")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_json_lines", "JSON Lines", false},
		{"valid_json_array", "JSON Array", false},
		{"valid_parquet", "Parquet", false},
		{"invalid_csv", "CSV", true},
		{"invalid_avro", "Avro", true},
		{"invalid_lowercase", "json lines", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("format"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestCompressionValidator tests the compression validator used in S3 destinations.
func TestCompressionValidator(t *testing.T) {
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
		{"invalid_bzip2", "bzip2", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("compression"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestIcebergCatalogTypeValidator tests the catalog type validator used in Iceberg destination.
func TestIcebergCatalogTypeValidator(t *testing.T) {
	v := stringvalidator.OneOf("rest", "hive", "glue")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_rest", "rest", false},
		{"valid_hive", "hive", false},
		{"valid_glue", "glue", false},
		{"invalid_s3", "s3", true},
		{"invalid_jdbc", "jdbc", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("catalog_type"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestKafkaDataFormatValidator tests the data format validator used in Kafka destination.
func TestKafkaDataFormatValidator(t *testing.T) {
	v := stringvalidator.OneOf("avro", "json")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_avro", "avro", false},
		{"valid_json", "json", false},
		{"invalid_protobuf", "protobuf", true},
		{"invalid_string", "string", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("output_data_format"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestDeleteHandlingModeValidator tests the delete handling mode validator used in destinations.
func TestDeleteHandlingModeValidator(t *testing.T) {
	v := stringvalidator.OneOf("none", "record_key", "record_value")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_none", "none", false},
		{"valid_record_key", "record_key", false},
		{"valid_record_value", "record_value", false},
		{"invalid_drop", "drop", true},
		{"invalid_tombstone", "tombstone", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("delete_handling_mode"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestHTTPSinkAuthTypeValidator tests the auth type validator used in HTTP sink destination.
func TestHTTPSinkAuthTypeValidator(t *testing.T) {
	v := stringvalidator.OneOf("none", "static", "oauth2")

	tests := []struct {
		name      string
		value     string
		wantError bool
	}{
		{"valid_none", "none", false},
		{"valid_static", "static", false},
		{"valid_oauth2", "oauth2", false},
		{"invalid_basic", "basic", true},
		{"invalid_bearer", "bearer", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("http_api_auth_type"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestTransformTopicFormatValidator tests the topic format validator used in transforms.
func TestTransformTopicFormatValidator(t *testing.T) {
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
		{"invalid_lowercase_json", "json", true},
		{"invalid_protobuf", "Protobuf", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.StringRequest{
				ConfigValue: types.StringValue(tt.value),
				Path:        path.Root("transforms_input_topic_type"),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %q: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestTasksMaxValidator tests the tasks_max validator used in various connectors.
func TestTasksMaxValidator(t *testing.T) {
	v := int64validator.Between(1, 10)

	tests := []struct {
		name      string
		value     int64
		wantError bool
	}{
		{"valid_min", 1, false},
		{"valid_middle", 5, false},
		{"valid_max", 10, false},
		{"invalid_zero", 0, true},
		{"invalid_negative", -1, true},
		{"invalid_above_max", 11, true},
		{"invalid_large", 100, true},
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
		})
	}
}

// TestDatabricksTasksMaxValidator tests the extended tasks_max validator for Databricks.
func TestDatabricksTasksMaxValidator(t *testing.T) {
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
		})
	}
}

// TestConsumerWaitTimeValidator tests the wait time validator for Databricks destination.
func TestConsumerWaitTimeValidator(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Int64Request{
				ConfigValue: types.Int64Value(tt.value),
				Path:        path.Root("consumer_wait_time_for_larger_batch_ms"),
			}
			resp := &validator.Int64Response{}
			v.ValidateInt64(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %d: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestDynamoDBParallelismValidator tests the parallelism validator for DynamoDB source.
func TestDynamoDBParallelismValidator(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Int64Request{
				ConfigValue: types.Int64Value(tt.value),
				Path:        path.Root("dynamodb_service_endpoint_parallelism"),
			}
			resp := &validator.Int64Response{}
			v.ValidateInt64(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %d: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestSnapshotParallelismValidator tests the snapshot parallelism validator.
func TestSnapshotParallelismValidator(t *testing.T) {
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

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %d: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestLargeTableThresholdValidator tests the large table threshold validator.
func TestLargeTableThresholdValidator(t *testing.T) {
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

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %d: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestEnrichAsyncConcurrencyValidator tests the concurrency validator for EnrichAsync transform.
func TestEnrichAsyncConcurrencyValidator(t *testing.T) {
	v := int64validator.Between(1, 50)

	tests := []struct {
		name      string
		value     int64
		wantError bool
	}{
		{"valid_min", 1, false},
		{"valid_default", 5, false},
		{"valid_max", 50, false},
		{"invalid_zero", 0, true},
		{"invalid_above_max", 51, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validator.Int64Request{
				ConfigValue: types.Int64Value(tt.value),
				Path:        path.Root("transforms_async_concurrency"),
			}
			resp := &validator.Int64Response{}
			v.ValidateInt64(context.Background(), req, resp)

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %d: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestMotherduckTasksMaxValidator tests the tasks_max validator for Motherduck destination.
func TestMotherduckTasksMaxValidator(t *testing.T) {
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

			assert.Equal(t, tt.wantError, resp.Diagnostics.HasError(),
				"value %d: expected error=%v, got error=%v", tt.value, tt.wantError, resp.Diagnostics.HasError())
		})
	}
}

// TestNullValueHandling verifies validators handle null/unknown values correctly.
func TestNullValueHandling(t *testing.T) {
	t.Run("string_validator_with_null", func(t *testing.T) {
		v := stringvalidator.OneOf("a", "b", "c")
		req := validator.StringRequest{
			ConfigValue: types.StringNull(),
			Path:        path.Root("test"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), req, resp)

		// Null values should not trigger validation errors (they are handled separately)
		assert.False(t, resp.Diagnostics.HasError(), "null value should not trigger validation error")
	})

	t.Run("string_validator_with_unknown", func(t *testing.T) {
		v := stringvalidator.OneOf("a", "b", "c")
		req := validator.StringRequest{
			ConfigValue: types.StringUnknown(),
			Path:        path.Root("test"),
		}
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), req, resp)

		// Unknown values should not trigger validation errors (they are deferred)
		assert.False(t, resp.Diagnostics.HasError(), "unknown value should not trigger validation error")
	})

	t.Run("int64_validator_with_null", func(t *testing.T) {
		v := int64validator.Between(1, 10)
		req := validator.Int64Request{
			ConfigValue: types.Int64Null(),
			Path:        path.Root("test"),
		}
		resp := &validator.Int64Response{}
		v.ValidateInt64(context.Background(), req, resp)

		// Null values should not trigger validation errors
		assert.False(t, resp.Diagnostics.HasError(), "null value should not trigger validation error")
	})

	t.Run("int64_validator_with_unknown", func(t *testing.T) {
		v := int64validator.Between(1, 10)
		req := validator.Int64Request{
			ConfigValue: types.Int64Unknown(),
			Path:        path.Root("test"),
		}
		resp := &validator.Int64Response{}
		v.ValidateInt64(context.Background(), req, resp)

		// Unknown values should not trigger validation errors
		assert.False(t, resp.Diagnostics.HasError(), "unknown value should not trigger validation error")
	})
}
