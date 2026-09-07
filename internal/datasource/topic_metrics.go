package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	ds "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

var _ ds.DataSource = &TopicMetricsDataSource{}

func NewTopicMetricsDataSource() ds.DataSource {
	return &TopicMetricsDataSource{}
}

type TopicMetricsDataSource struct {
	client api.StreamkapAPI
}

type TopicMetricsEntityModel struct {
	ID         types.String `tfsdk:"id"`
	EntityType types.String `tfsdk:"entity_type"`
	Connector  types.String `tfsdk:"connector"`
	TopicIDs   types.List   `tfsdk:"topic_ids"`
	TopicDBIDs types.List   `tfsdk:"topic_db_ids"`
}

type TopicMetricsResultModel struct {
	EntityID             types.String  `tfsdk:"entity_id"`
	TopicID              types.String  `tfsdk:"topic_id"`
	ID                   types.String  `tfsdk:"id"`
	PartitionCount       types.Int64   `tfsdk:"partition_count"`
	ReplicationFactor    types.Int64   `tfsdk:"replication_factor"`
	RetentionMs          types.Int64   `tfsdk:"retention_ms"`
	LastMessageTimestamp types.Int64   `tfsdk:"last_message_timestamp"`
	SnapshotStatusJSON   types.String  `tfsdk:"snapshot_status_json"`
	RecordErrorTotal     types.Int64   `tfsdk:"record_error_total"`
	MessagesIn           types.Int64   `tfsdk:"messages_in"`
	MessagesOut          types.Int64   `tfsdk:"messages_out"`
	BytesIn              types.Int64   `tfsdk:"bytes_in"`
	BytesOut             types.Int64   `tfsdk:"bytes_out"`
	Lag                  types.Int64   `tfsdk:"lag"`
	AvgLatencyMs         types.Float64 `tfsdk:"avg_latency_ms"`
}

type TopicMetricsDataSourceModel struct {
	Entities     []TopicMetricsEntityModel `tfsdk:"entities"`
	TimeInterval types.Int64               `tfsdk:"time_interval"`
	TimeUnit     types.String              `tfsdk:"time_unit"`
	Results      []TopicMetricsResultModel `tfsdk:"results"`
}

func (d *TopicMetricsDataSource) Metadata(ctx context.Context, req ds.MetadataRequest, resp *ds.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topic_metrics"
}

