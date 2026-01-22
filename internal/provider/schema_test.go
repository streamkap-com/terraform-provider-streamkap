package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/destination"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/pipeline"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/tag"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/topic"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/transform"
)

// Schema tests verify resource schemas have correct attribute properties.
// These run without API calls and catch schema definition bugs early.

// schemaTestCase defines a test case for schema validation.
type schemaTestCase struct {
	name                string
	resourceFactory     func() resource.Resource
	requiredAttrs       []string
	computedAttrs       []string
	sensitiveAttrs      []string
	optionalWithDefault []string // Attributes that are optional but have defaults (so also computed)
}

// getSchema retrieves the schema from a resource.
func getSchema(t *testing.T, res resource.Resource) schema.Schema {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), "schema should not have errors")
	return schemaResp.Schema
}

// verifyRequiredAttributes checks that required attributes exist and are marked required.
func verifyRequiredAttributes(t *testing.T, s schema.Schema, attrs []string, resourceName string) {
	t.Helper()
	for _, attr := range attrs {
		attrSchema, ok := s.Attributes[attr]
		assert.True(t, ok, "%s: attribute %q must exist", resourceName, attr)
		if ok {
			assert.True(t, attrSchema.IsRequired(), "%s: attribute %q must be required", resourceName, attr)
		}
	}
}

// verifyComputedAttributes checks that computed attributes exist and are marked computed.
func verifyComputedAttributes(t *testing.T, s schema.Schema, attrs []string, resourceName string) {
	t.Helper()
	for _, attr := range attrs {
		attrSchema, ok := s.Attributes[attr]
		assert.True(t, ok, "%s: attribute %q must exist", resourceName, attr)
		if ok {
			assert.True(t, attrSchema.IsComputed(), "%s: attribute %q must be computed", resourceName, attr)
		}
	}
}

// verifySensitiveAttributes checks that sensitive attributes exist and are marked sensitive.
func verifySensitiveAttributes(t *testing.T, s schema.Schema, attrs []string, resourceName string) {
	t.Helper()
	for _, attr := range attrs {
		attrSchema, ok := s.Attributes[attr]
		assert.True(t, ok, "%s: attribute %q must exist", resourceName, attr)
		if ok {
			assert.True(t, attrSchema.IsSensitive(), "%s: attribute %q must be sensitive", resourceName, attr)
		}
	}
}

// verifyOptionalWithDefault checks attributes that are optional but have defaults (so also computed).
func verifyOptionalWithDefault(t *testing.T, s schema.Schema, attrs []string, resourceName string) {
	t.Helper()
	for _, attr := range attrs {
		attrSchema, ok := s.Attributes[attr]
		assert.True(t, ok, "%s: attribute %q must exist", resourceName, attr)
		if ok {
			assert.True(t, attrSchema.IsOptional(), "%s: attribute %q must be optional", resourceName, attr)
			assert.True(t, attrSchema.IsComputed(), "%s: attribute %q must be computed (has default)", resourceName, attr)
		}
	}
}

// runSchemaTest executes schema validation for a given test case.
func runSchemaTest(t *testing.T, tc schemaTestCase) {
	t.Run(tc.name, func(t *testing.T) {
		res := tc.resourceFactory()
		s := getSchema(t, res)

		if len(tc.requiredAttrs) > 0 {
			verifyRequiredAttributes(t, s, tc.requiredAttrs, tc.name)
		}
		if len(tc.computedAttrs) > 0 {
			verifyComputedAttributes(t, s, tc.computedAttrs, tc.name)
		}
		if len(tc.sensitiveAttrs) > 0 {
			verifySensitiveAttributes(t, s, tc.sensitiveAttrs, tc.name)
		}
		if len(tc.optionalWithDefault) > 0 {
			verifyOptionalWithDefault(t, s, tc.optionalWithDefault, tc.name)
		}
	})
}

