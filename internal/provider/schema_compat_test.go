package provider

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/require"

	ds "github.com/streamkap-com/terraform-provider-streamkap/internal/datasource"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/client_credential"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/destination"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/kafka_user"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/pipeline"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/tag"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/topic"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/transform"
)

// SchemaSnapshot represents a saved schema for backwards compatibility testing.
type SchemaSnapshot struct {
	Version          string                   `json:"version"`
	Attributes       map[string]AttributeInfo `json:"attributes"`
	NestedAttributes map[string]AttributeInfo `json:"nested_attributes,omitempty"`
	Blocks           map[string]BlockInfo     `json:"blocks,omitempty"`
	BlockAttributes  map[string]AttributeInfo `json:"block_attributes,omitempty"`
}

// BlockInfo records a block's container shape. Attributes inside the block are
// flattened into BlockAttributes so their Required/Optional/Computed/Sensitive
// flags are checked with the same rules as top-level attributes.
type BlockInfo struct {
	NestingMode string `json:"nesting_mode"`
	Type        string `json:"type"`
}

// AttributeInfo captures the key properties of a schema attribute.
//
// Type is the framework type name (e.g. "basetypes.StringType"). Without it a
// String→Int64 flip — a hard breaking change, and one CLAUDE.md lists as *not*
// aliasable — passes every snapshot check silently.
type AttributeInfo struct {
	Required  bool   `json:"required"`
	Optional  bool   `json:"optional"`
	Computed  bool   `json:"computed"`
	Sensitive bool   `json:"sensitive"`
	Type      string `json:"type"`
}

// snapshotAttribute is the subset of the framework's attribute interface the
// snapshot needs. Resource and data-source attributes both satisfy it, which is
// what lets one extractor serve both schema flavours.
type snapshotAttribute interface {
	IsRequired() bool
	IsOptional() bool
	IsComputed() bool
	IsSensitive() bool
	GetType() attr.Type
}

func attributeInfo(a snapshotAttribute) AttributeInfo {
	return AttributeInfo{
		Required:  a.IsRequired(),
		Optional:  a.IsOptional(),
		Computed:  a.IsComputed(),
		Sensitive: a.IsSensitive(),
		Type:      a.GetType().String(),
	}
}

// schemaCompatTestCase defines a test case for schema backwards compatibility.
type schemaCompatTestCase struct {
	name            string
	snapshotFile    string
	resourceFactory func() resource.Resource
}

// dataSourceCompatTestCase defines a test case for data-source schema
// backwards compatibility.
type dataSourceCompatTestCase struct {
	name              string
	snapshotFile      string
	dataSourceFactory func() datasource.DataSource
}

// extractSchemaSnapshot extracts schema information into a snapshot structure.
func extractSchemaSnapshot(s schema.Schema) SchemaSnapshot {
	snapshot := SchemaSnapshot{
		Attributes:       make(map[string]AttributeInfo),
		NestedAttributes: make(map[string]AttributeInfo),
		Blocks:           make(map[string]BlockInfo),
		BlockAttributes:  make(map[string]AttributeInfo),
	}

	for name, attribute := range s.Attributes {
		snapshot.Attributes[name] = attributeInfo(attribute)
		addResourceNestedAttributes(snapshot.NestedAttributes, name, attribute)
	}
	for name, block := range s.Blocks {
		addResourceBlock(&snapshot, name, block)
	}

	return snapshot
}

// extractDataSourceSchemaSnapshot is extractSchemaSnapshot for data sources.
func extractDataSourceSchemaSnapshot(s dsschema.Schema) SchemaSnapshot {
	snapshot := SchemaSnapshot{
		Attributes:       make(map[string]AttributeInfo),
		NestedAttributes: make(map[string]AttributeInfo),
		Blocks:           make(map[string]BlockInfo),
		BlockAttributes:  make(map[string]AttributeInfo),
	}

	for name, attribute := range s.Attributes {
		snapshot.Attributes[name] = attributeInfo(attribute)
		addDataSourceNestedAttributes(snapshot.NestedAttributes, name, attribute)
	}
	for name, block := range s.Blocks {
		addDataSourceBlock(&snapshot, name, block)
	}

	return snapshot
}

