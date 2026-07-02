package connector

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type fakeModel struct {
	Name     types.String `tfsdk:"name"`
	Password types.String `tfsdk:"password"`
	Token    types.String `tfsdk:"token"`
	Region   types.String `tfsdk:"region"`
}

type fakeConfig struct{}

func (fakeConfig) GetSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name":     schema.StringAttribute{Required: true},
			"password": schema.StringAttribute{Optional: true, Computed: true, Sensitive: true},
			"token":    schema.StringAttribute{Required: true, Sensitive: true},
			"region":   schema.StringAttribute{Optional: true},
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

	got := r.sensitiveStringAttrNames()
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
