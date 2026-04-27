package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/destination"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/transform"
)

// TestIssue80_OptionalFieldsArePlanStable asserts that every Optional attribute
// on every generated resource schema is ALSO Computed. This guards against the
// bug class reported in GitHub issue #80: the Streamkap backend may dynamically
// backfill a user-facing field at apply time (e.g.
// `transforms.MarkColumnsAsOptional.fields.include.list` → "*" for MongoDB →
// PostgreSQL pipelines, see app/destinations/dynamic_utils.py). If a field is
// Optional but NOT Computed, a plan that sets it to null while state holds the
// backend-backfilled value surfaces as:
//
//	Error: Provider produced inconsistent result after apply
//	.<field>: was null, but now cty.StringVal("…")
//
// The generator was updated in the same commit as this test to emit
// Optional + Computed + UseStateForUnknown for every `required: false` field
// without a static default. This test fails if a future generator regression
// reintroduces Optional-only fields on any generated schema.
//
// Required and Optional+Computed-with-default fields are exempt — the backend
// can't invisibly backfill a value the user must provide, and fields with
// static defaults already settle the plan. Deprecated-alias attributes are
// also exempt (they're hand-wrapped with their own precedence rules).
//
// Map-override fields (the `map_string` / `map_nested` overrides in
// cmd/tfgen/overrides.json) are also exempt — see issue82MapOverrideFields
// below. Their Go model field is a plain Go map that physically cannot hold
// an unknown value, so they MUST stay Optional-only. They are pure
// `user_defined: true` config (verified against backend
// configuration.latest.json) and not subject to dynamic backfill, so the
// issue-#80 failure mode doesn't apply to them.
func TestIssue80_OptionalFieldsArePlanStable(t *testing.T) {
	factories := []struct {
		name string
		new  func() resource.Resource
	}{
		// Source connectors
		{"source_alloydb", source.NewAlloyDBResource},
		{"source_db2", source.NewDB2Resource},
		{"source_documentdb", source.NewDocumentDBResource},
		{"source_dynamodb", source.NewDynamoDBResource},
		{"source_elasticsearch", source.NewElasticsearchResource},
		{"source_kafkadirect", source.NewKafkaDirectResource},
		{"source_mariadb", source.NewMariaDBResource},
		{"source_mongodb", source.NewMongoDBResource},
		{"source_mongodbhosted", source.NewMongoDBHostedResource},
		{"source_mysql", source.NewMySQLResource},
		{"source_oracle", source.NewOracleResource},
		{"source_oracleaws", source.NewOracleAWSResource},
		{"source_planetscale", source.NewPlanetScaleResource},
		{"source_postgresql", source.NewPostgreSQLResource},
		{"source_redis", source.NewRedisResource},
		{"source_s3", source.NewS3SourceResource},
		{"source_sqlserver", source.NewSQLServerResource},
		{"source_supabase", source.NewSupabaseResource},
		{"source_vitess", source.NewVitessResource},
		{"source_webhook", source.NewWebhookResource},

		// Destination connectors
		{"destination_azblob", destination.NewAzBlobResource},
		{"destination_bigquery", destination.NewBigQueryResource},
		{"destination_clickhouse", destination.NewClickHouseResource},
		{"destination_cockroachdb", destination.NewCockroachDBResource},
		{"destination_databricks", destination.NewDatabricksResource},
		{"destination_db2", destination.NewDB2DestResource},
		{"destination_gcs", destination.NewGCSResource},
		{"destination_httpsink", destination.NewHTTPSinkResource},
		{"destination_iceberg", destination.NewIcebergResource},
		{"destination_kafka", destination.NewKafkaResource},
		{"destination_kafkadirect", destination.NewKafkaDirectDestResource},
		{"destination_motherduck", destination.NewMotherduckResource},
		{"destination_mysql", destination.NewMySQLDestResource},
		{"destination_oracle", destination.NewOracleDestResource},
		{"destination_pinecone", destination.NewPineconeDestResource},
		{"destination_postgresql", destination.NewPostgreSQLResource},
		{"destination_r2", destination.NewR2Resource},
		{"destination_redis", destination.NewRedisDestResource},
		{"destination_redshift", destination.NewRedshiftResource},
		{"destination_s3", destination.NewS3Resource},
		{"destination_snowflake", destination.NewSnowflakeResource},
		{"destination_sqlserver", destination.NewSQLServerDestResource},
		{"destination_starburst", destination.NewStarburstResource},
		{"destination_weaviate", destination.NewWeaviateResource},

		// Transforms
		{"transform_enrich", transform.NewEnrichResource},
		{"transform_enrich_async", transform.NewEnrichAsyncResource},
		{"transform_fan_out", transform.NewFanOutResource},
		{"transform_map_filter", transform.NewMapFilterResource},
		{"transform_rollup", transform.NewRollupResource},
		{"transform_sql_join", transform.NewSqlJoinResource},
		{"transform_topic_router", transform.NewTopicRouterResource},
	}

	for _, f := range factories {
		t.Run(f.name, func(t *testing.T) {
			ctx := context.Background()
			schemaResp := &resource.SchemaResponse{}
			f.new().Schema(ctx, resource.SchemaRequest{}, schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("schema errors: %v", schemaResp.Diagnostics)
			}

			for name, attr := range schemaResp.Schema.Attributes {
				if !attr.IsOptional() {
					continue
				}
				if attr.IsComputed() {
					continue
				}
				if attr.GetDeprecationMessage() != "" {
					continue
				}
				if issue82MapOverrideFields[f.name+"."+name] {
					continue
				}
				t.Errorf("%s.%s is Optional but not Computed — exposes issue-#80 "+
					"inconsistent-result-after-apply when the backend dynamically backfills this field",
					f.name, name)
			}
		})
	}
}

