package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

type apiFormProperty struct {
	Type     string   `json:"type"`
	Default  any      `json:"default"`
	Enum     []any    `json:"enum"`
	MinItems int      `json:"minItems"`
	Minimum  *float64 `json:"minimum"`
	Maximum  *float64 `json:"maximum"`
	Items    struct {
		Enum []any `json:"enum"`
	} `json:"items"`
}

type apiFormContract struct {
	OAuth      json.RawMessage `json:"oauth"`
	JSONSchema struct {
		Required   []string                   `json:"required"`
		Properties map[string]apiFormProperty `json:"properties"`
	} `json:"json_schema"`
	UISchema apiFormControl `json:"ui_schema"`
}

type apiFormRule struct {
	Effect    string `json:"effect"`
	Condition struct {
		Scope string `json:"scope"`
		// Schema is {"const": v}, {"enum": [...]} or {} (always true). A map
		// keeps a present `"const": false` distinct from an absent const.
		Schema map[string]any `json:"schema"`
	} `json:"condition"`
}

type apiFormControl struct {
	Type     string           `json:"type"`
	Scope    string           `json:"scope"`
	Elements []apiFormControl `json:"elements"`
	Rule     *apiFormRule     `json:"rule"`
}

func (control apiFormControl) rules() map[string]*apiFormRule {
	rules := map[string]*apiFormRule{}
	var visit func(apiFormControl)
	visit = func(item apiFormControl) {
		if item.Type == "Control" && item.Rule != nil {
			rules[strings.TrimPrefix(item.Scope, "#/properties/")] = item.Rule
		}
		for _, child := range item.Elements {
			visit(child)
		}
	}
	visit(control)
	return rules
}

// formValue renders a condition value the way the provider compares it at plan
// time: strings as they are, booleans as "true"/"false".
func formValue(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	case bool:
		if v {
			return "true", true
		}
		return "false", true
	}
	return "", false
}

// propertyValues lists every value a condition property can take, so a HIDE
// rule can be inverted into the values that show the control.
func propertyValues(property apiFormProperty) []string {
	if property.Type == "boolean" {
		return []string{"true", "false"}
	}
	values := make([]string, 0, len(property.Enum))
	for _, item := range property.Enum {
		if value, ok := formValue(item); ok {
			values = append(values, value)
		}
	}
	return values
}

// visibility resolves a rule to its condition field and the values of it that
// show the control. hidden reports a control that no value shows.
func (form *apiFormContract) visibility(rule *apiFormRule) (conditionField string, showWhen []string, hidden bool, err error) {
	conditionField = strings.TrimPrefix(rule.Condition.Scope, "#/properties/")
	if conditionField == "" || conditionField == rule.Condition.Scope {
		return "", nil, false, fmt.Errorf("unsupported condition scope %q", rule.Condition.Scope)
	}
	var matched []string
	matchesAll := false
	switch {
	case len(rule.Condition.Schema) == 0:
		matchesAll = true
	case rule.Condition.Schema["const"] != nil:
		value, ok := formValue(rule.Condition.Schema["const"])
		if !ok {
			return "", nil, false, fmt.Errorf("unsupported condition const %v", rule.Condition.Schema["const"])
		}
		matched = []string{value}
	case rule.Condition.Schema["enum"] != nil:
		items, _ := rule.Condition.Schema["enum"].([]any)
		for _, item := range items {
			value, ok := formValue(item)
			if !ok {
				return "", nil, false, fmt.Errorf("unsupported condition enum value %v", item)
			}
			matched = append(matched, value)
		}
		if len(matched) == 0 {
			return "", nil, false, fmt.Errorf("unsupported condition enum %v", rule.Condition.Schema["enum"])
		}
	default:
		return "", nil, false, fmt.Errorf("unsupported condition schema %v", rule.Condition.Schema)
	}
	switch rule.Effect {
	case "SHOW":
		if matchesAll {
			return "", nil, false, nil
		}
		return conditionField, matched, false, nil
	case "HIDE":
		if matchesAll {
			return conditionField, nil, true, nil
		}
		property, ok := form.JSONSchema.Properties[conditionField]
		if !ok {
			return "", nil, false, fmt.Errorf("no condition property %q", conditionField)
		}
		values := propertyValues(property)
		if len(values) == 0 {
			return "", nil, false, fmt.Errorf("condition property %q lists no values to invert HIDE against", conditionField)
		}
		for _, value := range values {
			if !slices.Contains(matched, value) {
				showWhen = append(showWhen, value)
			}
		}
		return conditionField, showWhen, len(showWhen) == 0, nil
	}
	return "", nil, false, fmt.Errorf("unsupported rule effect %q", rule.Effect)
}

