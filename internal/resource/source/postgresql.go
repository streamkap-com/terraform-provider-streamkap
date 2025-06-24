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
	_ res.Resource                = &SourcePostgreSQLResource{}
	_ res.ResourceWithConfigure   = &SourcePostgreSQLResource{}
	_ res.ResourceWithImportState = &SourcePostgreSQLResource{}
)

func NewSourcePostgreSQLResource() res.Resource {
	return &SourcePostgreSQLResource{connector_code: "postgresql"}
}

// SourcePostgreSQLResource defines the resource implementation.
type SourcePostgreSQLResource struct {
	client         api.StreamkapAPI
	connector_code string
}

// SourcePostgreSQLResourceModel describes the resource data model.
type SourcePostgreSQLResourceModel struct {
	ID                                      types.String `tfsdk:"id"`
	Name                                    types.String `tfsdk:"name"`
	Connector                               types.String `tfsdk:"connector"`
	DatabaseHostname                        types.String `tfsdk:"database_hostname"`
	DatabasePort                            types.Int64  `tfsdk:"database_port"`
	DatabaseUser                            types.String `tfsdk:"database_user"`
	DatabasePassword                        types.String `tfsdk:"database_password"`
	DatabaseDbname                          types.String `tfsdk:"database_dbname"`
	SnapshotReadOnly                        types.String `tfsdk:"snapshot_read_only"`
	DatabaseSSLMode                         types.String `tfsdk:"database_sslmode"`
	SchemaIncludeList                       types.String `tfsdk:"schema_include_list"`
	TableIncludeList                        types.String `tfsdk:"table_include_list"`
	SignalDataCollectionSchemaOrDatabase    types.String `tfsdk:"signal_data_collection_schema_or_database"`
	ColumnIncludeList                       types.String `tfsdk:"column_include_list"`
	ColumnExcludeList                       types.String `tfsdk:"column_exclude_list"`
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
	PredicatesIsTopicToEnrichPattern        types.String `tfsdk:"predicates_istopictoenrich_pattern"`
	InsertStaticKeyField1                   types.String `tfsdk:"insert_static_key_field_1"`
	InsertStaticKeyValue1                   types.String `tfsdk:"insert_static_key_value_1"`
	InsertStaticValueField1                 types.String `tfsdk:"insert_static_value_field_1"`
	InsertStaticValue1                      types.String `tfsdk:"insert_static_value_1"`
	InsertStaticKeyField2                   types.String `tfsdk:"insert_static_key_field_2"`
	InsertStaticKeyValue2                   types.String `tfsdk:"insert_static_key_value_2"`
	InsertStaticValueField2                 types.String `tfsdk:"insert_static_value_field_2"`
	InsertStaticValue2                      types.String `tfsdk:"insert_static_value_2"`
}

func (r *SourcePostgreSQLResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_postgresql"
}

