package destination

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
	_ res.Resource                = &DestinationKafkaResource{}
	_ res.ResourceWithConfigure   = &DestinationKafkaResource{}
	_ res.ResourceWithImportState = &DestinationKafkaResource{}
)

func NewDestinationKafkaResource() res.Resource {
	return &DestinationKafkaResource{connector_code: "kafka"}
}

// DestinationKafkaResource defines the resource implementation.
type DestinationKafkaResource struct {
	client         api.StreamkapAPI
	connector_code string
}

// DestinationKafkaResourceModel describes the resource data model.
type DestinationKafkaResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Connector          types.String `tfsdk:"connector"`
	KafkaSinkBootstrap types.String `tfsdk:"kafka_sink_bootstrap"`
	Format             types.String `tfsdk:"destination_format"`
	JsonSchemaEnable   types.Bool   `tfsdk:"json_schema_enable"`
	SchemaRegistryUrl  types.String `tfsdk:"schema_registry_url"`
	TopicPrefix        types.String `tfsdk:"topic_prefix"`
	TopicSuffix        types.String `tfsdk:"topic_suffix"`
}

func (r *DestinationKafkaResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination_kafka"
}

func (r *DestinationKafkaResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Destination Kafka resource",
		MarkdownDescription: "Destination Kafka resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Destination Kafka identifier",
				MarkdownDescription: "Destination Kafka identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Destination name",
				MarkdownDescription: "Destination name",
			},
			"connector": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"kafka_sink_bootstrap": schema.StringAttribute{
				Required:            true,
				Description:         "A comma-separated list of host and port pairs that are the addresses of the Destination Kafka brokers. This list should be in the form host1:port1,host2:port2,...",
				MarkdownDescription: "A comma-separated list of host and port pairs that are the addresses of the Destination Kafka brokers. This list should be in the form host1:port1,host2:port2,...",
			},
			"destination_format": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("json"),
				Description:         "The format to use when writing data to kafka",
				MarkdownDescription: "The format to use when writing data to kafka",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"avro",
						"json",
					),
				},
			},
			"json_schema_enable": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Include schema in json message",
				MarkdownDescription: "Include schema in json message",
			},
			"schema_registry_url": schema.StringAttribute{
				Optional:            true,
				Description:         "Destination kafka schema registry url",
				MarkdownDescription: "Kafka Hostname Or IP address",
			},
			"topic_prefix": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Prefix for destination topics",
				MarkdownDescription: "Prefix for destination topics",
			},
			"topic_suffix": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Suffix for destination topics",
				MarkdownDescription: "Suffix for destination topics",
			},
		},
	}
}

func (r *DestinationKafkaResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Destination Kafka Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *DestinationKafkaResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan DestinationKafkaResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	plan.Connector = types.StringValue(r.connector_code)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Pre CREATE ===> plan: "+fmt.Sprintf("%+v", plan))
	config, err := r.model2ConfigMap(plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Kafka destination",
			fmt.Sprintf("Unable to create Kafka destination, got error: %s", err),
		)
		return
	}

	tflog.Debug(ctx, "Pre CREATE ===> config: "+fmt.Sprintf("%+v", config))
	destination, err := r.client.CreateDestination(ctx, api.Destination{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Kafka destination",
			fmt.Sprintf("Unable to create Kafka destination, got error: %s", err),
		)
		return
	}
	tflog.Debug(ctx, "Post CREATE ===> config: "+fmt.Sprintf("%+v", destination.Config))

	plan.ID = types.StringValue(destination.ID)
	plan.Name = types.StringValue(destination.Name)
	plan.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(destination.Config, &plan)
	tflog.Debug(ctx, "Post CREATE ===> plan: "+fmt.Sprintf("%+v", plan))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DestinationKafkaResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state DestinationKafkaResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := state.ID.ValueString()
	destination, err := r.client.GetDestination(ctx, destinationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Kafka destination",
			fmt.Sprintf("Unable to read Kafka destination, got error: %s", err),
		)
		return
	}
	if destination == nil {
		resp.Diagnostics.AddError(
			"Error reading Kafka destination",
			fmt.Sprintf("Kafka destination %s does not exist", destinationID),
		)
		return
	}

	state.Name = types.StringValue(destination.Name)
	state.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(destination.Config, &state)
	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", state))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DestinationKafkaResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan DestinationKafkaResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.model2ConfigMap(plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Kafka destination",
			fmt.Sprintf("Unable to update Kafka destination, got error: %s", err),
		)
		return
	}

	destination, err := r.client.UpdateDestination(ctx, plan.ID.ValueString(), api.Destination{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Kafka destination",
			fmt.Sprintf("Unable to update Kafka destination, got error: %s", err),
		)
		return
	}

	// Update resource state with updated items
	plan.Name = types.StringValue(destination.Name)
	plan.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(destination.Config, &plan)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DestinationKafkaResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state DestinationKafkaResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDestination(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Kafka destination",
			fmt.Sprintf("Unable to delete Kafka destination, got error: %s", err),
		)
		return
	}
}

func (r *DestinationKafkaResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *DestinationKafkaResource) model2ConfigMap(model DestinationKafkaResourceModel) (map[string]any, error) {

	return map[string]any{
		"kafka.sink.bootstrap":             model.KafkaSinkBootstrap.ValueString(),
		"destination.format":               model.Format.ValueString(),
		"json.schema.enable":               model.JsonSchemaEnable.ValueBool(),
		"schema.registry.url.user.defined": model.SchemaRegistryUrl.ValueString(),
		"topic.prefix":                     model.TopicPrefix.ValueString(),
		"topic.suffix":                     model.TopicSuffix.ValueString(),
	}, nil
}

func (r *DestinationKafkaResource) configMap2Model(cfg map[string]any, model *DestinationKafkaResourceModel) {
	// Copy the config map to the model
	model.KafkaSinkBootstrap = helper.GetTfCfgString(cfg, "kafka.sink.bootstrap")
	model.Format = helper.GetTfCfgString(cfg, "destination.format")
	model.JsonSchemaEnable = helper.GetTfCfgBool(cfg, "json.schema.enable")
	model.SchemaRegistryUrl = helper.GetTfCfgString(cfg, "schema.registry.url.user.defined")
	model.TopicPrefix = helper.GetTfCfgString(cfg, "topic.prefix")
	model.TopicSuffix = helper.GetTfCfgString(cfg, "topic.suffix")
}
