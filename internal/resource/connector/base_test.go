package connector

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
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
	Lob      types.Bool   `tfsdk:"lob_enabled"`
	Port     types.Int64  `tfsdk:"port"`
	Chunk    types.Int64  `tfsdk:"chunk"`
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
			"lob_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"port": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(1521),
			},
			// Optional+Computed without a Default: planned Unknown, never refilled.
			"chunk": schema.Int64Attribute{Optional: true, Computed: true},
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

// TestDefaultedAttrNames locks the contract Create/Read/Update use to refill
// defaulted attributes the backend drops: only Optional+Computed string, bool
// and int64 attributes with a client-side Default qualify. Sensitive strings
// are handled separately and an Optional+Computed attribute without a Default
// is planned Unknown, so neither belongs here.
func TestDefaultedAttrNames(t *testing.T) {
	r := NewBaseConnectorResource(fakeConfig{}).(*BaseConnectorResource)

	got := shared.DefaultedAttrNames(r.config.GetSchema())
	sort.Strings(got)
	want := []string{"lob_enabled", "port", "scope"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultedAttrNames() = %v, want %v", got, want)
	}
}

// TestDefaultedNullEchoIsRefilled reproduces the Iceberg iceberg_catalog_scope
// and Oracle lob_enabled failures: the plan carries the Default, the backend
// drops the field because its gating condition is unmet and echoes null, and
// Terraform rejects the apply unless the planned value is put back. A non-null
// echo that differs from the plan must survive so a real mismatch still fails.
func TestDefaultedNullEchoIsRefilled(t *testing.T) {
	r := NewBaseConnectorResource(fakeConfig{}).(*BaseConnectorResource)
	model := &fakeModel{
		Name:  types.StringValue("dest"),
		Scope: types.StringValue("PRINCIPAL_ROLE:ALL"),
		Lob:   types.BoolValue(false),
		Port:  types.Int64Value(1521),
		Chunk: types.Int64Unknown(),
	}
	planned := shared.CaptureFields(model, shared.DefaultedAttrNames(r.config.GetSchema()))

	// API echo omitted every gated field.
	model.Scope = types.StringNull()
	model.Lob = types.BoolNull()
	model.Port = types.Int64Null()
	model.Chunk = types.Int64Null()
	shared.FillNullFields(model, planned)
	if model.Scope.ValueString() != "PRINCIPAL_ROLE:ALL" {
		t.Errorf("null string echo not refilled from plan: got %#v", model.Scope)
	}
	if model.Lob.IsNull() || model.Lob.ValueBool() {
		t.Errorf("null bool echo not refilled from plan: got %#v", model.Lob)
	}
	if model.Port.ValueInt64() != 1521 {
		t.Errorf("null int64 echo not refilled from plan: got %#v", model.Port)
	}
	if !model.Chunk.IsNull() {
		t.Errorf("an attribute without a Default must keep the null echo: got %#v", model.Chunk)
	}

	model.Scope = types.StringValue("other")
	model.Lob = types.BoolValue(true)
	shared.FillNullFields(model, planned)
	if model.Scope.ValueString() != "other" || !model.Lob.ValueBool() {
		t.Errorf("a differing non-null echo must be kept: got %#v / %#v", model.Scope, model.Lob)
	}
}
