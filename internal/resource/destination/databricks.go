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
	_ res.Resource                = &DestinationDatabricksResource{}
	_ res.ResourceWithConfigure   = &DestinationDatabricksResource{}
	_ res.ResourceWithImportState = &DestinationDatabricksResource{}
)

func NewDestinationDatabricksResource() res.Resource {
	return &DestinationDatabricksResource{connector_code: "databricks"}
}

// DestinationDatabricksResource defines the resource implementation.
type DestinationDatabricksResource struct {
	client         api.StreamkapAPI
	connector_code string
}

// DestinationDatabricksResourceModel describes the resource data model.
type DestinationDatabricksResourceModel struct {
	ID                            types.String            `tfsdk:"id"`
	Name                          types.String            `tfsdk:"name"`
	Connector                     types.String            `tfsdk:"connector"`
	ConnectionUrl                 types.String            `tfsdk:"connection_url"`
	DatabricksToken               types.String            `tfsdk:"databricks_token"`
	DatabricksCatalog             types.String            `tfsdk:"databricks_catalog"`
	TableNamePrefix               types.String            `tfsdk:"table_name_prefix"`
	IngestionMode                 types.String            `tfsdk:"ingestion_mode"`
	PartitionMode                 types.String            `tfsdk:"partition_mode"`
	HardDelete                    types.Bool              `tfsdk:"hard_delete"`
	SchemaEvolution               types.String            `tfsdk:"schema_evolution"`
	TasksMax                      types.Int64             `tfsdk:"tasks_max"`
}

func (r *DestinationDatabricksResource) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination_databricks"
}

func (r *DestinationDatabricksResource) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Destination Databricks resource",
		MarkdownDescription: "Destination Databricks resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Destination Databricks identifier",
				MarkdownDescription: "Destination Databricks identifier",
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
			"connection_url": schema.StringAttribute{
				Required:            true,
				Description:         "JDBC URL",
				MarkdownDescription: "JDBC URL",
			},
			"databricks_token": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "Token",
				MarkdownDescription: "Token",
			},
			"databricks_catalog": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("hive_metastore"),
				Description:         "Catalog Name. Make sure to change this to the correct cataog name",
				MarkdownDescription: "Catalog Name. Make sure to change this to the correct cataog name",
			},
			"table_name_prefix": schema.StringAttribute{
				Required:            true,
				Description:         "Schema for the associated table name",
				MarkdownDescription: "Schema for the associated table name",
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
			"partition_mode": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("by_topic"),
				Description:         "Partition tables or not",
				MarkdownDescription: "Partition tables or not",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"by_topic",
						"by_partition",
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
		},
	}
}

func (r *DestinationDatabricksResource) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Destination Databricks Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.client = client
}

func (r *DestinationDatabricksResource) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var plan DestinationDatabricksResourceModel

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
			"Error creating Databricks destination",
			fmt.Sprintf("Unable to create Databricks destination, got error: %s", err),
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

func (r *DestinationDatabricksResource) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var state DestinationDatabricksResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	destinationID := state.ID.ValueString()
	destination, err := r.client.GetDestination(ctx, destinationID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Databricks destination",
			fmt.Sprintf("Unable to read Databricks destination, got error: %s", err),
		)
		return
	}
	if destination == nil {
		resp.Diagnostics.AddError(
			"Error reading Databricks destination",
			fmt.Sprintf("Databricks destination %s does not exist", destinationID),
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

func (r *DestinationDatabricksResource) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var plan DestinationDatabricksResourceModel

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
			"Error updating Databricks destination",
			fmt.Sprintf("Unable to update Databricks destination, got error: %s", err),
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

func (r *DestinationDatabricksResource) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var state DestinationDatabricksResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDestination(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Databricks destination",
			fmt.Sprintf("Unable to delete Databricks destination, got error: %s", err),
		)
		return
	}
}

func (r *DestinationDatabricksResource) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helpers
func (r *DestinationDatabricksResource) model2ConfigMap(_ context.Context, model DestinationDatabricksResourceModel) map[string]any {

	configMap := map[string]any{
		"connection.url.user.defined": model.ConnectionUrl.ValueString(),
		"databricks.token":            model.DatabricksToken.ValueString(),
		"databricks.catalog.user.defined":              model.DatabricksCatalog.ValueString(),
		"table.name.prefix":           model.TableNamePrefix.ValueString(),
		"ingestion.mode":              model.IngestionMode.ValueString(),
		"partition.mode":              model.PartitionMode.ValueString(),
		"hard.delete":                 model.HardDelete.ValueBool(),
		"schema.evolution":            model.SchemaEvolution.ValueString(),
		"tasks.max":                   model.TasksMax.ValueInt64(),
	}

	return configMap
}

func (r *DestinationDatabricksResource) configMap2Model(ctx context.Context, cfg map[string]any, model *DestinationDatabricksResourceModel) {
	// Copy the config map to the model
	model.ConnectionUrl = helper.GetTfCfgString(cfg, "connection.url.user.defined")
	model.DatabricksToken = helper.GetTfCfgString(cfg, "databricks.token")
	model.DatabricksCatalog = helper.GetTfCfgString(cfg, "databricks.catalog.user.defined")
	model.TableNamePrefix = helper.GetTfCfgString(cfg, "table.name.prefix")
	model.IngestionMode = helper.GetTfCfgString(cfg, "ingestion.mode")
	model.PartitionMode = helper.GetTfCfgString(cfg, "partition.mode")
	model.HardDelete = helper.GetTfCfgBool(cfg, "hard.delete")
	model.SchemaEvolution = helper.GetTfCfgString(cfg, "schema.evolution")
	model.TasksMax = helper.GetTfCfgInt64(cfg, "tasks.max")
}
