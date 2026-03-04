package datasource

import (
	"context"
	"fmt"

	ds "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

var _ ds.DataSource = &RolesDataSource{}

func NewRolesDataSource() ds.DataSource {
	return &RolesDataSource{}
}

type RolesDataSource struct {
	client api.StreamkapAPI
}

type RoleDataModel struct {
	ID          types.String `tfsdk:"id"`
	Key         types.String `tfsdk:"key"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type RolesDataSourceModel struct {
	Roles []RoleDataModel `tfsdk:"roles"`
}

func (d *RolesDataSource) Metadata(ctx context.Context, req ds.MetadataRequest, resp *ds.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func (d *RolesDataSource) Schema(ctx context.Context, req ds.SchemaRequest, resp *ds.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all available roles that can be assigned to client credentials.",
		MarkdownDescription: "Lists all available **roles** that can be assigned to client credentials.\n\n" +
			"Use this data source to discover role IDs needed when creating " +
			"`streamkap_client_credential` resources.\n\n" +
			"[Documentation](https://docs.streamkap.com/streamkap-provider-for-terraform)",
		Blocks: map[string]schema.Block{
			"roles": schema.ListNestedBlock{
				Description:         "List of available roles.",
				MarkdownDescription: "List of available roles.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description:         "The unique identifier of the role.",
							MarkdownDescription: "The unique identifier of the role.",
							Computed:            true,
						},
						"key": schema.StringAttribute{
							Description:         "The key identifier of the role.",
							MarkdownDescription: "The key identifier of the role.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							Description:         "The display name of the role.",
							MarkdownDescription: "The display name of the role.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							Description:         "The description of the role.",
							MarkdownDescription: "The description of the role.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RolesDataSource) Configure(ctx context.Context, req ds.ConfigureRequest, resp *ds.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(api.StreamkapAPI)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Roles Data Source Configure Type",
			fmt.Sprintf("Expected api.StreamkapAPI, got: %T.", req.ProviderData),
		)
		return
	}
	d.client = client
}

func (d *RolesDataSource) Read(ctx context.Context, req ds.ReadRequest, resp *ds.ReadResponse) {
	var config RolesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := d.client.ListRoles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading roles", fmt.Sprintf("Unable to list roles: %s", err))
		return
	}

	roleModels := make([]RoleDataModel, len(roles))
	for i, role := range roles {
		roleModels[i] = RoleDataModel{
			ID:          types.StringValue(role.ID),
			Key:         types.StringValue(role.Key),
			Name:        types.StringValue(role.Name),
			Description: types.StringValue(role.Description),
		}
	}
	config.Roles = roleModels

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
