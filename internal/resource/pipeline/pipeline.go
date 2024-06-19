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
	_ res.Resource                = &PipelineResource{}
	_ res.ResourceWithConfigure   = &PipelineResource{}
	_ res.ResourceWithImportState = &PipelineResource{}
)

func NewPipelineResource() res.Resource {
	return &PipelineResource{}
}

// PipelineResource defines the res implementation.
type PipelineResource struct {
	client api.StreamkapAPI
}

// PipelineResourceModel describes the res data model.
type PipelineResourceModel struct {
	ID                types.String              `tfsdk:"id"`
	Name              types.String              `tfsdk:"name"`
	SnapshotNewTables types.Bool                `tfsdk:"snapshot_new_tables"`
	Source            *PipelineSourceModel      `tfsdk:"source"`
	Destination       *PipelineDestinationModel `tfsdk:"destination"`
	Transforms        []*PipelineTransformModel `tfsdk:"transforms"`
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

type PipelineTransformModel struct {
	ID     types.String   `tfsdk:"id"`
	Topics []types.String `tfsdk:"topics"`
}

func (r *PipelineResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

func (r *PipelineResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	transformsNestedObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id": types.StringType,
			"topics": types.ListType{
				ElemType: types.StringType,
			},
		},
	}

	defaultEmptyList, diags := types.ListValue(
		transformsNestedObjectType,
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
			"transforms": schema.ListNestedAttribute{
				Computed:            true,
				Optional:            true,
				Description:         "Pipeline transforms",
				MarkdownDescription: "Pipeline transforms",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:            true,
							Description:         "Transform identifier",
							MarkdownDescription: "Transform identifier",
						},
						"topics": schema.ListAttribute{
							Required:            true,
							ElementType:         types.StringType,
							Description:         "List of transform topics' names",
							MarkdownDescription: "List of transform topics' names",
						},
					},
				},
				Default: listdefault.StaticValue(defaultEmptyList),
			},
		},
	}
}

func (r *PipelineResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
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

func (r *PipelineResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan PipelineResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	payload, err := r.apiObjectFromModel(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating pipeline",
			fmt.Sprintf("Unable to create pipeline, got error: %s", err),
		)
		return
	}

	pipeline, err := r.client.CreatePipeline(ctx, *payload)
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

func (r *PipelineResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state PipelineResourceModel

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

func (r *PipelineResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan PipelineResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	payload, err := r.apiObjectFromModel(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating pipeline",
			fmt.Sprintf("Unable to update pipeline, got error: %s", err),
		)
		return
	}

	pipeline, err := r.client.UpdatePipeline(ctx, plan.ID.ValueString(), *payload)

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

func (r *PipelineResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state PipelineResourceModel

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

func (r *PipelineResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *PipelineResource) idxStringInSlice(a string, list []string) int {
	for idx, b := range list {
		if b == a {
			return idx
		}
	}
	return -1
}

func (r *PipelineResource) model2APITransforms(ctx context.Context, modelTransforms []*PipelineTransformModel) (res []*api.PipelineTransform, err error) {
	for _, modelTransform := range modelTransforms {
		transformID := modelTransform.ID.ValueString()
		transform, err := r.client.GetTransform(ctx, transformID)
		if err != nil {
			return nil, err
		}
		if transform == nil {
			return nil, fmt.Errorf("transform %s does not exist", transformID)
		}

		for _, modelTopic := range modelTransform.Topics {
			modelTopicStr := modelTopic.ValueString()
			if topicIdx := r.idxStringInSlice(modelTopicStr, transform.Topics); topicIdx >= 0 {
				res = append(res, &api.PipelineTransform{
					ID:        transform.ID,
					Name:      transform.Name,
					StartTime: transform.StartTime,
					Topic:     modelTopicStr,
					TopicID:   transform.TopicIDs[topicIdx],
				})
			} else {
				return nil, fmt.Errorf("topic %s not found in transform %s", modelTopicStr, transformID)
			}
		}
	}

	return res, nil
}

func (r *PipelineResource) api2ModelTransforms(apiTransforms []*api.PipelineTransform) (modelTransforms []*PipelineTransformModel, err error) {
	// Loop through all api.Transforms and fetch unwinded transform data
	if len(apiTransforms) == 0 {
		return nil, nil
	}

	modelTransforms = make([]*PipelineTransformModel, 1, len(apiTransforms))
	j := 0
	modelTransforms[j] = &PipelineTransformModel{
		ID:     types.StringValue(apiTransforms[j].ID),
		Topics: []types.String{types.StringValue(apiTransforms[j].Topic)},
	}

	for i, apiTransform := range apiTransforms {
		if i < 1 {
			continue
		}

		if modelTransform := modelTransforms[j]; apiTransform.ID == modelTransform.ID.ValueString() {
			modelTransform.Topics = append(modelTransform.Topics, types.StringValue(apiTransform.Topic))
		} else {
			modelTransforms = append(modelTransforms, &PipelineTransformModel{
				ID:     types.StringValue(apiTransform.ID),
				Topics: []types.String{types.StringValue(apiTransform.Topic)},
			})
			j++
		}
	}

	return modelTransforms, nil
}

func (r *PipelineResource) apiObjectFromModel(ctx context.Context, model PipelineResourceModel) (res *api.Pipeline, err error) {
	sourceTopics := []string{}
	for _, topic := range model.Source.Topics {
		sourceTopics = append(sourceTopics, topic.ValueString())
	}

	// Loop through all model.Transforms and fetch unwinded transform data
	apiTransforms, err := r.model2APITransforms(ctx, model.Transforms)
	if err != nil {
		// Log error and continue to next transform
		fmt.Printf("Error enriching model Transforms: %s\n", err)
		return nil, err
	}

	res = &api.Pipeline{
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
		Transforms: apiTransforms,
	}

	return res, nil
}

func (r *PipelineResource) modelFromAPIObject(apiObject api.Pipeline, model *PipelineResourceModel) error {
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

	transforms, err := r.api2ModelTransforms(apiObject.Transforms)
	if err != nil {
		return err
	}
	model.Transforms = transforms

	return nil
}
