package destination

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	_ res.Resource                = &DestinationPostgresqlResource{}
	_ res.ResourceWithConfigure   = &DestinationPostgresqlResource{}
	_ res.ResourceWithImportState = &DestinationPostgresqlResource{}
)

func NewDestinationPostgresqlResource() res.Resource {
	return &DestinationPostgresqlResource{connector_code: "postgresql"}
}

// DestinationPostgresqlResource defines the resource implementation.
type DestinationPostgresqlResource struct {
	client         api.StreamkapAPI
	connector_code string
}

// DestinationPostgresqlResourceModel describes the resource data model.
type DestinationPostgresqlResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Connector          types.String `tfsdk:"connector"`
	DatabaseHostname   types.String `tfsdk:"database_hostname"`
	DatabasePort       types.Int64  `tfsdk:"database_port"`
	DatabaseDbname     types.String `tfsdk:"database_dbname"`
	DatabaseUsername   types.String `tfsdk:"database_username"`
	DatabasePassword   types.String `tfsdk:"database_password"`
	DatabaseSchemaName types.String `tfsdk:"database_schema_name"`
	SchemaEvolution    types.String `tfsdk:"schema_evolution"`
	InsertMode         types.String `tfsdk:"insert_mode"`
	HardDelete         types.Bool   `tfsdk:"hard_delete"`
	PrimaryKeyMode     types.String `tfsdk:"primary_key_mode"`
	CustomPrimaryKey   types.String `tfsdk:"custom_primary_key"`
	TasksMax           types.Int64  `tfsdk:"tasks_max"`
	SSHEnabled         types.Bool   `tfsdk:"ssh_enabled"`
	SSHHost            types.String `tfsdk:"ssh_host"`
	SSHPort            types.String `tfsdk:"ssh_port"`
	SSHUser            types.String `tfsdk:"ssh_user"`
}

func (r *DestinationPostgresqlResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination_postgresql"
}

func (r *DestinationPostgresqlResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Destination Postgresql resource",
		MarkdownDescription: "Destination Postgresql resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Destination Postgresql identifier",
				MarkdownDescription: "Destination Postgresql identifier",
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
			"database_hostname": schema.StringAttribute{
				Required:            true,
				Description:         "PostgreSQL Hostname. For example, postgres.something.rds.amazonaws.com",
				MarkdownDescription: "PostgreSQL Hostname. For example, postgres.something.rds.amazonaws.com",
			},
			"database_port": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(5432),
				Description:         "PostgreSQL Port. For example, 5432",
				MarkdownDescription: "PostgreSQL Port. For example, 5432",
			},
			"database_dbname": schema.StringAttribute{
				Required:            true,
				Description:         "Database name",
				MarkdownDescription: "Database name",
			},
			"database_username": schema.StringAttribute{
				Required:            true,
				Description:         "Username to access Postgresql",
				MarkdownDescription: "Username to access Postgresql",
			},
			"database_password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "Password to access Postgresql",
				MarkdownDescription: "Password to access Postgresql",
			},
			"database_schema_name": schema.StringAttribute{
				Required:            true,
				Description:         "Schema for the associated table name",
				MarkdownDescription: "Schema for the associated table name",
			},
			"schema_evolution": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("basic"),
				Description:         "Controls how schema evolution is handled by the sink connector. For pipelines with pre-created destination tables, set to `none`",
				MarkdownDescription: "Controls how schema evolution is handled by the sink connector. For pipelines with pre-created destination tables, set to `none`",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"basic",
						"none",
					),
				},
			},
			"insert_mode": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("insert"),
				Description:         "Insert or upsert modes are available",
				MarkdownDescription: "Insert or upsert modes are available",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"insert",
						"upsert",
					),
				},
			},
			"hard_delete": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Specifies whether the connector processes DELETE or tombstone events and removes the corresponding row from the database",
				MarkdownDescription: "Specifies whether the connector processes DELETE or tombstone events and removes the corresponding row from the database",
			},
			"primary_key_mode": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("record_key"),
				Description:         "Specifies how the connector resolves the primary key columns from the event",
				MarkdownDescription: "Specifies how the connector resolves the primary key columns from the event",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"record_key",
						"record_value",
						"none",
					),
				},
			},
			"custom_primary_key": schema.StringAttribute{
				Optional:            true,
				Description:         "Either the name of the primary key column or a comma-separated list of fields to derive the primary key from.",
				MarkdownDescription: "Either the name of the primary key column or a comma-separated list of fields to derive the primary key from.",
			},

			"tasks_max": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(5),
				Description:         "The maximum number of active task",
				MarkdownDescription: "The maximum number of active task",
				Validators: []validator.Int64{
					int64validator.Between(1, 10),
				},
			},
			"ssh_enabled": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Connect via SSH tunnel",
				MarkdownDescription: "Connect via SSH tunnel",
			},
			"ssh_host": schema.StringAttribute{
				Optional:            true,
				Description:         "Hostname of the SSH server, only required if `ssh_enabled` is true",
				MarkdownDescription: "Hostname of the SSH server, only required if `ssh_enabled` is true",
			},
			"ssh_port": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("22"),
				Description:         "Port of the SSH server, only required if `ssh_enabled` is true",
				MarkdownDescription: "Port of the SSH server, only required if `ssh_enabled` is true",
			},
			"ssh_user": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("streamkap"),
				Description:         "User for connecting to the SSH server, only required if `ssh_enabled` is true",
				MarkdownDescription: "User for connecting to the SSH server, only required if `ssh_enabled` is true",
			},
		},
	}
}

