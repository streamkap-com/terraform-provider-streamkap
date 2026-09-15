package client_credential

import (
	"context"
	"testing"

	res "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// A Create or Update plan carries every Computed attribute without a plan
// modifier as unknown. `roles` is one of them, so the model must decode an
// unknown list; a Go slice cannot and fails with "Value Conversion Error".
func TestModelDecodesPlanWithUnknownRoles(t *testing.T) {
	ctx := context.Background()

	var schemaResp res.SchemaResponse
	(&ClientCredentialResource{}).Schema(ctx, res.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", schemaResp.Diagnostics)
	}

	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	values := map[string]tftypes.Value{}
	for name, attrType := range objType.AttributeTypes {
		values[name] = tftypes.NewValue(attrType, tftypes.UnknownValue)
	}
	values["role_ids"] = tftypes.NewValue(objType.AttributeTypes["role_ids"], []tftypes.Value{
		tftypes.NewValue(tftypes.String, "role-1"),
	})
	values["description"] = tftypes.NewValue(tftypes.String, "test")
	values["service_id"] = tftypes.NewValue(tftypes.String, nil)

	plan := tfsdk.Plan{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, values)}

	var model ClientCredentialResourceModel
	if diags := plan.Get(ctx, &model); diags.HasError() {
		t.Fatalf("plan with unknown roles must decode: %v", diags)
	}
	if !model.Roles.IsUnknown() {
		t.Fatalf("roles = %v, want unknown", model.Roles)
	}
}