func (r *SourcePostgreSQLResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
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
			"snapshot_read_only": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("Yes"),
				Description: "When connecting to a read replica PostgreSQL database, " +
					"this must be set to 'Yes' to support Streamkap snapshots",
				MarkdownDescription: "When connecting to a read replica PostgreSQL database, " +
					"this must be set to 'Yes' to support Streamkap snapshots",
				Validators: []validator.String{
					stringvalidator.OneOf("Yes", "No"),
				},
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
			"column_include_list": schema.StringAttribute{
				Optional: true,
				Description: "An optional, comma-separated list of regular expressions that match the fully-qualified names of columns " +
					"that should be included in change event record values. Fully-qualified names for columns " +
					"are of the form schemaName[.]tableName[.](columnName1|columnName2). " +
					"You can only specify either `column_include_list` or `column_exclude_list`, not both.",
				MarkdownDescription: "An optional, comma-separated list of regular expressions that match the fully-qualified names of columns " +
					"that should be included in change event record values. Fully-qualified names for columns " +
					"are of the form schemaName[.]tableName[.](columnName1|columnName2)" +
					"You can only specify either `column_include_list` or `column_exclude_list`, not both.",
			},
			"column_exclude_list": schema.StringAttribute{
				Optional: true,
				Description: "An optional, comma-separated list of regular expressions that match the fully-qualified names of columns " +
					"that should be excluded from change event record values. " +
					"Fully-qualified names for columns are of the form schemaName.tableName.columnName." +
					"You can only specify either `column_include_list` or `column_exclude_list`, not both.",
				MarkdownDescription: "An optional, comma-separated list of regular expressions that match the fully-qualified names of columns " +
					"that should be excluded from change event record values. " +
					"Fully-qualified names for columns are of the form schemaName.tableName.columnName." +
					"You can only specify either `column_include_list` or `column_exclude_list`, not both.",
			},
			"heartbeat_enabled": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Enable heartbeat to keep the pipeline healthy during low data volume",
				MarkdownDescription: "Enable heartbeat to keep the pipeline healthy during low data volume",
			},
			"heartbeat_data_collection_schema_or_database": schema.StringAttribute{
				Optional:            true,
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
			"predicates_istopictoenrich_pattern": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("$^"),
				Description:         "Regex pattern to match topics for enrichment",
				MarkdownDescription: "Regex pattern to match topics for enrichment",
			},
			"insert_static_key_field_1": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The name of the static field to be added to the message key.",
				MarkdownDescription: "The name of the static field to be added to the message key.",
			},
			"insert_static_key_value_1": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The value of the static field to be added to the message key.",
				MarkdownDescription: "The value of the static field to be added to the message key.",
			},
			"insert_static_value_field_1": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The name of the static field to be added to the message value.",
				MarkdownDescription: "The name of the static field to be added to the message value.",
			},
			"insert_static_value_1": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The value of the static field to be added to the message value.",
				MarkdownDescription: "The value of the static field to be added to the message value.",
			},
			"insert_static_key_field_2": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The name of the static field to be added to the message key.",
				MarkdownDescription: "The name of the static field to be added to the message key.",
			},
			"insert_static_key_value_2": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The value of the static field to be added to the message key.",
				MarkdownDescription: "The value of the static field to be added to the message key.",
			},
			"insert_static_value_field_2": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The name of the static field to be added to the message value.",
				MarkdownDescription: "The name of the static field to be added to the message value.",
			},
			"insert_static_value_2": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "The value of the static field to be added to the message value.",
				MarkdownDescription: "The value of the static field to be added to the message value.",
			},
		},
	}
}