func TestExtractSchemaSnapshotIncludesNestedShapes(t *testing.T) {
	ctx := context.Background()

	kafkaResponse := &resource.SchemaResponse{}
	kafka_user.NewKafkaUserResource().Schema(ctx, resource.SchemaRequest{}, kafkaResponse)
	require.False(t, kafkaResponse.Diagnostics.HasError())
	kafkaSnapshot := extractSchemaSnapshot(kafkaResponse.Schema)
	require.Equal(t, "list", kafkaSnapshot.Blocks["kafka_acls"].NestingMode)
	require.True(t, kafkaSnapshot.BlockAttributes["kafka_acls.topic_name"].Required)
	require.True(t, kafkaSnapshot.BlockAttributes["kafka_acls.resource"].Optional)

	pipelineResponse := &resource.SchemaResponse{}
	pipeline.NewPipelineResource().Schema(ctx, resource.SchemaRequest{}, pipelineResponse)
	require.False(t, pipelineResponse.Diagnostics.HasError())
	pipelineSnapshot := extractSchemaSnapshot(pipelineResponse.Schema)
	require.True(t, pipelineSnapshot.NestedAttributes["source.id"].Required)
	require.Equal(t, "single", pipelineSnapshot.Blocks["timeouts"].NestingMode)
	require.True(t, pipelineSnapshot.BlockAttributes["timeouts.create"].Optional)
}

func addResourceNestedAttributes(target map[string]AttributeInfo, prefix string, attribute schema.Attribute) {
	var attributes map[string]schema.Attribute
	switch a := attribute.(type) {
	case schema.ListNestedAttribute:
		attributes = a.NestedObject.Attributes
	case schema.SetNestedAttribute:
		attributes = a.NestedObject.Attributes
	case schema.MapNestedAttribute:
		attributes = a.NestedObject.Attributes
	case schema.SingleNestedAttribute:
		attributes = a.Attributes
	default:
		return
	}
	for name, nested := range attributes {
		path := prefix + "." + name
		target[path] = attributeInfo(nested)
		addResourceNestedAttributes(target, path, nested)
	}
}

func addDataSourceNestedAttributes(target map[string]AttributeInfo, prefix string, attribute dsschema.Attribute) {
	var attributes map[string]dsschema.Attribute
	switch a := attribute.(type) {
	case dsschema.ListNestedAttribute:
		attributes = a.NestedObject.Attributes
	case dsschema.SetNestedAttribute:
		attributes = a.NestedObject.Attributes
	case dsschema.MapNestedAttribute:
		attributes = a.NestedObject.Attributes
	case dsschema.SingleNestedAttribute:
		attributes = a.Attributes
	default:
		return
	}
	for name, nested := range attributes {
		path := prefix + "." + name
		target[path] = attributeInfo(nested)
		addDataSourceNestedAttributes(target, path, nested)
	}
}

func addResourceBlock(snapshot *SchemaSnapshot, path string, block schema.Block) {
	var attributes map[string]schema.Attribute
	var blocks map[string]schema.Block
	mode := ""
	switch b := block.(type) {
	case schema.ListNestedBlock:
		mode, attributes, blocks = "list", b.NestedObject.Attributes, b.NestedObject.Blocks
	case schema.SetNestedBlock:
		mode, attributes, blocks = "set", b.NestedObject.Attributes, b.NestedObject.Blocks
	case schema.SingleNestedBlock:
		mode, attributes, blocks = "single", b.Attributes, b.Blocks
	default:
		mode = "unknown"
	}
	snapshot.Blocks[path] = BlockInfo{NestingMode: mode, Type: block.Type().String()}
	for name, attribute := range attributes {
		attributePath := path + "." + name
		snapshot.BlockAttributes[attributePath] = attributeInfo(attribute)
		addResourceNestedAttributes(snapshot.BlockAttributes, attributePath, attribute)
	}
	for name, nested := range blocks {
		addResourceBlock(snapshot, path+"."+name, nested)
	}
}

