package source

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/helper"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ res.Resource                = &SourceKafkaDirectResource{}
	_ res.ResourceWithConfigure   = &SourceKafkaDirectResource{}
	_ res.ResourceWithImportState = &SourceKafkaDirectResource{}
)

func NewSourceKafkaDirectResource() res.Resource {
	return &SourceKafkaDirectResource{connector_code: "kafkadirect"}
}

// SourceKafkaDirectResource defines the resource implementation.
type SourceKafkaDirectResource struct {
	client         api.StreamkapAPI
	connector_code string
}

// SourceKafkaDirectResourceModel describes the resource data model.
type SourceKafkaDirectResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Connector        types.String `tfsdk:"connector"`
	TopicPrefix      types.String `tfsdk:"topic_prefix"`
	KafkaFormat      types.String `tfsdk:"kafka_format"`
	SchemasEnable    types.Bool   `tfsdk:"schemas_enable"`
	TopicIncludeList types.String `tfsdk:"topic_include_list"`
}

func (r *SourceKafkaDirectResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_kafkadirect"
}

func (r *SourceKafkaDirectResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Source Kafka Direct resource",
		MarkdownDescription: "Source Kafka Direct resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Source Kafka Direct identifier",
				MarkdownDescription: "Source Kafka Direct identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Source name",
				MarkdownDescription: "Source name",
			},
			"connector": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"topic_prefix": schema.StringAttribute{
				Required:            true,
				Description:         "Prefix for the topic",
				MarkdownDescription: "Prefix for the topic",
			},
			"kafka_format": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("string"),
				Description:         "The serialised format of the data written to the Kafka topic",
				MarkdownDescription: "The serialised format of the data written to the Kafka topic",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"string",
						"json",
					),
				},
			},
			"schemas_enable": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "If untoggled (default), Streamkap attempts to infer schema from your data - depending on the Destination. Otherwise, Streamkap assumes the Kafka message key and value contain `schema` and `payload` structures",
				MarkdownDescription: "If untoggled (default), Streamkap attempts to infer schema from your data - depending on the Destination. Otherwise, Streamkap assumes the Kafka message key and value contain `schema` and `payload` structures",
			},
			"topic_include_list": schema.StringAttribute{
				Required:            true,
				Description:         "Topics to sync",
				MarkdownDescription: "Topics to sync",
			},
		},
	}
}

func (r *SourceKafkaDirectResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Source Kafka Direct Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *SourceKafkaDirectResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan SourceKafkaDirectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	plan.Connector = types.StringValue(r.connector_code)

	if resp.Diagnostics.HasError() {
		return
	}

	config := r.model2ConfigMap(plan)

	source, err := r.client.CreateSource(ctx, api.Source{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Kafka Direct source",
			fmt.Sprintf("Unable to create Kafka Direct source, got error: %s", err),
		)
		return
	}
	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", source))
	plan.ID = types.StringValue(source.ID)
	plan.Name = types.StringValue(source.Name)
	plan.Connector = types.StringValue(source.Connector)
	r.configMap2Model(source.Config, &plan)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SourceKafkaDirectResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state SourceKafkaDirectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := state.ID.ValueString()
	source, err := r.client.GetSource(ctx, sourceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Kafka Direct source",
			fmt.Sprintf("Unable to read Kafka Direct source, got error: %s", err),
		)
		return
	}
	if source == nil {
		resp.Diagnostics.AddError(
			"Error reading Kafka Direct source",
			fmt.Sprintf("Kafka Direct source %s does not exist", sourceID),
		)
		return
	}

	state.Name = types.StringValue(source.Name)
	state.Connector = types.StringValue(source.Connector)
	r.configMap2Model(source.Config, &state)
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SourceKafkaDirectResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan SourceKafkaDirectResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", plan))
	config := r.model2ConfigMap(plan)

	source, err := r.client.UpdateSource(ctx, plan.ID.ValueString(), api.Source{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Kafka Direct source",
			fmt.Sprintf("Unable to update Kafka Direct source, got error: %s", err),
		)
		return
	}

	// Update resource state with updated items
	plan.Name = types.StringValue(source.Name)
	plan.Connector = types.StringValue(source.Connector)
	r.configMap2Model(source.Config, &plan)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SourceKafkaDirectResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state SourceKafkaDirectResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Kafka Direct source",
			fmt.Sprintf("Unable to delete Kafka Direct source, got error: %s", err),
		)
		return
	}
}

func (r *SourceKafkaDirectResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *SourceKafkaDirectResource) model2ConfigMap(model SourceKafkaDirectResourceModel) map[string]any {
	return map[string]any{
		"topic.prefix":                    model.TopicPrefix.ValueString(),
		"format":                          model.KafkaFormat.ValueString(),
		"topic.include.list.user.defined": model.TopicIncludeList.ValueString(),
		"schemas.enable":                  model.SchemasEnable.ValueBool(),
	}
}

func (r *SourceKafkaDirectResource) configMap2Model(cfg map[string]any, model *SourceKafkaDirectResourceModel) {
	// Copy the config map to the model
	model.TopicPrefix = helper.GetTfCfgString(cfg, "topic.prefix")
	model.KafkaFormat = helper.GetTfCfgString(cfg, "format")
	model.TopicIncludeList = helper.GetTfCfgString(cfg, "topic.include.list.user.defined")
	model.SchemasEnable = helper.GetTfCfgBool(cfg, "schemas.enable")
}
