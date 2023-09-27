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
	_ res.Resource                = &Pipeline{}
	_ res.ResourceWithImportState = &Pipeline{}
)

func NewPipelineResource() res.Resource {
	return &Pipeline{}
}

// Pipeline defines the res implementation.
type Pipeline struct {
	client api.StreamkapAPI
}

// PipelineModel describes the res data model.
type PipelineModel struct {
	ID         string   `json:"id" tfsdk:"id"`
	Name       string   `json:"name" tfsdk:"name" tfsdk:"name"`
	SubID      string   `json:"sub_id" tfsdk:"-" tfsdk:"-"`
	TenantID   string   `json:"tenant_id" tfsdk:"-"`
	Transforms []string `json:"transforms" tfsdk:"transforms"`
	TopicIDs   []string `json:"topic_ids" tfsdk:"-"`
	Topics     []string `json:"topics" tfsdk:"-"`
	Source     struct {
		ID           string `json:"id" tfsdk:"id"`
		Name         string `json:"name" tfsdk:"name"`
		SubID        string `json:"sub_id" tfsdk:"-"`
		TenantID     string `json:"tenant_id" tfsdk:"-"`
		Connector    string `json:"connector" tfsdk:"connector"`
		TaskStatuses struct {
			Field1 struct {
				Status string `json:"status" tfsdk:"-"`
			} `json:"0" tfsdk:"-"`
		} `json:"task_statuses" tfsdk:"-"`
		Tasks           []int    `json:"tasks" tfsdk:"-"`
		ConnectorStatus string   `json:"connector_status" tfsdk:"-"`
		Topics          []string `json:"topics" tfsdk:"topics"`
		Server          string   `json:"server" tfsdk:"server"`
	} `json:"source" tfsdk:"source"`
	Destination struct {
		ID           string `json:"id" tfsdk:"id"`
		Name         string `json:"name" tfsdk:"name"`
		SubID        string `json:"sub_id" tfsdk:"-"`
		TenantID     string `json:"tenant_id" tfsdk:"-"`
		Connector    string `json:"connector" tfsdk:"connector"`
		TaskStatuses struct {
			Field1 struct {
				Status string `json:"status" tfsdk:"status"`
			} `json:"0" tfsdk:"-"`
		} `json:"task_statuses" tfsdk:"-"`
		Tasks           []int  `json:"tasks" tfsdk:"-"`
		ConnectorStatus string `json:"connector_status" tfsdk:"-"`
	} `json:"destination" tfsdk:"destination"`
}

func (r *Pipeline) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

func (r *Pipeline) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Pipeline resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Pipeline identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Pipeline name",
			},
			"source": schema.ObjectAttribute{
				Required:            true,
				MarkdownDescription: "Pipeline source",
				AttributeTypes: map[string]attr.Type{
					"connector": types.StringType,
					"name":      types.StringType,
					"topics": types.ListType{
						ElemType: types.StringType,
					},
					"id": types.MapType{
						ElemType: types.StringType,
					},
				},
			},
			"destination": schema.ObjectAttribute{
				Required:            true,
				MarkdownDescription: "Pipeline source",
				AttributeTypes: map[string]attr.Type{
					"connector": types.StringType,
					"name":      types.StringType,
					"id": types.MapType{
						ElemType: types.StringType,
					},
				},
			},
			"transforms": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},
		},
	}
}

func (r *Pipeline) Configure(ctx context.Context, req res.ConfigureRequest, resp *res.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(api.StreamkapAPI)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Pipeline Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *Pipeline) Create(ctx context.Context, req res.CreateRequest, resp *res.CreateResponse) {
	var data PipelineModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	pipeline, err := r.client.CreatePipeline(ctx, api.CreatePipelineRequest{
		Name: data.Name,
		Destination: api.CreatePipelineDestination{
			Connector: data.Destination.Connector,
			Name:      data.Destination.Name,
			ID: struct {
				OID string `json:"$oid"`
			}(struct{ OID string }{
				OID: data.Destination.ID,
			}),
		},
		Source: api.CreatePipelineSource{
			Connector: data.Source.Connector,
			Name:      data.Source.Name,
			Topics:    data.Source.Topics,
			ID: struct {
				OID string `json:"$oid"`
			}(struct{ OID string }{
				OID: data.Source.ID,
			}),
		},
		Transforms: data.Transforms,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
		return
	}

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	data.ID = pipeline.ID

	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *Pipeline) Read(ctx context.Context, req res.ReadRequest, resp *res.ReadResponse) {
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

func (r *Pipeline) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
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

func (r *Pipeline) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
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

func (r *Pipeline) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
