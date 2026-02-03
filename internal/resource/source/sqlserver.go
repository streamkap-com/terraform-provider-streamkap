package source

import (
	"context"
	"fmt"
	"encoding/json"

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
	_ res.Resource                = &SourceSQLServerResource{}
	_ res.ResourceWithConfigure   = &SourceSQLServerResource{}
	_ res.ResourceWithImportState = &SourceSQLServerResource{}
)

func NewSourceSQLServerResource() res.Resource {
	return &SourceSQLServerResource{connector_code: "sqlserveraws"}
}

// SourceSQLServerResource defines the resource implementation.
type SourceSQLServerResource struct {
	client         api.StreamkapAPI
	connector_code string
}

// SourceSQLServerResourceModel describes the resource data model.
type SourceSQLServerResourceModel struct {
	ID                                      types.String `tfsdk:"id"`
	Name                                    types.String `tfsdk:"name"`
	Connector                               types.String `tfsdk:"connector"`
	DatabaseHostname                        types.String `tfsdk:"database_hostname"`
	DatabasePort                            types.Int64  `tfsdk:"database_port"`
	DatabaseUser                            types.String `tfsdk:"database_user"`
	DatabasePassword                        types.String `tfsdk:"database_password"`
	DatabaseName                            types.String `tfsdk:"database_dbname"`
	SchemaIncludeList                       types.String `tfsdk:"schema_include_list"`
	TableIncludeList                        types.String `tfsdk:"table_include_list"`
	SignalDataCollectionSchemaOrDatabase    types.String `tfsdk:"signal_data_collection_schema_or_database"`
	ColumnExcludeList                       types.String `tfsdk:"column_exclude_list"`
	HeartbeatEnabled                        types.Bool   `tfsdk:"heartbeat_enabled"`
	HeartbeatDataCollectionSchemaOrDatabase types.String `tfsdk:"heartbeat_data_collection_schema_or_database"`
	BinaryHandlingMode                      types.String `tfsdk:"binary_handling_mode"`
	InsertStaticKeyField                    types.String `tfsdk:"insert_static_key_field"`
	InsertStaticKeyValue                    types.String `tfsdk:"insert_static_key_value"`
	InsertStaticValueField                  types.String `tfsdk:"insert_static_value_field"`
	InsertStaticValue                       types.String `tfsdk:"insert_static_value"`
	SSHEnabled                              types.Bool   `tfsdk:"ssh_enabled"`
	SSHHost                                 types.String `tfsdk:"ssh_host"`
	SSHPort                                 types.String `tfsdk:"ssh_port"`
	SSHUser                                 types.String `tfsdk:"ssh_user"`
	SnapshotParallelism                     types.Int64 `tfsdk:"snapshot_parallelism"`
	SnapshotLargeTableThreshold             types.Int64 `tfsdk:"snapshot_large_table_threshold"`
	SnapshotCustomTableConfig               map[string]snapshotCustomTableConfigModel `tfsdk:"snapshot_custom_table_config"`
}

type snapshotCustomTableConfigModel struct {
	Chunks types.Int64 `tfsdk:"chunks"`
}

func (r *SourceSQLServerResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_sqlserver"
}

