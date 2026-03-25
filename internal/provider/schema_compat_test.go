package provider

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/destination"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/pipeline"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/tag"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/topic"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/transform"
)

// SchemaSnapshot represents a saved schema for backwards compatibility testing.
type SchemaSnapshot struct {
	Version    string                   `json:"version"`
	Attributes map[string]AttributeInfo `json:"attributes"`
}

// AttributeInfo captures the key properties of a schema attribute.
type AttributeInfo struct {
	Required  bool `json:"required"`
	Optional  bool `json:"optional"`
	Computed  bool `json:"computed"`
	Sensitive bool `json:"sensitive"`
}

// schemaCompatTestCase defines a test case for schema backwards compatibility.
type schemaCompatTestCase struct {
	name            string
	snapshotFile    string
	resourceFactory func() resource.Resource
}

// extractSchemaSnapshot extracts schema information into a snapshot structure.
func extractSchemaSnapshot(s schema.Schema) SchemaSnapshot {
	snapshot := SchemaSnapshot{
		Attributes: make(map[string]AttributeInfo),
	}

	for name, attr := range s.Attributes {
		snapshot.Attributes[name] = AttributeInfo{
			Required:  attr.IsRequired(),
			Optional:  attr.IsOptional(),
			Computed:  attr.IsComputed(),
			Sensitive: attr.IsSensitive(),
		}
	}

	return snapshot
}

// runSchemaCompatTest executes a schema backwards compatibility test.
// Breaking changes detected:
// - Removing a required attribute
// - Changing optional to required
// - Removing a computed attribute that users might reference
//
// Run UPDATE_SNAPSHOTS=1 to create new baseline after intentional changes.
func runSchemaCompatTest(t *testing.T, tc schemaCompatTestCase) {
	t.Helper()

	snapshotPath := filepath.Join("testdata", "schemas", tc.snapshotFile)

	// Get current schema
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res := tc.resourceFactory()
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), "schema should not have errors")

	currentSnapshot := extractSchemaSnapshot(schemaResp.Schema)

	// Update mode: save current schema as baseline
	if os.Getenv("UPDATE_SNAPSHOTS") != "" {
		currentSnapshot.Version = "v3.0.0" // Update version as needed
		data, err := json.MarshalIndent(currentSnapshot, "", "  ")
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Dir(snapshotPath), 0755)
		require.NoError(t, err)
		err = os.WriteFile(snapshotPath, data, 0644)
		require.NoError(t, err)
		t.Logf("Updated schema snapshot: %s", snapshotPath)
		return
	}

	// Compare mode: load baseline and check for breaking changes
	baselineData, err := os.ReadFile(snapshotPath)
	if os.IsNotExist(err) {
		t.Skipf("No baseline snapshot at %s. Run UPDATE_SNAPSHOTS=1 to create.", snapshotPath)
		return
	}
	require.NoError(t, err)

	var baseline SchemaSnapshot
	require.NoError(t, json.Unmarshal(baselineData, &baseline))

	// Track breaking changes
	breakingChanges := 0

	// Check for breaking changes
	for attrName, baseAttr := range baseline.Attributes {
		currentAttr, exists := currentSnapshot.Attributes[attrName]

		// Breaking: Required attribute removed
		if !exists && baseAttr.Required {
			t.Errorf("BREAKING CHANGE: Required attribute %q was removed", attrName)
			breakingChanges++
			continue
		}

		// Breaking: Optional changed to required
		if exists && baseAttr.Optional && !baseAttr.Required && currentAttr.Required {
			t.Errorf("BREAKING CHANGE: Attribute %q changed from optional to required", attrName)
			breakingChanges++
		}

		// Warning: Computed attribute removed (might break references)
		if !exists && baseAttr.Computed {
			t.Logf("WARNING: Computed attribute %q was removed - may break user references", attrName)
		}
	}

	// Info: New attributes (not breaking, just informational)
	newAttrs := 0
	for attrName := range currentSnapshot.Attributes {
		if _, exists := baseline.Attributes[attrName]; !exists {
			t.Logf("INFO: New attribute %q added", attrName)
			newAttrs++
		}
	}

	if breakingChanges == 0 {
		t.Logf("Schema compatibility check passed. Baseline: %d attrs, Current: %d attrs, New: %d",
			len(baseline.Attributes), len(currentSnapshot.Attributes), newAttrs)
	}
}

