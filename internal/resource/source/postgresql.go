package source

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
	_ res.Resource                = &SourcePostgreSQL{}
	_ res.ResourceWithConfigure   = &SourcePostgreSQL{}
	_ res.ResourceWithImportState = &SourcePostgreSQL{}
)

func NewSourcePostgreSQLResource() res.Resource {
	return &SourcePostgreSQL{connector_code: "postgresql"}
}

// SourcePostgreSQL defines the resource implementation.
type SourcePostgreSQL struct {
	client         api.StreamkapAPI
	connector_code string
}

// SourcePostgreSQLModel describes the resource data model.
type SourcePostgreSQLModel struct {
	ID                                      types.String `tfsdk:"id"`
	Name                                    types.String `tfsdk:"name"`
	Connector                               types.String `tfsdk:"connector"`
	DatabaseHostname                        types.String `tfsdk:"database_hostname"`
	DatabasePort                            types.Int64  `tfsdk:"database_port"`
	DatabaseUser                            types.String `tfsdk:"database_user"`
	DatabasePassword                        types.String `tfsdk:"database_password"`
	DatabaseDbname                          types.String `tfsdk:"database_dbname"`
	DatabaseSSLMode                         types.String `tfsdk:"database_sslmode"`
	SchemaIncludeList                       types.String `tfsdk:"schema_include_list"`
	TableIncludeList                        types.String `tfsdk:"table_include_list"`
	SignalDataCollectionSchemaOrDatabase    types.String `tfsdk:"signal_data_collection_schema_or_database"`
	HeartbeatEnabled                        types.Bool   `tfsdk:"heartbeat_enabled"`
	HeartbeatDataCollectionSchemaOrDatabase types.String `tfsdk:"heartbeat_data_collection_schema_or_database"`
	IncludeSourceDBNameInTableName          types.Bool   `tfsdk:"include_source_db_name_in_table_name"`
	SlotName                                types.String `tfsdk:"slot_name"`
	PublicationName                         types.String `tfsdk:"publication_name"`
	BinaryHandlingMode                      types.String `tfsdk:"binary_handling_mode"`
	SSHEnabled                              types.Bool   `tfsdk:"ssh_enabled"`
	SSHHost                                 types.String `tfsdk:"ssh_host"`
	SSHPort                                 types.String `tfsdk:"ssh_port"`
	SSHUser                                 types.String `tfsdk:"ssh_user"`
}

func (r *SourcePostgreSQL) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_postgresql"
}

func (r *SourcePostgreSQL) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Source PostgreSQL resource",
		MarkdownDescription: "Source PostgreSQL resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Source PostgreSQL identifier",
				MarkdownDescription: "Source PostgreSQL identifier",
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
			"database_user": schema.StringAttribute{
				Required:            true,
				Description:         "Username to access the database",
				MarkdownDescription: "Username to access the database",
			},
			"database_password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "Password to access the database",
				MarkdownDescription: "Password to access the database",
			},
			"database_dbname": schema.StringAttribute{
				Required:            true,
				Description:         "Database from which to stream data",
				MarkdownDescription: "Database from which to stream data",
			},
			"database_sslmode": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("require"),
				Description:         "Whether to use an encrypted connection to the PostgreSQL server",
				MarkdownDescription: "Whether to use an encrypted connection to the PostgreSQL server",
				Validators: []validator.String{
					stringvalidator.OneOf("require", "disable"),
				},
			},
			"schema_include_list": schema.StringAttribute{
				Required:            true,
				Description:         "Schemas to include",
				MarkdownDescription: "Schemas to include",
			},
			"table_include_list": schema.StringAttribute{
				Required:            true,
				Description:         "Source tables to sync",
				MarkdownDescription: "Source tables to sync",
			},
			"signal_data_collection_schema_or_database": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("public"),
				Description:         "Schema for signal data collection",
				MarkdownDescription: "Schema for signal data collection",
			},
			"heartbeat_enabled": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Enable heartbeat to keep the pipeline healthy during low data volume",
				MarkdownDescription: "Enable heartbeat to keep the pipeline healthy during low data volume",
			},
			"heartbeat_data_collection_schema_or_database": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "Schema for heartbeat data collection",
				MarkdownDescription: "Schema for heartbeat data collection",
			},
			"include_source_db_name_in_table_name": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Prefix topics with the database name",
				MarkdownDescription: "Prefix topics with the database name",
			},
			"slot_name": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("streamkap_pgoutput_slot"),
				Description:         "Replication slot name for the connector",
				MarkdownDescription: "Replication slot name for the connector",
			},
			"publication_name": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("streamkap_pub"),
				Description:         "Publication name for the connector",
				MarkdownDescription: "Publication name for the connector",
			},
			"binary_handling_mode": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("bytes"),
				Description:         "Representation of binary data for binary columns",
				MarkdownDescription: "Representation of binary data for binary columns",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"bytes",
						"base64",
						"base64-url-safe",
						"hex",
					),
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
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
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