// TestSourceSchemas validates schema definitions for source connectors.
func TestSourceSchemas(t *testing.T) {
	testCases := []schemaTestCase{
		{
			name:            "source_postgresql",
			resourceFactory: source.NewPostgreSQLResource,
			requiredAttrs: []string{
				"name",
				"database_hostname",
				"database_user",
				"database_password",
				"database_dbname",
				// signal_data_collection_schema_or_database is conditionally required (when snapshot_read_only="No")
				// heartbeat_data_collection_schema_or_database is conditionally required (when heartbeat_enabled=true and snapshot_read_only="No")
				"schema_include_list",
				"table_include_list",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
			sensitiveAttrs: []string{
				"database_password",
			},
			optionalWithDefault: []string{
				"database_port",
				"database_sslmode",
				"slot_name",
				"publication_name",
			},
		},
		{
			name:            "source_mysql",
			resourceFactory: source.NewMySQLResource,
			requiredAttrs: []string{
				"name",
				"database_hostname",
				"database_user",
				"database_password",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
			sensitiveAttrs: []string{
				"database_password",
			},
		},
		{
			name:            "source_mongodb",
			resourceFactory: source.NewMongoDBResource,
			requiredAttrs: []string{
				"name",
				"mongodb_connection_string",
				"database_include_list",
				"collection_include_list",
				"signal_data_collection_schema_or_database",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
			sensitiveAttrs: []string{
				"mongodb_connection_string",
			},
		},
		{
			name:            "source_dynamodb",
			resourceFactory: source.NewDynamoDBResource,
			requiredAttrs: []string{
				"name",
				"aws_region",
				"aws_access_key_id",
				"aws_secret_key",
				"s3_export_bucket_name",
				"table_include_list",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
			sensitiveAttrs: []string{
				"aws_access_key_id",
				"aws_secret_key",
			},
		},
		{
			name:            "source_sqlserver",
			resourceFactory: source.NewSQLServerResource,
			requiredAttrs: []string{
				"name",
				"database_hostname",
				"database_user",
				"database_password",
				"database_names",
				"schema_include_list",
				"table_include_list",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
			sensitiveAttrs: []string{
				"database_password",
			},
		},
		{
			name:            "source_kafkadirect",
			resourceFactory: source.NewKafkaDirectResource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
		},
	}

	for _, tc := range testCases {
		runSchemaTest(t, tc)
	}
}

// TestDestinationSchemas validates schema definitions for destination connectors.
func TestDestinationSchemas(t *testing.T) {
	testCases := []schemaTestCase{
		{
			name:            "destination_snowflake",
			resourceFactory: destination.NewSnowflakeResource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
			sensitiveAttrs: []string{
				"snowflake_private_key",
				"snowflake_private_key_passphrase",
			},
			optionalWithDefault: []string{
				"sfwarehouse",
				"snowflake_role_name",
				"ingestion_mode",
			},
		},
		{
			name:            "destination_clickhouse",
			resourceFactory: destination.NewClickHouseResource,
			requiredAttrs: []string{
				"name",
				"hostname",
				"connection_username",
				"connection_password",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
			sensitiveAttrs: []string{
				"connection_password",
			},
			optionalWithDefault: []string{
				"ingestion_mode",
				"port",
			},
		},
		{
			name:            "destination_databricks",
			resourceFactory: destination.NewDatabricksResource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
			sensitiveAttrs: []string{
				"databricks_token",
			},
			optionalWithDefault: []string{
				"ingestion_mode",
				"databricks_catalog",
			},
		},
		{
			name:            "destination_postgresql",
			resourceFactory: destination.NewPostgreSQLResource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
			sensitiveAttrs: []string{
				"connection_password",
			},
		},
		{
			name:            "destination_s3",
			resourceFactory: destination.NewS3Resource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
			sensitiveAttrs: []string{
				"aws_secret_access_key",
			},
			optionalWithDefault: []string{
				"aws_s3_region",
				"format",
			},
		},
		{
			name:            "destination_iceberg",
			resourceFactory: destination.NewIcebergResource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
		},
		{
			name:            "destination_kafka",
			resourceFactory: destination.NewKafkaResource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"connector",
			},
		},
	}

	for _, tc := range testCases {
		runSchemaTest(t, tc)
	}
}

// TestTransformSchemas validates schema definitions for transform connectors.
func TestTransformSchemas(t *testing.T) {
	testCases := []schemaTestCase{
		{
			name:            "transform_map_filter",
			resourceFactory: transform.NewMapFilterResource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"transform_type",
			},
			optionalWithDefault: []string{
				"transforms_language",
				"transforms_input_topic_pattern",
				"transforms_output_topic_pattern",
			},
		},
		{
			name:            "transform_enrich",
			resourceFactory: transform.NewEnrichResource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"transform_type",
			},
		},
		{
			name:            "transform_enrich_async",
			resourceFactory: transform.NewEnrichAsyncResource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"transform_type",
			},
		},
		{
			name:            "transform_sql_join",
			resourceFactory: transform.NewSqlJoinResource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"transform_type",
			},
		},
		{
			name:            "transform_rollup",
			resourceFactory: transform.NewRollupResource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"transform_type",
			},
		},
		{
			name:            "transform_fan_out",
			resourceFactory: transform.NewFanOutResource,
			requiredAttrs: []string{
				"name",
			},
			computedAttrs: []string{
				"id",
				"transform_type",
			},
		},
	}

	for _, tc := range testCases {
		runSchemaTest(t, tc)
	}
}

