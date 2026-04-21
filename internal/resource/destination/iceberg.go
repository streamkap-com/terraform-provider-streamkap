package destination

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	_ res.Resource                = &DestinationIcebergResource{}
	_ res.ResourceWithConfigure   = &DestinationIcebergResource{}
	_ res.ResourceWithImportState = &DestinationIcebergResource{}
)

func NewDestinationIcebergResource() res.Resource {
	return &DestinationIcebergResource{connector_code: "iceberg"}
}

// DestinationIcebergResource defines the resource implementation.
type DestinationIcebergResource struct {
	client         api.StreamkapAPI
	connector_code string
}

// DestinationIcebergResourceModel describes the resource data model.
type DestinationIcebergResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Connector        types.String `tfsdk:"connector"`
	CatalogType      types.String `tfsdk:"catalog_type"`
	CatalogName      types.String `tfsdk:"catalog_name"`
	CatalogURI       types.String `tfsdk:"catalog_uri"`
	AWSAccessKeyID   types.String `tfsdk:"aws_access_key"`
	AWSSecretKeyID   types.String `tfsdk:"aws_secret_key"`
	IAMRole          types.String `tfsdk:"aws_iam_role"`
	Region           types.String `tfsdk:"aws_region"`
	BucketPath       types.String `tfsdk:"bucket_path"`
	Schema           types.String `tfsdk:"schema"`
	InsertMode       types.String `tfsdk:"insert_mode"`
	PrimaryKeyFields types.String `tfsdk:"primary_key_fields"`
	QuoteIdentifiers types.Bool   `tfsdk:"quote_identifiers"`
}

func (r *DestinationIcebergResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination_iceberg"
}

func (r *DestinationIcebergResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Destination Iceberg resource",
		MarkdownDescription: "Destination Iceberg resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Destination Iceberg identifier",
				MarkdownDescription: "Destination Iceberg identifier",
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
			"catalog_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("rest"),
				Description:         "Type of Iceberg catalog.",
				MarkdownDescription: "Type of Iceberg catalog.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"rest", "hive", "glue",
					),
				},
			},
			"catalog_name": schema.StringAttribute{
				Optional:            true,
				Description:         "Iceberg catalog name. Required for rest and hive.",
				MarkdownDescription: "Iceberg catalog name. Required for rest and hive.",
			},
			"catalog_uri": schema.StringAttribute{
				Optional:            true,
				Description:         "Iceberg catalog uri. Required for rest and hive.",
				MarkdownDescription: "Iceberg catalog uri. Required for rest and hive.",
			},
			"aws_access_key": schema.StringAttribute{
				Optional:            true,
				Description:         "The AWS Access Key ID used to connect to S3. Required for rest and hive.",
				MarkdownDescription: "The AWS Access Key ID used to connect to S3. Required for rest and hive.",
			},
			"aws_secret_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				Description:         "The AWS Secret Access Key used to connect to Iceberg. Required for rest and hive.",
				MarkdownDescription: "The AWS Secret Access Key used to connect to Iceberg. Required for rest and hive.",
			},
			"aws_iam_role": schema.StringAttribute{
				Optional:            true,
				Description:         "AWS IAM role (e.g., arn:aws:iam:::role/). Required for glue.",
				MarkdownDescription: "AWS IAM role (e.g., arn:aws:iam:::role/). Required for glue.",
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
			"bucket_path": schema.StringAttribute{
				Required:            true,
				Description:         "The S3 Bucket path to use.",
				MarkdownDescription: "The S3 Bucket path to use.",
			},
			"schema": schema.StringAttribute{
				Required:            true,
				Description:         "Name of the database schema that contains the table (e.g., public, sales, analytics)..",
				MarkdownDescription: "Name of the database schema that contains the table (e.g., public, sales, analytics)..",
			},
			"insert_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("insert"),
				Description:         "Specifies the strategy used to insert events into the database",
				MarkdownDescription: "Specifies the strategy used to insert events into the database",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"insert",
						"upsert",
					),
				},
			},
			"primary_key_fields": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				Description:         "Optional (upsert). A comma-separated list of field names to use as record identifiers when key fields are not present in Kafka messages",
				MarkdownDescription: "Optional (upsert). A comma-separated list of field names to use as record identifiers when key fields are not present in Kafka messages",
			},
			"quote_identifiers": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				Description:         "Whether to quote identifiers in SQL statements",
				MarkdownDescription: "Whether to quote identifiers in SQL statements",
			},
		},
	}
}

