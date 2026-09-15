package source

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

type sqlServerEchoAPI struct {
	api.StreamkapAPI
	request api.Source
}

func (m *sqlServerEchoAPI) CreateSource(_ context.Context, source api.Source) (*api.Source, error) {
	m.request = source
	return m.echo(source), nil
}

func (m *sqlServerEchoAPI) UpdateSource(_ context.Context, _ string, source api.Source) (*api.Source, error) {
	m.request = source
	return m.echo(source), nil
}

func (m *sqlServerEchoAPI) GetSource(_ context.Context, _ string) (*api.Source, error) {
	return m.echo(m.request), nil
}

func (m *sqlServerEchoAPI) echo(source api.Source) *api.Source {
	config := make(map[string]any, len(source.Config))
	for key, value := range source.Config {
		config[key] = value
	}
	delete(config, "streamkap.snapshot.large.table.threshold")
	delete(config, "streamkap.snapshot.custom.table.config.user.defined")
	return &api.Source{ID: "source-id", Name: source.Name, Connector: source.Connector, Config: config}
}

func TestSQLServerRemovedSnapshotFieldsRoundTrip(t *testing.T) {
	ctx := context.Background()
	r := NewSourceSQLServerResource().(*SourceSQLServerResource)
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	s := schemaResp.Schema
	model := SourceSQLServerResourceModel{
		ID: types.StringNull(), Name: types.StringValue("test-source"), Connector: types.StringValue("sqlserveraws"),
		DatabaseHostname: types.StringValue("localhost"), DatabasePort: types.Int64Value(1433),
		DatabaseUser: types.StringValue("user"), DatabasePassword: types.StringValue("placeholder"),
		DatabaseName: types.StringValue("db"), SchemaIncludeList: types.StringValue("dbo"), TableIncludeList: types.StringValue("dbo.orders"),
		SignalDataCollectionSchemaOrDatabase: types.StringNull(), ColumnExcludeList: types.StringNull(),
		HeartbeatEnabled: types.BoolValue(false), HeartbeatDataCollectionSchemaOrDatabase: types.StringValue("streamkap"),
		BinaryHandlingMode: types.StringValue("bytes"), InsertStaticKeyField: types.StringNull(), InsertStaticKeyValue: types.StringNull(),
		InsertStaticValueField: types.StringNull(), InsertStaticValue: types.StringNull(),
		SSHEnabled: types.BoolValue(false), SSHHost: types.StringNull(), SSHPort: types.StringValue("22"), SSHUser: types.StringValue("streamkap"),
		SnapshotParallelism: types.Int64Value(1), SnapshotLargeTableThreshold: types.Int64Value(20000),
		SnapshotCustomTableConfig: map[string]snapshotCustomTableConfigModel{"dbo.orders": {Chunks: types.Int64Value(2)}},
	}
	plan := tfsdk.Plan{Schema: s}
	require.False(t, plan.Set(ctx, model).HasError())
	client := &sqlServerEchoAPI{}
	r.client = client
	var createResp resource.CreateResponse
	createResp.State = tfsdk.State{Schema: s}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
	require.False(t, createResp.Diagnostics.HasError(), "%v", createResp.Diagnostics)
	require.NotContains(t, client.request.Config, "streamkap.snapshot.large.table.threshold")
	require.NotContains(t, client.request.Config, "streamkap.snapshot.custom.table.config.user.defined")
	var created SourceSQLServerResourceModel
	require.False(t, createResp.State.Get(ctx, &created).HasError())
	require.Equal(t, int64(20000), created.SnapshotLargeTableThreshold.ValueInt64())
	require.Equal(t, int64(2), created.SnapshotCustomTableConfig["dbo.orders"].Chunks.ValueInt64())

	var readResp resource.ReadResponse
	readResp.State = createResp.State
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	require.False(t, readResp.Diagnostics.HasError(), "%v", readResp.Diagnostics)
	var refreshed SourceSQLServerResourceModel
	require.False(t, readResp.State.Get(ctx, &refreshed).HasError())
	require.Equal(t, int64(20000), refreshed.SnapshotLargeTableThreshold.ValueInt64())
	require.Equal(t, int64(2), refreshed.SnapshotCustomTableConfig["dbo.orders"].Chunks.ValueInt64())
	created.SnapshotLargeTableThreshold = types.Int64Null()
	created.SnapshotCustomTableConfig = nil
	importState := tfsdk.State{Schema: s}
	require.False(t, importState.Set(ctx, created).HasError())
	readResp = resource.ReadResponse{State: importState}
	r.Read(ctx, resource.ReadRequest{State: importState}, &readResp)
	require.False(t, readResp.Diagnostics.HasError(), "%v", readResp.Diagnostics)
	require.False(t, readResp.State.Get(ctx, &refreshed).HasError())
	require.Equal(t, int64(20000), refreshed.SnapshotLargeTableThreshold.ValueInt64())
	require.Nil(t, refreshed.SnapshotCustomTableConfig)

	model.ID = types.StringValue("source-id")
	plan = tfsdk.Plan{Schema: s}
	require.False(t, plan.Set(ctx, model).HasError())
	var updateResp resource.UpdateResponse
	updateResp.State = tfsdk.State{Schema: s}
	r.Update(ctx, resource.UpdateRequest{Plan: plan}, &updateResp)
	require.False(t, updateResp.Diagnostics.HasError(), "%v", updateResp.Diagnostics)
	require.NotContains(t, client.request.Config, "streamkap.snapshot.large.table.threshold")
	require.NotContains(t, client.request.Config, "streamkap.snapshot.custom.table.config.user.defined")
	var updated SourceSQLServerResourceModel
	require.False(t, updateResp.State.Get(ctx, &updated).HasError())
	require.Equal(t, int64(20000), updated.SnapshotLargeTableThreshold.ValueInt64())
	require.Equal(t, int64(2), updated.SnapshotCustomTableConfig["dbo.orders"].Chunks.ValueInt64())
}