func (r *SourcePostgreSQL) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Source PostgreSQL Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *SourcePostgreSQL) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan SourcePostgreSQLModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	plan.Connector = types.StringValue(r.connector_code)

	if resp.Diagnostics.HasError() {
		return
	}

	config := r.configMapFromModel(plan)

	source, err := r.client.CreateSource(ctx, api.Source{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating PostgreSQL source",
			fmt.Sprintf("Unable to create PostgreSQL source, got error: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(source.ID)
	plan.Name = types.StringValue(source.Name)
	plan.Connector = types.StringValue(source.Connector)
	r.modelFromConfigMap(source.Config, &plan)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SourcePostgreSQL) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state SourcePostgreSQLModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := state.ID.ValueString()
	source, err := r.client.GetSource(ctx, sourceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading PostgreSQL source",
			fmt.Sprintf("Unable to read PostgreSQL source, got error: %s", err),
		)
		return
	}
	if source == nil {
		resp.Diagnostics.AddError(
			"Error reading PostgreSQL source",
			fmt.Sprintf("PostgreSQL source %s does not exist", sourceID),
		)
		return
	}

	state.Name = types.StringValue(source.Name)
	state.Connector = types.StringValue(source.Connector)
	r.modelFromConfigMap(source.Config, &state)
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SourcePostgreSQL) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan SourcePostgreSQLModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", plan))
	config := r.configMapFromModel(plan)

	source, err := r.client.UpdateSource(ctx, plan.ID.ValueString(), api.Source{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating PostgreSQL source",
			fmt.Sprintf("Unable to update PostgreSQL source, got error: %s", err),
		)
		return
	}

	// Update resource state with updated items
	plan.Name = types.StringValue(source.Name)
	plan.Connector = types.StringValue(source.Connector)
	r.modelFromConfigMap(source.Config, &plan)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SourcePostgreSQL) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state SourcePostgreSQLModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting PostgreSQL source",
			fmt.Sprintf("Unable to delete PostgreSQL source, got error: %s", err),
		)
		return
	}
}

func (r *SourcePostgreSQL) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *SourcePostgreSQL) configMapFromModel(model SourcePostgreSQLModel) map[string]any {
	return map[string]any{
		"database.hostname.user.defined":                    model.DatabaseHostname.ValueString(),
		"database.port.user.defined":                        int(model.DatabasePort.ValueInt64()),
		"database.user":                                     model.DatabaseUser.ValueString(),
		"database.password":                                 model.DatabasePassword.ValueString(),
		"database.dbname":                                   model.DatabaseDbname.ValueString(),
		"database.sslmode":                                  model.DatabaseSSLMode.ValueString(),
		"schema.include.list":                               model.SchemaIncludeList.ValueString(),
		"table.include.list.user.defined":                   model.TableIncludeList.ValueString(),
		"signal.data.collection.schema.or.database":         model.SignalDataCollectionSchemaOrDatabase.ValueString(),
		"heartbeat.enabled":                                 model.HeartbeatEnabled.ValueBool(),
		"heartbeat.data.collection.schema.or.database":      model.HeartbeatDataCollectionSchemaOrDatabase.ValueString(),
		"include.source.db.name.in.table.name.user.defined": model.IncludeSourceDBNameInTableName.ValueBool(),
		"slot.name":                                         model.SlotName.ValueString(),
		"publication.name":                                  model.PublicationName.ValueString(),
		"binary.handling.mode":                              model.BinaryHandlingMode.ValueString(),
		"ssh.enabled":                                       model.SSHEnabled.ValueBool(),
		"ssh.host":                                          model.SSHHost.ValueString(),
		"ssh.port":                                          model.SSHPort.ValueString(),
		"ssh.user":                                          model.SSHUser.ValueString(),
	}
}

func (r *SourcePostgreSQL) modelFromConfigMap(cfg map[string]any, model *SourcePostgreSQLModel) {
	// Copy the config map to the model
	model.DatabaseHostname = helper.GetTfCfgString(cfg, "database.hostname.user.defined")
	model.DatabasePort = helper.GetTfCfgInt64(cfg, "database.port.user.defined")
	model.DatabaseUser = helper.GetTfCfgString(cfg, "database.user")
	model.DatabasePassword = helper.GetTfCfgString(cfg, "database.password")
	model.DatabaseDbname = helper.GetTfCfgString(cfg, "database.dbname")
	model.DatabaseSSLMode = helper.GetTfCfgString(cfg, "database.sslmode")
	model.SchemaIncludeList = helper.GetTfCfgString(cfg, "schema.include.list")
	model.TableIncludeList = helper.GetTfCfgString(cfg, "table.include.list.user.defined")
	model.SignalDataCollectionSchemaOrDatabase = helper.GetTfCfgString(cfg, "signal.data.collection.schema.or.database")
	model.HeartbeatEnabled = helper.GetTfCfgBool(cfg, "heartbeat.enabled")
	model.HeartbeatDataCollectionSchemaOrDatabase = helper.GetTfCfgString(cfg, "heartbeat.data.collection.schema.or.database")
	model.IncludeSourceDBNameInTableName = helper.GetTfCfgBool(cfg, "include.source.db.name.in.table.name.user.defined")
	model.SlotName = helper.GetTfCfgString(cfg, "slot.name")
	model.PublicationName = helper.GetTfCfgString(cfg, "publication.name")
	model.BinaryHandlingMode = helper.GetTfCfgString(cfg, "binary.handling.mode")
	model.SSHEnabled = helper.GetTfCfgBool(cfg, "ssh.enabled")
	model.SSHHost = helper.GetTfCfgString(cfg, "ssh.host")
	model.SSHPort = helper.GetTfCfgString(cfg, "ssh.port")
	model.SSHUser = helper.GetTfCfgString(cfg, "ssh.user")
}