func addDataSourceBlock(snapshot *SchemaSnapshot, path string, block dsschema.Block) {
	var attributes map[string]dsschema.Attribute
	var blocks map[string]dsschema.Block
	mode := ""
	switch b := block.(type) {
	case dsschema.ListNestedBlock:
		mode, attributes, blocks = "list", b.NestedObject.Attributes, b.NestedObject.Blocks
	case dsschema.SetNestedBlock:
		mode, attributes, blocks = "set", b.NestedObject.Attributes, b.NestedObject.Blocks
	case dsschema.SingleNestedBlock:
		mode, attributes, blocks = "single", b.Attributes, b.Blocks
	default:
		mode = "unknown"
	}
	snapshot.Blocks[path] = BlockInfo{NestingMode: mode, Type: block.Type().String()}
	for name, attribute := range attributes {
		attributePath := path + "." + name
		snapshot.BlockAttributes[attributePath] = attributeInfo(attribute)
		addDataSourceNestedAttributes(snapshot.BlockAttributes, attributePath, attribute)
	}
	for name, nested := range blocks {
		addDataSourceBlock(snapshot, path+"."+name, nested)
	}
}

// runSchemaCompatTest executes a schema backwards compatibility test.
// Breaking changes detected:
// - Removing a required attribute
// - Changing optional to required
// - Changing an attribute's type
// - Removing a computed attribute that users might reference
//
// Run UPDATE_SNAPSHOTS=1 to create new baseline after intentional changes.
func runSchemaCompatTest(t *testing.T, tc schemaCompatTestCase) {
	t.Helper()

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res := tc.resourceFactory()
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), "schema should not have errors")

	compareAgainstSnapshot(t, tc.snapshotFile, extractSchemaSnapshot(schemaResp.Schema))
}

// runDataSourceSchemaCompatTest is runSchemaCompatTest for data sources.
func runDataSourceSchemaCompatTest(t *testing.T, tc dataSourceCompatTestCase) {
	t.Helper()

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds := tc.dataSourceFactory()
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), "schema should not have errors")

	compareAgainstSnapshot(t, tc.snapshotFile, extractDataSourceSchemaSnapshot(schemaResp.Schema))
}

// compareAgainstSnapshot writes or checks a snapshot for either schema flavour.
func compareAgainstSnapshot(t *testing.T, snapshotFile string, currentSnapshot SchemaSnapshot) {
	t.Helper()

	snapshotPath := filepath.Join("testdata", "schemas", snapshotFile)

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

	breakingChanges := checkAttributeCompatibility(t, "attribute", baseline.Attributes, currentSnapshot.Attributes)
	breakingChanges += checkAttributeCompatibility(t, "nested attribute", baseline.NestedAttributes, currentSnapshot.NestedAttributes)
	breakingChanges += checkAttributeCompatibility(t, "block attribute", baseline.BlockAttributes, currentSnapshot.BlockAttributes)

	// Drift: the snapshot is the schema of record that humans and tooling read
	// from the repo. Additive changes are not breaking, but treating them as
	// merely informational lets a baseline rot silently — `tags` went missing
	// from 48 snapshots for two months that way. Any divergence fails here;
	// `make snapshots` accepts it.
	var added, removed, changed []string
	collectDrift("attribute", baseline.Attributes, currentSnapshot.Attributes, &added, &removed, &changed)
	collectDrift("nested_attribute", baseline.NestedAttributes, currentSnapshot.NestedAttributes, &added, &removed, &changed)
	collectDrift("block", baseline.Blocks, currentSnapshot.Blocks, &added, &removed, &changed)
	collectDrift("block_attribute", baseline.BlockAttributes, currentSnapshot.BlockAttributes, &added, &removed, &changed)

	if len(added)+len(removed)+len(changed) > 0 {
		sort.Strings(added)
		sort.Strings(removed)
		sort.Strings(changed)
		t.Errorf("schema snapshot %s is stale: added=%v removed=%v changed=%v\n"+
			"Run `make snapshots` and review the diff before committing.",
			snapshotFile, added, removed, changed)
		return
	}

	if breakingChanges == 0 {
		t.Logf("Schema compatibility check passed. %d attrs, snapshot in sync.",
			len(currentSnapshot.Attributes))
	}
}