// TestPipelineSchema validates the pipeline resource schema.
func TestPipelineSchema(t *testing.T) {
	res := pipeline.NewPipelineResource()
	s := getSchema(t, res)

	// Required attributes
	verifyRequiredAttributes(t, s, []string{
		"name",
		"source",
		"destination",
	}, "pipeline")

	// Computed attributes
	verifyComputedAttributes(t, s, []string{
		"id",
	}, "pipeline")

	// Optional with default
	verifyOptionalWithDefault(t, s, []string{
		"snapshot_new_tables",
	}, "pipeline")
}

// TestTopicSchema validates the topic resource schema.
func TestTopicSchema(t *testing.T) {
	res := topic.NewTopicResource()
	s := getSchema(t, res)

	// Required attributes
	verifyRequiredAttributes(t, s, []string{
		"topic_id",
		"partition_count",
	}, "topic")
}

// TestTagSchema validates the tag resource schema.
func TestTagSchema(t *testing.T) {
	res := tag.NewTagResource()
	s := getSchema(t, res)

	// Required attributes
	verifyRequiredAttributes(t, s, []string{
		"name",
	}, "tag")

	// Computed attributes
	verifyComputedAttributes(t, s, []string{
		"id",
	}, "tag")
}

// TestAllSourceSchemasHaveCommonFields verifies all source resources have standard fields.
func TestAllSourceSchemasHaveCommonFields(t *testing.T) {
	sourceFactories := map[string]func() resource.Resource{
		"postgresql":     source.NewPostgreSQLResource,
		"mysql":          source.NewMySQLResource,
		"mongodb":        source.NewMongoDBResource,
		"dynamodb":       source.NewDynamoDBResource,
		"sqlserver":      source.NewSQLServerResource,
		"kafkadirect":    source.NewKafkaDirectResource,
		"alloydb":        source.NewAlloyDBResource,
		"db2":            source.NewDB2Resource,
		"documentdb":     source.NewDocumentDBResource,
		"elasticsearch":  source.NewElasticsearchResource,
		"mariadb":        source.NewMariaDBResource,
		"mongodb_hosted": source.NewMongoDBHostedResource,
		"oracle":         source.NewOracleResource,
		"oracle_aws":     source.NewOracleAWSResource,
		"planetscale":    source.NewPlanetScaleResource,
		"redis":          source.NewRedisResource,
		"s3":             source.NewS3SourceResource,
		"supabase":       source.NewSupabaseResource,
		"vitess":         source.NewVitessResource,
		"webhook":        source.NewWebhookResource,
	}

	for name, factory := range sourceFactories {
		t.Run(name, func(t *testing.T) {
			res := factory()
			s := getSchema(t, res)

			// All sources must have these common fields
			commonComputedAttrs := []string{"id", "connector"}
			commonRequiredAttrs := []string{"name"}

			verifyRequiredAttributes(t, s, commonRequiredAttrs, "source_"+name)
			verifyComputedAttributes(t, s, commonComputedAttrs, "source_"+name)
		})
	}
}

