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
	_ res.Resource                = &DestinationSnowflakeResource{}
	_ res.ResourceWithConfigure   = &DestinationSnowflakeResource{}
	_ res.ResourceWithImportState = &DestinationSnowflakeResource{}
)

func NewDestinationSnowflakeResource() res.Resource {
	return &DestinationSnowflakeResource{connector_code: "snowflake"}
}

// DestinationSnowflakeResource defines the resource implementation.
type DestinationSnowflakeResource struct {
	client         api.StreamkapAPI
	connector_code string
}

// DestinationSnowflakeResourceModel describes the resource data model.
type DestinationSnowflakeResourceModel struct {
	ID                            types.String `tfsdk:"id"`
	Name                          types.String `tfsdk:"name"`
	Connector                     types.String `tfsdk:"connector"`
	SnowflakeUrlName              types.String `tfsdk:"snowflake_url_name"`
	SnowflakeUserName             types.String `tfsdk:"snowflake_user_name"`
	SnowflakePrivateKey           types.String `tfsdk:"snowflake_private_key"`
	SnowflakePrivateKeyPassphrase types.String `tfsdk:"snowflake_private_key_passphrase"`
	Sfwarehouse                   types.String `tfsdk:"sfwarehouse"`
	SnowflakeDatabaseName         types.String `tfsdk:"snowflake_database_name"`
	SnowflakeSchemaName           types.String `tfsdk:"snowflake_schema_name"`
	SnowflakeRoleName             types.String `tfsdk:"snowflake_role_name"`
	IngestionMode                 types.String `tfsdk:"ingestion_mode"`
	HardDelete                    types.Bool   `tfsdk:"hard_delete"`
	SchemaEvolution               types.String `tfsdk:"schema_evolution"`
	UseHybridTables               types.Bool   `tfsdk:"use_hybrid_tables"`
	ApplyDynamicTableScript       types.Bool   `tfsdk:"apply_dynamic_table_script"`
	DynamicTableTargetLag         types.Int64  `tfsdk:"dynamic_table_target_lag"`
	CleanupTaskSchedule           types.Int64  `tfsdk:"cleanup_task_schedule"`
	CreateSQLExecute              types.String `tfsdk:"create_sql_execute"`
	CreateSQLData                 types.String `tfsdk:"create_sql_data"`
	SQLTableName                  types.String `tfsdk:"sql_table_name"`
}

func (r *DestinationSnowflakeResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination_snowflake"
}

