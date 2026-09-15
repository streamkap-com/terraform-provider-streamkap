package transform

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/constants"
)

// fakeDeployClient accepts the deploy calls and answers job-status polls from
// a fixed script, so deployFromPlan can be driven without a backend.
type fakeDeployClient struct {
	api.StreamkapAPI
	statuses    []string
	statusCalls int
}

func (f *fakeDeployClient) DeployTransformPreview(context.Context, string, string, string) error {
	return nil
}

func (f *fakeDeployClient) DeployTransformLive(context.Context, string, string) error {
	return nil
}

func (f *fakeDeployClient) GetTransformJobStatus(context.Context, string) (*api.TransformJobStatus, error) {
	f.statusCalls++
	idx := f.statusCalls - 1
	if idx >= len(f.statuses) {
		idx = len(f.statuses) - 1
	}
	return &api.TransformJobStatus{Status: f.statuses[idx]}, nil
}

// newDeployFixture builds a map/filter plan with deploy = true and a state that
// already holds the saved transform, which is the situation deployFromPlan
// runs in after Create has stored the ID.
func newDeployFixture(t *testing.T, client api.StreamkapAPI) (*BaseTransformResource, tfsdk.Plan, *tfsdk.State) {
	t.Helper()
	ctx := context.Background()

	r := &BaseTransformResource{client: client, config: &MapFilterConfig{}}
	// Schema() adds deploy/replay_window/connector_status and the timeouts block
	// on top of the generated schema; go through it so the plan matches.
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	schema := schemaResp.Schema

	objType := schema.Type().TerraformType(ctx).(tftypes.Object)
	nullValues := func() map[string]tftypes.Value {
		values := map[string]tftypes.Value{}
		for name, attrType := range objType.AttributeTypes {
			values[name] = tftypes.NewValue(attrType, nil)
		}
		return values
	}

	planValues := nullValues()
	planValues["deploy"] = tftypes.NewValue(tftypes.Bool, true)
	plan := tfsdk.Plan{Schema: schema, Raw: tftypes.NewValue(objType, planValues)}

	stateValues := nullValues()
	stateValues["id"] = tftypes.NewValue(tftypes.String, "transform-1")
	state := &tfsdk.State{Schema: schema, Raw: tftypes.NewValue(objType, stateValues)}

	return r, plan, state
}

func TestDeployFromPlan_TerminalFailureIsAnErrorAndKeepsState(t *testing.T) {
	if testing.Short() {
		t.Skip("polls the fake job status on the 5s deployment ticker")
	}
	client := &fakeDeployClient{statuses: []string{constants.JobStatusFailed}}
	r, plan, state := newDeployFixture(t, client)

	var diags diag.Diagnostics
	r.deployFromPlan(context.Background(), "transform-1", plan, &diags, state)

	if !diags.HasError() {
		t.Fatal("a FAILED deployment must fail the apply, not warn")
	}
	if !strings.Contains(diags.Errors()[0].Detail(), constants.JobStatusFailed) {
		t.Fatalf("error must name the Flink status, got %q", diags.Errors()[0].Detail())
	}

	var id, status types.String
	state.GetAttribute(context.Background(), path.Root("id"), &id)
	state.GetAttribute(context.Background(), path.Root("connector_status"), &status)
	if id.ValueString() != "transform-1" {
		t.Fatalf("saved transform ID must stay in state so the failure can be recovered, got %v", id)
	}
	if status.ValueString() != constants.JobStatusFailed {
		t.Fatalf("connector_status = %v, want %s", status, constants.JobStatusFailed)
	}
}

func TestDeployFromPlan_TimeoutIsAnErrorAndSkipsStatusRead(t *testing.T) {
	client := &fakeDeployClient{statuses: []string{"DEPLOYING"}}
	r, plan, state := newDeployFixture(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	var diags diag.Diagnostics
	r.deployFromPlan(ctx, "transform-1", plan, &diags, state)

	if !diags.HasError() {
		t.Fatal("running out the deployment timeout must fail the apply")
	}
	if !strings.Contains(diags.Errors()[0].Detail(), "timed out") {
		t.Fatalf("error must say the wait timed out, got %q", diags.Errors()[0].Detail())
	}
	if client.statusCalls != 0 {
		t.Fatalf("status read after a spent context must be skipped, got %d calls", client.statusCalls)
	}

	var status types.String
	state.GetAttribute(context.Background(), path.Root("connector_status"), &status)
	if status.ValueString() != constants.JobStatusUnknown {
		t.Fatalf("connector_status = %v, want %s when no status was observed", status, constants.JobStatusUnknown)
	}
}
