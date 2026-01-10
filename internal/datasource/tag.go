package datasource

import (
	"context"
	"fmt"

	ds "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ ds.DataSource = &TagDataSource{}

func NewTagDataSource() ds.DataSource {
	return &TagDataSource{}
}

// TagDataSource defines the data source implementation.
type TagDataSource struct {
	client api.StreamkapAPI
}

// TagDataSourceModel describes the data source data model.
type TagDataSourceModel struct {
	ID          types.String   `tfsdk:"id"`
	Name        types.String   `tfsdk:"name"`
	Type        []types.String `tfsdk:"type"`
	Description types.String   `tfsdk:"description"`
	System      types.Bool     `tfsdk:"system"`
	Custom      types.Bool     `tfsdk:"custom"`
}

func (d *TagDataSource) Metadata(ctx context.Context, req ds.MetadataRequest, resp *ds.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag"
}

func (d *TagDataSource) Schema(ctx context.Context, req ds.SchemaRequest, resp *ds.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves information about a Streamkap tag by ID.",
		MarkdownDescription: "Retrieves information about a **Streamkap tag** by ID.\n\n" +
			"Use this data source to look up tag details for use in other resources.\n\n" +
			"[Documentation](https://docs.streamkap.com/streamkap-provider-for-terraform)",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The unique identifier of the tag to retrieve. Example: '670e5ca40afe1d3983ce0c22' for Development, '670e5bab0d119c0d1f8cda9d' for Production.",
				MarkdownDescription: "The unique identifier of the tag to retrieve. Example: `670e5ca40afe1d3983ce0c22` for Development, `670e5bab0d119c0d1f8cda9d` for Production.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "Display name of the tag. Example: 'Development' or 'Production'.",
				MarkdownDescription: "Display name of the tag. Example: `Development` or `Production`.",
				Computed:            true,
			},
			"type": schema.ListAttribute{
				Description:         "List of entity types this tag can be applied to (e.g., 'source', 'destination', 'pipeline').",
				MarkdownDescription: "List of entity types this tag can be applied to (e.g., `source`, `destination`, `pipeline`).",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"description": schema.StringAttribute{
				Description:         "Human-readable description of the tag's purpose.",
				MarkdownDescription: "Human-readable description of the tag's purpose.",
				Computed:            true,
			},
			"system": schema.BoolAttribute{
				Description:         "Whether this is a built-in system tag (true) or user-created (false).",
				MarkdownDescription: "Whether this is a built-in system tag (`true`) or user-created (`false`).",
				Computed:            true,
			},
			"custom": schema.BoolAttribute{
				Description:         "Whether this is a custom user-defined tag (true) or system-provided (false).",
				MarkdownDescription: "Whether this is a custom user-defined tag (`true`) or system-provided (`false`).",
				Computed:            true,
			},
		},
	}
}

func (d *TagDataSource) Configure(ctx context.Context, req ds.ConfigureRequest, resp *ds.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(api.StreamkapAPI)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Tag Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *TagDataSource) Read(ctx context.Context, req ds.ReadRequest, resp *ds.ReadResponse) {
	var state TagDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	TagID := state.ID.ValueString()
	Tag, err := d.client.GetTag(ctx, TagID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Tag",
			fmt.Sprintf("Unable to read Tag, got error: %s", err),
		)
		return
	}
	if Tag == nil {
		resp.Diagnostics.AddError(
			"Error reading Tag",
			fmt.Sprintf("Tag %s does not exist", TagID),
		)
		return
	}

	d.modelFromAPIObject(*Tag, &state)
	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Helpers
func (r *TagDataSource) modelFromAPIObject(apiObject api.Tag, model *TagDataSourceModel) {
	// Copy the API Object to the model
	model.ID = types.StringValue(apiObject.ID)
	model.Name = types.StringValue(apiObject.Name)

	tagTypes := []types.String{}
	for _, t := range apiObject.Type {
		tagTypes = append(tagTypes, types.StringValue(t))
	}

	model.Type = tagTypes
	model.Description = types.StringValue(apiObject.Description)
	model.System = types.BoolValue(apiObject.System)
	model.Custom = types.BoolPointerValue(apiObject.Custom)
}