// TestSchemaBackwardsCompatibility_SourcePostgreSQL verifies no breaking changes to PostgreSQL source schema.
func TestSchemaBackwardsCompatibility_SourcePostgreSQL(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_postgresql",
		snapshotFile:    "source_postgresql_v1.json",
		resourceFactory: source.NewPostgreSQLResource,
	})
}

// TestSchemaBackwardsCompatibility_SourceMySQL verifies no breaking changes to MySQL source schema.
func TestSchemaBackwardsCompatibility_SourceMySQL(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_mysql",
		snapshotFile:    "source_mysql_v1.json",
		resourceFactory: source.NewMySQLResource,
	})
}

// TestSchemaBackwardsCompatibility_SourceMongoDB verifies no breaking changes to MongoDB source schema.
func TestSchemaBackwardsCompatibility_SourceMongoDB(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_mongodb",
		snapshotFile:    "source_mongodb_v1.json",
		resourceFactory: source.NewMongoDBResource,
	})
}

// TestSchemaBackwardsCompatibility_SourceDynamoDB verifies no breaking changes to DynamoDB source schema.
func TestSchemaBackwardsCompatibility_SourceDynamoDB(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_dynamodb",
		snapshotFile:    "source_dynamodb_v1.json",
		resourceFactory: source.NewDynamoDBResource,
	})
}

// TestSchemaBackwardsCompatibility_SourceSQLServer verifies no breaking changes to SQL Server source schema.
func TestSchemaBackwardsCompatibility_SourceSQLServer(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_sqlserver",
		snapshotFile:    "source_sqlserver_v1.json",
		resourceFactory: source.NewSQLServerResource,
	})
}

// TestSchemaBackwardsCompatibility_SourceKafkaDirect verifies no breaking changes to KafkaDirect source schema.
func TestSchemaBackwardsCompatibility_SourceKafkaDirect(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_kafkadirect",
		snapshotFile:    "source_kafkadirect_v1.json",
		resourceFactory: source.NewKafkaDirectResource,
	})
}

// TestSchemaBackwardsCompatibility_DestinationSnowflake verifies no breaking changes to Snowflake destination schema.
func TestSchemaBackwardsCompatibility_DestinationSnowflake(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_snowflake",
		snapshotFile:    "destination_snowflake_v1.json",
		resourceFactory: destination.NewSnowflakeResource,
	})
}

// TestSchemaBackwardsCompatibility_DestinationClickHouse verifies no breaking changes to ClickHouse destination schema.
func TestSchemaBackwardsCompatibility_DestinationClickHouse(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_clickhouse",
		snapshotFile:    "destination_clickhouse_v1.json",
		resourceFactory: destination.NewClickHouseResource,
	})
}

// TestSchemaBackwardsCompatibility_DestinationDatabricks verifies no breaking changes to Databricks destination schema.
func TestSchemaBackwardsCompatibility_DestinationDatabricks(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_databricks",
		snapshotFile:    "destination_databricks_v1.json",
		resourceFactory: destination.NewDatabricksResource,
	})
}

// TestSchemaBackwardsCompatibility_DestinationPostgreSQL verifies no breaking changes to PostgreSQL destination schema.
func TestSchemaBackwardsCompatibility_DestinationPostgreSQL(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_postgresql",
		snapshotFile:    "destination_postgresql_v1.json",
		resourceFactory: destination.NewPostgreSQLResource,
	})
}

// TestSchemaBackwardsCompatibility_DestinationS3 verifies no breaking changes to S3 destination schema.
func TestSchemaBackwardsCompatibility_DestinationS3(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_s3",
		snapshotFile:    "destination_s3_v1.json",
		resourceFactory: destination.NewS3Resource,
	})
}