// TestAllDestinationSchemasHaveCommonFields verifies all destination resources have standard fields.
func TestAllDestinationSchemasHaveCommonFields(t *testing.T) {
	destFactories := map[string]func() resource.Resource{
		"snowflake":   destination.NewSnowflakeResource,
		"clickhouse":  destination.NewClickHouseResource,
		"databricks":  destination.NewDatabricksResource,
		"postgresql":  destination.NewPostgreSQLResource,
		"s3":          destination.NewS3Resource,
		"iceberg":     destination.NewIcebergResource,
		"kafka":       destination.NewKafkaResource,
		"azblob":      destination.NewAzBlobResource,
		"bigquery":    destination.NewBigQueryResource,
		"cockroachdb": destination.NewCockroachDBResource,
		"db2":         destination.NewDB2DestResource,
		"gcs":         destination.NewGCSResource,
		"httpsink":    destination.NewHTTPSinkResource,
		"kafkadirect": destination.NewKafkaDirectDestResource,
		"motherduck":  destination.NewMotherduckResource,
		"mysql":       destination.NewMySQLDestResource,
		"oracle":      destination.NewOracleDestResource,
		"r2":          destination.NewR2Resource,
		"redis":       destination.NewRedisDestResource,
		"redshift":    destination.NewRedshiftResource,
		"sqlserver":   destination.NewSQLServerDestResource,
		"starburst":   destination.NewStarburstResource,
	}

	for name, factory := range destFactories {
		t.Run(name, func(t *testing.T) {
			res := factory()
			s := getSchema(t, res)

			// All destinations must have these common fields
			commonComputedAttrs := []string{"id", "connector"}
			commonRequiredAttrs := []string{"name"}

			verifyRequiredAttributes(t, s, commonRequiredAttrs, "destination_"+name)
			verifyComputedAttributes(t, s, commonComputedAttrs, "destination_"+name)
		})
	}
}

// TestAllTransformSchemasHaveCommonFields verifies all transform resources have standard fields.
func TestAllTransformSchemasHaveCommonFields(t *testing.T) {
	transformFactories := map[string]func() resource.Resource{
		"map_filter":   transform.NewMapFilterResource,
		"enrich":       transform.NewEnrichResource,
		"enrich_async": transform.NewEnrichAsyncResource,
		"sql_join":     transform.NewSqlJoinResource,
		"rollup":       transform.NewRollupResource,
		"fan_out":      transform.NewFanOutResource,
	}

	for name, factory := range transformFactories {
		t.Run(name, func(t *testing.T) {
			res := factory()
			s := getSchema(t, res)

			// All transforms must have these common fields
			commonComputedAttrs := []string{"id", "transform_type"}
			commonRequiredAttrs := []string{"name"}

			verifyRequiredAttributes(t, s, commonRequiredAttrs, "transform_"+name)
			verifyComputedAttributes(t, s, commonComputedAttrs, "transform_"+name)
		})
	}
}

// TestSchemaHasDescriptions verifies that all attributes have descriptions for documentation.
func TestSchemaHasDescriptions(t *testing.T) {
	// Test a few key resources to ensure they have descriptions
	resources := map[string]resource.Resource{
		"source_postgresql":     source.NewPostgreSQLResource(),
		"destination_snowflake": destination.NewSnowflakeResource(),
		"transform_map_filter":  transform.NewMapFilterResource(),
		"pipeline":              pipeline.NewPipelineResource(),
	}

	for name, res := range resources {
		t.Run(name, func(t *testing.T) {
			s := getSchema(t, res)

			// Schema itself should have a description
			assert.NotEmpty(t, s.Description, "%s: schema should have Description", name)
			assert.NotEmpty(t, s.MarkdownDescription, "%s: schema should have MarkdownDescription", name)

			// Check a few key attributes have descriptions
			for attrName, attr := range s.Attributes {
				// Skip checking all attributes, just ensure the pattern is followed
				if attrName == "id" || attrName == "name" {
					assert.NotEmpty(t, attr.GetDescription(), "%s.%s: attribute should have Description", name, attrName)
					assert.NotEmpty(t, attr.GetMarkdownDescription(), "%s.%s: attribute should have MarkdownDescription", name, attrName)
				}
			}
		})
	}
}