func (r *SourceSQLServerResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Source SQLServer resource",
		MarkdownDescription: "Source SQLServer resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Source SQLServer identifier",
				MarkdownDescription: "Source SQLServer identifier",
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
				Description:         "SQLServer Hostname. For example, sqlserverdb.something.rds.amazonaws.com",
				MarkdownDescription: "SQLServer Hostname. For example, sqlserverdb.something.rds.amazonaws.com",
			},
			"database_port": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(1433),
				Description:         "SQLServer Port. For example, 1433",
				MarkdownDescription: "SQLServer Port. For example, 1433",
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
				Description:         "Source Databases",
				MarkdownDescription: "Source Databases",
			},
			"schema_include_list": schema.StringAttribute{
				Required:            true,
				Description:         "Source schemas to sync",
				MarkdownDescription: "Source schemas to sync",
			},
			"table_include_list": schema.StringAttribute{
				Required:            true,
				Description:         "Source tables to sync",
				MarkdownDescription: "Source tables to sync",
			},
			"signal_data_collection_schema_or_database": schema.StringAttribute{
				Optional:            true,
				Description:         "Signal table location in schema.table format (e.g., 'dbo.streamkap_signal'). For backwards compatibility, you can specify just the schema name.",
				MarkdownDescription: "Signal table location in `schema.table` format (e.g., `dbo.streamkap_signal`). For backwards compatibility, you can specify just the schema name.",
			},
			"column_exclude_list": schema.StringAttribute{
				Optional:            true,
				Description:         "Comma separated list of columns blacklist regular expressions, format schema[.]table[.](column1|column2|etc)",
				MarkdownDescription: "Comma separated list of columns blacklist regular expressions, format schema[.]table[.](column1|column2|etc)",
			},
			"heartbeat_enabled": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Heartbeats are used to keep the pipeline healthy when there is a low volume of data at times.",
				MarkdownDescription: "Heartbeats are used to keep the pipeline healthy when there is a low volume of data at times.",
			},
			"heartbeat_data_collection_schema_or_database": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("streamkap"),
				Description:         "Heartbeat Table Database",
				MarkdownDescription: "Heartbeat Table Database",
			},
			"insert_static_key_field": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The name of the static field to be added to the message key.",
				MarkdownDescription: "The name of the static field to be added to the message key.",
			},
			"insert_static_key_value": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The value of the static field to be added to the message key.",
				MarkdownDescription: "The value of the static field to be added to the message key.",
			},
			"insert_static_value_field": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The name of the static field to be added to the message value.",
				MarkdownDescription: "The name of the static field to be added to the message value.",
			},
			"insert_static_value": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The value of the static field to be added to the message value.",
				MarkdownDescription: "The value of the static field to be added to the message value.",
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
			"snapshot_parallelism": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(1),
				Description:         "How many parallel chunk requests to send to the source DB",
				MarkdownDescription: "How many parallel chunk requests to send to the source DB",
				Validators: []validator.Int64{
					int64validator.Between(1, 10),
				},
			},
			"snapshot_large_table_threshold": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(20000),
				Description:         "The threshold in MB for a Large Table to require multiple chunks to be read in parallel",
				MarkdownDescription: "The threshold in MB for a Large Table to require multiple chunks to be read in parallel",
				Validators: []validator.Int64{
					int64validator.Between(1, 64000),
				},
			},
			"snapshot_custom_table_config": schema.MapNestedAttribute{
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"chunks": schema.Int64Attribute{
							Required:   true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
					},
				},
				Description:         "Explicitly set nb of parallel chunks for tables. Format: {\"db.Some_Tbl\": {\"chunks\": 5}}. This allows manual settings for parallelization when stats are outdated and estimated table size cannot be computed reliably",
				MarkdownDescription: "Explicitly set nb of parallel chunks for tables. Format: {\"db.Some_Tbl\": {\"chunks\": 5}}. This allows manual settings for parallelization when stats are outdated and estimated table size cannot be computed reliably",
			},
		},
	}
}

func (r *SourceSQLServerResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Source SQLServer Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *SourceSQLServerResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan SourceSQLServerResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	plan.Connector = types.StringValue(r.connector_code)

	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.model2ConfigMap(plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting SQLServer source config",
			fmt.Sprintf("Unable to convert SQLServer source config, got error: %s", err),
		)
		return
	}

	source, err := r.client.CreateSource(ctx, api.Source{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})


	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating SQLServer source",
			fmt.Sprintf("Unable to create SQLServer source, got error: %s", err),
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

