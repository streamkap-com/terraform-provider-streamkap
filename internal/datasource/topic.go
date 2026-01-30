package datasource

import (
	"context"
	"fmt"

	ds "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

var _ ds.DataSource = &TopicDataSource{}

func NewTopicDataSource() ds.DataSource {
	return &TopicDataSource{}
}

type TopicDataSource struct {
	client api.StreamkapAPI
}

type TopicDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	EntityID      types.String `tfsdk:"entity_id"`
	EntityName    types.String `tfsdk:"entity_name"`
	EntityType    types.String `tfsdk:"entity_type"`
	Partitions    types.Int64  `tfsdk:"partitions"`
	RetentionMs   types.Int64  `tfsdk:"retention_ms"`
	CleanupPolicy types.String `tfsdk:"cleanup_policy"`
	Prefix        types.String `tfsdk:"prefix"`
	Serialization types.String `tfsdk:"serialization"`
}

func (d *TopicDataSource) Metadata(ctx context.Context, req ds.MetadataRequest, resp *ds.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topic"
}

func (d *TopicDataSource) Schema(ctx context.Context, req ds.SchemaRequest, resp *ds.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves detailed information about a single Streamkap Kafka topic.",
		MarkdownDescription: "Retrieves detailed information about a single **Streamkap Kafka topic**.\n\n" +
			"Use this data source to look up topic details including Kafka configuration.\n\n" +
			"[Documentation](https://docs.streamkap.com/streamkap-provider-for-terraform)",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the topic to retrieve.",
				MarkdownDescription: "The unique identifier of the topic to retrieve.",
				Required:            true,
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
			"partitions": schema.Int64Attribute{
				Description:         "Number of Kafka partitions.",
				MarkdownDescription: "Number of Kafka partitions.",
				Computed:            true,
			},
			"retention_ms": schema.Int64Attribute{
				Description:         "Message retention time in milliseconds.",
				MarkdownDescription: "Message retention time in milliseconds.",
				Computed:            true,
			},
			"cleanup_policy": schema.StringAttribute{
				Description:         "Kafka cleanup policy (delete, compact).",
				MarkdownDescription: "Kafka cleanup policy (`delete`, `compact`).",
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
		},
	}
}

func (d *TopicDataSource) Configure(ctx context.Context, req ds.ConfigureRequest, resp *ds.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Topic Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *TopicDataSource) Read(ctx context.Context, req ds.ReadRequest, resp *ds.ReadResponse) {
	var config TopicDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	topicID := config.ID.ValueString()
	topic, err := d.client.GetTopicDetailed(ctx, topicID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading topic",
			fmt.Sprintf("Unable to read topic %s: %s", topicID, err),
		)
		return
	}
	if topic == nil {
		resp.Diagnostics.AddError(
			"Topic not found",
			fmt.Sprintf("Topic %s does not exist", topicID),
		)
		return
	}

	// Map response to model
	config.ID = types.StringValue(topic.ID)
	config.Name = types.StringValue(topic.Name)
	config.Prefix = types.StringPointerValue(topic.Prefix)
	config.Serialization = types.StringPointerValue(topic.Serialization)

	if topic.Entity != nil {
		config.EntityID = types.StringValue(topic.Entity.EntityID)
		config.EntityName = types.StringValue(topic.Entity.Name)
		config.EntityType = types.StringValue(topic.Entity.EntityType)
	} else {
		config.EntityID = types.StringNull()
		config.EntityName = types.StringNull()
		config.EntityType = types.StringNull()
	}

	if topic.Kafka != nil {
		config.Partitions = types.Int64Value(int64(topic.Kafka.Partitions))
		if topic.Kafka.Configs != nil {
			if topic.Kafka.Configs.RetentionMs != nil {
				config.RetentionMs = types.Int64Value(*topic.Kafka.Configs.RetentionMs)
			} else {
				config.RetentionMs = types.Int64Null()
			}
			config.CleanupPolicy = types.StringPointerValue(topic.Kafka.Configs.CleanupPolicy)
		} else {
			config.RetentionMs = types.Int64Null()
			config.CleanupPolicy = types.StringNull()
		}
	} else {
		config.Partitions = types.Int64Null()
		config.RetentionMs = types.Int64Null()
		config.CleanupPolicy = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
