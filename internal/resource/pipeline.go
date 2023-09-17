package resource

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
	Id            string   `json:"id"`
	Name          string   `json:"name"`
	SubId         string   `json:"sub_id"`
	TenantId      string   `json:"tenant_id"`
	Transforms    []string `json:"transforms"`
	TopicIds      []string `json:"topic_ids"`
	Topics        []string `json:"topics"`
	InlineMetrics struct {
		Source struct {
			ConnectorStatus []struct {
				Timestamp string `json:"timestamp"`
				Value     string `json:"value"`
			} `json:"connector_status"`
			Latency []struct {
				Timestamp string `json:"timestamp"`
				Value     string `json:"value"`
			} `json:"latency"`
			SnapshotState []struct {
				Timestamp string `json:"timestamp"`
				Value     string `json:"value"`
			} `json:"snapshotState"`
			State []struct {
				Timestamp string `json:"timestamp"`
				Value     string `json:"value"`
			} `json:"state"`
			StreamingState []struct {
				Timestamp string `json:"timestamp"`
				Value     string `json:"value"`
			} `json:"streamingState"`
		} `json:"source"`
		Destination struct {
			ConnectorStatus []struct {
				Timestamp string `json:"timestamp"`
				Value     string `json:"value"`
			} `json:"connector_status"`
			Latency []struct {
				Timestamp string `json:"timestamp"`
				Value     string `json:"value"`
			} `json:"latency"`
		} `json:"destination"`
		Pipeline struct {
			ConnectorStatus []struct {
				Timestamp string `json:"timestamp"`
				Value     string `json:"value"`
			} `json:"connector_status"`
			Latency []struct {
				Timestamp string `json:"timestamp"`
				Value     string `json:"value"`
			} `json:"latency"`
		} `json:"pipeline"`
	} `json:"inline_metrics"`
	Source struct {
		Id           string `json:"id"`
		Name         string `json:"name"`
		SubId        string `json:"sub_id"`
		TenantId     string `json:"tenant_id"`
		Connector    string `json:"connector"`
		TaskStatuses struct {
			Field1 struct {
				Status string `json:"status"`
			} `json:"0"`
		} `json:"task_statuses"`
		Tasks           []int    `json:"tasks"`
		ConnectorStatus string   `json:"connector_status"`
		Topics          []string `json:"topics"`
		Server          string   `json:"server"`
	} `json:"source"`
	Destination struct {
		Id           string `json:"id"`
		Name         string `json:"name"`
		SubId        string `json:"sub_id"`
		TenantId     string `json:"tenant_id"`
		Connector    string `json:"connector"`
		TaskStatuses struct {
			Field1 struct {
				Status string `json:"status"`
			} `json:"0"`
		} `json:"task_statuses"`
		Tasks           []int  `json:"tasks"`
		ConnectorStatus string `json:"connector_status"`
	} `json:"destination"`
}

func (r *Pipeline) Metadata(ctx context.Context, req res.MetadataRequest, resp *res.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pipeline"
}

func (r *Pipeline) Schema(ctx context.Context, req res.SchemaRequest, resp *res.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Pipeline res",

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
			"sub_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Pipeline sub_id",
			},
			"tenant_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Pipeline tenant_id",
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
	httpResp, err := r.client.CreateDestination(ctx, api.CreateDestinationRequest{
		Name: data.Name,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
		return
	}
	source := httpResp.Data[0]

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	data.Id = source.Id
	data.Name = source.Name

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a res")

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
