package datasource

import (
	"context"
	"fmt"

	ds "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ ds.DataSource = &TopicsDataSource{}

func NewTopicsDataSource() ds.DataSource {
	return &TopicsDataSource{}
}

// TopicsDataSource defines the data source implementation.
type TopicsDataSource struct {
	client api.StreamkapAPI
}

// TopicsDataSourceModel describes the data source data model.
type TopicsDataSourceModel struct {
	EntityType types.String        `tfsdk:"entity_type"`
	EntityIDs  types.List          `tfsdk:"entity_ids"`
	Topics     []TopicDetailsModel `tfsdk:"topics"`
	Total      types.Int64         `tfsdk:"total"`
}

// TopicDetailsModel represents a single topic in the list
type TopicDetailsModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	EntityID      types.String `tfsdk:"entity_id"`
	EntityName    types.String `tfsdk:"entity_name"`
	EntityType    types.String `tfsdk:"entity_type"`
	Prefix        types.String `tfsdk:"prefix"`
	Serialization types.String `tfsdk:"serialization"`
	Messages7D    types.Int64  `tfsdk:"messages_7d"`
	Messages30D   types.Int64  `tfsdk:"messages_30d"`
}

func (d *TopicsDataSource) Metadata(ctx context.Context, req ds.MetadataRequest, resp *ds.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topics"
}

func (d *TopicsDataSource) Schema(ctx context.Context, req ds.SchemaRequest, resp *ds.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves a list of Streamkap Kafka topics with optional filtering.",
		MarkdownDescription: "Retrieves a list of **Streamkap Kafka topics** with optional filtering.\n\n" +
			"Use this data source to discover topics across your Streamkap cluster.\n\n" +
			"[Documentation](https://docs.streamkap.com/streamkap-provider-for-terraform)",

		Attributes: map[string]schema.Attribute{
			"entity_type": schema.StringAttribute{
				Description:         "Filter topics by entity type. Valid values: sources, transforms, destinations.",
				MarkdownDescription: "Filter topics by entity type. Valid values: `sources`, `transforms`, `destinations`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("sources", "transforms", "destinations"),
				},
			},
			"entity_ids": schema.ListAttribute{
				Description:         "Filter topics by specific entity IDs.",
				MarkdownDescription: "Filter topics by specific entity IDs.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"total": schema.Int64Attribute{
				Description:         "Total number of topics matching the filter.",
				MarkdownDescription: "Total number of topics matching the filter.",
				Computed:            true,
			},
			"topics": schema.ListNestedAttribute{
				Description:         "List of topics matching the filter criteria.",
				MarkdownDescription: "List of topics matching the filter criteria.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description:         "Unique topic identifier.",
							MarkdownDescription: "Unique topic identifier.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							Description:         "Topic name.",
							MarkdownDescription: "Topic name.",
							Computed:            true,
						},
						"entity_id": schema.StringAttribute{
							Description:         "ID of the entity (source/transform/destination) that owns this topic.",
							MarkdownDescription: "ID of the entity (source/transform/destination) that owns this topic.",
							Computed:            true,
						},
						"entity_name": schema.StringAttribute{
							Description:         "Name of the entity that owns this topic.",
							MarkdownDescription: "Name of the entity that owns this topic.",
							Computed:            true,
						},
						"entity_type": schema.StringAttribute{
							Description:         "Type of entity: sources, transforms, or destinations.",
							MarkdownDescription: "Type of entity: `sources`, `transforms`, or `destinations`.",
							Computed:            true,
						},
						"prefix": schema.StringAttribute{
							Description:         "Topic name prefix.",
							MarkdownDescription: "Topic name prefix.",
							Computed:            true,
						},
						"serialization": schema.StringAttribute{
							Description:         "Message serialization format.",
							MarkdownDescription: "Message serialization format.",
							Computed:            true,
						},
						"messages_7d": schema.Int64Attribute{
							Description:         "Number of messages in the last 7 days.",
							MarkdownDescription: "Number of messages in the last 7 days.",
							Computed:            true,
						},
						"messages_30d": schema.Int64Attribute{
							Description:         "Number of messages in the last 30 days.",
							MarkdownDescription: "Number of messages in the last 30 days.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *TopicsDataSource) Configure(ctx context.Context, req ds.ConfigureRequest, resp *ds.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Topics Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *TopicsDataSource) Read(ctx context.Context, req ds.ReadRequest, resp *ds.ReadResponse) {
	var config TopicsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API params
	params := &api.TopicListParams{}
	if !config.EntityType.IsNull() {
		params.EntityType = config.EntityType.ValueString()
	}
	if !config.EntityIDs.IsNull() {
		var entityIDs []string
		resp.Diagnostics.Append(config.EntityIDs.ElementsAs(ctx, &entityIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		params.EntityIDs = entityIDs
	}

	// Call API
	result, err := d.client.ListTopics(ctx, params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading topics",
			fmt.Sprintf("Unable to list topics: %s", err),
		)
		return
	}

	// Map response to model
	config.Total = types.Int64Value(int64(result.Total))
	config.Topics = make([]TopicDetailsModel, len(result.Result))
	for i, topic := range result.Result {
		config.Topics[i] = TopicDetailsModel{
			ID:            types.StringValue(topic.ID),
			Name:          types.StringValue(topic.Name),
			Prefix:        types.StringPointerValue(topic.Prefix),
			Serialization: types.StringPointerValue(topic.Serialization),
		}
		if topic.Entity != nil {
			config.Topics[i].EntityID = types.StringValue(topic.Entity.EntityID)
			config.Topics[i].EntityName = types.StringValue(topic.Entity.Name)
			config.Topics[i].EntityType = types.StringValue(topic.Entity.EntityType)
		} else {
			config.Topics[i].EntityID = types.StringNull()
			config.Topics[i].EntityName = types.StringNull()
			config.Topics[i].EntityType = types.StringNull()
		}
		if topic.Messages7D != nil {
			config.Topics[i].Messages7D = types.Int64Value(*topic.Messages7D)
		} else {
			config.Topics[i].Messages7D = types.Int64Null()
		}
		if topic.Messages30D != nil {
			config.Topics[i].Messages30D = types.Int64Value(*topic.Messages30D)
		} else {
			config.Topics[i].Messages30D = types.Int64Null()
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