func (r *DestinationIcebergResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Destination Iceberg Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *DestinationIcebergResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan DestinationIcebergResourceModel

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
			"Error creating Iceberg destination",
			fmt.Sprintf("Unable to create Iceberg destination, got error: %s", err),
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

func (r *DestinationIcebergResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state DestinationIcebergResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := state.ID.ValueString()
	destination, err := r.client.GetDestination(ctx, destinationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Iceberg destination",
			fmt.Sprintf("Unable to read Iceberg destination, got error: %s", err),
		)
		return
	}
	if destination == nil {
		resp.Diagnostics.AddError(
			"Error reading Iceberg destination",
			fmt.Sprintf("Iceberg destination %s does not exist", destinationID),
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

func (r *DestinationIcebergResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan DestinationIcebergResourceModel

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
			"Error updating Iceberg destination",
			fmt.Sprintf("Unable to update Iceberg destination, got error: %s", err),
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

func (r *DestinationIcebergResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state DestinationIcebergResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDestination(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Iceberg destination",
			fmt.Sprintf("Unable to delete Iceberg destination, got error: %s", err),
		)
		return
	}
}

func (r *DestinationIcebergResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *DestinationIcebergResource) model2ConfigMap(model DestinationIcebergResourceModel) map[string]any {
	configMap := map[string]any{
		"iceberg.catalog.type":                       model.CatalogType.ValueString(),
		"iceberg.catalog.name":                       model.CatalogName.ValueString(),
		"iceberg.catalog.uri":                        model.CatalogURI.ValueString(),
		"iceberg.catalog.s3.access-key-id":           model.AWSAccessKeyID.ValueString(),
		"iceberg.catalog.s3.secret-access-key":       model.AWSSecretKeyID.ValueString(),
		"iceberg.catalog.client.assume-role.arn":     model.IAMRole.ValueString(),
		"iceberg.catalog.client.region.user.defined": model.Region.ValueString(),
		"iceberg.catalog.warehouse":                  model.BucketPath.ValueString(),
		"table.name.prefix":                          model.Schema.ValueString(),
		"insert.mode.user.defined":                   model.InsertMode.ValueString(),
		"iceberg.tables.default-id-columns":          model.PrimaryKeyFields.ValueString(),
		"quote.identifiers":                          model.QuoteIdentifiers.ValueBool(),
	}

	return configMap
}

func (r *DestinationIcebergResource) configMap2Model(cfg map[string]any, model *DestinationIcebergResourceModel, ctx context.Context) {
	// Copy the config map to the model
	model.CatalogType = helper.GetTfCfgString(cfg, "iceberg.catalog.type")
	model.CatalogName = helper.GetTfCfgString(cfg, "iceberg.catalog.name")
	model.CatalogURI = helper.GetTfCfgString(cfg, "iceberg.catalog.uri")
	model.AWSAccessKeyID = helper.GetTfCfgString(cfg, "iceberg.catalog.s3.access-key-id")
	model.AWSSecretKeyID = helper.GetTfCfgString(cfg, "iceberg.catalog.s3.secret-access-key")
	model.IAMRole = helper.GetTfCfgString(cfg, "iceberg.catalog.client.assume-role.arn")
	model.Region = helper.GetTfCfgString(cfg, "iceberg.catalog.client.region.user.defined")
	model.BucketPath = helper.GetTfCfgString(cfg, "iceberg.catalog.warehouse")
	model.Schema = helper.GetTfCfgString(cfg, "table.name.prefix")
	model.InsertMode = helper.GetTfCfgString(cfg, "insert.mode.user.defined")
	model.PrimaryKeyFields = helper.GetTfCfgString(cfg, "iceberg.tables.default-id-columns")
	model.QuoteIdentifiers = helper.GetTfCfgBool(cfg, "quote.identifiers")
}