func checkAttributeCompatibility(t *testing.T, kind string, baseline, current map[string]AttributeInfo) int {
	t.Helper()
	breakingChanges := 0
	for name, before := range baseline {
		after, exists := current[name]
		if !exists && before.Required {
			t.Errorf("BREAKING CHANGE: Required %s %q was removed", kind, name)
			breakingChanges++
			continue
		}
		if exists && before.Optional && !before.Required && after.Required {
			t.Errorf("BREAKING CHANGE: %s %q changed from optional to required", kind, name)
			breakingChanges++
		}
		// A credential losing Sensitive is caught by the generic drift check too,
		// but that reports it as a stale snapshot whose stated remedy is
		// `make snapshots` — which would silently bless the downgrade. Call it
		// out separately so it cannot be resolved by regenerating.
		if exists && before.Sensitive && !after.Sensitive {
			t.Errorf("BREAKING CHANGE (SECURITY): %s %q is no longer Sensitive; its value would now appear in plan output and state as plaintext. "+
				"Do NOT resolve this with `make snapshots` — restore the flag at its source (tfgen isSecretField, or the backend field's encrypt/control).", kind, name)
			breakingChanges++
		}
		if exists && before.Type != "" && before.Type != after.Type {
			t.Errorf("BREAKING CHANGE: %s %q changed type from %s to %s", kind, name, before.Type, after.Type)
			breakingChanges++
		}
		if !exists && before.Computed {
			t.Logf("WARNING: Computed %s %q was removed - may break user references", kind, name)
		}
	}
	return breakingChanges
}

func collectDrift[T comparable](kind string, baseline, current map[string]T, added, removed, changed *[]string) {
	for name, after := range current {
		before, exists := baseline[name]
		switch {
		case !exists:
			*added = append(*added, kind+"."+name)
		case before != after:
			*changed = append(*changed, kind+"."+name)
		}
	}
	for name := range baseline {
		if _, exists := current[name]; !exists {
			*removed = append(*removed, kind+"."+name)
		}
	}
}

// TestEveryResourceHasSchemaSnapshot fails when a registered resource or data
// source has no snapshot baseline.
//
// runSchemaCompatTest skips silently when the snapshot file is absent, so a new
// resource — or one whose test case was never written — gets zero drift
// protection while the suite stays green. Four webhook sources went unprotected
// that way, which is how an unmarked `api_key` shipped.
//
// Data-source snapshots carry a `datasource_` prefix: `streamkap_topic` and
// `streamkap_tag` exist as both a resource and a data source, so the bare
// TypeName is not a unique file name.
func TestEveryResourceHasSchemaSnapshot(t *testing.T) {
	ctx := context.Background()
	p := &streamkapProvider{}

	for _, factory := range p.Resources(ctx) {
		metaResp := &resource.MetadataResponse{}
		factory().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "streamkap"}, metaResp)

		name := strings.TrimPrefix(metaResp.TypeName, "streamkap_")
		snapshotPath := filepath.Join("testdata", "schemas", name+"_v1.json")

		if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
			t.Errorf("resource %q has no schema snapshot at %s; add a "+
				"TestSchemaBackwardsCompatibility_* case and run `make snapshots`",
				metaResp.TypeName, snapshotPath)
		}
	}

	for _, factory := range p.DataSources(ctx) {
		metaResp := &datasource.MetadataResponse{}
		factory().Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "streamkap"}, metaResp)

		name := strings.TrimPrefix(metaResp.TypeName, "streamkap_")
		snapshotPath := filepath.Join("testdata", "schemas", "datasource_"+name+"_v1.json")

		if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
			t.Errorf("data source %q has no schema snapshot at %s; add a "+
				"TestSchemaBackwardsCompatibility_DataSource* case and run `make snapshots`",
				metaResp.TypeName, snapshotPath)
		}
	}
}

// --- Data sources ---

func TestSchemaBackwardsCompatibility_DataSourceTransform(t *testing.T) {
	runDataSourceSchemaCompatTest(t, dataSourceCompatTestCase{
		name:              "datasource_transform",
		snapshotFile:      "datasource_transform_v1.json",
		dataSourceFactory: ds.NewTransformDataSource,
	})
}

