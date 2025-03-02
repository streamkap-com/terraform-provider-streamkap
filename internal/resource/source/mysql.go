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
	_ res.Resource                = &SourceMySQLResource{}
	_ res.ResourceWithConfigure   = &SourceMySQLResource{}
	_ res.ResourceWithImportState = &SourceMySQLResource{}
)

func NewSourceMySQLResource() res.Resource {
	return &SourceMySQLResource{connector_code: "mysql"}
}

// SourceMySQLResource defines the resource implementation.
type SourceMySQLResource struct {
	client         api.StreamkapAPI
	connector_code string
}

// SourceMySQLResourceModel describes the resource data model.
type SourceMySQLResourceModel struct {
	ID                                      types.String `tfsdk:"id"`
	Name                                    types.String `tfsdk:"name"`
	Connector                               types.String `tfsdk:"connector"`
	DatabaseHostname                        types.String `tfsdk:"database_hostname"`
	DatabasePort                            types.Int64  `tfsdk:"database_port"`
	DatabaseUser                            types.String `tfsdk:"database_user"`
	DatabasePassword                        types.String `tfsdk:"database_password"`
	DatabaseIncludeList                     types.String `tfsdk:"database_include_list"`
	TableIncludeList                        types.String `tfsdk:"table_include_list"`
	SignalDataCollectionSchemaOrDatabase    types.String `tfsdk:"signal_data_collection_schema_or_database"`
	ColumnIncludeList                       types.String `tfsdk:"column_include_list"`
	HeartbeatEnabled                        types.Bool   `tfsdk:"heartbeat_enabled"`
	HeartbeatDataCollectionSchemaOrDatabase types.String `tfsdk:"heartbeat_data_collection_schema_or_database"`
	SnapshotGTID                            types.Bool   `tfsdk:"snapshot_gtid"`
	BinaryHandlingMode                      types.String `tfsdk:"binary_handling_mode"`
	DatabaseConnectionTimezone              types.String `tfsdk:"database_connection_timezone"`
	SSHEnabled                              types.Bool   `tfsdk:"ssh_enabled"`
	SSHHost                                 types.String `tfsdk:"ssh_host"`
	SSHPort                                 types.String `tfsdk:"ssh_port"`
	SSHUser                                 types.String `tfsdk:"ssh_user"`
}

func (r *SourceMySQLResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_mysql"
}