// oauthMode derives how the vendor's Connect flow is selected. The form does
// not name the mode value, so a vendor with an auth_mode choice must offer
// OAuth as "oauth"; a vendor without one signs in only through Connect.
func (form *apiFormContract) oauthMode() (APIOAuth, error) {
	if len(form.OAuth) == 0 || string(form.OAuth) == "null" {
		return APIOAuth{}, nil
	}
	var flow struct {
		ClientFields []string `json:"client_fields"`
	}
	if err := json.Unmarshal(form.OAuth, &flow); err != nil {
		return APIOAuth{}, fmt.Errorf("has an unreadable oauth block: %w", err)
	}
	property, ok := form.JSONSchema.Properties["auth_mode"]
	if !ok {
		return APIOAuth{Enabled: true, ClientFields: flow.ClientFields}, nil
	}
	if !slices.Contains(propertyValues(property), "oauth") {
		return APIOAuth{}, fmt.Errorf("declares an OAuth flow, but auth_mode offers no \"oauth\" value")
	}
	authModeDefault, _ := formValue(property.Default)
	return APIOAuth{Enabled: true, AuthModeField: "auth_mode", AuthModeValue: "oauth", AuthModeDefault: authModeDefault, ClientFields: flow.ClientFields}, nil
}

func applyAPIFormContract(config *ConnectorConfig, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read API source form %s: %w", path, err)
	}
	var form apiFormContract
	if err := json.Unmarshal(data, &form); err != nil {
		return fmt.Errorf("parse API source form %s: %w", path, err)
	}
	if len(form.JSONSchema.Properties) == 0 {
		return fmt.Errorf("API source form %s has no json_schema.properties", path)
	}
	required := make(map[string]bool, len(form.JSONSchema.Required))
	for _, name := range form.JSONSchema.Required {
		required[name] = true
	}
	if config.APIOAuth, err = form.oauthMode(); err != nil {
		return fmt.Errorf("API source form %s %w", path, err)
	}
	rules := form.UISchema.rules()
	for i := range config.Config {
		entry := &config.Config[i]
		if !entry.UserDefined {
			continue
		}
		property, ok := form.JSONSchema.Properties[entry.Name]
		if !ok {
			return fmt.Errorf("API source form %s is missing property %q", path, entry.Name)
		}
		var conditionField string
		var showWhen []string
		if rule := rules[entry.Name]; rule != nil {
			var hidden bool
			if conditionField, showWhen, hidden, err = form.visibility(rule); err != nil {
				return fmt.Errorf("API source form %s has an unsupported rule for %q: %w", path, entry.Name, err)
			}
			if hidden {
				entry.UserDefined = false
				continue
			}
		}
		entry.APIFormShow = len(showWhen) > 0
		needsInput := required[entry.Name] && !entry.APIFormShow
		if entry.APIFormShow {
			conditionDefault, _ := formValue(form.JSONSchema.Properties[conditionField].Default)
			config.APIConditions = append(config.APIConditions, APICondition{Field: entry.Name, ConditionField: conditionField, ConditionValues: showWhen, ConditionDefault: conditionDefault, Required: required[entry.Name]})
		}
		entry.Required = &needsInput
		entry.Value.Default = property.Default
		entry.APIFormMinItems = property.MinItems
		if entry.Value.Min == nil {
			entry.Value.Min = property.Minimum
		}
		if entry.Value.Max == nil {
			entry.Value.Max = property.Maximum
		}
		if entry.Value.Control == "multi-select" && len(property.Items.Enum) == 0 {
			entry.Value.RawValues = nil
		}
	}
	if config.APIOAuth.Enabled {
		if config.GetEntryByName(oauthGrantField) != nil {
			return fmt.Errorf("API source form %s: backend config already declares %q", path, oauthGrantField)
		}
		config.Config = append(config.Config, oauthGrantEntry(config.DisplayName))
	}
	return nil
}

const oauthGrantField = "oauth_grant_id"

// oauthGrantEntry is the create/update handoff for the Connect flow. The
// backend accepts it in config and swaps it for the credentials the flow
// stored; it is not a config-model field, so neither artifact declares it.
func oauthGrantEntry(displayName string) ConfigEntry {
	required := false
	return ConfigEntry{
		Name:        oauthGrantField,
		DisplayName: "OAuth grant ID",
		Description: "Single-use grant from the Connect with " + displayName + " flow, which Terraform cannot complete itself. " +
			"Finish Connect in the Streamkap UI, or with the CLI (`streamkap sources start-source-oauth-connect`, open the returned authorize_url, then `streamkap sources poll-source-oauth-grant`), " +
			"and set the returned grant here to create the source or to reconnect it. The grant expires within minutes and is spent when the source is saved. Leave it unset on an imported source.",
		UserDefined: true,
		Required:    &required,
		Encrypt:     true,
		Value:       ValueObject{Control: "password"},
		APIFormShow: true,
	}
}