// issue82MapOverrideFields lists the `map_string` / `map_nested` override
// attributes that MUST stay `Optional: true` only (no Computed). The Go model
// type for these fields is a plain Go map (`map[string]types.String` or
// `map[string]<NestedModel>`) which physically cannot hold an unknown value,
// and Computed forces the framework to plan an unknown on Create. See
// TestIssue82_MapOverridesAreOptionalOnly for the affirmative assertion.
var issue82MapOverrideFields = map[string]bool{
	"destination_snowflake.auto_qa_dedupe_table_mapping": true,
	"destination_clickhouse.topics_config_map":           true,
	"source_sqlserver.snapshot_custom_table_config":      true,
}

// TestIssue82_MapOverridesAreOptionalOnly is the regression guard for issue
// #82: the three map-override fields above must be `Optional: true` and NOT
// `Computed: true`, otherwise the framework crashes on Create with
// "Received unknown value, however the target type cannot handle unknown
// values" (path: <map field>, target type: map[string]…).
//
// If a future codegen change reintroduces Computed on these, this test
// catches it before users do.
func TestIssue82_MapOverridesAreOptionalOnly(t *testing.T) {
	cases := []struct {
		resourceName string
		attrName     string
		newResource  func() resource.Resource
	}{
		{"destination_snowflake", "auto_qa_dedupe_table_mapping", destination.NewSnowflakeResource},
		{"destination_clickhouse", "topics_config_map", destination.NewClickHouseResource},
		{"source_sqlserver", "snapshot_custom_table_config", source.NewSQLServerResource},
	}

	for _, tc := range cases {
		t.Run(tc.resourceName+"."+tc.attrName, func(t *testing.T) {
			ctx := context.Background()
			schemaResp := &resource.SchemaResponse{}
			tc.newResource().Schema(ctx, resource.SchemaRequest{}, schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("schema errors: %v", schemaResp.Diagnostics)
			}

			attr, ok := schemaResp.Schema.Attributes[tc.attrName]
			if !ok {
				t.Fatalf("attribute %s not found on %s", tc.attrName, tc.resourceName)
			}
			if !attr.IsOptional() {
				t.Errorf("%s.%s must be Optional", tc.resourceName, tc.attrName)
			}
			if attr.IsComputed() {
				t.Errorf("%s.%s must NOT be Computed — Go map type can't hold unknown (issue #82)",
					tc.resourceName, tc.attrName)
			}
		})
	}
}