// TestSchemaBackwardsCompatibility_DestinationIceberg verifies no breaking changes to Iceberg destination schema.
func TestSchemaBackwardsCompatibility_DestinationIceberg(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_iceberg",
		snapshotFile:    "destination_iceberg_v1.json",
		resourceFactory: destination.NewIcebergResource,
	})
}

// TestSchemaBackwardsCompatibility_DestinationKafka verifies no breaking changes to Kafka destination schema.
func TestSchemaBackwardsCompatibility_DestinationKafka(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_kafka",
		snapshotFile:    "destination_kafka_v1.json",
		resourceFactory: destination.NewKafkaResource,
	})
}

// TestSchemaBackwardsCompatibility_TransformMapFilter verifies no breaking changes to MapFilter transform schema.
func TestSchemaBackwardsCompatibility_TransformMapFilter(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "transform_map_filter",
		snapshotFile:    "transform_map_filter_v1.json",
		resourceFactory: transform.NewMapFilterResource,
	})
}

// TestSchemaBackwardsCompatibility_TransformEnrich verifies no breaking changes to Enrich transform schema.
func TestSchemaBackwardsCompatibility_TransformEnrich(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "transform_enrich",
		snapshotFile:    "transform_enrich_v1.json",
		resourceFactory: transform.NewEnrichResource,
	})
}

// TestSchemaBackwardsCompatibility_TransformSqlJoin verifies no breaking changes to SQL Join transform schema.
func TestSchemaBackwardsCompatibility_TransformSqlJoin(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "transform_sql_join",
		snapshotFile:    "transform_sql_join_v1.json",
		resourceFactory: transform.NewSqlJoinResource,
	})
}

// --- Sources (missing) ---

func TestSchemaBackwardsCompatibility_SourceAlloyDB(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_alloydb",
		snapshotFile:    "source_alloydb_v1.json",
		resourceFactory: source.NewAlloyDBResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceDB2(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_db2",
		snapshotFile:    "source_db2_v1.json",
		resourceFactory: source.NewDB2Resource,
	})
}

