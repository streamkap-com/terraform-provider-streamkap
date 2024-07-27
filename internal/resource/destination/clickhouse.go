package destination

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
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
	_ res.Resource                = &DestinationClickHouseResource{}
	_ res.ResourceWithConfigure   = &DestinationClickHouseResource{}
	_ res.ResourceWithImportState = &DestinationClickHouseResource{}
)

func NewDestinationClickHouseResource() res.Resource {
	return &DestinationClickHouseResource{connector_code: "clickhouse"}
}

// DestinationClickHouseResource defines the resource implementation.
type DestinationClickHouseResource struct {
	client         api.StreamkapAPI
	connector_code string
}

// DestinationClickHouseResourceModel describes the resource data model.
type DestinationClickHouseResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Connector          types.String `tfsdk:"connector"`
	IngestionMode      types.String `tfsdk:"ingestion_mode"`
	TasksMax           types.Int64  `tfsdk:"tasks_max"`
	Hostname           types.String `tfsdk:"hostname"`
	ConnectionUsername types.String `tfsdk:"connection_username"`
	ConnectionPassword types.String `tfsdk:"connection_password"`
	Port               types.String `tfsdk:"port"`
	Database           types.String `tfsdk:"database"`
	SSL                types.Bool   `tfsdk:"ssl"`
}

func (r *DestinationClickHouseResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination_clickhouse"
}

func (r *DestinationClickHouseResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Destination ClickHouse resource",
		MarkdownDescription: "Destination ClickHouse resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Destination ClickHouse identifier",
				MarkdownDescription: "Destination ClickHouse identifier",
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
			"ingestion_mode": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("upsert"),
				Description:         "Upsert or append modes are available",
				MarkdownDescription: "Upsert or append modes are available",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"upsert",
						"append",
					),
				},
			},
			"tasks_max": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(5),
				Description:         "The maximum number of active task",
				MarkdownDescription: "The maximum number of active task",
			},
			"hostname": schema.StringAttribute{
				Required:            true,
				Description:         "ClickHouse Hostname Or IP address",
				MarkdownDescription: "ClickHouse Hostname Or IP address",
			},
			"connection_username": schema.StringAttribute{
				Required:            true,
				Description:         "Username to access ClickHouse",
				MarkdownDescription: "Username to access ClickHouse",
			},
			"connection_password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "Password to access the ClickHouse",
				MarkdownDescription: "Password to access the ClickHouse",
			},
			"port": schema.StringAttribute{
				Required:            true,
				Description:         "ClickHouse Port. For example, 8443",
				MarkdownDescription: "ClickHouse Port. For example, 8443",
			},
			"database": schema.StringAttribute{
				Optional:            true,
				Description:         "ClickHouse database",
				MarkdownDescription: "ClickHouse database",
			},
			"ssl": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				Description:         "Enable TLS for network connections",
				MarkdownDescription: "Enable TLS for network connections",
			},
		},
	}
}

func (r *DestinationClickHouseResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Destination ClickHouse Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *DestinationClickHouseResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan DestinationClickHouseResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	plan.Connector = types.StringValue(r.connector_code)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", plan))
	config := r.model2ConfigMap(plan)

	destination, err := r.client.CreateDestination(ctx, api.Destination{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ClickHouse destination",
			fmt.Sprintf("Unable to create ClickHouse destination, got error: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(destination.ID)
	plan.Name = types.StringValue(destination.Name)
	plan.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(destination.Config, &plan)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DestinationClickHouseResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state DestinationClickHouseResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := state.ID.ValueString()
	destination, err := r.client.GetDestination(ctx, destinationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading ClickHouse destination",
			fmt.Sprintf("Unable to read ClickHouse destination, got error: %s", err),
		)
		return
	}
	if destination == nil {
		resp.Diagnostics.AddError(
			"Error reading ClickHouse destination",
			fmt.Sprintf("ClickHouse destination %s does not exist", destinationID),
		)
		return
	}

	state.Name = types.StringValue(destination.Name)
	state.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(destination.Config, &state)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DestinationClickHouseResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan DestinationClickHouseResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", plan))
	config := r.model2ConfigMap(plan)

	destination, err := r.client.UpdateDestination(ctx, plan.ID.ValueString(), api.Destination{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating ClickHouse destination",
			fmt.Sprintf("Unable to update ClickHouse destination, got error: %s", err),
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

func (r *DestinationClickHouseResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state DestinationClickHouseResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDestination(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting ClickHouse destination",
			fmt.Sprintf("Unable to delete ClickHouse destination, got error: %s", err),
		)
		return
	}
}

func (r *DestinationClickHouseResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *DestinationClickHouseResource) model2ConfigMap(model DestinationClickHouseResourceModel) map[string]any {
	return map[string]any{
		"ingestion.mode":      model.IngestionMode.ValueString(),
		"tasks.max":           model.TasksMax.ValueInt64(),
		"hostname":            model.Hostname.ValueString(),
		"connection.username": model.ConnectionUsername.ValueString(),
		"connection.password": model.ConnectionPassword.ValueString(),
		"port":                model.Port.ValueString(),
		"database":            model.Database.ValueStringPointer(),
		"ssl":                 model.SSL.ValueBool(),
	}
}

func (r *DestinationClickHouseResource) configMap2Model(cfg map[string]any, model *DestinationClickHouseResourceModel) {
	// Copy the config map to the model
	model.IngestionMode = helper.GetTfCfgString(cfg, "ingestion.mode")
	model.TasksMax = helper.GetTfCfgInt64(cfg, "tasks.max")
	model.Hostname = helper.GetTfCfgString(cfg, "hostname")
	model.ConnectionUsername = helper.GetTfCfgString(cfg, "connection.username")
	model.ConnectionPassword = helper.GetTfCfgString(cfg, "connection.password")
	model.Port = helper.GetTfCfgString(cfg, "port")
	model.Database = helper.GetTfCfgString(cfg, "database")
	model.SSL = helper.GetTfCfgBool(cfg, "ssl")
}
