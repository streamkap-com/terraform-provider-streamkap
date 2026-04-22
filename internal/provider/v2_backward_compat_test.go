package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
)

// TestV2BackwardCompat_SourceAttributes verifies that every attribute present in
// v2.1.19 source schemas still resolves against the current (v3.x) schema, either
// as a direct attribute or as a deprecated alias. This catches silent regressions
// where a codegen refresh or manual edit removes an old name that v2.1.19 users
// still have in their Terraform config.
//
// The attribute lists are the exact set of schema.Attributes keys present on each
// connector in tag v2.1.19 (see `git show v2.1.19:internal/resource/source/<c>.go`).
// When this test fails, either:
//  1. Restore the missing attribute as a deprecated alias (preferred), or
//  2. Intentionally remove support (update this list) and document in MIGRATION.md.
func TestV2BackwardCompat_SourceAttributes(t *testing.T) {
	cases := []struct {
		name       string
		newFn      func() resource.Resource
		v219Attrs  []string
		exceptions map[string]string // attr → reason it's OK to be missing
	}{
		{
			name:      "source_postgresql",
			newFn:     source.NewPostgreSQLResource,
			v219Attrs: postgresqlV219Attrs,
		},
		{
			name:      "source_mysql",
			newFn:     source.NewMySQLResource,
			v219Attrs: mysqlV219Attrs,
		},
		{
			name:      "source_mongodb",
			newFn:     source.NewMongoDBResource,
			v219Attrs: mongodbV219Attrs,
		},
		{
			name:      "source_sqlserver",
			newFn:     source.NewSQLServerResource,
			v219Attrs: sqlserverV219Attrs,
			exceptions: map[string]string{
				// v2.1.19 `database_dbname` → `database.dbname` API field was replaced
				// with v3.x `database_names` → `database.names` (different API field,
				// multi-DB support). Cannot be aliased in the provider; backend contract changed.
				"database_dbname": "backend API field renamed database.dbname → database.names (multi-DB support); requires config migration, see MIGRATION.md",
				// v2.1.19 snapshot_custom_table_config was a map<string, {chunks}>. v3.x
				// replaced it with streamkap_snapshot_custom_table_config as a JSON string.
				// Not a scalar rename; cannot be aliased as a deprecated string alias.
				"snapshot_custom_table_config": "type changed from map-of-objects to JSON string (streamkap_snapshot_custom_table_config); see MIGRATION.md",
			},
		},
		{
			name:      "source_dynamodb",
			newFn:     source.NewDynamoDBResource,
			v219Attrs: dynamodbV219Attrs,
			exceptions: map[string]string{
				// v2.1.19 `table_include_list_user_defined` was Required. v3.x schema
				// renames it to `table_include_list` (same API field). Aliasing a
				// Required field is complex; the rename is documented in MIGRATION.md.
				"table_include_list_user_defined": "renamed to table_include_list (Required field, config migration required)",
			},
		},
		{
			name:      "source_kafkadirect",
			newFn:     source.NewKafkaDirectResource,
			v219Attrs: kafkadirectV219Attrs,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := tc.newFn()

			resp := &resource.SchemaResponse{}
			r.Schema(context.Background(), resource.SchemaRequest{}, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("schema request returned errors: %v", resp.Diagnostics)
			}

			for _, attr := range tc.v219Attrs {
				if _, exists := resp.Schema.Attributes[attr]; !exists {
					if reason, ok := tc.exceptions[attr]; ok {
						t.Logf("%s: %q is intentionally missing: %s", tc.name, attr, reason)
						continue
					}
					t.Errorf("%s: v2.1.19 attribute %q is missing from current schema — add as deprecated alias or document in the test's exceptions map + MIGRATION.md", tc.name, attr)
				}
			}
		})
	}
}

// v2.1.19 attribute lists, snapshotted from:
//   git show v2.1.19:internal/resource/source/<connector>.go
// Any addition/removal here should correspond to a deliberate compatibility decision.