func (r *DestinationSnowflakeResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Destination Snowflake resource",
		MarkdownDescription: "Destination Snowflake resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Destination Snowflake identifier",
				MarkdownDescription: "Destination Snowflake identifier",
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
			"snowflake_url_name": schema.StringAttribute{
				Required:            true,
				Description:         "The URL for accessing your Snowflake account. This URL must include your account identifier. Note that the protocol (https://) and port number are optional.",
				MarkdownDescription: "The URL for accessing your Snowflake account. This URL must include your account identifier. Note that the protocol (https://) and port number are optional.",
			},
			"snowflake_user_name": schema.StringAttribute{
				Required:            true,
				Description:         "User login name for the Snowflake account.",
				MarkdownDescription: "User login name for the Snowflake account.",
			},
			"snowflake_private_key": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "The private key to authenticate the user. Include only the key, not the header or footer. If the key is split across multiple lines, remove the line breaks.",
				MarkdownDescription: "The private key to authenticate the user. Include only the key, not the header or footer. If the key is split across multiple lines, remove the line breaks.",
			},
			"snowflake_private_key_passphrase": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				Description:         "If the value is not empty, this phrase is used to try to decrypt the private key.",
				MarkdownDescription: "If the value is not empty, this phrase is used to try to decrypt the private key.",
			},
			"sfwarehouse": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("STREAMKAP_WH"),
				Description:         "The name of the Snowflake warehouse.",
				MarkdownDescription: "The name of the Snowflake warehouse.",
			},
			"snowflake_database_name": schema.StringAttribute{
				Required:            true,
				Description:         "The name of the database that contains the table to insert rows into.",
				MarkdownDescription: "The name of the database that contains the table to insert rows into.",
			},
			"snowflake_schema_name": schema.StringAttribute{
				Required:            true,
				Description:         "The name of the schema that contains the table to insert rows into.",
				MarkdownDescription: "The name of the schema that contains the table to insert rows into.",
			},
			"snowflake_role_name": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("STREAMKAP_ROLE"),
				Description:         "The name of an existing role with necessary privileges (for Streamkap) assigned to the Username.",
				MarkdownDescription: "The name of an existing role with necessary privileges (for Streamkap) assigned to the Username.",
			},
			"ingestion_mode": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("append"),
				Description:         "`upsert` or `append` modes are available",
				MarkdownDescription: "`upsert` or `append` modes are available",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"upsert",
						"append",
					),
				},
			},
			"hard_delete": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Specifies whether the connector processes DELETE or tombstone events and removes the corresponding row from the database (applies to `upsert` only)",
				MarkdownDescription: "Specifies whether the connector processes DELETE or tombstone events and removes the corresponding row from the database (applies to `upsert` only)",
			},
			"schema_evolution": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("basic"),
				Description:         "Controls how schema evolution is handled by the sink connector. For pipelines with pre-created destination tables, set to `none`",
				MarkdownDescription: "Controls how schema evolution is handled by the sink connector. For pipelines with pre-created destination tables, set to `none`",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"none",
						"basic",
					),
				},
			},
			"use_hybrid_tables": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Specifies whether the connector should create Hybrid Tables (applies to `upsert` only)",
				MarkdownDescription: "Specifies whether the connector should create Hybrid Tables (applies to `upsert` only)",
			},
			"apply_dynamic_table_script": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Specifies whether the connector should create Dyanmic Tables & Cleanup Task (applies to `append` mode only)",
				MarkdownDescription: "Specifies whether the connector should create Dyanmic Tables & Cleanup Task (applies to `append` mode only)",
			},
			"dynamic_table_target_lag": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(15),
				Description:         "Target lag for dynamic tables in minutes (applies to `append` mode only)",
				MarkdownDescription: "Target lag for dynamic tables in minutes (applies to `append` mode only)",
				Validators: []validator.Int64{
					int64validator.Between(0, 2147483647),
				},
			},
			"cleanup_task_schedule": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(60),
				Description:         "Schedule for cleanup task in minutes (applies to `append` mode only)",
				MarkdownDescription: "Schedule for cleanup task in minutes (applies to `append` mode only)",
				Validators: []validator.Int64{
					int64validator.Between(0, 2147483647),
				},
			},
			"create_sql_execute": schema.StringAttribute{
				Computed: true,
				Optional: true,
				Default: stringdefault.StaticString("CREATE OR REPLACE DYNAMIC TABLE {{table}}_DT TARGET_LAG='{{targetLag}} minutes' WAREHOUSE={{warehouse}} " +
					"AS SELECT * EXCLUDE dedupe_id FROM( SELECT *, ROW_NUMBER() OVER (PARTITION BY {{primaryKeyColumns}} ORDER BY _streamkap_ts_ms DESC, _streamkap_offset DESC) AS dedupe_id " +
					"FROM \"{{table}}\" ) WHERE dedupe_id = 1 AND __deleted = 'false';\n" +
					"CREATE OR REPLACE TASK {{table}}_CT WAREHOUSE={{warehouse}} SCHEDULE='{{schedule}} minutes' TASK_AUTO_RETRY_ATTEMPTS=3 ALLOW_OVERLAPPING_EXECUTION=FALSE " +
					"AS DELETE FROM \"{{table}}\" WHERE NOT EXISTS ( SELECT 1 FROM ( SELECT {{primaryKeyColumns}}, MAX(_streamkap_ts_ms) AS max_timestamp FROM \"{{table}}\" GROUP BY {{primaryKeyColumns}} ) AS subquery " +
					"WHERE {{{keyColumnsAndCondition}}} AND \"{{table}}\"._streamkap_ts_ms = subquery.max_timestamp);\nALTER TASK {{table}}_CT RESUME"),
				Description:         "Custom SQL mustache template to be run the first time a record is streamed for each table.",
				MarkdownDescription: "Custom SQL mustache template to be run the first time a record is streamed for each table.",
			},
			"create_sql_data": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("{\n    \"TABLE_DATA\": {\n        \"my-table-name\": {\n            \"someTableSpecificKey\": \"someTableSpecificValue\"\n        }\n    }\n}"),
				Description:         "Custom SQL mustache template input JSON data. Use TABLE_DATA dictionary to set table specific data.",
				MarkdownDescription: "Custom SQL mustache template input JSON data. Use TABLE_DATA dictionary to set table specific data.",
			},
			"sql_table_name": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("{{table}}_DT"),
				Description:         "Dynamic Table Name mustache template. Can be used as `{{`{`}}{{`{`}}dynamicTableName{{`}`}}{{`}`}}` in dynamic table creation SQL. It can use input JSON data for more complex mappings and logic.",
				MarkdownDescription: "Dynamic Table Name mustache template. Can be used as `{{`{`}}{{`{`}}dynamicTableName{{`}`}}{{`}`}}` in dynamic table creation SQL. It can use input JSON data for more complex mappings and logic.",
			},
		},
	}
}

