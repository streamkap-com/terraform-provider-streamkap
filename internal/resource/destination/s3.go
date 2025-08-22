package destination

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
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
	_ res.Resource                = &DestinationS3Resource{}
	_ res.ResourceWithConfigure   = &DestinationS3Resource{}
	_ res.ResourceWithImportState = &DestinationS3Resource{}
)

func NewDestinationS3Resource() res.Resource {
	return &DestinationS3Resource{connector_code: "s3"}
}

// DestinationS3Resource defines the resource implementation.
type DestinationS3Resource struct {
	client         api.StreamkapAPI
	connector_code string
}

// DestinationS3ResourceModel describes the resource data model.
type DestinationS3ResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Connector        types.String `tfsdk:"connector"`
	AWSAccessKeyID   types.String `tfsdk:"aws_access_key"`
	AWSSecretKeyID   types.String `tfsdk:"aws_secret_key"`
	Region           types.String `tfsdk:"aws_region"`
	BucketName       types.String `tfsdk:"bucket_name"`
	Format           types.String `tfsdk:"format"`
	FilenameTemplate types.String `tfsdk:"filename_template"`
	FilenamePrefix   types.String `tfsdk:"filename_prefix"`
	CompressionType  types.String `tfsdk:"compression_type"`
	OutputFields     types.List   `tfsdk:"output_fields"`
}

func (r *DestinationS3Resource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination_s3"
}

func (r *DestinationS3Resource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Destination S3 resource",
		MarkdownDescription: "Destination S3 resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Destination S3 identifier",
				MarkdownDescription: "Destination S3 identifier",
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
			"aws_access_key": schema.StringAttribute{
				Required:            true,
				Description:         "The AWS Access Key ID used to connect to S3.",
				MarkdownDescription: "The AWS Access Key ID used to connect to S3.",
			},
			"aws_secret_key": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "The AWS Secret Access Key used to connect to S3.",
				MarkdownDescription: "The AWS Secret Access Key used to connect to S3.",
			},
			"aws_region": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("us-west-2"),
				Description:         "The AWS region to be used",
				MarkdownDescription: "The AWS region to be used",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"ap-south-1",
						"eu-west-2",
						"eu-west-1",
						"ap-northeast-2",
						"ap-northeast-1",
						"ca-central-1",
						"sa-east-1",
						"cn-north-1",
						"us-gov-west-1",
						"ap-southeast-1",
						"ap-southeast-2",
						"eu-central-1",
						"us-east-1",
						"us-east-2",
						"us-west-1",
						"us-west-2",
					),
				},
			},
			"bucket_name": schema.StringAttribute{
				Required:            true,
				Description:         "The S3 Bucket to use.",
				MarkdownDescription: "The S3 Bucket to use.",
			},
			"format": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("JSON Array"),
				Description:         "The format to use when writing data to the store.",
				MarkdownDescription: "The format to use when writing data to the store.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"JSON Lines",
						"JSON Array",
						"Parquet",
					),
				},
			},
			"filename_template": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("{{topic}}-{{partition}}-{{start_offset}}"),
				Description:         "The format of the filename. See documentation for more information about formatting options.",
				MarkdownDescription: "The format of the filename. See documentation for more information about formatting options.",
			},
			"filename_prefix": schema.StringAttribute{
				Optional:            true,
				Description:         "Prefix for the filename. Prefixes can be used to specify a directory for the file (e.g. dir1/dir2/).",
				MarkdownDescription: "Prefix for the filename. Prefixes can be used to specify a directory for the file (e.g. dir1/dir2/).",
			},
			"compression_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("gzip"),
				Description:         "Compression type for files written to S3.",
				MarkdownDescription: "Compression type for files written to S3.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"none",
						"gzip",
						"snappy",
						"zstd",
					),
				},
			},
			"output_fields": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{types.StringValue("value")})),
				Description:         "A comma separated list of fields to include in output? Options to include key, offset, timestamp, value, headers.",
				MarkdownDescription: "A comma separated list of fields to include in output? Options to include key, offset, timestamp, value, headers.",
				Validators: []validator.List{
					listvalidator.ValueStringsAre(validator.String(
						stringvalidator.OneOf(
							"key",
							"offset",
							"timestamp",
							"value",
							"headers",
						),
					)),
				},
			},
		},
	}
}