// Each list is the exact output of:
//   git show v2.1.19:internal/resource/source/<connector>.go |
//     awk '/schema\.Schema{/{s=1} s{print}' | grep -oE '"[a-z_][a-z_0-9]*":' | sort -u
// minus nested-attribute names (like `chunks` inside snapshot_custom_table_config).

var postgresqlV219Attrs = []string{
	"binary_handling_mode", "column_exclude_list", "column_include_list", "connector",
	"database_dbname", "database_hostname", "database_password", "database_port",
	"database_sslmode", "database_user",
	"heartbeat_data_collection_schema_or_database", "heartbeat_enabled",
	"id", "include_source_db_name_in_table_name",
	"insert_static_key_field_1", "insert_static_key_field_2",
	"insert_static_key_value_1", "insert_static_key_value_2",
	"insert_static_value_1", "insert_static_value_2",
	"insert_static_value_field_1", "insert_static_value_field_2",
	"name", "predicates_istopictoenrich_pattern", "publication_name",
	"schema_include_list", "signal_data_collection_schema_or_database",
	"slot_name", "snapshot_read_only",
	"ssh_enabled", "ssh_host", "ssh_port", "ssh_user",
	"table_include_list",
}

var mysqlV219Attrs = []string{
	"binary_handling_mode", "column_exclude_list", "column_include_list", "connector",
	"database_connection_timezone", "database_hostname", "database_include_list",
	"database_password", "database_port", "database_user",
	"heartbeat_data_collection_schema_or_database", "heartbeat_enabled",
	"id",
	"insert_static_key_field_1", "insert_static_key_field_2",
	"insert_static_key_value_1", "insert_static_key_value_2",
	"insert_static_value_1", "insert_static_value_2",
	"insert_static_value_field_1", "insert_static_value_field_2",
	"name", "predicates_istopictoenrich_pattern",
	"signal_data_collection_schema_or_database", "snapshot_gtid",
	"ssh_enabled", "ssh_host", "ssh_port", "ssh_user",
	"table_include_list",
}

var mongodbV219Attrs = []string{
	"array_encoding", "collection_include_list", "connector",
	"database_include_list", "id",
	"insert_static_key_field_1", "insert_static_key_field_2",
	"insert_static_key_value_1", "insert_static_key_value_2",
	"insert_static_value_1", "insert_static_value_2",
	"insert_static_value_field_1", "insert_static_value_field_2",
	"mongodb_connection_string", "name", "nested_document_encoding",
	"predicates_istopictoenrich_pattern",
	"signal_data_collection_schema_or_database",
	"ssh_enabled", "ssh_host", "ssh_port", "ssh_user",
}

var sqlserverV219Attrs = []string{
	"binary_handling_mode", "column_exclude_list", "connector",
	"database_dbname", "database_hostname", "database_password", "database_port",
	"database_user",
	"heartbeat_data_collection_schema_or_database", "heartbeat_enabled",
	"id",
	"insert_static_key_field", "insert_static_key_value",
	"insert_static_value", "insert_static_value_field",
	"name", "schema_include_list",
	"signal_data_collection_schema_or_database",
	"snapshot_custom_table_config", "snapshot_large_table_threshold",
	"snapshot_parallelism",
	"ssh_enabled", "ssh_host", "ssh_port", "ssh_user",
	"table_include_list",
}

var dynamodbV219Attrs = []string{
	"array_encoding_json", "aws_access_key_id", "aws_region", "aws_secret_key",
	"batch_size", "connector", "dynamodb_service_endpoint",
	"full_export_expiration_time_ms", "id",
	"incremental_snapshot_chunk_size", "incremental_snapshot_max_threads",
	"name", "poll_timeout_ms", "s3_export_bucket_name",
	"signal_kafka_poll_timeout_ms", "struct_encoding_json",
	"table_include_list_user_defined", "tasks_max",
}

var kafkadirectV219Attrs = []string{
	"connector", "format", "id", "kafka_format", "name",
	"schemas_enable", "topic_include_list", "topic_prefix",
}