func (r *DestinationPostgresqlResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Destination Postgresql Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *DestinationPostgresqlResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan DestinationPostgresqlResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	plan.Connector = types.StringValue(r.connector_code)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Pre CREATE ===> plan: "+fmt.Sprintf("%+v", plan))
	config := r.model2ConfigMap(plan)

	tflog.Debug(ctx, "Pre CREATE ===> config: "+fmt.Sprintf("%+v", config))
	destination, err := r.client.CreateDestination(ctx, api.Destination{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Postgresql destination",
			fmt.Sprintf("Unable to create Postgresql destination, got error: %s", err),
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

func (r *DestinationPostgresqlResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state DestinationPostgresqlResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := state.ID.ValueString()
	destination, err := r.client.GetDestination(ctx, destinationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Postgresql destination",
			fmt.Sprintf("Unable to read Postgresql destination, got error: %s", err),
		)
		return
	}
	if destination == nil {
		resp.Diagnostics.AddError(
			"Error reading Postgresql destination",
			fmt.Sprintf("Postgresql destination %s does not exist", destinationID),
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

func (r *DestinationPostgresqlResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan DestinationPostgresqlResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	config := r.model2ConfigMap(plan)

	destination, err := r.client.UpdateDestination(ctx, plan.ID.ValueString(), api.Destination{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Postgresql destination",
			fmt.Sprintf("Unable to update Postgresql destination, got error: %s", err),
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

func (r *DestinationPostgresqlResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state DestinationPostgresqlResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDestination(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Postgresql destination",
			fmt.Sprintf("Unable to delete Postgresql destination, got error: %s", err),
		)
		return
	}
}

func (r *DestinationPostgresqlResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *DestinationPostgresqlResource) model2ConfigMap(model DestinationPostgresqlResourceModel) map[string]any {
	configMap := map[string]any{
		"database.hostname.user.defined": model.DatabaseHostname.ValueString(),
		"database.port.user.defined":     int(model.DatabasePort.ValueInt64()), //? need to convert to 32???
		"database.database.user.defined": model.DatabaseDbname.ValueString(),
		"connection.username":            model.DatabaseUsername.ValueString(),
		"connection.password":            model.DatabasePassword.ValueString(),
		"table.name.prefix":              model.DatabaseSchemaName.ValueString(),
		"schema.evolution":               model.SchemaEvolution.ValueString(),
		"insert.mode":                    model.InsertMode.ValueString(),
		"delete.enabled":                 model.HardDelete.ValueBool(),
		"primary.key.mode":               model.PrimaryKeyMode.ValueString(),
		"primary.key.fields":             model.CustomPrimaryKey.ValueStringPointer(),
		"tasks.max":                      model.TasksMax.ValueInt64(),
		"ssh.enabled":                    model.SSHEnabled.ValueBool(),
		"ssh.host":                       model.SSHHost.ValueStringPointer(),
		"ssh.port":                       model.SSHPort.ValueString(),
		"ssh.user":                       model.SSHUser.ValueString(),
	}

	return configMap
}

func (r *DestinationPostgresqlResource) configMap2Model(cfg map[string]any, model *DestinationPostgresqlResourceModel) {
	// Copy the config map to the model
	model.DatabaseHostname = helper.GetTfCfgString(cfg, "database.hostname.user.defined")
	model.DatabasePort = helper.GetTfCfgInt64(cfg, "database.port.user.defined")
	model.DatabaseDbname = helper.GetTfCfgString(cfg, "database.database.user.defined")
	model.DatabaseUsername = helper.GetTfCfgString(cfg, "connection.username")
	model.DatabasePassword = helper.GetTfCfgString(cfg, "connection.password")
	model.DatabaseSchemaName = helper.GetTfCfgString(cfg, "table.name.prefix")
	model.SchemaEvolution = helper.GetTfCfgString(cfg, "schema.evolution")
	model.InsertMode = helper.GetTfCfgString(cfg, "insert.mode")
	model.HardDelete = helper.GetTfCfgBool(cfg, "delete.enabled")
	model.PrimaryKeyMode = helper.GetTfCfgString(cfg, "primary.key.mode")
	model.CustomPrimaryKey = helper.GetTfCfgString(cfg, "primary.key.fields")
	model.TasksMax = helper.GetTfCfgInt64(cfg, "tasks.max")
	model.SSHEnabled = helper.GetTfCfgBool(cfg, "ssh.enabled")
	model.SSHHost = helper.GetTfCfgString(cfg, "ssh.host")
	model.SSHPort = helper.GetTfCfgString(cfg, "ssh.port")
	model.SSHUser = helper.GetTfCfgString(cfg, "ssh.user")
}