func (r *SourceMySQLResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Source MySQL resource",
		MarkdownDescription: "Source MySQL resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Source MySQL identifier",
				MarkdownDescription: "Source MySQL identifier",
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
				Description:         "MySQL Hostname. For example, mysqldb.something.rds.amazonaws.com",
				MarkdownDescription: "MySQL Hostname. For example, mysqldb.something.rds.amazonaws.com",
			},
			"database_port": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(3306),
				Description:         "MySQL Port. For example, 3306",
				MarkdownDescription: "MySQL Port. For example, 3306",
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
			"database_include_list": schema.StringAttribute{
				Required:            true,
				Description:         "Source Databases",
				MarkdownDescription: "Source Databases",
			},
			"table_include_list": schema.StringAttribute{
				Required:            true,
				Description:         "Source tables to sync",
				MarkdownDescription: "Source tables to sync",
			},
			"signal_data_collection_schema_or_database": schema.StringAttribute{
				Optional:            true,
				Description:         "Schema for signal data collection. If connector is in read-only mode (snapshot_gtid=\"Yes\"), set this to null.",
				MarkdownDescription: "Schema for signal data collection. If connector is in read-only mode (snapshot_gtid=\"Yes\"), set this to null.",
			},
			"column_include_list": schema.StringAttribute{
				Optional:            true,
				Description:         "Comma separated list of columns whitelist regular expressions, format schema[.]table[.](column1|column2|etc)",
				MarkdownDescription: "Comma separated list of columns whitelist regular expressions, format schema[.]table[.](column1|column2|etc)",
			},
			"heartbeat_enabled": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Heartbeats are used to keep the pipeline healthy when there is a low volume of data at times.",
				MarkdownDescription: "Heartbeats are used to keep the pipeline healthy when there is a low volume of data at times.",
			},
			"heartbeat_data_collection_schema_or_database": schema.StringAttribute{
				Optional:            true,
				Description:         "Heartbeat Table Database",
				MarkdownDescription: "Heartbeat Table Database",
			},
			"database_connection_timezone": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("SERVER"),
				Description:         "Set the connection timezone. If set to SERVER, the source will detect the connection time zone from the values configured on the MySQL server session variables 'time_zone' or 'system_time_zone'",
				MarkdownDescription: "Set the connection timezone. If set to SERVER, the source will detect the connection time zone from the values configured on the MySQL server session variables 'time_zone' or 'system_time_zone'",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"SERVER",
						"UTC",
						"Africa/Cairo",
						" Asia/Riyadh",
						"Africa/Casablanca",
						"Asia/Seoul",
						"Africa/Harare",
						"Asia/Shanghai",
						"Africa/Monrovia",
						"Asia/Singapore",
						"Africa/Nairobi",
						"Asia/Taipei",
						"Africa/Tripoli",
						"Asia/Tehran",
						"Africa/Windhoek",
						"Asia/Tokyo",
						"America/Araguaina",
						"Asia/Ulaanbaatar",
						"America/Asuncion",
						"Asia/Vladivostok",
						"America/Bogota",
						"Asia/Yakutsk",
						"America/Buenos_Aires",
						"Asia/Yerevan",
						"America/Caracas",
						"Atlantic/Azores",
						"America/Chihuahua",
						"Australia/Adelaide",
						"America/Cuiaba",
						"Australia/Brisbane",
						"America/Denver",
						"Australia/Darwin",
						"America/Fortaleza",
						"Australia/Hobart",
						"America/Guatemala",
						"Australia/Perth",
						"America/Halifax",
						"Australia/Sydney",
						"America/Manaus",
						"Brazil/East",
						"America/Matamoros",
						"Canada/Newfoundland",
						"America/Monterrey",
						"Canada/Saskatchewan",
						"America/Montevideo",
						"Canada/Yukon",
						"America/Phoenix",
						"Europe/Amsterdam",
						"America/Santiago",
						"Europe/Athens",
						"America/Tijuana",
						"Europe/Dublin",
						"Asia/Amman",
						"Europe/Helsinki",
						"Asia/Ashgabat",
						"Europe/Istanbul",
						"Asia/Baghdad",
						"Europe/Kaliningrad",
						"Asia/Baku",
						"Europe/Moscow",
						"Asia/Bangkok",
						"Europe/Paris",
						"Asia/Beirut",
						"Europe/Prague",
						"Asia/Calcutta",
						"Europe/Sarajevo",
						"Asia/Damascus",
						"Pacific/Auckland",
						"Asia/Dhaka",
						"Pacific/Fiji",
						"Asia/Irkutsk",
						"Pacific/Guam",
						"Asia/Jerusalem",
						"Pacific/Honolulu",
						"Asia/Kabul",
						"Pacific/Samoa",
						"Asia/Karachi",
						"US/Alaska",
						"Asia/Kathmandu",
						"US/Central",
						"Asia/Krasnoyarsk",
						"US/Eastern",
						"Asia/Magadan",
						"US/East-Indiana",
						"Asia/Muscat",
						"US/Pacific",
						"Asia/Novosibirsk",
					),
				},
			},
			"snapshot_gtid": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(true),
				Description:         "GTID snapshots are read only but require some prerequisite settings, including enabling GTID on the source database. See the documentation for more details.",
				MarkdownDescription: "GTID snapshots are read only but require some prerequisite settings, including enabling GTID on the source database. See the documentation for more details.",
			},
			"binary_handling_mode": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("bytes"),
				Description:         "Specifies how the data for binary columns e.g. blob, binary, varbinary should be represented. This setting depends on what the destination is. See the documentation for more details.",
				MarkdownDescription: "Specifies how the data for binary columns e.g. blob, binary, varbinary should be represented. This setting depends on what the destination is. See the documentation for more details.",
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

