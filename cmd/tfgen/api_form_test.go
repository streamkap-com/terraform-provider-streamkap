package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyAPIFormContract(t *testing.T) {
	config := &ConnectorConfig{Config: []ConfigEntry{
		{Name: "resources", UserDefined: true, Value: ValueObject{Control: "multi-select", Default: []any{"Account"}, RawValues: []any{"Account"}}},
		{Name: "client_id", UserDefined: true, Value: ValueObject{Control: "string"}},
		{Name: "refresh_token", UserDefined: true, Value: ValueObject{Control: "password"}},
	}}
	form := `{"json_schema":{"required":["resources","client_id"],"properties":{"resources":{"type":"array","items":{"type":"string"}},"client_id":{"type":"string"},"auth_mode":{"type":"string","default":"service"},"refresh_token":{"type":"string"}}},"ui_schema":{"elements":[{"type":"Control","scope":"#/properties/resources"},{"type":"Control","scope":"#/properties/client_id","rule":{"effect":"SHOW","condition":{"scope":"#/properties/auth_mode","schema":{"const":"service"}}}},{"type":"Control","scope":"#/properties/refresh_token","rule":{"effect":"HIDE"}}]}}`
	form = strings.Replace(form, `"type":"array"`, `"type":"array","minItems":1`, 1)
	path := filepath.Join(t.TempDir(), "form.schema.json")
	if err := os.WriteFile(path, []byte(form), 0644); err != nil {
		t.Fatal(err)
	}
	if err := applyAPIFormContract(config, path); err != nil {
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
	if len(config.APIRequirements) != 1 || config.APIRequirements[0] != (APIRequirement{Field: "client_id", ConditionField: "auth_mode", ConditionValue: "service", ConditionDefault: "service"}) {
		t.Errorf("conditional requirements = %#v", config.APIRequirements)
	}
	if refreshToken.UserDefined {
		t.Error("hidden system-managed credential must not be configured in Terraform")
	}
}

func TestApplyAPIFormContractRequiresForm(t *testing.T) {
	config := &ConnectorConfig{Config: []ConfigEntry{{Name: "token", UserDefined: true}}}
	if err := applyAPIFormContract(config, filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("missing API source form must fail generation")
	}
}
