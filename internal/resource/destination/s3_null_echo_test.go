package destination

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

type s3EchoAPI struct {
	api.StreamkapAPI
	request api.Destination
}

func (m *s3EchoAPI) CreateDestination(_ context.Context, destination api.Destination) (*api.Destination, error) {
	m.request = destination
	return m.echo(destination), nil
}

func (m *s3EchoAPI) UpdateDestination(_ context.Context, _ string, destination api.Destination) (*api.Destination, error) {
	m.request = destination
	return m.echo(destination), nil
}

func (m *s3EchoAPI) GetDestination(_ context.Context, _ string) (*api.Destination, error) {
	return m.echo(m.request), nil
}

func (m *s3EchoAPI) echo(destination api.Destination) *api.Destination {
	config := make(map[string]any, len(destination.Config))
	for key, value := range destination.Config {
		config[key] = value
	}
	delete(config, "file.name.prefix")
	return &api.Destination{ID: "destination-id", Name: destination.Name, Connector: destination.Connector, Config: config}
}

func TestS3RemovedFilenamePrefixRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := NewDestinationS3Resource().(*DestinationS3Resource)
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	outputFields, diags := types.ListValueFrom(ctx, types.StringType, []string{"value"})
	require.False(t, diags.HasError())
	model := DestinationS3ResourceModel{
		ID: types.StringNull(), Name: types.StringValue("test-destination"), Connector: types.StringValue("s3"),
		AWSAccessKeyID: types.StringValue("placeholder"), AWSSecretKeyID: types.StringValue("placeholder"),
		Region: types.StringValue("us-west-2"), BucketName: types.StringValue("bucket"),
		Format: types.StringValue("JSON Array"), FilenameTemplate: types.StringValue("{{topic}}-{{partition}}-{{start_offset}}"),
		FilenamePrefix: types.StringValue("archive/"), CompressionType: types.StringValue("gzip"), OutputFields: outputFields,
	}
	plan := tfsdk.Plan{Schema: s}
	require.False(t, plan.Set(ctx, model).HasError())
	client := &s3EchoAPI{}
	r.client = client
	var createResp resource.CreateResponse
	createResp.State = tfsdk.State{Schema: s}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
	require.False(t, createResp.Diagnostics.HasError(), "%v", createResp.Diagnostics)
	require.NotContains(t, client.request.Config, "file.name.prefix")
	var created DestinationS3ResourceModel
	require.False(t, createResp.State.Get(ctx, &created).HasError())
	require.Equal(t, "archive/", created.FilenamePrefix.ValueString())

	var readResp resource.ReadResponse
	readResp.State = createResp.State
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	require.False(t, readResp.Diagnostics.HasError(), "%v", readResp.Diagnostics)
	var refreshed DestinationS3ResourceModel
	require.False(t, readResp.State.Get(ctx, &refreshed).HasError())
	require.Equal(t, "archive/", refreshed.FilenamePrefix.ValueString())
	created.FilenamePrefix = types.StringNull()
	importState := tfsdk.State{Schema: s}
	require.False(t, importState.Set(ctx, created).HasError())
	readResp = resource.ReadResponse{State: importState}
	r.Read(ctx, resource.ReadRequest{State: importState}, &readResp)
	require.False(t, readResp.Diagnostics.HasError(), "%v", readResp.Diagnostics)
	require.False(t, readResp.State.Get(ctx, &refreshed).HasError())
	require.Equal(t, "", refreshed.FilenamePrefix.ValueString())

	model.ID = types.StringValue("destination-id")
	plan = tfsdk.Plan{Schema: s}
	require.False(t, plan.Set(ctx, model).HasError())
	var updateResp resource.UpdateResponse
	updateResp.State = tfsdk.State{Schema: s}
	r.Update(ctx, resource.UpdateRequest{Plan: plan}, &updateResp)
	require.False(t, updateResp.Diagnostics.HasError(), "%v", updateResp.Diagnostics)
	require.NotContains(t, client.request.Config, "file.name.prefix")
	var updated DestinationS3ResourceModel
	require.False(t, updateResp.State.Get(ctx, &updated).HasError())
	require.Equal(t, "archive/", updated.FilenamePrefix.ValueString())
}