func (r *SourceSQLServerResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state SourceSQLServerResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := state.ID.ValueString()
	source, err := r.client.GetSource(ctx, sourceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading SQLServer source",
			fmt.Sprintf("Unable to read SQLServer source, got error: %s", err),
		)
		return
	}
	if source == nil {
		resp.Diagnostics.AddError(
			"Error reading SQLServer source",
			fmt.Sprintf("SQLServer source %s does not exist", sourceID),
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

func (r *SourceSQLServerResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan SourceSQLServerResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", plan))
	config, err := r.model2ConfigMap(plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error converting SQLServer source config",
			fmt.Sprintf("Unable to convert SQLServer source config, got error: %s", err),
		)
		return
	}

	source, err := r.client.UpdateSource(ctx, plan.ID.ValueString(), api.Source{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating SQLServer source",
			fmt.Sprintf("Unable to update SQLServer source, got error: %s", err),
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

func (r *SourceSQLServerResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state SourceSQLServerResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting SQLServer source",
			fmt.Sprintf("Unable to delete SQLServer source, got error: %s", err),
		)
		return
	}
}

func (r *SourceSQLServerResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *SourceSQLServerResource) model2ConfigMap(model SourceSQLServerResourceModel) (map[string]any, error) {

	var snapshotCustomTableConfigStr string
	snapshotCustomTableConfigJSON := make(map[string]map[string]int64)
	if len(model.SnapshotCustomTableConfig) != 0 {
		for table, chunks := range model.SnapshotCustomTableConfig {
			_, err := json.Marshal(map[string]int64{
				"chunks": chunks.Chunks.ValueInt64(),
			})
			if err != nil {
				return nil, err
			}
			chunksMap := map[string]int64{
				"chunks": chunks.Chunks.ValueInt64(),
			}
			snapshotCustomTableConfigJSON[table] = chunksMap
		}

		snapshotCustomTableConfig, err := json.Marshal(snapshotCustomTableConfigJSON)
		snapshotCustomTableConfigStr = string(snapshotCustomTableConfig)
		if err != nil {
			return nil, err
		}
	}

	var expectedSnapshotCustomTableConfig *string

	if snapshotCustomTableConfigStr == "" {
		expectedSnapshotCustomTableConfig = nil
	} else {
		expectedSnapshotCustomTableConfig = &snapshotCustomTableConfigStr
	}

	configMap := map[string]any{
		"database.hostname.user.defined":               model.DatabaseHostname.ValueString(),
		"database.port.user.defined":                   int(model.DatabasePort.ValueInt64()),
		"database.user":                                model.DatabaseUser.ValueString(),
		"database.password":                            model.DatabasePassword.ValueString(),
		"database.names":                               model.DatabaseName.ValueString(),
		"schema.include.list":                          model.SchemaIncludeList.ValueString(),
		"table.include.list.user.defined":              model.TableIncludeList.ValueString(),
		"signal.data.collection.schema.or.database":    model.SignalDataCollectionSchemaOrDatabase.ValueStringPointer(),
		"column.exclude.list.user.defined":             model.ColumnExcludeList.ValueStringPointer(),
		"heartbeat.enabled":                            model.HeartbeatEnabled.ValueBool(),
		"heartbeat.data.collection.schema.or.database": model.HeartbeatDataCollectionSchemaOrDatabase.ValueStringPointer(),
		"transforms.InsertStaticKey1.static.field":      model.InsertStaticKeyField.ValueString(),
		"transforms.InsertStaticKey1.static.value":      model.InsertStaticKeyValue.ValueString(),
		"transforms.InsertStaticValue1.static.field":    model.InsertStaticValueField.ValueString(),
		"transforms.InsertStaticValue1.static.value":    model.InsertStaticValue.ValueString(),
		"binary.handling.mode":                         model.BinaryHandlingMode.ValueString(),
		"ssh.enabled":                                  model.SSHEnabled.ValueBool(),
		"ssh.host":                                     model.SSHHost.ValueStringPointer(),
		"ssh.port":                                     model.SSHPort.ValueString(),
		"ssh.user":                                     model.SSHUser.ValueString(),
		"streamkap.snapshot.parallelism":               model.SnapshotParallelism.ValueInt64(),
		"streamkap.snapshot.large.table.threshold":     model.SnapshotLargeTableThreshold.ValueInt64(),
		"streamkap.snapshot.custom.table.config.user.defined": expectedSnapshotCustomTableConfig,
	}

	return configMap, nil
}

func (r *SourceSQLServerResource) configMap2Model(cfg map[string]any, model *SourceSQLServerResourceModel) (err error) {
	// Copy the config map to the model
	model.DatabaseHostname = helper.GetTfCfgString(cfg, "database.hostname.user.defined")
	model.DatabasePort = helper.GetTfCfgInt64(cfg, "database.port.user.defined")
	model.DatabaseUser = helper.GetTfCfgString(cfg, "database.user")
	model.DatabasePassword = helper.GetTfCfgString(cfg, "database.password")
	model.DatabaseName = helper.GetTfCfgString(cfg, "database.names")
	model.SchemaIncludeList = helper.GetTfCfgString(cfg, "schema.include.list")
	model.TableIncludeList = helper.GetTfCfgString(cfg, "table.include.list.user.defined")
	model.SignalDataCollectionSchemaOrDatabase = helper.GetTfCfgString(cfg, "signal.data.collection.schema.or.database")
	model.ColumnExcludeList = helper.GetTfCfgString(cfg, "column.exclude.list.user.defined")
	model.HeartbeatEnabled = helper.GetTfCfgBool(cfg, "heartbeat.enabled")
	model.HeartbeatDataCollectionSchemaOrDatabase = helper.GetTfCfgString(cfg, "heartbeat.data.collection.schema.or.database")
	model.InsertStaticKeyField = helper.GetTfCfgString(cfg, "transforms.InsertStaticKey1.static.field")
	model.InsertStaticKeyValue = helper.GetTfCfgString(cfg, "transforms.InsertStaticKey1.static.value")
	model.InsertStaticValueField = helper.GetTfCfgString(cfg, "transforms.InsertStaticValue1.static.field")
	model.InsertStaticValue = helper.GetTfCfgString(cfg, "transforms.InsertStaticValue1.static.value")
	model.BinaryHandlingMode = helper.GetTfCfgString(cfg, "binary.handling.mode")
	model.SSHEnabled = helper.GetTfCfgBool(cfg, "ssh.enabled")
	model.SSHHost = helper.GetTfCfgString(cfg, "ssh.host")
	model.SSHPort = helper.GetTfCfgString(cfg, "ssh.port")
	model.SSHUser = helper.GetTfCfgString(cfg, "ssh.user")
	model.SnapshotParallelism = helper.GetTfCfgInt64(cfg, "streamkap.snapshot.parallelism")
	model.SnapshotLargeTableThreshold = helper.GetTfCfgInt64(cfg, "streamkap.snapshot.large.table.threshold")

	snapshotCustomTableConfigStr := helper.GetTfCfgString(cfg, "streamkap.snapshot.custom.table.config.user.defined").ValueString()
	snapshotCustomTableConfig := make(map[string]snapshotCustomTableConfigModel)

	snapshotCustomTableConfigPartialJSON := make(map[string]int64)
	err = json.Unmarshal([]byte(snapshotCustomTableConfigStr), &snapshotCustomTableConfigPartialJSON)
	if err != nil {
		return
	}

	snapshotCustomTableConfigJSON := make(map[string]map[string]int64)
	for table, chunks := range snapshotCustomTableConfigPartialJSON {
		// chunksMap := make(map[string]int64)
		// err = json.Unmarshal([]byte(chunks), &chunksMap)
		// if err != nil {
		// 	return
		// }
		chunksMap := map[string]int64{
			"chunks": chunks,
		}
		snapshotCustomTableConfigJSON[table] = chunksMap
	}

	for table, chunks := range snapshotCustomTableConfigJSON {
		snapshotCustomTableConfig[table] = snapshotCustomTableConfigModel{
			Chunks: types.Int64Value(chunks["chunks"]),
		}
	}
	model.SnapshotCustomTableConfig = snapshotCustomTableConfig
	return
}