func (r *DestinationSnowflakeResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Destination Snowflake Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *DestinationSnowflakeResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan DestinationSnowflakeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	plan.Connector = types.StringValue(r.connector_code)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Pre CREATE ===> plan: "+fmt.Sprintf("%+v", plan))
	config := r.model2ConfigMap(ctx, plan)

	tflog.Debug(ctx, "Pre CREATE ===> config: "+fmt.Sprintf("%+v", config))
	destination, err := r.client.CreateDestination(ctx, api.Destination{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Snowflake destination",
			fmt.Sprintf("Unable to create Snowflake destination, got error: %s", err),
		)
		return
	}
	tflog.Debug(ctx, "Post CREATE ===> config: "+fmt.Sprintf("%+v", destination.Config))

	plan.ID = types.StringValue(destination.ID)
	plan.Name = types.StringValue(destination.Name)
	plan.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(ctx, destination.Config, &plan)
	tflog.Debug(ctx, "Post CREATE ===> plan: "+fmt.Sprintf("%+v", plan))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DestinationSnowflakeResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state DestinationSnowflakeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := state.ID.ValueString()
	destination, err := r.client.GetDestination(ctx, destinationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Snowflake destination",
			fmt.Sprintf("Unable to read Snowflake destination, got error: %s", err),
		)
		return
	}
	if destination == nil {
		resp.Diagnostics.AddError(
			"Error reading Snowflake destination",
			fmt.Sprintf("Snowflake destination %s does not exist", destinationID),
		)
		return
	}

	state.Name = types.StringValue(destination.Name)
	state.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(ctx, destination.Config, &state)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DestinationSnowflakeResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan DestinationSnowflakeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Pre UPDATE ===> plan: "+fmt.Sprintf("%+v", plan))
	config := r.model2ConfigMap(ctx, plan)

	tflog.Debug(ctx, "Pre UPDATE ===> config: "+fmt.Sprintf("%+v", config))
	destination, err := r.client.UpdateDestination(ctx, plan.ID.ValueString(), api.Destination{
		Name:      plan.Name.ValueString(),
		Connector: plan.Connector.ValueString(),
		Config:    config,
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating Snowflake destination",
			fmt.Sprintf("Unable to update Snowflake destination, got error: %s", err),
		)
		return
	}
	tflog.Debug(ctx, "Post UPDATE ===> config: "+fmt.Sprintf("%+v", destination.Config))

	// Update resource state with updated items
	plan.Name = types.StringValue(destination.Name)
	plan.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(ctx, destination.Config, &plan)
	tflog.Debug(ctx, "Post UPDATE ===> plan: "+fmt.Sprintf("%+v", plan))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DestinationSnowflakeResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state DestinationSnowflakeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDestination(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Snowflake destination",
			fmt.Sprintf("Unable to delete Snowflake destination, got error: %s", err),
		)
		return
	}
}