func (d *TopicMetricsDataSource) Schema(ctx context.Context, req ds.SchemaRequest, resp *ds.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves metrics for specific Streamkap Kafka topics.",
		MarkdownDescription: "Retrieves metrics for specific **Streamkap Kafka topics**.\n\n" +
			"Use this data source to query broker metadata and recent status for topics.\n\n" +
			"[Documentation](https://docs.streamkap.com/streamkap-provider-for-terraform)",

		Attributes: map[string]schema.Attribute{
			"entities": schema.ListNestedAttribute{
				Description:         "List of entities with their topic IDs to get metrics for.",
				MarkdownDescription: "List of entities with their topic IDs to get metrics for.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description:         "Entity ID (source, transform, or destination ID).",
							MarkdownDescription: "Entity ID (source, transform, or destination ID).",
							Required:            true,
						},
						"entity_type": schema.StringAttribute{
							Description:         "Entity type. Valid values: sources, transforms, destinations.",
							MarkdownDescription: "Entity type. Valid values: `sources`, `transforms`, `destinations`.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("sources", "transforms", "destinations"),
							},
						},
						"connector": schema.StringAttribute{
							Description:         "Connector type (e.g., postgresql, snowflake, map_filter).",
							MarkdownDescription: "Connector type (e.g., `postgresql`, `snowflake`, `map_filter`).",
							Required:            true,
						},
						"topic_ids": schema.ListAttribute{
							Description:         "List of topic IDs for this entity.",
							MarkdownDescription: "List of topic IDs for this entity.",
							Required:            true,
							ElementType:         types.StringType,
						},
						"topic_db_ids": schema.ListAttribute{
							Description:         "List of topic database IDs corresponding by position to topic_ids. Required for sources and transforms; use an empty list for destinations.",
							MarkdownDescription: "List of topic database IDs corresponding by position to `topic_ids`. Required for sources and transforms; use an empty list for destinations.",
							Required:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
			"time_interval": schema.Int64Attribute{
				Description:         "Deprecated. Accepted for configuration compatibility but ignored by the current API.",
				MarkdownDescription: "**Deprecated:** Accepted for configuration compatibility but ignored by the current API.",
				Optional:            true,
				DeprecationMessage:  "time_interval is ignored because the topic table metrics API no longer supports time aggregation.",
			},
			"time_unit": schema.StringAttribute{
				Description:         "Deprecated. Accepted for configuration compatibility but ignored by the current API.",
				MarkdownDescription: "**Deprecated:** Accepted for configuration compatibility but ignored by the current API.",
				Optional:            true,
				DeprecationMessage:  "time_unit is ignored because the topic table metrics API no longer supports time aggregation.",
				Validators: []validator.String{
					stringvalidator.OneOf("minutes", "hours", "days"),
				},
			},
			"results": schema.ListNestedAttribute{
				Description:         "Metrics results for each topic.",
				MarkdownDescription: "Metrics results for each topic.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"entity_id": schema.StringAttribute{
							Description:         "Entity ID associated with the requested topic. Null if multiple requested entities share the topic.",
							MarkdownDescription: "Entity ID associated with the requested topic. Null if multiple requested entities share the topic.",
							Computed:            true,
						},
						"topic_id": schema.StringAttribute{
							Description:         "Topic ID.",
							MarkdownDescription: "Topic ID.",
							Computed:            true,
						},
						"id": schema.StringAttribute{
							Description:         "Topic identifier returned in the response row.",
							MarkdownDescription: "Topic identifier returned in the response row.",
							Computed:            true,
						},
						"partition_count": schema.Int64Attribute{
							Description:         "Kafka partition count. Null when broker metadata is unavailable.",
							MarkdownDescription: "Kafka partition count. Null when broker metadata is unavailable.",
							Computed:            true,
						},
						"replication_factor": schema.Int64Attribute{
							Description:         "Kafka replication factor. Null when broker metadata is unavailable.",
							MarkdownDescription: "Kafka replication factor. Null when broker metadata is unavailable.",
							Computed:            true,
						},
						"retention_ms": schema.Int64Attribute{
							Description:         "Kafka retention period in milliseconds. Null when broker metadata is unavailable.",
							MarkdownDescription: "Kafka retention period in milliseconds. Null when broker metadata is unavailable.",
							Computed:            true,
						},
						"last_message_timestamp": schema.Int64Attribute{
							Description:         "Unix timestamp in milliseconds of the latest message. Null when unavailable.",
							MarkdownDescription: "Unix timestamp in milliseconds of the latest message. Null when unavailable.",
							Computed:            true,
						},
						"snapshot_status_json": schema.StringAttribute{
							Description:         "Snapshot status entries as JSON. The backend may add fields to these entries.",
							MarkdownDescription: "Snapshot status entries as JSON. The backend may add fields to these entries.",
							Computed:            true,
						},
						"record_error_total": schema.Int64Attribute{
							Description:         "Latest record error total. Null when ClickHouse metrics are unavailable.",
							MarkdownDescription: "Latest record error total. Null when ClickHouse metrics are unavailable.",
							Computed:            true,
						},
						"messages_in":  deprecatedUnavailableMetric("messages_in"),
						"messages_out": deprecatedUnavailableMetric("messages_out"),
						"bytes_in":     deprecatedUnavailableMetric("bytes_in"),
						"bytes_out":    deprecatedUnavailableMetric("bytes_out"),
						"lag":          deprecatedUnavailableMetric("lag"),
						"avg_latency_ms": schema.Float64Attribute{
							Description:         "Deprecated. Always null because the current API does not return average latency.",
							MarkdownDescription: "**Deprecated:** Always null because the current API does not return average latency.",
							Computed:            true,
							DeprecationMessage:  "avg_latency_ms is unavailable from the current topic table metrics API.",
						},
					},
				},
			},
		},
	}
}

