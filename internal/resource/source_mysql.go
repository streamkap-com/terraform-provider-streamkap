package resource

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ res.Resource                = &SourceMySQL{}
	_ res.ResourceWithImportState = &SourceMySQL{}
)

func NewSourceMySQLResource() res.Resource {
	return &SourceMySQL{}
}

// SourceMySQL defines the resource implementation.
type SourceMySQL struct {
	client         api.StreamkapAPI
	configurations []api.SourceConfigurationResponse
}

// SourceModel describes the resource data model.
type SourceModel struct {
	Id        types.String `json:"id" tfsdk:"id"`
	Name      types.String `json:"name" tfsdk:"name"`
	Connector types.String `json:"connector" tfsdk:"connector"`
	Config    types.Object `json:"config" tfsdk:"config"`
}

func (r *SourceMySQL) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_mysql"
}

func (r *SourceMySQL) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Source MySQL resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Source MySQL identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "SourceMySQL name",
			},
			"connector": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "SourceMySQL connector",
			},
			"config": schema.ObjectAttribute{
				Required:            true,
				MarkdownDescription: "SourceMySQL config",
				AttributeTypes: map[string]attr.Type{
					"database.hostname.user.defined":            types.StringType,
					"database.port":                             types.StringType,
					"database.user":                             types.StringType,
					"database.password":                         types.StringType,
					"database.include.list.user.defined":        types.StringType,
					"table.include.list.user.defined":           types.StringType,
					"signal.data.collection.schema.or.database": types.StringType,
					"database.connectionTimeZone":               types.StringType,
					"snapshot.gtid":                             types.StringType,
					"snapshot.mode.user.defined":                types.StringType,
					"binary.handling.mode":                      types.StringType,
					"incremental.snapshot.chunk.size":           types.Int64Type,
					"max.batch.size":                            types.Int64Type,
				},
			},
		},
	}
}

func (r *SourceMySQL) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected SourceMySQL Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	configurations, err := client.ListSourceConfigurations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list source configurations, got error: %s", err))
	}

	r.client = client
	r.configurations = configurations
}

func (r *SourceMySQL) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var data SourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	config := api.CreateSourceConfig{
		DatabaseHostnameUserDefined:          data.Config.Attributes()["database.hostname.user.defined"].String(),
		DatabasePort:                         data.Config.Attributes()["database.port"].String(),
		DatabaseUser:                         data.Config.Attributes()["database.user"].String(),
		DatabasePassword:                     data.Config.Attributes()["database.password"].String(),
		DatabaseIncludeListUserDefined:       data.Config.Attributes()["database.include.list.user.defined"].String(),
		TableIncludeListUserDefined:          data.Config.Attributes()["table.include.list.user.defined"].String(),
		SignalDataCollectionSchemaOrDatabase: data.Config.Attributes()["signal.data.collection.schema.or.database"].String(),
		DatabaseConnectionTimeZone:           data.Config.Attributes()["database.connectionTimeZone"].String(),
		SnapshotGtid:                         data.Config.Attributes()["snapshot.gtid"].String(),
		SnapshotModeUserDefined:              data.Config.Attributes()["snapshot.mode.user.defined"].String(),
		BinaryHandlingMode:                   data.Config.Attributes()["binary.handling.mode"].String(),
	}
	chunkSize, err := strconv.Atoi(data.Config.Attributes()["incremental.snapshot.chunk.size"].String())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get chunk size, got error: %s", err))
		return
	}
	config.IncrementalSnapshotChunkSize = chunkSize

	maxSize, err := strconv.Atoi(data.Config.Attributes()["max.batch.size"].String())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get max batch size, got error: %s", err))
		return
	}
	config.MaxBatchSize = maxSize

	httpResp, err := r.client.CreateSource(ctx, api.CreateSourceRequest{
		Name:      data.Name.String(),
		Connector: data.Connector.String(),
		Config:    config,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
		return
	}
	source := httpResp.Data[0]

	data.Id = types.StringValue(source.ID)
	data.Name = types.StringValue(source.Name)
	data.Connector = types.StringValue(source.Connector)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceMySQL) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var data SourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceMySQL) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var data SourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.client.GetSource(ctx, data.Id.String())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
		return
	}

	data.Id = types.StringValue(httpResp.ID)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceMySQL) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var data SourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}

func (r *SourceMySQL) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
