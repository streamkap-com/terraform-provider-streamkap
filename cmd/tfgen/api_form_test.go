package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestApplyAPIFormContract(t *testing.T) {
	config := &ConnectorConfig{Config: []ConfigEntry{
		{Name: "resources", UserDefined: true, Value: ValueObject{Control: "multi-select", Default: []any{"Account"}, RawValues: []any{"Account"}}},
		{Name: "client_id", UserDefined: true, Value: ValueObject{Control: "string"}},
		{Name: "refresh_token", UserDefined: true, Value: ValueObject{Control: "password"}},
	}}
	form := `{"json_schema":{"required":["resources","client_id"],"properties":{"resources":{"type":"array","items":{"type":"string"}},"client_id":{"type":"string"},"auth_mode":{"type":"string","default":"service"},"refresh_token":{"type":"string"}}},"ui_schema":{"elements":[{"type":"Control","scope":"#/properties/resources"},{"type":"Control","scope":"#/properties/client_id","rule":{"effect":"SHOW","condition":{"scope":"#/properties/auth_mode","schema":{"const":"service"}}}},{"type":"Control","scope":"#/properties/refresh_token","rule":{"effect":"HIDE","condition":{"scope":"#/properties/auth_mode","schema":{}}}}]}}`
	form = strings.Replace(form, `"type":"array"`, `"type":"array","minItems":1`, 1)
	path := filepath.Join(t.TempDir(), "form.schema.json")
	if err := os.WriteFile(path, []byte(form), 0644); err != nil {
		t.Fatal(err)
	}
	if err := applyAPIFormContract(config, "vendor", path); err != nil {
		t.Fatal(err)
	}
	resources, clientID, refreshToken := config.Config[0], config.Config[1], config.Config[2]
	if !resources.IsRequired() || resources.HasDefault() || len(resources.Value.RawValues) != 0 {
		t.Errorf("resources must be required without a UI prefill or exhaustive choice list: %#v", resources)
	}
	if resources.APIFormMinItems != 1 {
		t.Errorf("resources minimum = %d, want 1", resources.APIFormMinItems)
	}
	field := NewGenerator(t.TempDir(), "source").entryToFieldData(&resources)
	if !field.HasValidators || field.Validators != "listvalidator.SizeAtLeast(1)" {
		t.Errorf("resources validators = %q, want list size minimum", field.Validators)
	}
	if !strings.Contains(field.Description, "Requires at least one item.") {
		t.Errorf("resources description omits minimum: %q", field.Description)
	}
	if clientID.IsRequired() {
		t.Error("mode-dependent credential must not be unconditionally required")
	}
	credential := NewGenerator(t.TempDir(), "source").entryToFieldData(&clientID)
	if credential.Required || !credential.Optional || !credential.Computed || !credential.NeedsPlanMod {
		t.Errorf("mode-dependent credential must be Optional+Computed with UseStateForUnknown: %#v", credential)
	}
	if len(config.APIConditions) != 1 || !reflect.DeepEqual(config.APIConditions[0], APICondition{Field: "client_id", ConditionField: "auth_mode", ConditionValues: []string{"service"}, ConditionDefault: "service", Required: true}) {
		t.Errorf("conditions = %#v", config.APIConditions)
	}
	if refreshToken.UserDefined {
		t.Error("hidden system-managed credential must not be configured in Terraform")
	}
}

func TestApplyAPIFormContractRequiresForm(t *testing.T) {
	config := &ConnectorConfig{Config: []ConfigEntry{{Name: "token", UserDefined: true}}}
	if err := applyAPIFormContract(config, "vendor", filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("missing API source form must fail generation")
	}
}

