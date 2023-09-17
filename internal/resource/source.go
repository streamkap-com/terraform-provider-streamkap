package resource

import (
	"context"
	"fmt"

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
	_ res.Resource                = &Source{}
	_ res.ResourceWithImportState = &Source{}
)

func NewSourceResource() res.Resource {
	return &Source{}
}

// Source defines the resource implementation.
type Source struct {
	client         api.StreamkapAPI
	configurations []api.SourceConfigurationResponse
}

// SourceModel describes the resource data model.
type SourceModel struct {
	Id        types.String           `json:"id" tfsdk:"id"`
	Name      types.String           `json:"name" tfsdk:"name"`
	Connector types.String           `json:"connector" tfsdk:"connector"`
	Config    map[string]interface{} `json:"config" tfsdk:"config"`
}

func (r *Source) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source"
}

func (r *Source) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Source resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Source identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source name",
			},
			"connector": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source connector",
			},
			"config": schema.ObjectAttribute{
				Required:            true,
				MarkdownDescription: "Source config",
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
					"incremental.snapshot.chunk.size":           types.StringType,
					"max.batch.size":                            types.StringType,
				},
			},
		},
	}
}

func (r *Source) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Source Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	configurations, err := client.ListSourceConfigurations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list source configurations, got error: %s", err))
	}
	fmt.Println("configurations: ", configurations)

	r.client = client
	r.configurations = configurations
}

func (r *Source) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var data SourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	httpResp, err := r.client.CreateSource(ctx, api.CreateSourceRequest{
		Name:      data.Name.String(),
		Connector: data.Connector.String(),
		Config: api.CreateSourceConfig{
			DatabaseHostnameUserDefined:          data.Config["database.hostname.user.defined"].(string),
			DatabasePort:                         data.Config["database.port"].(string),
			DatabaseUser:                         data.Config["database.user"].(string),
			DatabasePassword:                     data.Config["database.password"].(string),
			DatabaseIncludeListUserDefined:       data.Config["database.include.list.user.defined"].(string),
			TableIncludeListUserDefined:          data.Config["table.include.list.user.defined"].(string),
			SignalDataCollectionSchemaOrDatabase: data.Config["signal.data.collection.schema.or.database"].(string),
			DatabaseConnectionTimeZone:           data.Config["database.connectionTimeZone"].(string),
			SnapshotGtid:                         data.Config["snapshot.gtid"].(string),
			SnapshotModeUserDefined:              data.Config["snapshot.mode.user.defined"].(string),
			BinaryHandlingMode:                   data.Config["binary.handling.mode"].(string),
			IncrementalSnapshotChunkSize:         data.Config["incremental.snapshot.chunk.size"].(int),
			MaxBatchSize:                         data.Config["max.batch.size"].(int),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
		return
	}
	source := httpResp.Data[0]

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	data.Id = types.StringValue(source.Id)
	data.Name = types.StringValue(source.Name)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Source) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
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

func (r *Source) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var data SourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Source) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
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

func (r *Source) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
