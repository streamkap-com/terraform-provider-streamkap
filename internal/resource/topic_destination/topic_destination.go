package topic_destination

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

type Resource struct {
	client api.StreamkapAPI
}

type Model struct {
	ID            types.String `tfsdk:"id"`
	TopicID       types.String `tfsdk:"topic_id"`
	DestinationID types.String `tfsdk:"destination_id"`
	BindingID     types.String `tfsdk:"binding_id"`
}

var _ resource.Resource = (*Resource)(nil)
var _ resource.ResourceWithConfigure = (*Resource)(nil)
var _ resource.ResourceWithImportState = (*Resource)(nil)

func NewResource() resource.Resource { return &Resource{} }

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_topic_destination"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Sends one API source topic to a destination.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true, Description: "Composite topic and destination identifier."},
			"topic_id":       schema.StringAttribute{Required: true, Description: "Full API source topic identifier.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"destination_id": schema.StringAttribute{Required: true, Description: "Destination connector ID.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"binding_id":     schema.StringAttribute{Computed: true, Description: "Managed binding ID shared by topics from the same source and destination."},
		},
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError("Unexpected topic destination client", fmt.Sprintf("Expected api.StreamkapAPI, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	link, err := r.client.AttachTopicDestination(ctx, model.TopicID.ValueString(), model.DestinationID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error attaching topic to destination", err.Error())
		return
	}
	model.ID = types.StringValue(compositeID(model.TopicID.ValueString(), model.DestinationID.ValueString()))
	model.BindingID = types.StringValue(link.BindingID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if model.TopicID.IsNull() || model.DestinationID.IsNull() {
		topicID, destinationID, err := parseCompositeID(model.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Invalid topic destination ID", err.Error())
			return
		}
		model.TopicID = types.StringValue(topicID)
		model.DestinationID = types.StringValue(destinationID)
	}
	link, err := r.client.GetTopicDestination(ctx, model.TopicID.ValueString(), model.DestinationID.ValueString())
	if api.IsNotFound(err) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading topic destination", err.Error())
		return
	}
	model.ID = types.StringValue(compositeID(model.TopicID.ValueString(), model.DestinationID.ValueString()))
	model.BindingID = types.StringValue(link.BindingID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *Resource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Topic destination cannot be updated", "Changing topic_id or destination_id requires replacement.")
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model Model
	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DetachTopicDestination(ctx, model.TopicID.ValueString(), model.DestinationID.ValueString()); err != nil && !api.IsNotFound(err) {
		resp.Diagnostics.AddError("Error detaching topic from destination", err.Error())
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	topicID, destinationID, err := parseCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid topic destination import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("topic_id"), topicID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("destination_id"), destinationID)...)
}

func compositeID(topicID, destinationID string) string { return topicID + "|" + destinationID }

func parseCompositeID(id string) (string, string, error) {
	parts := strings.Split(id, "|")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected <topic_id>|<destination_id>, got %q", id)
	}
	return parts[0], parts[1], nil
}