func (r *SourcePostgreSQLResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
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

func (r *SourcePostgreSQLResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan SourcePostgreSQLResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	plan.Connector = types.StringValue(r.connector_code)

	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.model2ConfigMap(plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating PostgreSQL source",
			fmt.Sprintf("Unable to create PostgreSQL source, got error: %s", err),
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
			"Error creating PostgreSQL source",
			fmt.Sprintf("Unable to create PostgreSQL source, got error: %s", err),
		)
		return
	}

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

func (r *SourcePostgreSQLResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state SourcePostgreSQLResourceModel

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
	r.configMap2Model(source.Config, &state)
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SourcePostgreSQLResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan SourcePostgreSQLResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", plan))
	config, err := r.model2ConfigMap(plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating PostgreSQL source",
			fmt.Sprintf("Unable to update PostgreSQL source, got error: %s", err),
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
			"Error updating PostgreSQL source",
			fmt.Sprintf("Unable to update PostgreSQL source, got error: %s", err),
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

func (r *SourcePostgreSQLResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state SourcePostgreSQLResourceModel

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

func (r *SourcePostgreSQLResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *SourcePostgreSQLResource) model2ConfigMap(model SourcePostgreSQLResourceModel) (map[string]any, error) {
	if !model.ColumnExcludeList.IsNull() && !model.ColumnIncludeList.IsNull() {
		return nil, fmt.Errorf("only one of column_include_list or column_exclude_list can be set")
	}

	configMap := map[string]any{
		"database.hostname.user.defined":                    model.DatabaseHostname.ValueString(),
		"database.port.user.defined":                        int(model.DatabasePort.ValueInt64()),
		"database.user":                                     model.DatabaseUser.ValueString(),
		"database.password":                                 model.DatabasePassword.ValueString(),
		"database.dbname":                                   model.DatabaseDbname.ValueString(),
		"snapshot.read.only.user.defined":                   model.SnapshotReadOnly.ValueString(),
		"database.sslmode":                                  model.DatabaseSSLMode.ValueString(),
		"schema.include.list":                               model.SchemaIncludeList.ValueString(),
		"table.include.list.user.defined":                   model.TableIncludeList.ValueString(),
		"signal.data.collection.schema.or.database":         model.SignalDataCollectionSchemaOrDatabase.ValueString(),
		"column.include.list.toggled":                       true,
		"heartbeat.enabled":                                 model.HeartbeatEnabled.ValueBool(),
		"heartbeat.data.collection.schema.or.database":      model.HeartbeatDataCollectionSchemaOrDatabase.ValueStringPointer(),
		"include.source.db.name.in.table.name.user.defined": model.IncludeSourceDBNameInTableName.ValueBool(),
		"slot.name":                                         model.SlotName.ValueString(),
		"publication.name":                                  model.PublicationName.ValueString(),
		"binary.handling.mode":                              model.BinaryHandlingMode.ValueString(),
		"ssh.enabled":                                       model.SSHEnabled.ValueBool(),
		"ssh.host":                                          model.SSHHost.ValueStringPointer(),
		"ssh.port":                                          model.SSHPort.ValueString(),
		"ssh.user":                                          model.SSHUser.ValueString(),
		"predicates.IsTopicToEnrich.pattern":    model.PredicatesIsTopicToEnrichPattern.ValueString(),
		"transforms.InsertStaticKey1.static.field":      model.InsertStaticKeyField1.ValueString(),
		"transforms.InsertStaticKey1.static.value":      model.InsertStaticKeyValue1.ValueString(),
		"transforms.InsertStaticValue1.static.field":    model.InsertStaticValueField1.ValueString(),
		"transforms.InsertStaticValue1.static.value":    model.InsertStaticValue1.ValueString(),
		"transforms.InsertStaticKey2.static.field":      model.InsertStaticKeyField2.ValueString(),
		"transforms.InsertStaticKey2.static.value":      model.InsertStaticKeyValue2.ValueString(),
		"transforms.InsertStaticValue2.static.field":    model.InsertStaticValueField2.ValueString(),
		"transforms.InsertStaticValue2.static.value":    model.InsertStaticValue2.ValueString(),
	}

	if !model.ColumnIncludeList.IsNull() {
		configMap["column.include.list.toggled"] = true
	} else if !model.ColumnExcludeList.IsNull() {
		configMap["column.include.list.toggled"] = false
	}
	configMap["column.include.list.user.defined"] = model.ColumnIncludeList.ValueStringPointer()
	configMap["column.exclude.list.user.defined"] = model.ColumnExcludeList.ValueStringPointer()

	return configMap, nil
}

func (r *SourcePostgreSQLResource) configMap2Model(cfg map[string]any, model *SourcePostgreSQLResourceModel) {
	// Copy the config map to the model
	model.DatabaseHostname = helper.GetTfCfgString(cfg, "database.hostname.user.defined")
	model.DatabasePort = helper.GetTfCfgInt64(cfg, "database.port.user.defined")
	model.DatabaseUser = helper.GetTfCfgString(cfg, "database.user")
	model.DatabasePassword = helper.GetTfCfgString(cfg, "database.password")
	model.DatabaseDbname = helper.GetTfCfgString(cfg, "database.dbname")
	model.SnapshotReadOnly = helper.GetTfCfgString(cfg, "snapshot.read.only.user.defined")
	model.DatabaseSSLMode = helper.GetTfCfgString(cfg, "database.sslmode")
	model.SchemaIncludeList = helper.GetTfCfgString(cfg, "schema.include.list")
	model.TableIncludeList = helper.GetTfCfgString(cfg, "table.include.list.user.defined")
	model.SignalDataCollectionSchemaOrDatabase = helper.GetTfCfgString(cfg, "signal.data.collection.schema.or.database")
	model.ColumnIncludeList = helper.GetTfCfgString(cfg, "column.include.list.user.defined")
	model.ColumnExcludeList = helper.GetTfCfgString(cfg, "column.exclude.list.user.defined")
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
	model.PredicatesIsTopicToEnrichPattern = helper.GetTfCfgString(cfg, "predicates.IsTopicToEnrich.pattern")
	model.InsertStaticKeyField1 = helper.GetTfCfgString(cfg, "transforms.InsertStaticKey1.static.field")
	model.InsertStaticKeyValue1 = helper.GetTfCfgString(cfg, "transforms.InsertStaticKey1.static.value")
	model.InsertStaticValueField1 = helper.GetTfCfgString(cfg, "transforms.InsertStaticValue1.static.field")
	model.InsertStaticValue1 = helper.GetTfCfgString(cfg, "transforms.InsertStaticValue1.static.value")
	model.InsertStaticKeyField2 = helper.GetTfCfgString(cfg, "transforms.InsertStaticKey2.static.field")
	model.InsertStaticKeyValue2 = helper.GetTfCfgString(cfg, "transforms.InsertStaticKey2.static.value")
	model.InsertStaticValueField2 = helper.GetTfCfgString(cfg, "transforms.InsertStaticValue2.static.field")
	model.InsertStaticValue2 = helper.GetTfCfgString(cfg, "transforms.InsertStaticValue2.static.value")
}