func TestSchemaBackwardsCompatibility_DataSourceTag(t *testing.T) {
	runDataSourceSchemaCompatTest(t, dataSourceCompatTestCase{
		name:              "datasource_tag",
		snapshotFile:      "datasource_tag_v1.json",
		dataSourceFactory: ds.NewTagDataSource,
	})
}

func TestSchemaBackwardsCompatibility_DataSourceTags(t *testing.T) {
	runDataSourceSchemaCompatTest(t, dataSourceCompatTestCase{
		name:              "datasource_tags",
		snapshotFile:      "datasource_tags_v1.json",
		dataSourceFactory: ds.NewTagsDataSource,
	})
}

func TestSchemaBackwardsCompatibility_DataSourceTopics(t *testing.T) {
	runDataSourceSchemaCompatTest(t, dataSourceCompatTestCase{
		name:              "datasource_topics",
		snapshotFile:      "datasource_topics_v1.json",
		dataSourceFactory: ds.NewTopicsDataSource,
	})
}

func TestSchemaBackwardsCompatibility_DataSourceTopic(t *testing.T) {
	runDataSourceSchemaCompatTest(t, dataSourceCompatTestCase{
		name:              "datasource_topic",
		snapshotFile:      "datasource_topic_v1.json",
		dataSourceFactory: ds.NewTopicDataSource,
	})
}

func TestSchemaBackwardsCompatibility_DataSourceRoles(t *testing.T) {
	runDataSourceSchemaCompatTest(t, dataSourceCompatTestCase{
		name:              "datasource_roles",
		snapshotFile:      "datasource_roles_v1.json",
		dataSourceFactory: ds.NewRolesDataSource,
	})
}

func TestSchemaBackwardsCompatibility_DataSourceTopicMetrics(t *testing.T) {
	runDataSourceSchemaCompatTest(t, dataSourceCompatTestCase{
		name:              "datasource_topic_metrics",
		snapshotFile:      "datasource_topic_metrics_v1.json",
		dataSourceFactory: ds.NewTopicMetricsDataSource,
	})
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

// The remaining webhook sources carry an `api_key` that the backend spec does not
// flag as a secret; tfgen forces it Sensitive. Without a baseline here, a
// regression would go unnoticed — runSchemaCompatTest skips when no snapshot exists.

func TestSchemaBackwardsCompatibility_SourceSalesforceWebhook(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_salesforce_webhook",
		snapshotFile:    "source_salesforce_webhook_v1.json",
		resourceFactory: source.NewSalesforceWebhookResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceZendeskWebhook(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_zendesk_webhook",
		snapshotFile:    "source_zendesk_webhook_v1.json",
		resourceFactory: source.NewZendeskWebhookResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceShopifyWebhook(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_shopify_webhook",
		snapshotFile:    "source_shopify_webhook_v1.json",
		resourceFactory: source.NewShopifyWebhookResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceStripeWebhook(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_stripe_webhook",
		snapshotFile:    "source_stripe_webhook_v1.json",
		resourceFactory: source.NewStripeWebhookResource,
	})
}

func TestSchemaBackwardsCompatibility_SourceInformix(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "source_informix",
		snapshotFile:    "source_informix_v1.json",
		resourceFactory: source.NewInformixResource,
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

func TestSchemaBackwardsCompatibility_TransformTopicRouter(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "transform_topic_router",
		snapshotFile:    "transform_topic_router_v1.json",
		resourceFactory: transform.NewTopicRouterResource,
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

func TestSchemaBackwardsCompatibility_KafkaUser(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "kafka_user",
		snapshotFile:    "kafka_user_v1.json",
		resourceFactory: kafka_user.NewKafkaUserResource,
	})
}

func TestSchemaBackwardsCompatibility_ClientCredential(t *testing.T) {
	runSchemaCompatTest(t, schemaCompatTestCase{
		name:            "client_credential",
		snapshotFile:    "client_credential_v1.json",
		resourceFactory: client_credential.NewClientCredentialResource,
	})
}
