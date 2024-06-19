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
var _ ds.DataSource = &TransformDataSource{}

func NewTransformDataSource() ds.DataSource {
	return &TransformDataSource{}
}

// TransformDataSource defines the data source implementation.
type TransformDataSource struct {
	client api.StreamkapAPI
}

// TransformDataSourceModel describes the data source data model.
type TransformDataSourceModel struct {
	ID        types.String   `tfsdk:"id"`
	Name      types.String   `tfsdk:"name"`
	StartTime types.String   `tfsdk:"start_time"`
	TopicIDs  []types.String `tfsdk:"topic_ids"`
	TopicMap  []TopicModel   `tfsdk:"topic_map"`
}

type TopicModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *TransformDataSource) Metadata(ctx context.Context, req ds.MetadataRequest, resp *ds.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transform"
}

func (d *TransformDataSource) Schema(ctx context.Context, req ds.SchemaRequest, resp *ds.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Tranform data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "Transform identifier",
				MarkdownDescription: "Transform identifier",
				Required:            true,
			},
			"name": schema.StringAttribute{
				Description:         "Transform name",
				MarkdownDescription: "Transform name",
				Computed:            true,
			},
			"start_time": schema.StringAttribute{
				Description:         "Start time",
				MarkdownDescription: "Start time",
				Computed:            true,
			},
			"topic_ids": schema.ListAttribute{
				Description:         "List of topic identifiers",
				MarkdownDescription: "List of topic identifiers",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"topic_map": schema.ListNestedAttribute{
				Computed:            true,
				Description:         "List of topic object, with id and name for each topic",
				MarkdownDescription: "List of topic object, with id and name for each topic",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"name": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *TransformDataSource) Configure(ctx context.Context, req ds.ConfigureRequest, resp *ds.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(api.StreamkapAPI)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Transform Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *TransformDataSource) Read(ctx context.Context, req ds.ReadRequest, resp *ds.ReadResponse) {
	var state TransformDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	transformID := state.ID.ValueString()
	transform, err := d.client.GetTransform(ctx, transformID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading transform",
			fmt.Sprintf("Unable to read transform, got error: %s", err),
		)
		return
	}
	if transform == nil {
		resp.Diagnostics.AddError(
			"Error reading transform",
			fmt.Sprintf("transform %s does not exist", transformID),
		)
		return
	}

	d.modelFromAPIObject(*transform, &state)

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Helpers
func (r *TransformDataSource) modelFromAPIObject(apiObject api.Transform, model *TransformDataSourceModel) {
	// Copy the API Object to the model
	model.ID = types.StringValue(apiObject.ID)
	model.Name = types.StringValue(apiObject.Name)
	model.StartTime = types.StringPointerValue(apiObject.StartTime)

	topicIDs := []types.String{}
	for _, topicID := range apiObject.TopicIDs {
		topicIDs = append(topicIDs, types.StringValue(topicID))
	}
	model.TopicIDs = topicIDs

	topicMap := []TopicModel{}
	for i, topicID := range apiObject.TopicIDs {
		topicMap = append(topicMap, TopicModel{
			ID:   types.StringValue(topicID),
			Name: types.StringValue(apiObject.Topics[i]),
		})
	}
	model.TopicMap = topicMap
}
