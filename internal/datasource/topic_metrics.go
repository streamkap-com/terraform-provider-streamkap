package datasource

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	ds "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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
	EntityID     types.String  `tfsdk:"entity_id"`
	TopicID      types.String  `tfsdk:"topic_id"`
	MessagesIn   types.Int64   `tfsdk:"messages_in"`
	MessagesOut  types.Int64   `tfsdk:"messages_out"`
	BytesIn      types.Int64   `tfsdk:"bytes_in"`
	BytesOut     types.Int64   `tfsdk:"bytes_out"`
	Lag          types.Int64   `tfsdk:"lag"`
	AvgLatencyMs types.Float64 `tfsdk:"avg_latency_ms"`
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
			"Use this data source to query throughput and latency metrics for topics.\n\n" +
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
							Description:         "List of topic database IDs (MongoDB ObjectIds).",
							MarkdownDescription: "List of topic database IDs (MongoDB ObjectIds).",
							Required:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
			"time_interval": schema.Int64Attribute{
				Description:         "Time interval for aggregation. Defaults to 1.",
				MarkdownDescription: "Time interval for aggregation. Defaults to `1`.",
				Optional:            true,
			},
			"time_unit": schema.StringAttribute{
				Description:         "Time unit for interval. Valid values: minutes, hours, days. Defaults to hours.",
				MarkdownDescription: "Time unit for interval. Valid values: `minutes`, `hours`, `days`. Defaults to `hours`.",
				Optional:            true,
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
							Description:         "Entity ID.",
							MarkdownDescription: "Entity ID.",
							Computed:            true,
						},
						"topic_id": schema.StringAttribute{
							Description:         "Topic ID.",
							MarkdownDescription: "Topic ID.",
							Computed:            true,
						},
						"messages_in": schema.Int64Attribute{
							Description:         "Number of messages received.",
							MarkdownDescription: "Number of messages received.",
							Computed:            true,
						},
						"messages_out": schema.Int64Attribute{
							Description:         "Number of messages sent.",
							MarkdownDescription: "Number of messages sent.",
							Computed:            true,
						},
						"bytes_in": schema.Int64Attribute{
							Description:         "Bytes received.",
							MarkdownDescription: "Bytes received.",
							Computed:            true,
						},
						"bytes_out": schema.Int64Attribute{
							Description:         "Bytes sent.",
							MarkdownDescription: "Bytes sent.",
							Computed:            true,
						},
						"lag": schema.Int64Attribute{
							Description:         "Consumer lag.",
							MarkdownDescription: "Consumer lag.",
							Computed:            true,
						},
						"avg_latency_ms": schema.Float64Attribute{
							Description:         "Average latency in milliseconds.",
							MarkdownDescription: "Average latency in milliseconds.",
							Computed:            true,
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
	if !config.TimeInterval.IsNull() {
		val := int(config.TimeInterval.ValueInt64())
		apiReq.TimeInterval = &val
	}
	if !config.TimeUnit.IsNull() {
		val := config.TimeUnit.ValueString()
		apiReq.TimeUnit = &val
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

	// Flatten results - initialize with null values for optional fields
	var results []TopicMetricsResultModel
	for entityID, topicMetrics := range metrics {
		for topicID, m := range topicMetrics {
			result := TopicMetricsResultModel{
				EntityID:     types.StringValue(entityID),
				TopicID:      types.StringValue(topicID),
				MessagesIn:   types.Int64Null(),
				MessagesOut:  types.Int64Null(),
				BytesIn:      types.Int64Null(),
				BytesOut:     types.Int64Null(),
				Lag:          types.Int64Null(),
				AvgLatencyMs: types.Float64Null(),
			}
			if m.MessagesIn != nil {
				result.MessagesIn = types.Int64Value(*m.MessagesIn)
			}
			if m.MessagesOut != nil {
				result.MessagesOut = types.Int64Value(*m.MessagesOut)
			}
			if m.BytesIn != nil {
				result.BytesIn = types.Int64Value(*m.BytesIn)
			}
			if m.BytesOut != nil {
				result.BytesOut = types.Int64Value(*m.BytesOut)
			}
			if m.Lag != nil {
				result.Lag = types.Int64Value(*m.Lag)
			}
			if m.AvgLatencyMs != nil {
				result.AvgLatencyMs = types.Float64Value(*m.AvgLatencyMs)
			}
			results = append(results, result)
		}
	}
	config.Results = results

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
