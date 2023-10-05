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
	client api.StreamkapAPI
}

// SourceModel describes the resource data model.
type SourceModel struct {
	ID        types.String    `json:"id" tfsdk:"id"`
	Name      types.String    `json:"name" tfsdk:"name"`
	Connector types.String    `json:"connector" tfsdk:"connector"`
	Config    jsontypes.Exact `json:"config" tfsdk:"config"`
	Instance  jsontypes.Exact `json:"instance" tfsdk:"-"`
}

func (r *Source) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source"
}

func (r *Source) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
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
				MarkdownDescription: "Source name",
			},
			"connector": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source connector",
			},
			"config": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source config",
				CustomType:          jsontypes.ExactType{},
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
	r.client = client
}

func (r *Source) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var data SourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "===> config: "+data.Config.String())
	var config map[string]interface{}
	data.Config.Unmarshal(&config)
	source, err := r.client.CreateSource(ctx, api.CreateSourceRequest{
		Name:      data.Name.ValueString(),
		Connector: data.Connector.ValueString(),
		Config:    config,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create source, got error: %s", err))
		return
	}

	data.ID = types.StringValue(source.ID)
	data.Name = types.StringValue(source.Name)
	data.Connector = types.StringValue(source.Connector)

	sourceString, err := json.Marshal(source)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse source, got error: %s", err))
		return
	}
	data.Instance = jsontypes.NewExactValue(string(sourceString))
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

	source, err := r.client.GetSource(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source, got error: %s", err))
		return
	}
	if source != nil {
		sourceString, err := json.Marshal(source[0])
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse source, got error: %s", err))
			return
		}
		data.Instance = jsontypes.NewExactValue(string(sourceString))
	}
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

	source, err := r.client.GetSource(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get example, got error: %s", err))
		return
	}
	if len(source) == 0 {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get source, got error: %s", err))
		return
	}

	var currentInstance api.Source
	data.Instance.Unmarshal(currentInstance)
	diff := cmp.Diff(source[0], currentInstance, cmp.AllowUnexported())
	if diff != "" {
		var config map[string]interface{}
		data.Config.Unmarshal(&config)
		updatedSource, err := r.client.UpdateSource(ctx, api.CreateSourceRequest{
			Name:      data.Name.ValueString(),
			Connector: data.Connector.ValueString(),
			Config:    config,
			ID:        data.ID.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update source, got error: %s", err))
			return
		}
		sourceString, err := json.Marshal(updatedSource)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to parse source, got error: %s", err))
			return
		}
		data.Instance = jsontypes.NewExactValue(string(sourceString))
	}

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

	err := r.client.DeleteSource(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete source, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted a resource")
}

func (r *Source) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