func (r *DestinationS3Resource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Destination S3 Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *DestinationS3Resource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan DestinationS3ResourceModel

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
			"Error creating S3 destination",
			fmt.Sprintf("Unable to create S3 destination, got error: %s", err),
		)
		return
	}
	tflog.Debug(ctx, "Post CREATE ===> config: "+fmt.Sprintf("%+v", destination.Config))

	plan.ID = types.StringValue(destination.ID)
	plan.Name = types.StringValue(destination.Name)
	plan.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(destination.Config, &plan, ctx)
	tflog.Debug(ctx, "Post CREATE ===> plan: "+fmt.Sprintf("%+v", plan))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DestinationS3Resource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state DestinationS3ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := state.ID.ValueString()
	destination, err := r.client.GetDestination(ctx, destinationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading S3 destination",
			fmt.Sprintf("Unable to read S3 destination, got error: %s", err),
		)
		return
	}
	if destination == nil {
		resp.Diagnostics.AddError(
			"Error reading S3 destination",
			fmt.Sprintf("S3 destination %s does not exist", destinationID),
		)
		return
	}

	state.Name = types.StringValue(destination.Name)
	state.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(destination.Config, &state, ctx)
	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", state))

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DestinationS3Resource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan DestinationS3ResourceModel

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
			"Error updating S3 destination",
			fmt.Sprintf("Unable to update S3 destination, got error: %s", err),
		)
		return
	}

	// Update resource state with updated items
	plan.Name = types.StringValue(destination.Name)
	plan.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(destination.Config, &plan, ctx)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *DestinationS3Resource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state DestinationS3ResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDestination(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting S3 destination",
			fmt.Sprintf("Unable to delete S3 destination, got error: %s", err),
		)
		return
	}
}

func (r *DestinationS3Resource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *DestinationS3Resource) model2ConfigMap(model DestinationS3ResourceModel) map[string]any {
	configMap := map[string]any{
		"aws.access.key.id":                 model.AWSAccessKeyID.ValueString(),
		"aws.secret.access.key":             model.AWSSecretKeyID.ValueString(),
		"aws.s3.region":                     model.Region.ValueString(),
		"aws.s3.bucket.name":                model.BucketName.ValueString(),
		"format.user.defined":               model.Format.ValueString(),
		"file.name.template":                model.FilenameTemplate.ValueString(),
		"file.name.prefix":                  model.FilenamePrefix.ValueString(),
		"file.compression.type":             model.CompressionType.ValueString(),
		"format.output.fields.user.defined": model.OutputFields,
	}

	return configMap
}

func (r *DestinationS3Resource) configMap2Model(cfg map[string]any, model *DestinationS3ResourceModel, ctx context.Context) {
	// Copy the config map to the model
	model.AWSAccessKeyID = helper.GetTfCfgString(cfg, "aws.access.key.id")
	model.AWSSecretKeyID = helper.GetTfCfgString(cfg, "aws.secret.access.key")
	model.Region = helper.GetTfCfgString(cfg, "aws.s3.region")
	model.BucketName = helper.GetTfCfgString(cfg, "aws.s3.bucket.name")
	model.Format = helper.GetTfCfgString(cfg, "aws.format.user.defined")
	model.FilenameTemplate = helper.GetTfCfgString(cfg, "file.name.template")
	model.FilenamePrefix = helper.GetTfCfgString(cfg, "file.name.prefix")
	model.CompressionType = helper.GetTfCfgString(cfg, "file.compression.type")
	model.OutputFields = helper.GetTfCfgListString(ctx, cfg, "format.output.fields.user.defined")
}
