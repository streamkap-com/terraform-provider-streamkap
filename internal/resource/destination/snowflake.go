package destination

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	SnowflakeDatabaseName         types.String `tfsdk:"snowflake_database_name"`
	SnowflakeSchemaName           types.String `tfsdk:"snowflake_schema_name"`
	SnowflakeRoleName             types.String `tfsdk:"snowflake_role_name"`
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
				Required:            true,
				Sensitive:           true,
				Description:         "If the value is not empty, this phrase is used to try to decrypt the private key.",
				MarkdownDescription: "If the value is not empty, this phrase is used to try to decrypt the private key.",
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

	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", plan))
	config := r.model2ConfigMap(plan)

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

	plan.ID = types.StringValue(destination.ID)
	plan.Name = types.StringValue(destination.Name)
	plan.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(destination.Config, &plan)

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
	r.configMap2Model(destination.Config, &state)

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

	tflog.Info(ctx, "===> config: "+fmt.Sprintf("%+v", plan))
	config := r.model2ConfigMap(plan)

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

	// Update resource state with updated items
	plan.Name = types.StringValue(destination.Name)
	plan.Connector = types.StringValue(destination.Connector)
	r.configMap2Model(destination.Config, &plan)

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
func (r *DestinationSnowflakeResource) model2ConfigMap(model DestinationSnowflakeResourceModel) map[string]any {
	return map[string]any{
		"snowflake.url.name":               model.SnowflakeUrlName.ValueString(),
		"snowflake.user.name":              model.SnowflakeUserName.ValueString(),
		"snowflake.private.key":            model.SnowflakePrivateKey.ValueString(),
		"snowflake.private.key.passphrase": model.SnowflakePrivateKeyPassphrase.ValueString(),
		"snowflake.database.name":          model.SnowflakeDatabaseName.ValueString(),
		"snowflake.schema.name":            model.SnowflakeSchemaName.ValueString(),
		"snowflake.role.name":              model.SnowflakeRoleName.ValueString(),
	}
}

func (r *DestinationSnowflakeResource) configMap2Model(cfg map[string]any, model *DestinationSnowflakeResourceModel) {
	// Copy the config map to the model
	model.SnowflakeUrlName = helper.GetTfCfgString(cfg, "snowflake.url.name")
	model.SnowflakeUserName = helper.GetTfCfgString(cfg, "snowflake.user.name")
	model.SnowflakePrivateKey = helper.GetTfCfgString(cfg, "snowflake.private.key")
	model.SnowflakePrivateKeyPassphrase = helper.GetTfCfgString(cfg, "snowflake.private.key.passphrase")
	model.SnowflakeDatabaseName = helper.GetTfCfgString(cfg, "snowflake.database.name")
	model.SnowflakeSchemaName = helper.GetTfCfgString(cfg, "snowflake.schema.name")
	model.SnowflakeRoleName = helper.GetTfCfgString(cfg, "snowflake.role.name")
}
