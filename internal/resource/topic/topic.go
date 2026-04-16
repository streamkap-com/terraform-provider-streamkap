package topic

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ res.Resource                = &TopicResource{}
	_ res.ResourceWithConfigure   = &TopicResource{}
	_ res.ResourceWithImportState = &TopicResource{}
)

func NewTopicResource() res.Resource {
	return &TopicResource{}
}

// TopicResource defines the resource implementation.
type TopicResource struct {
	client         api.StreamkapAPI
}

// TopicResourceModel describes the resource data model.
type TopicResourceModel struct {
	TopicID            types.String  `tfsdk:"topic_id"`
	PartitionCount     types.Int64   `tfsdk:"partition_count"`
}

func (r *TopicResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topic"
}

func (r *TopicResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Streamkap Kafka topic's partition count.",
		MarkdownDescription: "Manages a **Streamkap Kafka topic's partition count**.\n\n" +
			"This resource allows you to modify the partition count of an existing Kafka topic " +
			"in your Streamkap cluster. Use this to scale topic throughput.\n\n" +
			"**Note:** Partition count can only be increased, not decreased.\n\n" +
			"[Documentation](https://docs.streamkap.com/streamkap-provider-for-terraform)",
		Attributes: map[string]schema.Attribute{
			"topic_id": schema.StringAttribute{
				Description:         "The Kafka topic identifier. Format: <source-id>.<schema>.<table> for CDC topics.",
				MarkdownDescription: "The Kafka topic identifier. Format: `<source-id>.<schema>.<table>` for CDC topics.",
				Required:            true,
			},
			"partition_count": schema.Int64Attribute{
				Required:            true,
				Description:         "Number of partitions for the topic. Can only be increased, not decreased. Higher values allow more parallel consumers.",
				MarkdownDescription: "Number of partitions for the topic. Can only be increased, not decreased. Higher values allow more parallel consumers.",
			},
		},
	}
}

func (r *TopicResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Topic Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *TopicResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan TopicResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Pre CREATE ===> plan: "+fmt.Sprintf("%+v", plan))

	topic, err := r.client.UpdateTopic(ctx, plan.TopicID.ValueString(), api.Topic{
		TopicID: plan.TopicID.ValueString(),
		PartitionCount: int(plan.PartitionCount.ValueInt64()),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating topic",
			fmt.Sprintf("Unable to update topic, got error: %s", err),
		)
		return
	}
	tflog.Debug(ctx, "Post CREATE ===> config: "+fmt.Sprintf("%+v", topic))

	if err := r.configMap2Model(*topic, &plan); err != nil {
		resp.Diagnostics.AddError("Error mapping topic response", err.Error())
		return
	}
	tflog.Debug(ctx, "Post CREATE ===> plan: "+fmt.Sprintf("%+v", plan))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *TopicResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state TopicResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	topicID := state.TopicID.ValueString()
	topic, err := r.client.GetTopic(ctx, topicID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading topic",
			fmt.Sprintf("Unable to read topic, got error: %s", err),
		)
		return
	}
	if topic == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if err := r.configMap2Model(*topic, &state); err != nil {
		resp.Diagnostics.AddError("Error mapping topic response", err.Error())
		return
	}
	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", state))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *TopicResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan TopicResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	topic, err := r.client.UpdateTopic(ctx, plan.TopicID.ValueString(), api.Topic{
		TopicID: plan.TopicID.ValueString(),
		PartitionCount: int(plan.PartitionCount.ValueInt64()),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating topic",
			fmt.Sprintf("Unable to update topic, got error: %s", err),
		)
		return
	}

	// Update resource state with updated items
	if err := r.configMap2Model(*topic, &plan); err != nil {
		resp.Diagnostics.AddError("Error mapping topic response", err.Error())
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *TopicResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	// This resource manages partition count only, not the topic lifecycle.
	// The underlying Kafka topic is owned by its producer (source or transform)
	// and must not be deleted here. Destroying the resource simply drops it
	// from Terraform state; the topic remains in Kafka.
	var state TopicResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Removing topic from Terraform state (partition-count-only resource; Kafka topic is not deleted): "+state.TopicID.ValueString())
}

func (r *TopicResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("topic_id"), req, resp)
}

func (r *TopicResource) configMap2Model(cfg api.Topic, model *TopicResourceModel) (err error) {
	// Copy the config map to the model
	model.TopicID = types.StringValue(cfg.TopicID)
	model.PartitionCount = types.Int64Value(int64(cfg.PartitionCount))

	return
}