func (d *TopicMetricsDataSource) Configure(ctx context.Context, req ds.ConfigureRequest, resp *ds.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected TopicMetrics Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *TopicMetricsDataSource) Read(ctx context.Context, req ds.ReadRequest, resp *ds.ReadResponse) {
	var config TopicMetricsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request
	apiReq := api.TopicTableMetricsRequest{
		Entities: make([]api.TopicMetricsEntity, len(config.Entities)),
	}
	for i, e := range config.Entities {
		// Extract topic IDs from types.List
		var topicIDs []string
		resp.Diagnostics.Append(e.TopicIDs.ElementsAs(ctx, &topicIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Extract topic DB IDs from types.List
		var topicDBIDs []string
		resp.Diagnostics.Append(e.TopicDBIDs.ElementsAs(ctx, &topicDBIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		apiReq.Entities[i] = api.TopicMetricsEntity{
			ID:         e.ID.ValueString(),
			EntityType: e.EntityType.ValueString(),
			Connector:  e.Connector.ValueString(),
			TopicIDs:   topicIDs,
			TopicDBIDs: topicDBIDs,
		}
	}
	// Call API
	metrics, err := d.client.GetTopicTableMetrics(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading topic metrics",
			fmt.Sprintf("Unable to get topic metrics: %s", err),
		)
		return
	}

	config.Results = flattenTopicTableMetrics(metrics, apiReq.Entities, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func deprecatedUnavailableMetric(name string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Description:         fmt.Sprintf("Deprecated. Always null because the current API does not return %s.", name),
		MarkdownDescription: fmt.Sprintf("**Deprecated:** Always null because the current API does not return `%s`.", name),
		Computed:            true,
		DeprecationMessage:  fmt.Sprintf("%s is unavailable from the current topic table metrics API.", name),
	}
}

func flattenTopicTableMetrics(metrics api.TopicTableMetricsResponse, entities []api.TopicMetricsEntity, diags *diag.Diagnostics) []TopicMetricsResultModel {
	entityByTopicID := make(map[string]types.String)
	for _, entity := range entities {
		for _, topicID := range entity.TopicIDs {
			if existing, exists := entityByTopicID[topicID]; !exists {
				entityByTopicID[topicID] = types.StringValue(entity.ID)
			} else if existing.ValueString() != entity.ID {
				entityByTopicID[topicID] = types.StringNull()
			}
		}
	}

	topicIDs := make([]string, 0, len(metrics))
	for topicID := range metrics {
		topicIDs = append(topicIDs, topicID)
	}
	sort.Strings(topicIDs)

	results := make([]TopicMetricsResultModel, 0, len(topicIDs))
	for _, topicID := range topicIDs {
		row := metrics[topicID]
		entityID := types.StringNull()
		if value, ok := entityByTopicID[topicID]; ok {
			entityID = value
		}
		snapshotStatus, err := json.Marshal(row.SnapshotStatus)
		if err != nil {
			diags.AddError("Error mapping topic metrics", fmt.Sprintf("Unable to encode snapshot status for topic %s: %s", topicID, err))
			continue
		}
		results = append(results, TopicMetricsResultModel{
			EntityID:             entityID,
			TopicID:              types.StringValue(topicID),
			ID:                   types.StringValue(row.ID),
			PartitionCount:       types.Int64PointerValue(row.Kafka.PartitionCount),
			ReplicationFactor:    types.Int64PointerValue(row.Kafka.ReplicationFactor),
			RetentionMs:          types.Int64PointerValue(row.Kafka.RetentionMs),
			LastMessageTimestamp: types.Int64PointerValue(row.LastMessageTimestamp),
			SnapshotStatusJSON:   types.StringValue(string(snapshotStatus)),
			RecordErrorTotal:     types.Int64PointerValue(row.RecordErrorTotal),
			MessagesIn:           types.Int64Null(),
			MessagesOut:          types.Int64Null(),
			BytesIn:              types.Int64Null(),
			BytesOut:             types.Int64Null(),
			Lag:                  types.Int64Null(),
			AvgLatencyMs:         types.Float64Null(),
		})
	}
	return results
}