func TestSchemaBackwardsCompatibility_SourceDocumentDB(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_documentdb",
		snapshotFile:    "source_documentdb_v1.json",
		resourceFactory: source.NewDocumentDBResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceElasticsearch(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_elasticsearch",
		snapshotFile:    "source_elasticsearch_v1.json",
		resourceFactory: source.NewElasticsearchResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceMariaDB(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_mariadb",
		snapshotFile:    "source_mariadb_v1.json",
		resourceFactory: source.NewMariaDBResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceMongoDBHosted(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_mongodbhosted",
		snapshotFile:    "source_mongodbhosted_v1.json",
		resourceFactory: source.NewMongoDBHostedResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceOracle(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_oracle",
		snapshotFile:    "source_oracle_v1.json",
		resourceFactory: source.NewOracleResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceOracleAWS(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_oracleaws",
		snapshotFile:    "source_oracleaws_v1.json",
		resourceFactory: source.NewOracleAWSResource,
	})
}

func TestSchemaBackwardsCompatibility_SourcePlanetScale(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_planetscale",
		snapshotFile:    "source_planetscale_v1.json",
		resourceFactory: source.NewPlanetScaleResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceRedis(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_redis",
		snapshotFile:    "source_redis_v1.json",
		resourceFactory: source.NewRedisResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceS3(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_s3",
		snapshotFile:    "source_s3_v1.json",
		resourceFactory: source.NewS3SourceResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceSupabase(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_supabase",
		snapshotFile:    "source_supabase_v1.json",
		resourceFactory: source.NewSupabaseResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceVitess(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_vitess",
		snapshotFile:    "source_vitess_v1.json",
		resourceFactory: source.NewVitessResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceWebhook(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_webhook",
		snapshotFile:    "source_webhook_v1.json",
		resourceFactory: source.NewWebhookResource,
	})
}

// --- Destinations (missing) ---

func TestSchemaBackwardsCompatibility_DestinationAzBlob(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_azblob",
		snapshotFile:    "destination_azblob_v1.json",
		resourceFactory: destination.NewAzBlobResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationBigQuery(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_bigquery",
		snapshotFile:    "destination_bigquery_v1.json",
		resourceFactory: destination.NewBigQueryResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationCockroachDB(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_cockroachdb",
		snapshotFile:    "destination_cockroachdb_v1.json",
		resourceFactory: destination.NewCockroachDBResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationDB2(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_db2",
		snapshotFile:    "destination_db2_v1.json",
		resourceFactory: destination.NewDB2DestResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationGCS(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_gcs",
		snapshotFile:    "destination_gcs_v1.json",
		resourceFactory: destination.NewGCSResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationHTTPSink(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_httpsink",
		snapshotFile:    "destination_httpsink_v1.json",
		resourceFactory: destination.NewHTTPSinkResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationKafkaDirect(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_kafkadirect",
		snapshotFile:    "destination_kafkadirect_v1.json",
		resourceFactory: destination.NewKafkaDirectDestResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationMotherduck(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_motherduck",
		snapshotFile:    "destination_motherduck_v1.json",
		resourceFactory: destination.NewMotherduckResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationMySQL(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_mysql",
		snapshotFile:    "destination_mysql_v1.json",
		resourceFactory: destination.NewMySQLDestResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationOracle(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_oracle",
		snapshotFile:    "destination_oracle_v1.json",
		resourceFactory: destination.NewOracleDestResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationR2(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_r2",
		snapshotFile:    "destination_r2_v1.json",
		resourceFactory: destination.NewR2Resource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationRedis(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_redis",
		snapshotFile:    "destination_redis_v1.json",
		resourceFactory: destination.NewRedisDestResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationRedshift(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_redshift",
		snapshotFile:    "destination_redshift_v1.json",
		resourceFactory: destination.NewRedshiftResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationSQLServer(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_sqlserver",
		snapshotFile:    "destination_sqlserver_v1.json",
		resourceFactory: destination.NewSQLServerDestResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationStarburst(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_starburst",
		snapshotFile:    "destination_starburst_v1.json",
		resourceFactory: destination.NewStarburstResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationWeaviate(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_weaviate",
		snapshotFile:    "destination_weaviate_v1.json",
		resourceFactory: destination.NewWeaviateResource,
	})
}

func TestSchemaBackwardsCompatibility_DestinationPinecone(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "destination_pinecone",
		snapshotFile:    "destination_pinecone_v1.json",
		resourceFactory: destination.NewPineconeDestResource,
	})
}

// --- Transforms (missing) ---

func TestSchemaBackwardsCompatibility_TransformEnrichAsync(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "transform_enrich_async",
		snapshotFile:    "transform_enrich_async_v1.json",
		resourceFactory: transform.NewEnrichAsyncResource,
	})
}

func TestSchemaBackwardsCompatibility_TransformRollup(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "transform_rollup",
		snapshotFile:    "transform_rollup_v1.json",
		resourceFactory: transform.NewRollupResource,
	})
}

func TestSchemaBackwardsCompatibility_TransformFanOut(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "transform_fan_out",
		snapshotFile:    "transform_fan_out_v1.json",
		resourceFactory: transform.NewFanOutResource,
	})
}

// --- Non-connector resources ---

func TestSchemaBackwardsCompatibility_Pipeline(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "pipeline",
		snapshotFile:    "pipeline_v1.json",
		resourceFactory: pipeline.NewPipelineResource,
	})
}

func TestSchemaBackwardsCompatibility_Topic(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "topic",
		snapshotFile:    "topic_v1.json",
		resourceFactory: topic.NewTopicResource,
	})
}

func TestSchemaBackwardsCompatibility_Tag(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "tag",
		snapshotFile:    "tag_v1.json",
		resourceFactory: tag.NewTagResource,
	})
}

