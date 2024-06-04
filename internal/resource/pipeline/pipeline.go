package pipeline

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ res.Resource                = &Pipeline{}
	_ res.ResourceWithConfigure   = &Pipeline{}
	_ res.ResourceWithImportState = &Pipeline{}
)

func NewPipelineResource() res.Resource {
	return &Pipeline{}
}

// Pipeline defines the res implementation.
type Pipeline struct {
	client api.StreamkapAPI
}

// PipelineModel describes the res data model.
type PipelineModel struct {
	ID                types.String              `tfsdk:"id"`
	Name              types.String              `tfsdk:"name"`
	SnapshotNewTables types.Bool                `tfsdk:"snapshot_new_tables"`
	Source            *PipelineSourceModel      `tfsdk:"source"`
	Destination       *PipelineDestinationModel `tfsdk:"destination"`
	Transforms        []types.String            `tfsdk:"transforms"`
}

type PipelineSourceModel struct {
	ID        types.String   `tfsdk:"id"`
	Name      types.String   `tfsdk:"name"`
	Connector types.String   `tfsdk:"connector"`
	Topics    []types.String `tfsdk:"topics"`
}

type PipelineDestinationModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Connector types.String `tfsdk:"connector"`
}

func (r *Pipeline) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

func (r *Pipeline) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	defaultEmptyList, diags := types.ListValue(
		types.StringType,
		[]attr.Value{},
	)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description:         "Pipeline resource",
		MarkdownDescription: "Pipeline resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Pipeline identifier",
				MarkdownDescription: "Pipeline identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Pipeline name",
				MarkdownDescription: "Pipeline name",
			},
			"snapshot_new_tables": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(true),
				Description:         "Whether to snapshot new tables (topics) or not",
				MarkdownDescription: "Whether to snapshot new tables (topics) or not",
			},
			"source": schema.ObjectAttribute{
				Required:            true,
				Description:         "Pipeline source",
				MarkdownDescription: "Pipeline source",
				AttributeTypes: map[string]attr.Type{
					"id":        types.StringType,
					"name":      types.StringType,
					"connector": types.StringType,
					"topics": types.ListType{
						ElemType: types.StringType,
					},
				},
			},
			"destination": schema.ObjectAttribute{
				Required:            true,
				Description:         "Pipeline destination",
				MarkdownDescription: "Pipeline destination",
				AttributeTypes: map[string]attr.Type{
					"id":        types.StringType,
					"name":      types.StringType,
					"connector": types.StringType,
				},
			},
			"transforms": schema.ListAttribute{
				Computed:    true,
				Optional:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(defaultEmptyList),
			},
		},
	}
}

func (r *Pipeline) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Pipeline Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *Pipeline) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan PipelineModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	pipeline, err := r.client.CreatePipeline(ctx, r.APIObjectFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating pipeline",
			fmt.Sprintf("Unable to create pipeline, got error: %s", err),
		)
		return
	}

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	r.modelFromAPIObject(*pipeline, &plan)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Pipeline) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state PipelineModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	pipelineID := state.ID.ValueString()
	pipeline, err := r.client.GetPipeline(ctx, pipelineID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading pipeline",
			fmt.Sprintf("Unable to read pipeline, got error: %s", err),
		)
		return
	}
	if pipeline == nil {
		resp.Diagnostics.AddError(
			"Error reading pipeline",
			fmt.Sprintf("Pipeline %s does not exist", pipelineID),
		)
		return
	}

	r.modelFromAPIObject(*pipeline, &state)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Pipeline) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan PipelineModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pipeline, err := r.client.UpdatePipeline(ctx, plan.ID.ValueString(), r.APIObjectFromModel(plan))

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating pipeline",
			fmt.Sprintf("Unable to update pipeline, got error: %s", err),
		)
		return
	}

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	r.modelFromAPIObject(*pipeline, &plan)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Pipeline) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state PipelineModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePipeline(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting pipeline",
			fmt.Sprintf("Unable to delete pipeline, got error: %s", err),
		)
		return
	}
}

func (r *Pipeline) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *Pipeline) APIObjectFromModel(model PipelineModel) api.Pipeline {
	sourceTopics := []string{}
	for _, topic := range model.Source.Topics {
		sourceTopics = append(sourceTopics, topic.ValueString())
	}

	transforms := []string{}
	for _, transform := range model.Transforms {
		transforms = append(transforms, transform.ValueString())
	}

	return api.Pipeline{
		Name:              model.Name.ValueString(),
		SnapshotNewTables: model.SnapshotNewTables.ValueBool(),
		Destination: api.PipelineDestination{
			ID:        model.Destination.ID.ValueString(),
			Name:      model.Destination.Name.ValueString(),
			Connector: model.Destination.Connector.ValueString(),
		},
		Source: api.PipelineSource{
			ID:        model.Source.ID.ValueString(),
			Name:      model.Source.Name.ValueString(),
			Connector: model.Source.Connector.ValueString(),
			Topics:    sourceTopics,
		},
		Transforms: transforms,
	}
}

func (r *Pipeline) modelFromAPIObject(apiObject api.Pipeline, model *PipelineModel) {
	// Copy the API Object to the model
	model.ID = types.StringValue(apiObject.ID)
	model.Name = types.StringValue(apiObject.Name)

	model.SnapshotNewTables = types.BoolValue(apiObject.SnapshotNewTables)

	sourceTopics := []types.String{}
	for _, topic := range apiObject.Source.Topics {
		sourceTopics = append(sourceTopics, types.StringValue(topic))
	}
	model.Source = &PipelineSourceModel{
		ID:        types.StringValue(apiObject.Source.ID),
		Name:      types.StringValue(apiObject.Source.Name),
		Connector: types.StringValue(apiObject.Source.Connector),
		Topics:    sourceTopics,
	}

	model.Destination = &PipelineDestinationModel{
		ID:        types.StringValue(apiObject.Destination.ID),
		Name:      types.StringValue(apiObject.Destination.Name),
		Connector: types.StringValue(apiObject.Destination.Connector),
	}

	transforms := []types.String{}
	for _, transform := range apiObject.Transforms {
		transforms = append(transforms, types.StringValue(transform))
	}
	model.Transforms = transforms
}
