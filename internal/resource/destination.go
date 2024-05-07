package resource

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jinzhu/copier"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ res.Resource                = &Destination{}
	_ res.ResourceWithImportState = &Destination{}
)

func NewDestinationResource() res.Resource {
	return &Destination{}
}

// Destination defines the resource implementation.
type Destination struct {
	client api.StreamkapAPI
}

// DestinationModel describes the resource data model.
type DestinationModel struct {
	ID        basetypes.StringValue `json:"id" tfsdk:"id"`
	Name      *string               `json:"name" tfsdk:"name"`
	Connector *string               `json:"connector" tfsdk:"connector"`
	Config    jsontypes.Exact       `json:"config" tfsdk:"config"`
}

func (r *Destination) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination"
}

func (r *Destination) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Destination resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Destination identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Destination name",
			},
			"connector": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Destination connector",
			},
			"config": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Destination config",
				CustomType:          jsontypes.ExactType{},
			},
		},
	}
}

func (r *Destination) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Destination Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *Destination) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var data DestinationModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var config map[string]interface{}
	data.Config.Unmarshal(&config)
	destination, err := r.client.CreateDestination(ctx, api.CreateDestinationRequest{
		Name:      data.Name,
		Connector: data.Connector,
		Config:    config,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create destination, got error: %s", err))
		return
	}

	data.ID = types.StringValue(destination.ID)
	data.Name = &destination.Name
	data.Connector = &destination.Connector

	sourceString, err := json.Marshal(config)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse destination, got error: %s", err))
		return
	}
	data.Config = jsontypes.NewExactValue(string(sourceString))
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Destination) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
	var data SourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	destination, err := r.client.GetDestination(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read destination, got error: %s", err))
		return
	}
	if destination != nil {
		sourceString, err := json.Marshal(destination[0])
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse destination, got error: %s", err))
			return
		}
		copier.CopyWithOption(&data, &destination[0], copier.Option{DeepCopy: true})
		data.Config = jsontypes.NewExactValue(string(sourceString))
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Destination) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
	var data SourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	destination, err := r.client.GetDestination(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get destination, got error: %s", err))
		return
	}
	if len(destination) == 0 {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get destination, got error: %s", err))
		return
	}

	var currentInstance api.Source
	copier.CopyWithOption(&currentInstance, &destination[0], copier.Option{DeepCopy: true})
	json.Unmarshal([]byte(data.Config.String()), &currentInstance.Config)
	diff := cmp.Diff(destination[0], currentInstance, cmp.AllowUnexported())
	if diff != "" {
		var config map[string]interface{}
		data.Config.Unmarshal(&config)
		fmt.Println("config", config)
		updatedSource, err := r.client.UpdateDestination(ctx, api.CreateDestinationRequest{
			Name:      data.Name,
			Connector: data.Connector,
			Config:    config,
			ID:        data.ID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update destination, got error: %s", err))
			return
		}
		sourceString, err := json.Marshal(updatedSource.Config)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse source, got error: %s", err))
			return
		}
		copier.CopyWithOption(&data, &updatedSource, copier.Option{DeepCopy: true})
		data.Config = jsontypes.NewExactValue(string(sourceString))
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Destination) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
	var data SourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDestination(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete destination, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a resource")
}

func (r *Destination) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
