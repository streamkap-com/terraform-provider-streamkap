package tag

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

var (
	_ res.Resource                = &TagResource{}
	_ res.ResourceWithConfigure   = &TagResource{}
	_ res.ResourceWithImportState = &TagResource{}
)

func NewTagResource() res.Resource {
	return &TagResource{}
}

type TagResource struct {
	client api.StreamkapAPI
}

type TagResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	Type        []types.String `tfsdk:"type"`
	System      types.Bool     `tfsdk:"system"`
	Custom      types.Bool     `tfsdk:"custom"`
}

func (r *TagResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (r *TagResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Tag resource for organizing Streamkap entities",
		MarkdownDescription: "Tag resource for organizing Streamkap entities",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Tag identifier",
				MarkdownDescription: "Tag identifier",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "Tag name",
				MarkdownDescription: "Tag name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				Description:         "Tag description",
				MarkdownDescription: "Tag description",
				Optional:            true,
			},
			"type": schema.ListAttribute{
				Description:         "List of entity types this tag applies to",
				MarkdownDescription: "List of entity types this tag applies to (e.g., sources, destinations, pipelines)",
				Required:            true,
				ElementType:         types.StringType,
			},
			"system": schema.BoolAttribute{
				Description:         "Is the tag a system tag (read-only)",
				MarkdownDescription: "Is the tag a system tag (read-only)",
				Computed:            true,
			},
			"custom": schema.BoolAttribute{
				Description:         "Is the tag a custom tag (read-only)",
				MarkdownDescription: "Is the tag a custom tag (read-only)",
				Computed:            true,
			},
		},
	}
}

func (r *TagResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Tag Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T.", req.ProviderData),
		)
		return
	}
	r.client = client
}

func (r *TagResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan TagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagTypes := make([]string, len(plan.Type))
	for i, t := range plan.Type {
		tagTypes[i] = t.ValueString()
	}

	tag, err := r.client.CreateTag(ctx, api.Tag{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Type:        tagTypes,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating tag", fmt.Sprintf("Unable to create tag: %s", err))
		return
	}

	r.modelFromAPIObject(*tag, &plan)
	tflog.Info(ctx, "Created tag: "+tag.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *TagResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state TagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag, err := r.client.GetTag(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading tag", fmt.Sprintf("Unable to read tag: %s", err))
		return
	}
	if tag == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.modelFromAPIObject(*tag, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TagResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan TagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagTypes := make([]string, len(plan.Type))
	for i, t := range plan.Type {
		tagTypes[i] = t.ValueString()
	}

	tag, err := r.client.UpdateTag(ctx, plan.ID.ValueString(), api.Tag{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Type:        tagTypes,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating tag", fmt.Sprintf("Unable to update tag: %s", err))
		return
	}

	r.modelFromAPIObject(*tag, &plan)
	tflog.Info(ctx, "Updated tag: "+tag.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *TagResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state TagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTag(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting tag", fmt.Sprintf("Unable to delete tag: %s", err))
		return
	}
	tflog.Info(ctx, "Deleted tag: "+state.ID.ValueString())
}

func (r *TagResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *TagResource) modelFromAPIObject(apiObject api.Tag, model *TagResourceModel) {
	model.ID = types.StringValue(apiObject.ID)
	model.Name = types.StringValue(apiObject.Name)
	model.Description = types.StringValue(apiObject.Description)

	tagTypes := make([]types.String, len(apiObject.Type))
	for i, t := range apiObject.Type {
		tagTypes[i] = types.StringValue(t)
	}
	model.Type = tagTypes

	model.System = types.BoolValue(apiObject.System)
	model.Custom = types.BoolPointerValue(apiObject.Custom)
}