func (r *DestinationSnowflakeResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *DestinationSnowflakeResource) model2ConfigMap(_ context.Context, model DestinationSnowflakeResourceModel) map[string]any {
	configMap := map[string]any{
		"snowflake.url.name":                       model.SnowflakeUrlName.ValueString(),
		"snowflake.user.name":                      model.SnowflakeUserName.ValueString(),
		"snowflake.private.key":                    model.SnowflakePrivateKey.ValueString(),
		"snowflake.private.key.passphrase.secured": true,
		"snowflake.private.key.passphrase":         model.SnowflakePrivateKeyPassphrase.ValueStringPointer(),
		"sfwarehouse":                              model.Sfwarehouse.ValueString(),
		"snowflake.database.name":                  model.SnowflakeDatabaseName.ValueString(),
		"snowflake.schema.name":                    model.SnowflakeSchemaName.ValueString(),
		"snowflake.role.name":                      model.SnowflakeRoleName.ValueString(),
		"ingestion.mode":                           model.IngestionMode.ValueString(),
		"hard.delete":                              model.HardDelete.ValueBool(),
		"schema.evolution":                         model.SchemaEvolution.ValueString(),
		"use.hybrid.tables":                        model.UseHybridTables.ValueBool(),
		"apply.dynamic.table.script":               model.ApplyDynamicTableScript.ValueBool(),
		"dynamic.table.target.lag":                 model.DynamicTableTargetLag.ValueInt64(),
		"cleanup.task.schedule":                    model.CleanupTaskSchedule.ValueInt64(),
		"create.sql.execute":                       model.CreateSQLExecute.ValueString(),
		"create.sql.data":                          model.CreateSQLData.ValueString(),
		"sql.table.name":                           model.SQLTableName.ValueString(),
	}

	if model.SnowflakePrivateKeyPassphrase.IsNull() {
		configMap["snowflake.private.key.passphrase.secured"] = false
	}

	return configMap
}

func (r *DestinationSnowflakeResource) configMap2Model(_ context.Context, cfg map[string]any, model *DestinationSnowflakeResourceModel) {
	// Copy the config map to the model
	model.SnowflakeUrlName = helper.GetTfCfgString(cfg, "snowflake.url.name")
	model.SnowflakeUserName = helper.GetTfCfgString(cfg, "snowflake.user.name")
	model.SnowflakePrivateKey = helper.GetTfCfgString(cfg, "snowflake.private.key")
	model.SnowflakePrivateKeyPassphrase = helper.GetTfCfgString(cfg, "snowflake.private.key.passphrase")
	model.Sfwarehouse = helper.GetTfCfgString(cfg, "sfwarehouse")
	model.SnowflakeDatabaseName = helper.GetTfCfgString(cfg, "snowflake.database.name")
	model.SnowflakeSchemaName = helper.GetTfCfgString(cfg, "snowflake.schema.name")
	model.SnowflakeRoleName = helper.GetTfCfgString(cfg, "snowflake.role.name")
	model.IngestionMode = helper.GetTfCfgString(cfg, "ingestion.mode")
	model.HardDelete = helper.GetTfCfgBool(cfg, "hard.delete")
	model.SchemaEvolution = helper.GetTfCfgString(cfg, "schema.evolution")
	model.UseHybridTables = helper.GetTfCfgBool(cfg, "use.hybrid.tables")
	model.ApplyDynamicTableScript = helper.GetTfCfgBool(cfg, "apply.dynamic.table.script")
	model.DynamicTableTargetLag = helper.GetTfCfgInt64(cfg, "dynamic.table.target.lag")
	model.CleanupTaskSchedule = helper.GetTfCfgInt64(cfg, "cleanup.task.schedule")
	model.CreateSQLExecute = helper.GetTfCfgString(cfg, "create.sql.execute")
	model.CreateSQLData = helper.GetTfCfgString(cfg, "create.sql.data")
	model.SQLTableName = helper.GetTfCfgString(cfg, "sql.table.name")
}
