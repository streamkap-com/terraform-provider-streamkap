package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type apiFormContract struct {
	OAuth      json.RawMessage `json:"oauth"`
	JSONSchema struct {
		Required   []string `json:"required"`
		Properties map[string]struct {
			Default  any `json:"default"`
			MinItems int `json:"minItems"`
			Items    struct {
				Enum []any `json:"enum"`
			} `json:"items"`
		} `json:"properties"`
	} `json:"json_schema"`
	UISchema apiFormControl `json:"ui_schema"`
}

type apiFormControl struct {
	Type     string           `json:"type"`
	Scope    string           `json:"scope"`
	Elements []apiFormControl `json:"elements"`
	Rule     *struct {
		Effect    string `json:"effect"`
		Condition struct {
			Scope  string `json:"scope"`
			Schema struct {
				Const any `json:"const"`
			} `json:"schema"`
		} `json:"condition"`
	} `json:"rule"`
}

func (control apiFormControl) rules() map[string]apiFormControl {
	rules := map[string]apiFormControl{}
	var visit func(apiFormControl)
	visit = func(item apiFormControl) {
		if item.Type == "Control" && item.Rule != nil {
			rules[strings.TrimPrefix(item.Scope, "#/properties/")] = item
		}
		for _, child := range item.Elements {
			visit(child)
		}
	}
	visit(control)
	return rules
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
	rules := form.UISchema.rules()
	config.APIOAuth = len(form.OAuth) > 0 && string(form.OAuth) != "null"
	for i := range config.Config {
		entry := &config.Config[i]
		if !entry.UserDefined {
			continue
		}
		property, ok := form.JSONSchema.Properties[entry.Name]
		if !ok {
			return fmt.Errorf("API source form %s is missing property %q", path, entry.Name)
		}
		rule := rules[entry.Name].Rule
		if rule != nil && rule.Effect == "HIDE" {
			entry.UserDefined = false
			continue
		}
		entry.APIFormShow = rule != nil && rule.Effect == "SHOW"
		needsInput := required[entry.Name] && !entry.APIFormShow
		if required[entry.Name] && entry.APIFormShow {
			conditionField := strings.TrimPrefix(rule.Condition.Scope, "#/properties/")
			conditionValue, ok := rule.Condition.Schema.Const.(string)
			if !ok || conditionField == "" || conditionField == rule.Condition.Scope {
				return fmt.Errorf("API source form %s has unsupported requirement for %q", path, entry.Name)
			}
			conditionProperty, ok := form.JSONSchema.Properties[conditionField]
			if !ok {
				return fmt.Errorf("API source form %s has no condition property %q", path, conditionField)
			}
			conditionDefault, _ := conditionProperty.Default.(string)
			config.APIRequirements = append(config.APIRequirements, APIRequirement{Field: entry.Name, ConditionField: conditionField, ConditionValue: conditionValue, ConditionDefault: conditionDefault})
		}
		entry.Required = &needsInput
		entry.Value.Default = property.Default
		entry.APIFormMinItems = property.MinItems
		if entry.Value.Control == "multi-select" && len(property.Items.Enum) == 0 {
			entry.Value.RawValues = nil
		}
	}
	return nil
}
