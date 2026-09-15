package provider

import (
	"context"
	"maps"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/destination"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/shared"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
)

func TestDeprecatedAliasPlan(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		connector            connector.ConnectorConfig
		attribute, canonical string
		typ                  tftypes.Type
		value, want          any
	}{
		{"KafkaDirect alias", &source.KafkaDirectConfig{}, "kafka_format", "format", tftypes.String, "json", "json"},
		{"KafkaDirect canonical", &source.KafkaDirectConfig{}, "format", "format", tftypes.String, "avro", "avro"},
		{"KafkaDirect default", &source.KafkaDirectConfig{}, "format", "format", tftypes.String, nil, "string"},
		{"KafkaDirect unknown alias", &source.KafkaDirectConfig{}, "kafka_format", "format", tftypes.String, tftypes.UnknownValue, tftypes.UnknownValue},
		{"PostgreSQL predicate", &source.PostgreSQLConfig{}, "predicates_istopictoenrich_pattern", "predicates_is_topic_to_enrich_pattern", tftypes.String, "orders", "orders"},
		{"MySQL predicate", &source.MySQLConfig{}, "predicates_istopictoenrich_pattern", "predicates_is_topic_to_enrich_pattern", tftypes.String, "orders", "orders"},
		{"MySQL timezone", &source.MySQLConfig{}, "database_connection_timezone", "database_connection_time_zone", tftypes.String, "UTC", "UTC"},
		{"MongoDB predicate", &source.MongoDBConfig{}, "predicates_istopictoenrich_pattern", "predicates_is_topic_to_enrich_pattern", tftypes.String, "orders", "orders"},
		{"MongoDB array", &source.MongoDBConfig{}, "array_encoding", "transforms_unwrap_array_encoding", tftypes.String, "array", "array"},
		{"MongoDB document", &source.MongoDBConfig{}, "nested_document_encoding", "transforms_unwrap_document_encoding", tftypes.String, "json", "json"},
		{"SQL Server parallelism", &source.SQLServerConfig{}, "snapshot_parallelism", "streamkap_snapshot_parallelism", tftypes.Number, 2, int64(2)},
		{"Snowflake schema creation", &destination.SnowflakeConfig{}, "auto_schema_creation", "create_schema_auto", tftypes.Bool, false, false},
	} {
		for _, operation := range []string{"create", "update", "destroy"} {
			t.Run(tc.name+"/"+operation, func(t *testing.T) {
				ctx := context.Background()
				connectorConfig := tc.connector
				var schemaResponse resource.SchemaResponse
				connector.NewBaseConnectorResource(connectorConfig).Schema(ctx, resource.SchemaRequest{}, &schemaResponse)
				s := schemaResponse.Schema
				cfg := configWithOnlyAttrSet(ctx, s, tc.attribute, tftypes.NewValue(tc.typ, tc.value))
				typ := s.Type().TerraformType(ctx)
				config, err := tfprotov6.NewDynamicValue(typ, cfg.Raw)
				require.NoError(t, err)
				prior, err := tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, nil))
				require.NoError(t, err)
				proposed := config
				if operation != "create" {
					var values map[string]tftypes.Value
					require.NoError(t, cfg.Raw.As(&values))
					values = maps.Clone(values)
					priorValues := maps.Clone(values)
					priorValues["id"] = tftypes.NewValue(tftypes.String, "existing")
					var old any = "old"
					if tc.typ.Is(tftypes.Number) {
						old = 1
					}
					if tc.typ.Is(tftypes.Bool) {
						old = true
					}
					for name, apiField := range connectorConfig.GetFieldMappings() {
						if apiField == connectorConfig.GetFieldMappings()[tc.canonical] {
							priorValues[name] = tftypes.NewValue(tc.typ, old)
						}
					}
					prior, err = tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, priorValues))
					require.NoError(t, err)
					for name, attribute := range s.Attributes {
						if values[name].IsNull() && attribute.IsComputed() {
							values[name] = priorValues[name]
						}
					}
					proposed, err = tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, values))
					require.NoError(t, err)
					if operation == "destroy" {
						proposed, err = tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, nil))
						require.NoError(t, err)
						config = proposed
					}
				}
				server := providerserver.NewProtocol6(New("test")())()
				resp, err := server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
					TypeName: "streamkap_" + connectorConfig.GetResourceName(), Config: &config,
					PriorState: &prior, ProposedNewState: &proposed,
				})
				require.NoError(t, err)
				for _, d := range resp.Diagnostics {
					require.NotEqual(t, tfprotov6.DiagnosticSeverityError, d.Severity, "%s: %s", d.Summary, d.Detail)
				}
				require.NotNil(t, resp.PlannedState)
				planned, err := resp.PlannedState.Unmarshal(typ)
				require.NoError(t, err)
				if operation == "destroy" {
					require.True(t, planned.IsNull())
					return
				}
				var attrs map[string]tftypes.Value
				require.NoError(t, planned.As(&attrs))
				require.True(t, attrs[tc.canonical].Equal(tftypes.NewValue(tc.typ, tc.want)), "planned %s: %s", tc.canonical, attrs[tc.canonical])
				if tc.value != nil {
					for name, apiField := range connectorConfig.GetFieldMappings() {
						if apiField == connectorConfig.GetFieldMappings()[tc.canonical] {
							require.True(t, attrs[name].Equal(tftypes.NewValue(tc.typ, tc.want)), "planned %s: %s", name, attrs[name])
						}
					}
				}
				if tc.want == tftypes.UnknownValue {
					return
				}
				model := connectorConfig.NewModelInstance()
				plan := tfsdk.Plan{Schema: s, Raw: planned}
				require.False(t, plan.Get(ctx, model).HasError())
				for range 100 {
					configMap, err := shared.ModelToConfigMap(ctx, model, connectorConfig.GetFieldMappings(), nil)
					require.NoError(t, err)
					require.Equal(t, tc.want, configMap[connectorConfig.GetFieldMappings()[tc.canonical]])
				}
				// Simulate an API echo, then check the next plan preserves computed state.
				applied := maps.Clone(attrs)
				for name, value := range applied {
					if !value.IsKnown() {
						applied[name] = tftypes.NewValue(value.Type(), nil)
					}
				}
				applied["id"] = tftypes.NewValue(tftypes.String, "existing")
				applied["connector_status"] = tftypes.NewValue(tftypes.String, "RUNNING")
				appliedRaw := tftypes.NewValue(typ, applied)
				prior, err = tfprotov6.NewDynamicValue(typ, appliedRaw)
				require.NoError(t, err)
				var next map[string]tftypes.Value
				require.NoError(t, cfg.Raw.As(&next))
				next = maps.Clone(next)
				for name, attribute := range s.Attributes {
					if next[name].IsNull() && attribute.IsComputed() {
						next[name] = applied[name]
					}
				}
				proposed, err = tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, next))
				require.NoError(t, err)
				resp, err = server.PlanResourceChange(ctx, &tfprotov6.PlanResourceChangeRequest{
					TypeName: "streamkap_" + connectorConfig.GetResourceName(), Config: &config,
					PriorState: &prior, ProposedNewState: &proposed,
				})
				require.NoError(t, err)
				for _, d := range resp.Diagnostics {
					require.NotEqual(t, tfprotov6.DiagnosticSeverityError, d.Severity, "%s: %s", d.Summary, d.Detail)
				}
				require.NotNil(t, resp.PlannedState)
				nextPlan, err := resp.PlannedState.Unmarshal(typ)
				require.NoError(t, err)
				require.True(t, nextPlan.Equal(appliedRaw), "follow-up plan changed unchanged state")

			})
		}
	}
}
