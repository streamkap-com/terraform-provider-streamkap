package provider

import (
	"context"
	// "fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	// "github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	ds "github.com/streamkap-com/terraform-provider-streamkap/internal/datasource"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/destination"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/pipeline"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/source"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/topic"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ provider.Provider = &streamkapProvider{}
)

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &streamkapProvider{
			version: version,
		}
	}
}

// streamkapProvider is the provider implementation.
type streamkapProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
	client  api.StreamkapAPI
}

type streamkapProviderModel struct {
	Host     types.String `tfsdk:"host"`
	ClientID types.String `tfsdk:"client_id"`
	Secret   types.String `tfsdk:"secret"`
}

// Metadata returns the provider type name.
func (p *streamkapProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "streamkap"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *streamkapProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description:         "The Streamkap API host. If not set, Streamkap will use environment variable `STREAMKAP_HOST`. Defaults to https://api.streamkap.com if both are not set.",
				MarkdownDescription: "The Streamkap API host. If not set, Streamkap will use environment variable `STREAMKAP_HOST`. Defaults to https://api.streamkap.com if both are not set.",
				Optional:            true,
			},
			"client_id": schema.StringAttribute{
				Description:         "The Streamkap API client_id. If not set, Streamkap will use environment variable `STREAMKAP_CLIENT_ID`",
				MarkdownDescription: "The Streamkap API client_id. If not set, Streamkap will use environment variable `STREAMKAP_CLIENT_ID`",
				Optional:            true,
			},
			"secret": schema.StringAttribute{
				Description:         "The Streamkap API secret. If not set, Streamkap will use environment variable `STREAMKAP_SECRET`",
				MarkdownDescription: "The Streamkap API secret. If not set, Streamkap will use environment variable `STREAMKAP_SECRET`",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

// Configure prepares a Streamkap API client for data sources and resources.
func (p *streamkapProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config streamkapProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown Streamkap API Host",
			"The provider cannot create the Streamkap API client as there is an unknown configuration value for the Streamkap API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the STREAMKAP_HOST environment variable.",
		)
	}

	if config.ClientID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_id"),
			"Unknown Streamkap API client_id",
			"The provider cannot create the Streamkap API client as there is an unknown configuration value for the Streamkap API client_id. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the STREAMKAP_CLIENT_ID environment variable.",
		)
	}

	if config.Secret.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("secret"),
			"Unknown Streamkap API secret",
			"The provider cannot create the Streamkap API client as there is an unknown configuration value for the Streamkap API secret. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the STREAMKAP_SECRET environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	host := os.Getenv("STREAMKAP_HOST")
	clientID := os.Getenv("STREAMKAP_CLIENT_ID")
	secret := os.Getenv("STREAMKAP_SECRET")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.ClientID.IsNull() {
		clientID = config.ClientID.ValueString()
	}

	if !config.Secret.IsNull() {
		secret = config.Secret.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.
	if host == "" {
		host = "https://api.streamkap.com"
	}

	if clientID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("client_id"),
			"Missing Streamkap API client_id",
			"The provider cannot create the Streamkap API client as there is a missing or empty value for the Streamkap API client_id. "+
				"Set the client_id value in the configuration or use the STREAMKAP_CLIENT_ID environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if secret == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("secret"),
			"Missing Streamkap API secret",
			"The provider cannot create the Streamkap API client as there is a missing or empty value for the Streamkap API secret. "+
				"Set the secret value in the configuration or use the STREAMKAP_SECRET environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	p.client = api.NewClient(&api.Config{
		BaseURL: host,
	})
	// Create a new Streamkap client using the configuration values
	token, err := p.client.GetAccessToken(clientID, secret)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Streamkap API Client",
			"An unexpected error occurred when creating the Streamkap API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Streamkap Client Error: "+err.Error(),
		)
		return
	}
	p.client.SetToken(token)

	// Make the Streamkap client available during TokenDS and Resource
	// type Configure methods.
	resp.DataSourceData = p.client
	resp.ResourceData = p.client
}

// DataSources defines the data sources implemented in the provider.
func (p *streamkapProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		ds.NewTransformDataSource,
		ds.NewTagDataSource,
	}
}

// Resources defines the resources implemented in the provider.
func (p *streamkapProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		source.NewSourceMongoDBResource,
		source.NewSourceMySQLResource,
		source.NewSourcePostgreSQLResource,
		source.NewSourceDynamoDBResource,
		source.NewSourceSQLServerResource,
		destination.NewDestinationSnowflakeResource,
		destination.NewDestinationClickHouseResource,
		destination.NewDestinationDatabricksResource,
		pipeline.NewPipelineResource,
		topic.NewTopicResource,
	}
}
