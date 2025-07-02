package source

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	FullExportExpirationTimeMS    types.Int64  `tfsdk:"full_export_expiration_time_ms"`
	SignalKafkaPollTimeoutMS      types.Int64  `tfsdk:"signal_kafka_poll_timeout_ms"`
	ArrayEncodingJson             types.Bool   `tfsdk:"array_encoding_json"`
	StructEncodingJson            types.Bool   `tfsdk:"struct_encoding_json"`
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
				Optional:            true,
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
			"array_encoding_json": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(true),
				Description:         "Force nested lists as JSON string",
				MarkdownDescription: "Force nested lists as JSON string",
			},
			"struct_encoding_json": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(true),
				Description:         "Force nested maps as JSON string",
				MarkdownDescription: "Force nested maps as JSON string",
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

	config := r.model2ConfigMap(plan)

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
	r.configMap2Model(source.Config, &plan)

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
	r.configMap2Model(source.Config, &state)
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
	config := r.model2ConfigMap(plan)

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
	r.configMap2Model(source.Config, &plan)

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
func (r *SourceDynamoDBResource) model2ConfigMap(model SourceDynamoDBResourceModel) map[string]any {
	return map[string]any{
		"aws.region":                       model.AWSRegion.ValueString(),
		"aws.access.key.id":                model.AWSAccessKeyID.ValueString(),
		"aws.secret.key":                   model.AWSSecretKey.ValueString(),
		"s3.export.bucket.name":            model.S3ExportBucketName.ValueString(),
		"table.include.list.user.defined":  model.TableIncludeListUserDefined.ValueString(),
		"batch.size":                       int(model.BatchSize.ValueInt64()),
		"dynamodb.service.endpoint":        model.DynamoDBServiceEndpoint.ValueStringPointer(),
		"poll.timeout.ms":                  int(model.PollTimeoutMS.ValueInt64()),
		"incremental.snapshot.chunk.size":  int(model.IncrementalSnapshotChunkSize.ValueInt64()),
		"incremental.snapshot.max.threads": int(model.IncrementalSnapshotMaxThreads.ValueInt64()),
		"full.export.expiration.time.ms":   int(model.FullExportExpirationTimeMS.ValueInt64()),
		"signal.kafka.poll.timeout.ms":     int(model.SignalKafkaPollTimeoutMS.ValueInt64()),
		"array.encoding.json":              model.ArrayEncodingJson.ValueBool(),
		"struct.encoding.json":             model.StructEncodingJson.ValueBool(),
	}
}

func (r *SourceDynamoDBResource) configMap2Model(cfg map[string]any, model *SourceDynamoDBResourceModel) {
	// Copy the config map to the model
	model.AWSRegion = helper.GetTfCfgString(cfg, "aws.region")
	model.AWSAccessKeyID = helper.GetTfCfgString(cfg, "aws.access.key.id")
	model.AWSSecretKey = helper.GetTfCfgString(cfg, "aws.secret.key")
	model.S3ExportBucketName = helper.GetTfCfgString(cfg, "s3.export.bucket.name")
	model.TableIncludeListUserDefined = helper.GetTfCfgString(cfg, "table.include.list.user.defined")
	model.BatchSize = helper.GetTfCfgInt64(cfg, "batch.size")
	model.DynamoDBServiceEndpoint = helper.GetTfCfgString(cfg, "dynamodb.service.endpoint")
	model.PollTimeoutMS = helper.GetTfCfgInt64(cfg, "poll.timeout.ms")
	model.IncrementalSnapshotChunkSize = helper.GetTfCfgInt64(cfg, "incremental.snapshot.chunk.size")
	model.IncrementalSnapshotMaxThreads = helper.GetTfCfgInt64(cfg, "incremental.snapshot.max.threads")
	model.FullExportExpirationTimeMS = helper.GetTfCfgInt64(cfg, "full.export.expiration.time.ms")
	model.SignalKafkaPollTimeoutMS = helper.GetTfCfgInt64(cfg, "signal.kafka.poll.timeout.ms")
	model.ArrayEncodingJson = helper.GetTfCfgBool(cfg, "array.encoding.json")
	model.StructEncodingJson = helper.GetTfCfgBool(cfg, "struct.encoding.json")
}
