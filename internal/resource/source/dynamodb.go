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
	_ res.Resource                = &SourceDynamoDBResource{}
	_ res.ResourceWithConfigure   = &SourceDynamoDBResource{}
	_ res.ResourceWithImportState = &SourceDynamoDBResource{}
)

func NewSourceDynamoDBResource() res.Resource {
	return &SourceDynamoDBResource{connector_code: "dynamodb"}
}

// SourceDynamoDBResource defines the resource implementation.
type SourceDynamoDBResource struct {
	client         api.StreamkapAPI
	connector_code string
}

// SourceDynamoDBResourceModel describes the resource data model.
type SourceDynamoDBResourceModel struct {
	ID                            types.String `tfsdk:"id"`
	Name                          types.String `tfsdk:"name"`
	Connector                     types.String `tfsdk:"connector"`
	AWSRegion                     types.String `tfsdk:"aws_region"`
	AWSAccessKeyID                types.String `tfsdk:"aws_access_key_id"`
	AWSSecretKey                  types.String `tfsdk:"aws_secret_key"`
	S3ExportBucketName            types.String `tfsdk:"s3_export_bucket_name"`
	TableIncludeListUserDefined   types.String `tfsdk:"table_include_list_user_defined"`
	BatchSize                     types.Int64  `tfsdk:"batch_size"`
	DynamoDBServiceEndpoint       types.String `tfsdk:"dynamodb_service_endpoint"`
	PollTimeoutMS                 types.Int64  `tfsdk:"poll_timeout_ms"`
	IncrementalSnapshotChunkSize  types.Int64  `tfsdk:"incremental_snapshot_chunk_size"`
	IncrementalSnapshotMaxThreads types.Int64  `tfsdk:"incremental_snapshot_max_threads"`
	IncrementalSnapshotIntervalMS types.Int64  `tfsdk:"incremental_snapshot_interval_ms"`
	FullExportExpirationTimeMS    types.Int64  `tfsdk:"full_export_expiration_time_ms"`
	SignalKafkaPollTimeoutMS      types.Int64  `tfsdk:"signal_kafka_poll_timeout_ms"`
}

func (r *SourceDynamoDBResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_dynamodb"
}

func (r *SourceDynamoDBResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Source DynamoDB resource",
		MarkdownDescription: "Source DynamoDB resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Source DynamoDB identifier",
				MarkdownDescription: "Source DynamoDB identifier",
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
			"aws_region": schema.StringAttribute{
				Required:            true,
				Description:         "AWS Region",
				MarkdownDescription: "AWS Region",
			},
			"aws_access_key_id": schema.StringAttribute{
				Required:            true,
				Description:         "AWS Access Key ID",
				MarkdownDescription: "AWS Access Key ID",
			},
			"aws_secret_key": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "AWS Secret Key",
				MarkdownDescription: "AWS Secret Key",
			},
			"s3_export_bucket_name": schema.StringAttribute{
				Required:            true,
				Description:         "used for backfill (snapshot)",
				MarkdownDescription: "used for backfill (snapshot)",
			},
			"table_include_list_user_defined": schema.StringAttribute{
				Required:            true,
				Description:         "Source tables to sync.",
				MarkdownDescription: "Source tables to sync.",
			},
			"batch_size": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(1024),
				Description:         "Batch size to fetch records.",
				MarkdownDescription: "Batch size to fetch records.",
			},
			"dynamodb_service_endpoint": schema.StringAttribute{
				Required:            true,
				Description:         "Dynamodb Service Endpoint (optional)",
				MarkdownDescription: "Dynamodb Service Endpoint (optional)",
			},
			"poll_timeout_ms": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(1000),
				Description:         "Poll Timeout (ms)",
				MarkdownDescription: "Poll Timeout (ms)",
			},
			"incremental_snapshot_chunk_size": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(32768),
				Description:         "Incremental snapshot chunk size",
				MarkdownDescription: "Incremental snapshot chunk size",
			},
			"incremental_snapshot_max_threads": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(8),
				Description:         "Incremental snapshot max threads",
				MarkdownDescription: "Incremental snapshot max threads",
			},
			"incremental_snapshot_interval_ms": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(8),
				Description:         "Incremental snapshot chunk size (ms)",
				MarkdownDescription: "Incremental snapshot chunk size (ms)",
			},
			"full_export_expiration_time_ms": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(86400000),
				Description:         "Full Export Expiration Time (ms)",
				MarkdownDescription: "Full Export Expiration Time (ms)",
			},
			"signal_kafka_poll_timeout_ms": schema.Int64Attribute{
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(1000),
				Description:         "Signal Kafka Poll Timeout (ms)",
				MarkdownDescription: "Signal Kafka Poll Timeout (ms)",
			},
		},
	}
}

func (r *SourceDynamoDBResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Source DynamoDB Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *SourceDynamoDBResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan SourceDynamoDBResourceModel

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
			"Error creating DynamoDB source",
			fmt.Sprintf("Unable to create DynamoDB source, got error: %s", err),
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

func (r *SourceDynamoDBResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state SourceDynamoDBResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	sourceID := state.ID.ValueString()
	source, err := r.client.GetSource(ctx, sourceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading DynamoDB source",
			fmt.Sprintf("Unable to read DynamoDB source, got error: %s", err),
		)
		return
	}
	if source == nil {
		resp.Diagnostics.AddError(
			"Error reading DynamoDB source",
			fmt.Sprintf("DynamoDB source %s does not exist", sourceID),
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

func (r *SourceDynamoDBResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan SourceDynamoDBResourceModel

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
			"Error updating DynamoDB source",
			fmt.Sprintf("Unable to update DynamoDB source, got error: %s", err),
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

func (r *SourceDynamoDBResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state SourceDynamoDBResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting DynamoDB source",
			fmt.Sprintf("Unable to delete DynamoDB source, got error: %s", err),
		)
		return
	}
}

func (r *SourceDynamoDBResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *SourceDynamoDBResource) configMapFromModel(model SourceDynamoDBResourceModel) map[string]any {
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
		"column.include.list.user.defined":                  model.ColumnIncludeList.ValueString(),
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

func (r *SourceDynamoDBResource) modelFromConfigMap(cfg map[string]any, model *SourceDynamoDBResourceModel) {
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
	model.ColumnIncludeList = helper.GetTfCfgString(cfg, "column.include.list.user.defined")
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