// TestIssue80_OptionalComputedFieldsHaveUseStateForUnknown ensures the
// companion plan modifier is present. Without it, every Optional+Computed field
// the user hasn't set shows "(known after apply)" on every refresh, creating
// noisy plans. See cmd/tfgen/generator.go for the generator rule that pairs
// these.
func TestIssue80_OptionalComputedFieldsHaveUseStateForUnknown(t *testing.T) {
	// Spot-check on four representative schemas — the full matrix is exercised
	// indirectly via `go test ./cmd/tfgen/...` which verifies NeedsPlanMod for
	// any Optional+Computed-without-default entry the generator produces.
	factories := []struct {
		name string
		new  func() resource.Resource
	}{
		{"destination_postgresql", destination.NewPostgreSQLResource},
		{"destination_snowflake", destination.NewSnowflakeResource},
		{"source_postgresql", source.NewPostgreSQLResource},
		{"source_mongodb", source.NewMongoDBResource},
	}

	for _, f := range factories {
		t.Run(f.name, func(t *testing.T) {
			ctx := context.Background()
			schemaResp := &resource.SchemaResponse{}
			f.new().Schema(ctx, resource.SchemaRequest{}, schemaResp)
			if schemaResp.Diagnostics.HasError() {
				t.Fatalf("schema errors: %v", schemaResp.Diagnostics)
			}

			for name, attr := range schemaResp.Schema.Attributes {
				if !(attr.IsOptional() && attr.IsComputed()) {
					continue
				}
				if attr.GetDeprecationMessage() != "" {
					continue
				}
				if hasDefault(attr) {
					continue
				}
				if !hasUseStateForUnknown(attr) {
					t.Errorf("%s.%s is Optional+Computed without a Default, "+
						"but is missing UseStateForUnknown — will show "+
						"\"(known after apply)\" on every refresh",
						f.name, name)
				}
			}
		})
	}
}

func hasDefault(attr schema.Attribute) bool {
	switch a := attr.(type) {
	case schema.StringAttribute:
		return a.Default != nil
	case schema.Int64Attribute:
		return a.Default != nil
	case schema.BoolAttribute:
		return a.Default != nil
	case schema.ListAttribute:
		return a.Default != nil
	case schema.SetAttribute:
		return a.Default != nil
	case schema.MapAttribute:
		return a.Default != nil
	}
	return false
}

func hasUseStateForUnknown(attr schema.Attribute) bool {
	switch a := attr.(type) {
	case schema.StringAttribute:
		return anyUseStateForUnknown[planmodifier.String](a.PlanModifiers)
	case schema.Int64Attribute:
		return anyUseStateForUnknown[planmodifier.Int64](a.PlanModifiers)
	case schema.BoolAttribute:
		return anyUseStateForUnknown[planmodifier.Bool](a.PlanModifiers)
	case schema.ListAttribute:
		return anyUseStateForUnknown[planmodifier.List](a.PlanModifiers)
	case schema.SetAttribute:
		return anyUseStateForUnknown[planmodifier.Set](a.PlanModifiers)
	case schema.MapAttribute:
		return anyUseStateForUnknown[planmodifier.Map](a.PlanModifiers)
	case schema.MapNestedAttribute:
		return anyUseStateForUnknown[planmodifier.Map](a.PlanModifiers)
	}
	return false
}

// anyUseStateForUnknown detects the framework's UseStateForUnknown modifier by
// its Description text. The Plan Framework doesn't expose a .Name() method on
// modifiers; all built-in UseStateForUnknown variants return a description
// that includes the phrase "will not change". This is pragmatic, not beautiful.
func anyUseStateForUnknown[PM interface {
	Description(context.Context) string
}](mods []PM) bool {
	for _, pm := range mods {
		if containsIgnoreCase(pm.Description(context.Background()), "will not change") {
			return true
		}
	}
	return false
}

func containsIgnoreCase(s, substr string) bool {
	if substr == "" {
		return true
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			a := s[i+j]
			b := substr[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
