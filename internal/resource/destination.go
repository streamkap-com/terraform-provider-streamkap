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
	Id           string `json:"id" tfsdk:"id"`
	Name         string `json:"name" tfsdk:"name"`
	SubId        string `json:"sub_id" tfsdk:"sub_id"`
	TenantId     string `json:"tenant_id" tfsdk:"tenant_id"`
	Connector    string `json:"connector" tfsdk:"connector"`
	TaskStatuses struct {
		Field1 struct {
			Status string `json:"status" tfsdk:"status"`
		} `json:"0" tfsdk:"field_1"`
	} `json:"task_statuses" tfsdk:"task_statuses"`
	Tasks           []int    `json:"tasks" tfsdk:"tasks"`
	ConnectorStatus string   `json:"connector_status" tfsdk:"connector_status"`
	TopicIds        []string `json:"topic_ids" tfsdk:"topic_ids"`
	Topics          []string `json:"topics" tfsdk:"topics"`
	InlineMetrics   struct {
		ConnectorStatus []struct {
			Timestamp string `json:"timestamp" tfsdk:"timestamp"`
			Value     string `json:"value" tfsdk:"value"`
		} `json:"connector_status" tfsdk:"connector_status"`
		Latency []struct {
			Timestamp string `json:"timestamp" tfsdk:"timestamp"`
			Value     string `json:"value" tfsdk:"value"`
		} `json:"latency" tfsdk:"latency"`
		SourceRecordWriteTotal []struct {
			Timestamp string `json:"timestamp" tfsdk:"timestamp"`
			Value     int    `json:"value" tfsdk:"value"`
		} `json:"sourceRecordWriteTotal" tfsdk:"source_record_write_total"`
	} `json:"inline_metrics" tfsdk:"inline_metrics"`
	Config struct {
		Key string `json:"key" tfsdk:"key"`
	} `json:"config" tfsdk:"config"`
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

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	httpResp, err := r.client.CreateDestination(ctx, api.CreateDestinationRequest{
		Name:      data.Name,
		Connector: data.Connector,
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

func (r *Destination) Update(ctx context.Context, req res.UpdateRequest, resp *res.UpdateResponse) {
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

func (r *Destination) Delete(ctx context.Context, req res.DeleteRequest, resp *res.DeleteResponse) {
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

func (r *Destination) ImportState(ctx context.Context, req res.ImportStateRequest, resp *res.ImportStateResponse) {
	res.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
