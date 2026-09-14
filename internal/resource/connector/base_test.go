package connector

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/shared"
)

type fakeModel struct {
	Name     types.String `tfsdk:"name"`
	Password types.String `tfsdk:"password"`
	Token    types.String `tfsdk:"token"`
	Region   types.String `tfsdk:"region"`
	Scope    types.String `tfsdk:"scope"`
}

func TestExtractValueHook_PreservesExplicitEmptyMap(t *testing.T) {
	r := &BaseConnectorResource{}
	hook := r.extractValueHook()

	empty := map[string]types.String{}
	got, handled := hook(context.Background(), reflect.ValueOf(empty))
	if !handled {
		t.Fatal("map value was not handled")
	}
	gotMap, ok := got.(map[string]string)
	if !ok || gotMap == nil || len(gotMap) != 0 {
		t.Fatalf("explicit empty map became %#v, want non-nil empty map", got)
	}

	var unset map[string]types.String
	got, handled = hook(context.Background(), reflect.ValueOf(unset))
	if !handled || got != nil {
		t.Fatalf("unset map became %#v (handled=%v), want nil", got, handled)
	}
}

type fakeConfig struct{}

func (fakeConfig) GetSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name":     schema.StringAttribute{Required: true},
			"password": schema.StringAttribute{Optional: true, Computed: true, Sensitive: true},
			"token":    schema.StringAttribute{Required: true, Sensitive: true},
			"region":   schema.StringAttribute{Optional: true},
			"scope": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("PRINCIPAL_ROLE:ALL"),
			},
		},
	}
}
func (fakeConfig) GetFieldMappings() map[string]string { return map[string]string{} }
func (fakeConfig) GetConnectorType() ConnectorType     { return ConnectorTypeSource }
func (fakeConfig) GetConnectorCode() string            { return "fake" }
func (fakeConfig) GetResourceName() string             { return "fake" }
func (fakeConfig) NewModelInstance() any               { return &fakeModel{} }

// TestSensitiveStringAttrNames locks the contract that Create/Update use to
// decide which fields to restore from the plan: every Sensitive string
// attribute, regardless of Required vs Optional+Computed, and nothing else.
func TestSensitiveStringAttrNames(t *testing.T) {
	r := NewBaseConnectorResource(fakeConfig{}).(*BaseConnectorResource)

	got := shared.SensitiveStringAttrNames(r.config.GetSchema())
	want := map[string]bool{"password": true, "token": true}
	if len(got) != len(want) {
		t.Fatalf("sensitiveStringAttrNames() = %v, want exactly %v", got, []string{"password", "token"})
	}
	for _, name := range got {
		if !want[name] {
			t.Errorf("unexpected sensitive attr %q", name)
		}
	}
}

// TestDefaultedStringAttrNames locks the contract Create/Read/Update use to
// refill defaulted strings the backend drops: only Optional+Computed string
// attributes with a client-side Default qualify.
func TestDefaultedStringAttrNames(t *testing.T) {
	r := NewBaseConnectorResource(fakeConfig{}).(*BaseConnectorResource)

	got := shared.DefaultedStringAttrNames(r.config.GetSchema())
	if len(got) != 1 || got[0] != "scope" {
		t.Fatalf("DefaultedStringAttrNames() = %v, want exactly [scope]", got)
	}
}

// TestDefaultedStringNullEchoIsRefilled reproduces the Iceberg
// iceberg_catalog_scope failure: the plan carries the Default, the backend
// drops the field because its gating condition is unmet and echoes null, and
// Terraform rejects the apply unless the planned value is put back. A non-null
// echo that differs from the plan must survive so a real mismatch still fails.
func TestDefaultedStringNullEchoIsRefilled(t *testing.T) {
	r := NewBaseConnectorResource(fakeConfig{}).(*BaseConnectorResource)
	model := &fakeModel{Name: types.StringValue("dest"), Scope: types.StringValue("PRINCIPAL_ROLE:ALL")}
	planned := shared.CaptureStringFields(model, shared.DefaultedStringAttrNames(r.config.GetSchema()))

	model.Scope = types.StringNull() // API echo omitted the field
	shared.FillNullStringFields(model, planned)
	if model.Scope.ValueString() != "PRINCIPAL_ROLE:ALL" {
		t.Errorf("null echo not refilled from plan: got %#v", model.Scope)
	}

	model.Scope = types.StringValue("other")
	shared.FillNullStringFields(model, planned)
	if model.Scope.ValueString() != "other" {
		t.Errorf("a differing non-null echo must be kept: got %#v", model.Scope)
	}
}
