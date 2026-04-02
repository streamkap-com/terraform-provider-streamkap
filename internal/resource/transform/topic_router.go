package transform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/helper"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ res.Resource                = &TransformTopicRouterResource{}
	_ res.ResourceWithConfigure   = &TransformTopicRouterResource{}
	_ res.ResourceWithImportState = &TransformTopicRouterResource{}
)

func NewTransformTopicRouterResource() res.Resource {
	return &TransformTopicRouterResource{}
}

// TransformTopicRouterResource defines the resource implementation.
type TransformTopicRouterResource struct {
	client api.StreamkapAPI
}

// TransformTopicRouterResourceModel describes the resource data model.
type TransformTopicRouterResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	TopicPrefix     types.String `tfsdk:"topic_prefix"`
	InputTopicIDs   types.String `tfsdk:"input_topic_ids"`
	KcClusterId     types.String `tfsdk:"kc_cluster_id"`
	TransformType   types.String `tfsdk:"transform_type"`
}

func (r *TransformTopicRouterResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transform_topic_router"
}

func (r *TransformTopicRouterResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Transform Topic Router resource — merges topics via KC connector with byte passthrough",
		MarkdownDescription: "Transform Topic Router resource — merges topics via KC Kafka destination connector with byte passthrough (no value deserialization).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Transform identifier",
				MarkdownDescription: "Transform identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Transform name",
				MarkdownDescription: "Transform name",
			},
			"topic_prefix": schema.StringAttribute{
				Required:            true,
				Description:         "Output topic prefix for merged topics (e.g. 'merged.dbo.')",
				MarkdownDescription: "Output topic prefix for merged topics (e.g. `merged.dbo.`)",
			},
			"input_topic_ids": schema.StringAttribute{
				Required:            true,
				Description:         "Comma-separated list of input topic IDs to merge",
				MarkdownDescription: "Comma-separated list of input topic IDs to merge",
			},
			"kc_cluster_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "KC cluster ID to deploy the connector to",
				MarkdownDescription: "KC cluster ID to deploy the connector to. Empty for default cluster.",
			},
			"transform_type": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *TransformTopicRouterResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Transform Topic Router Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *TransformTopicRouterResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan TransformTopicRouterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Pre CREATE topic_router ===> plan: "+fmt.Sprintf("%+v", plan))

	transform, err := r.client.CreateTransform(ctx, api.Transform{
		Name:          plan.Name.ValueString(),
		TransformType: "topic_router",
		Config: map[string]any{
			"transforms.name":                      plan.Name.ValueString(),
			"transforms.input.topic.pattern":        plan.InputTopicIDs.ValueString(),
			"transforms.output.topic.pattern":       "",
			"transforms.input.serialization.format":  "Any",
			"transforms.output.serialization.format": "Any",
		},
		Implementation: map[string]any{
			"topic_prefix":  plan.TopicPrefix.ValueString(),
			"kc_cluster_id": plan.KcClusterId.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating topic_router transform",
			fmt.Sprintf("Unable to create topic_router transform, got error: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(transform.ID)
	plan.TransformType = types.StringValue(transform.TransformType)

	// Deploy live immediately after creation
	err = r.client.DeployTransformLive(ctx, transform.ID, "0")
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Transform created but deploy failed",
			fmt.Sprintf("Transform %s was created but live deployment failed: %s. You may need to deploy manually.", transform.ID, err),
		)
	}

	tflog.Debug(ctx, "Post CREATE topic_router ===> plan: "+fmt.Sprintf("%+v", plan))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *TransformTopicRouterResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state TransformTopicRouterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	transformID := state.ID.ValueString()
	transform, err := r.client.GetTransform(ctx, transformID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading topic_router transform",
			fmt.Sprintf("Unable to read topic_router transform, got error: %s", err),
		)
		return
	}
	if transform == nil {
		resp.Diagnostics.AddError(
			"Error reading topic_router transform",
			fmt.Sprintf("Topic router transform %s does not exist", transformID),
		)
		return
	}

	state.Name = types.StringValue(transform.Name)
	state.TransformType = types.StringValue(transform.TransformType)

	// Read implementation fields
	if transform.Implementation != nil {
		if v, ok := transform.Implementation["topic_prefix"]; ok && v != nil {
			state.TopicPrefix = helper.GetTfCfgString(transform.Implementation, "topic_prefix")
		}
		if v, ok := transform.Implementation["kc_cluster_id"]; ok && v != nil {
			state.KcClusterId = helper.GetTfCfgString(transform.Implementation, "kc_cluster_id")
		}
	}

	// Read input topic pattern from config
	if transform.Config != nil {
		state.InputTopicIDs = helper.GetTfCfgString(transform.Config, "transforms.input.topic.pattern")
	}

	tflog.Info(ctx, "===> topic_router state: "+fmt.Sprintf("%+v", state))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TransformTopicRouterResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan TransformTopicRouterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	transform, err := r.client.UpdateTransform(ctx, plan.ID.ValueString(), api.Transform{
		Name:          plan.Name.ValueString(),
		TransformType: "topic_router",
		Config: map[string]any{
			"transforms.name":                      plan.Name.ValueString(),
			"transforms.input.topic.pattern":        plan.InputTopicIDs.ValueString(),
			"transforms.output.topic.pattern":       "",
			"transforms.input.serialization.format":  "Any",
			"transforms.output.serialization.format": "Any",
		},
		Implementation: map[string]any{
			"topic_prefix":  plan.TopicPrefix.ValueString(),
			"kc_cluster_id": plan.KcClusterId.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating topic_router transform",
			fmt.Sprintf("Unable to update topic_router transform, got error: %s", err),
		)
		return
	}

	plan.Name = types.StringValue(transform.Name)
	plan.TransformType = types.StringValue(transform.TransformType)

	// Re-deploy after update
	err = r.client.DeployTransformLive(ctx, plan.ID.ValueString(), "0")
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Transform updated but deploy failed",
			fmt.Sprintf("Transform %s was updated but live deployment failed: %s. You may need to deploy manually.", plan.ID.ValueString(), err),
		)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TransformTopicRouterResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state TransformTopicRouterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTransform(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting topic_router transform",
			fmt.Sprintf("Unable to delete topic_router transform, got error: %s", err),
		)
		return
	}
}

func (r *TransformTopicRouterResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