func writeForm(t *testing.T, form string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "form.schema.json")
	if err := os.WriteFile(path, []byte(form), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// A HIDE rule names the modes a field is absent from; the field is shown, and
// required, for every other value of the condition field.
func TestApplyAPIFormContractInvertsConditionalHide(t *testing.T) {
	config := &ConnectorConfig{DisplayName: "Vendor", Config: []ConfigEntry{
		{Name: "auth_mode", UserDefined: true, Value: ValueObject{Control: "one-select"}},
		{Name: "client_id", UserDefined: true, Value: ValueObject{Control: "string"}},
		{Name: "refresh_token", UserDefined: true, Value: ValueObject{Control: "password"}},
		{Name: "key", UserDefined: true, Value: ValueObject{Control: "password"}},
		{Name: "system", UserDefined: true, Value: ValueObject{Control: "password"}},
		{Name: "window", UserDefined: true, Value: ValueObject{Control: "number"}},
	}}
	path := writeForm(t, `{"oauth":{"environments":[],"client_fields":["client_id","refresh_token"]},"json_schema":{"required":["client_id","refresh_token","key"],"properties":{
		"auth_mode":{"type":"string","enum":["oauth","key","refresh"],"default":"key"},
		"client_id":{"type":"string"},"refresh_token":{"type":"string"},"key":{"type":"string"},"system":{"type":"string"},
		"window":{"type":"integer","minimum":1,"maximum":90,"default":30}}},
		"ui_schema":{"elements":[
		{"type":"Control","scope":"#/properties/client_id","rule":{"effect":"HIDE","condition":{"scope":"#/properties/auth_mode","schema":{"const":"key"}}}},
		{"type":"Control","scope":"#/properties/refresh_token","rule":{"effect":"SHOW","condition":{"scope":"#/properties/auth_mode","schema":{"enum":["refresh"]}}}},
		{"type":"Control","scope":"#/properties/key","rule":{"effect":"HIDE","condition":{"scope":"#/properties/auth_mode","schema":{"enum":["oauth","refresh"]}}}},
		{"type":"Control","scope":"#/properties/system","rule":{"effect":"HIDE","condition":{"scope":"#/properties/auth_mode","schema":{}}}}]}}`)
	if err := applyAPIFormContract(config, "vendor", path); err != nil {
		t.Fatal(err)
	}
	want := []APICondition{
		{Field: "client_id", ConditionField: "auth_mode", ConditionValues: []string{"oauth", "refresh"}, ConditionDefault: "key", Required: true},
		{Field: "refresh_token", ConditionField: "auth_mode", ConditionValues: []string{"refresh"}, ConditionDefault: "key", Required: true},
		{Field: "key", ConditionField: "auth_mode", ConditionValues: []string{"key"}, ConditionDefault: "key", Required: true},
	}
	if !reflect.DeepEqual(config.APIConditions, want) {
		t.Errorf("conditions = %#v, want %#v", config.APIConditions, want)
	}
	if !config.Config[1].UserDefined || !config.Config[1].APIFormShow || config.Config[1].IsRequired() {
		t.Errorf("a field hidden in one mode must stay configurable and conditional: %#v", config.Config[1])
	}
	if config.Config[4].UserDefined {
		t.Error("a field hidden in every mode must not be configurable")
	}
	if !reflect.DeepEqual(config.APIOAuth, APIOAuth{Enabled: true, AuthModeField: "auth_mode", AuthModeValue: "oauth", AuthModeDefault: "key", ConnectCommand: "`streamkap sources start-source-oauth-connect vendor --body <file>`, the file holding your OAuth client's client_id and refresh_token as JSON"}) {
		t.Errorf("oauth = %#v", config.APIOAuth)
	}
	grant := config.GetEntryByName(oauthGrantField)
	if grant == nil || !grant.UserDefined || !grant.Encrypt || grant.IsRequired() || !grant.APIFormShow {
		t.Fatalf("OAuth vendors need an optional, sensitive grant handoff field: %#v", grant)
	}
	g := NewGenerator(t.TempDir(), "source")
	g.apiSource = true
	window := g.entryToFieldData(&config.Config[5])
	if window.Validators != "int64validator.Between(1, 90)" {
		t.Errorf("form minimum/maximum must bound the number input: %q", window.Validators)
	}
}

func TestApplyAPIFormContractOAuthOnlyVendor(t *testing.T) {
	config := &ConnectorConfig{DisplayName: "Vendor", Config: []ConfigEntry{
		{Name: "subdomain", UserDefined: true, Value: ValueObject{Control: "string"}},
	}}
	path := writeForm(t, `{"oauth":{"host_field":"subdomain"},"json_schema":{"required":["subdomain"],"properties":{"subdomain":{"type":"string"}}},"ui_schema":{"elements":[]}}`)
	if err := applyAPIFormContract(config, "vendor", path); err != nil {
		t.Fatal(err)
	}
	want := APIOAuth{Enabled: true, HostField: "subdomain", ConnectCommand: "`streamkap sources start-source-oauth-connect vendor --environment <subdomain>`"}
	if !reflect.DeepEqual(config.APIOAuth, want) {
		t.Errorf("a vendor without auth_mode signs in only through OAuth, on the host the grant is for: %#v", config.APIOAuth)
	}
}

func TestApplyAPIFormContractRejectsUnsupportedRules(t *testing.T) {
	for name, form := range map[string]string{
		"oauth without an oauth mode": `{"oauth":{},"json_schema":{"properties":{"auth_mode":{"type":"string","enum":["token"]},"token":{"type":"string"}}},"ui_schema":{}}`,
		"hide against an open value":  `{"json_schema":{"properties":{"region":{"type":"string"},"token":{"type":"string"}}},"ui_schema":{"elements":[{"type":"Control","scope":"#/properties/token","rule":{"effect":"HIDE","condition":{"scope":"#/properties/region","schema":{"const":"eu"}}}}]}}`,
		"unknown condition keyword":   `{"json_schema":{"properties":{"auth_mode":{"type":"string","enum":["a","b"]},"token":{"type":"string"}}},"ui_schema":{"elements":[{"type":"Control","scope":"#/properties/token","rule":{"effect":"SHOW","condition":{"scope":"#/properties/auth_mode","schema":{"pattern":"a"}}}}]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			config := &ConnectorConfig{Config: []ConfigEntry{{Name: "token", UserDefined: true, Value: ValueObject{Control: "password"}}}}
			if err := applyAPIFormContract(config, "vendor", writeForm(t, form)); err == nil {
				t.Fatal("generation must stop on a form rule it cannot express")
			}
		})
	}
}

// The artifacts describe a deployment that provisions Connect. Where that is
// the default mode, a deployment without Connect serves another default, so
// the generated auth_mode has none and must be set.
func TestApplyAPIFormContractRequiresAuthModeDefaultingToOAuth(t *testing.T) {
	config := &ConnectorConfig{Config: []ConfigEntry{
		{Name: "auth_mode", UserDefined: true, Value: ValueObject{Control: "one-select", Default: "oauth"}},
		{Name: "key", UserDefined: true, Value: ValueObject{Control: "password"}},
	}}
	path := writeForm(t, `{"oauth":{},"json_schema":{"required":["key"],"properties":{"auth_mode":{"type":"string","enum":["oauth","key"],"default":"oauth"},"key":{"type":"string"}}},
		"ui_schema":{"elements":[{"type":"Control","scope":"#/properties/key","rule":{"effect":"SHOW","condition":{"scope":"#/properties/auth_mode","schema":{"const":"key"}}}}]}}`)
	if err := applyAPIFormContract(config, "vendor", path); err != nil {
		t.Fatal(err)
	}
	mode := config.Config[0]
	if !mode.IsRequired() || mode.HasDefault() || config.APIOAuth.AuthModeDefault != "" || config.APIConditions[0].ConditionDefault != "" {
		t.Errorf("auth_mode must be required with no default: %#v %#v %#v", mode, config.APIOAuth, config.APIConditions)
	}
}

func TestApplyAPIFormContractDependentRequired(t *testing.T) {
	config := &ConnectorConfig{Config: []ConfigEntry{
		{Name: "app_id", UserDefined: true, Value: ValueObject{Control: "string"}},
		{Name: "app_secret", UserDefined: true, Value: ValueObject{Control: "password"}},
	}}
	path := writeForm(t, `{"json_schema":{"properties":{"app_id":{"type":"string"},"app_secret":{"type":"string"}},"dependentRequired":{"app_secret":["app_id"],"app_id":["app_secret"]}},"ui_schema":{}}`)
	if err := applyAPIFormContract(config, "vendor", path); err != nil {
		t.Fatal(err)
	}
	want := []APIDependency{{Field: "app_id", Requires: []string{"app_secret"}}, {Field: "app_secret", Requires: []string{"app_id"}}}
	if !reflect.DeepEqual(config.APIDependencies, want) {
		t.Errorf("dependencies = %#v", config.APIDependencies)
	}
	bad := writeForm(t, `{"json_schema":{"properties":{"app_id":{"type":"string"}},"dependentRequired":{"app_id":["app_secret"]}},"ui_schema":{}}`)
	if err := applyAPIFormContract(&ConnectorConfig{}, "vendor", bad); err == nil {
		t.Error("a dependency on an undeclared property must stop generation")
	}
}
