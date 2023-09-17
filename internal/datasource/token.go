package datasource

import (
	"context"
	"fmt"

	ds "github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ ds.DataSource = &TokenDS{}

func NewDataSource() ds.DataSource {
	return &TokenDS{}
}

// TokenDS defines the data source implementation.
type TokenDS struct {
	token *api.Token
}

// TokenModel describes the data source data model.
type TokenModel struct {
	AccessToken  types.String `tfsdk:"access_token"`
	RefreshToken types.String `tfsdk:"refresh_token"`
	ExpiresIn    types.Int64  `tfsdk:"expires_in"`
	Expires      types.String `tfsdk:"expires"`
}

func (d *TokenDS) Metadata(ctx context.Context, req ds.MetadataRequest, resp *ds.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

func (d *TokenDS) Schema(ctx context.Context, req ds.SchemaRequest, resp *ds.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Token data source",

		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Access Token",
				Computed:            true,
			},
			"refresh_token": schema.StringAttribute{
				MarkdownDescription: "Refresh Token",
				Computed:            true,
			},
			"expires_in": schema.Int64Attribute{
				MarkdownDescription: "Token expires in duration",
				Computed:            true,
			},
			"expires": schema.StringAttribute{
				MarkdownDescription: "Token expires at time",
				Computed:            true,
			},
		},
	}
}

func (d *TokenDS) Configure(ctx context.Context, req ds.ConfigureRequest, resp *ds.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	token, ok := req.ProviderData.(*api.Token)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.token = token
}

func (d *TokenDS) Read(ctx context.Context, req ds.ReadRequest, resp *ds.ReadResponse) {
	var data TokenModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	data.AccessToken = types.StringValue(d.token.AccessToken)
	data.RefreshToken = types.StringValue(d.token.RefreshToken)
	data.Expires = types.StringValue(d.token.Expires)
	data.ExpiresIn = types.Int64Value(d.token.ExpiresIn)

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