func (r *SourceMySQLResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Source MySQL Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *SourceMySQLResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan SourceMySQLResourceModel

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
			"Error creating MySQL source",
			fmt.Sprintf("Unable to create MySQL source, got error: %s", err),
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

func (r *SourceMySQLResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state SourceMySQLResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := state.ID.ValueString()
	source, err := r.client.GetSource(ctx, sourceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading MySQL source",
			fmt.Sprintf("Unable to read MySQL source, got error: %s", err),
		)
		return
	}
	if source == nil {
		resp.Diagnostics.AddError(
			"Error reading MySQL source",
			fmt.Sprintf("MySQL source %s does not exist", sourceID),
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

func (r *SourceMySQLResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan SourceMySQLResourceModel

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
			"Error updating MySQL source",
			fmt.Sprintf("Unable to update MySQL source, got error: %s", err),
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

func (r *SourceMySQLResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state SourceMySQLResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting MySQL source",
			fmt.Sprintf("Unable to delete MySQL source, got error: %s", err),
		)
		return
	}
}

func (r *SourceMySQLResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *SourceMySQLResource) model2ConfigMap(model SourceMySQLResourceModel) map[string]any {
	snapshotGTIDStr := "Yes"
	if !model.SnapshotGTID.ValueBool() {
		snapshotGTIDStr = "No"
	}
	return map[string]any{
		"database.hostname.user.defined":               model.DatabaseHostname.ValueString(),
		"database.port.user.defined":                   int(model.DatabasePort.ValueInt64()),
		"database.user":                                model.DatabaseUser.ValueString(),
		"database.password":                            model.DatabasePassword.ValueString(),
		"database.include.list.user.defined":           model.DatabaseIncludeList.ValueString(),
		"table.include.list.user.defined":              model.TableIncludeList.ValueString(),
		"signal.data.collection.schema.or.database":    model.SignalDataCollectionSchemaOrDatabase.ValueStringPointer(),
		"column.include.list.user.defined":             model.ColumnIncludeList.ValueStringPointer(),
		"heartbeat.enabled":                            model.HeartbeatEnabled.ValueBool(),
		"heartbeat.data.collection.schema.or.database": model.HeartbeatDataCollectionSchemaOrDatabase.ValueStringPointer(),
		"database.connectionTimeZone":                 model.DatabaseConnectionTimezone.ValueString(),
		"snapshot.gtid":                                snapshotGTIDStr,
		"binary.handling.mode":                         model.BinaryHandlingMode.ValueString(),
		"ssh.enabled":                                  model.SSHEnabled.ValueBool(),
		"ssh.host":                                     model.SSHHost.ValueStringPointer(),
		"ssh.port":                                     model.SSHPort.ValueString(),
		"ssh.user":                                     model.SSHUser.ValueString(),
	}
}

func (r *SourceMySQLResource) configMap2Model(cfg map[string]any, model *SourceMySQLResourceModel) {
	// Copy the config map to the model
	model.DatabaseHostname = helper.GetTfCfgString(cfg, "database.hostname.user.defined")
	model.DatabasePort = helper.GetTfCfgInt64(cfg, "database.port.user.defined")
	model.DatabaseUser = helper.GetTfCfgString(cfg, "database.user")
	model.DatabasePassword = helper.GetTfCfgString(cfg, "database.password")
	model.DatabaseIncludeList = helper.GetTfCfgString(cfg, "database.include.list.user.defined")
	model.TableIncludeList = helper.GetTfCfgString(cfg, "table.include.list.user.defined")
	model.SignalDataCollectionSchemaOrDatabase = helper.GetTfCfgString(cfg, "signal.data.collection.schema.or.database")
	model.ColumnIncludeList = helper.GetTfCfgString(cfg, "column.include.list.user.defined")
	model.HeartbeatEnabled = helper.GetTfCfgBool(cfg, "heartbeat.enabled")
	model.HeartbeatDataCollectionSchemaOrDatabase = helper.GetTfCfgString(cfg, "heartbeat.data.collection.schema.or.database")
	model.DatabaseConnectionTimezone = helper.GetTfCfgString(cfg, "database.connectionTimeZone")
	model.SnapshotGTID = types.BoolValue(helper.GetTfCfgString(cfg, "snapshot.gtid").ValueString() == "Yes")
	model.BinaryHandlingMode = helper.GetTfCfgString(cfg, "binary.handling.mode")
	model.SSHEnabled = helper.GetTfCfgBool(cfg, "ssh.enabled")
	model.SSHHost = helper.GetTfCfgString(cfg, "ssh.host")
	model.SSHPort = helper.GetTfCfgString(cfg, "ssh.port")
	model.SSHUser = helper.GetTfCfgString(cfg, "ssh.user")
}
