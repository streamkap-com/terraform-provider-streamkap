package smt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// ConsumerRequirements is the producer-derived inventory consumers use to
// refuse a catalog whose schema or UI needs capabilities they cannot support.
// The provider represents the structural schema. Semantic constraints such as
// enum, pattern, bounds and format remain server-validated; their presence is
// inventoried here but does not claim a Terraform plan-time validator.
type ConsumerRequirements struct {
	Version          int      `json:"version"`
	SchemaKeywords   []string `json:"schema_keywords"`
	SchemaTypes      []string `json:"schema_types"`
	SchemaFormats    []string `json:"schema_formats"`
	UIElements       []string `json:"ui_elements"`
	UIControls       []string `json:"ui_controls"`
	SecretAddressing []string `json:"secret_addressing"`
}

var acceptedSchemaKeywords = map[string]struct{}{
	"type": {}, "properties": {}, "required": {}, "additionalProperties": {}, "items": {}, "anyOf": {},
	"enum": {}, "const": {}, "title": {}, "description": {}, "default": {}, "examples": {},
	"deprecated": {}, "writeOnly": {}, "format": {}, "minimum": {}, "maximum": {},
	"exclusiveMinimum": {}, "exclusiveMaximum": {}, "multipleOf": {}, "minLength": {}, "maxLength": {},
	"pattern": {}, "minItems": {}, "maxItems": {}, "uniqueItems": {}, "minProperties": {}, "maxProperties": {},
}

var acceptedSchemaTypes = map[string]struct{}{
	"object": {}, "array": {}, "string": {}, "integer": {}, "number": {}, "boolean": {}, "null": {},
}

var requirementKeys = map[string]struct{}{
	"version": {}, "schema_keywords": {}, "schema_types": {}, "schema_formats": {},
	"ui_elements": {}, "ui_controls": {}, "secret_addressing": {},
}

type capabilityInventory struct {
	keywords map[string]struct{}
	types    map[string]struct{}
	formats  map[string]struct{}
	elements map[string]struct{}
	controls map[string]struct{}
}

func validateConsumerRequirements(rawType json.RawMessage, spec *TypeSpec) error {
	var raw struct {
		DataSchema           json.RawMessage `json:"data_schema"`
		UISchema             json.RawMessage `json:"ui_schema"`
		ConsumerRequirements json.RawMessage `json:"consumer_requirements"`
	}
	if err := json.Unmarshal(rawType, &raw); err != nil {
		return fmt.Errorf("decode declaration: %w", err)
	}
	if err := decodeClosedRequirements(raw.ConsumerRequirements, &spec.ConsumerRequirements); err != nil {
		return err
	}

	inv := capabilityInventory{
		keywords: map[string]struct{}{}, types: map[string]struct{}{}, formats: map[string]struct{}{},
		elements: map[string]struct{}{}, controls: map[string]struct{}{},
	}
	dataSchema, err := decodeRawObject(raw.DataSchema, "data_schema")
	if err != nil {
		return err
	}
	if err := inv.walkSchema(dataSchema, "", true); err != nil {
		return err
	}
	if len(raw.UISchema) > 0 && !bytes.Equal(bytes.TrimSpace(raw.UISchema), []byte("null")) {
		uiSchema, err := decodeRawObject(raw.UISchema, "ui_schema")
		if err != nil {
			return err
		}
		if err := inv.walkUI(uiSchema, "/ui_schema"); err != nil {
			return err
		}
	}

	derivedSecretAddressing := []string{}
	if len(spec.SecretPaths) > 0 {
		derivedSecretAddressing = append(derivedSecretAddressing, "json-pointer-v1")
	}
	if len(spec.RowKeys) > 0 {
		derivedSecretAddressing = append(derivedSecretAddressing, "keyed-rows-v1")
	}
	checks := []struct {
		name     string
		declared []string
		derived  []string
	}{
		{"schema_keywords", spec.ConsumerRequirements.SchemaKeywords, sortedKeys(inv.keywords)},
		{"schema_types", spec.ConsumerRequirements.SchemaTypes, sortedKeys(inv.types)},
		{"schema_formats", spec.ConsumerRequirements.SchemaFormats, sortedKeys(inv.formats)},
		{"ui_elements", spec.ConsumerRequirements.UIElements, sortedKeys(inv.elements)},
		{"ui_controls", spec.ConsumerRequirements.UIControls, sortedKeys(inv.controls)},
		{"secret_addressing", spec.ConsumerRequirements.SecretAddressing, derivedSecretAddressing},
	}
	for _, check := range checks {
		if err := validateCanonicalList(check.name, check.declared); err != nil {
			return err
		}
		if !equalStrings(check.declared, check.derived) {
			return fmt.Errorf("consumer_requirements.%s mismatch: declared %v, derived %v", check.name, check.declared, check.derived)
		}
	}
	return nil
}

func decodeClosedRequirements(raw json.RawMessage, out *ConsumerRequirements) error {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fmt.Errorf("consumer_requirements is missing")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("consumer_requirements: expected an object: %w", err)
	}
	for name := range fields {
		if _, ok := requirementKeys[name]; !ok {
			return fmt.Errorf("consumer_requirements contains unknown member %q", name)
		}
	}
	for name := range requirementKeys {
		if _, ok := fields[name]; !ok {
			return fmt.Errorf("consumer_requirements.%s is missing", name)
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("consumer_requirements: %w", err)
	}
	if out.Version != 1 {
		return fmt.Errorf("consumer_requirements.version must be 1, got %d", out.Version)
	}
	return nil
}

func decodeRawObject(raw json.RawMessage, name string) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%s is missing", name)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", name)
	}
	return object, nil
}

func (inv *capabilityInventory) walkSchema(node map[string]any, path string, requireClosedObjects bool) error {
	for keyword := range node {
		if _, ok := acceptedSchemaKeywords[keyword]; !ok {
			return fmt.Errorf("unsupported schema keyword %q at %s", keyword, displayPath(path))
		}
		inv.keywords[keyword] = struct{}{}
	}
	typeName := ""
	if rawType, ok := node["type"]; ok {
		typeName, ok = rawType.(string)
		if !ok {
			return fmt.Errorf("schema type at %s must be one string", displayPath(path))
		}
		if _, ok := acceptedSchemaTypes[typeName]; !ok {
			return fmt.Errorf("unsupported schema type %q at %s", typeName, displayPath(path))
		}
		inv.types[typeName] = struct{}{}
	}
	if rawFormat, ok := node["format"]; ok {
		format, ok := rawFormat.(string)
		if !ok {
			return fmt.Errorf("schema format at %s must be a string", displayPath(path))
		}
		inv.formats[format] = struct{}{}
	}
	if rawAdditional, ok := node["additionalProperties"]; ok {
		additional, ok := rawAdditional.(bool)
		if !ok || additional {
			return fmt.Errorf("schema additionalProperties at %s must be false", displayPath(path))
		}
	} else if requireClosedObjects && typeName == "object" {
		return fmt.Errorf("schema additionalProperties at %s must be false", displayPath(path))
	}
	if rawProperties, ok := node["properties"]; ok {
		properties, ok := rawProperties.(map[string]any)
		if !ok {
			return fmt.Errorf("schema properties at %s must be an object", displayPath(path))
		}
		for name, rawProperty := range properties {
			property, ok := rawProperty.(map[string]any)
			if !ok {
				return fmt.Errorf("schema property at %s must be an object", displayPath(joinPath(path, name)))
			}
			if err := inv.walkSchema(property, joinPath(path, name), requireClosedObjects); err != nil {
				return err
			}
		}
	}
	if rawItems, ok := node["items"]; ok {
		items, ok := rawItems.(map[string]any)
		if !ok {
			return fmt.Errorf("schema items at %s must be an object", displayPath(path))
		}
		if err := inv.walkSchema(items, joinPath(path, "*"), requireClosedObjects); err != nil {
			return err
		}
	}
	if rawAnyOf, ok := node["anyOf"]; ok {
		branches, ok := rawAnyOf.([]any)
		if !ok {
			return fmt.Errorf("schema anyOf at %s must be an array", displayPath(path))
		}
		for i, rawBranch := range branches {
			branch, ok := rawBranch.(map[string]any)
			if !ok {
				return fmt.Errorf("schema anyOf branch %d at %s must be an object", i, displayPath(path))
			}
			if err := inv.walkSchema(branch, path, requireClosedObjects); err != nil {
				return err
			}
		}
	}
	return nil
}

var numericSchemaKeywords = []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "multipleOf"}

func validateRawNumericType(rawType json.RawMessage) error {
	var raw struct {
		DataSchema json.RawMessage `json:"data_schema"`
		UISchema   json.RawMessage `json:"ui_schema"`
	}
	if err := json.Unmarshal(rawType, &raw); err != nil {
		return fmt.Errorf("decode declaration: %w", err)
	}
	dataSchema, err := decodeRawObject(raw.DataSchema, "data_schema")
	if err != nil {
		return err
	}
	if err := validateRawNumericSchema(dataSchema, "", true); err != nil {
		return err
	}
	if len(raw.UISchema) == 0 || bytes.Equal(bytes.TrimSpace(raw.UISchema), []byte("null")) {
		return nil
	}
	uiSchema, err := decodeRawObject(raw.UISchema, "ui_schema")
	if err != nil {
		return err
	}
	return validateRawUINumericSchemas(uiSchema, "/ui_schema", dataSchema)
}

func validateRawNumericSchema(node map[string]any, path string, requireIntegerBounds bool) error {
	typeName := schemaType(node)
	if err := validateRepresentableNumericSchema(node, path, typeName, requireIntegerBounds); err != nil {
		return err
	}
	if properties, ok := node["properties"].(map[string]any); ok {
		for name, rawProperty := range properties {
			if property, ok := rawProperty.(map[string]any); ok {
				if err := validateRawNumericSchema(property, joinPath(path, name), requireIntegerBounds); err != nil {
					return err
				}
			}
		}
	}
	if items, ok := node["items"].(map[string]any); ok {
		if err := validateRawNumericSchema(items, joinPath(path, "*"), requireIntegerBounds); err != nil {
			return err
		}
	}
	if branches, ok := node["anyOf"].([]any); ok {
		for _, rawBranch := range branches {
			if branch, ok := rawBranch.(map[string]any); ok {
				if err := validateRawNumericSchema(branch, path, requireIntegerBounds); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateRawUINumericSchemas(node map[string]any, path string, dataSchema map[string]any) error {
	if rule, ok := node["rule"].(map[string]any); ok {
		if condition, ok := rule["condition"].(map[string]any); ok {
			if err := validateRawNumericCondition(condition, path+"/rule/condition", dataSchema); err != nil {
				return err
			}
		}
	}
	if elements, ok := node["elements"].([]any); ok {
		for i, rawElement := range elements {
			if element, ok := rawElement.(map[string]any); ok {
				if err := validateRawUINumericSchemas(element, fmt.Sprintf("%s/elements/%d", path, i), dataSchema); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func validateRawNumericCondition(condition map[string]any, path string, dataSchema map[string]any) error {
	if schema, ok := condition["schema"].(map[string]any); ok {
		context := map[string]any(nil)
		if scope, ok := condition["scope"].(string); ok {
			context, _ = resolveSchemaScope(dataSchema, scope)
		}
		if err := validateContextualNumericSchema(schema, path+"/schema", context); err != nil {
			return err
		}
	}
	if conditions, ok := condition["conditions"].([]any); ok {
		for i, rawChild := range conditions {
			if child, ok := rawChild.(map[string]any); ok {
				if err := validateRawNumericCondition(child, fmt.Sprintf("%s/conditions/%d", path, i), dataSchema); err != nil {
					return err
				}
			}
		}
	}
	if child, ok := condition["condition"].(map[string]any); ok {
		return validateRawNumericCondition(child, path+"/condition", dataSchema)
	}
	return nil
}

func validateContextualNumericSchema(node map[string]any, path string, context map[string]any) error {
	contextSchema := context
	contextType := ""
	if context != nil {
		contextSchema, contextType = literalContext(context, schemaType(context))
	}
	typeName := schemaType(node)
	if typeName == "" {
		typeName = contextType
	}
	if err := validateRepresentableNumericSchema(node, path, typeName, false); err != nil {
		return err
	}
	if properties, ok := node["properties"].(map[string]any); ok {
		contextProperties, _ := contextSchema["properties"].(map[string]any)
		for name, rawProperty := range properties {
			property, ok := rawProperty.(map[string]any)
			if !ok {
				continue
			}
			childContext, _ := contextProperties[name].(map[string]any)
			if err := validateContextualNumericSchema(property, joinPath(path, name), childContext); err != nil {
				return err
			}
		}
	}
	if items, ok := node["items"].(map[string]any); ok {
		contextItems, _ := contextSchema["items"].(map[string]any)
		if err := validateContextualNumericSchema(items, joinPath(path, "*"), contextItems); err != nil {
			return err
		}
	}
	if branches, ok := node["anyOf"].([]any); ok {
		for _, rawBranch := range branches {
			if branch, ok := rawBranch.(map[string]any); ok {
				if err := validateContextualNumericSchema(branch, path, context); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func resolveSchemaScope(root map[string]any, scope string) (map[string]any, bool) {
	if !strings.HasPrefix(scope, "#/") {
		return nil, false
	}
	var current any = root
	for _, token := range strings.Split(strings.TrimPrefix(scope, "#/"), "/") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[UnescapeToken(token)]
		if !ok {
			return nil, false
		}
	}
	resolved, ok := current.(map[string]any)
	return resolved, ok
}

func validateRepresentableNumericSchema(node map[string]any, path, typeName string, requireIntegerBounds bool) error {
	if typeName == "integer" && requireIntegerBounds {
		minimum, ok, err := exactSchemaInteger(node, "minimum", path)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("schema integer minimum at %s is required", displayPath(path))
		}
		maximum, ok, err := exactSchemaInteger(node, "maximum", path)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("schema integer maximum at %s is required", displayPath(path))
		}
		if minimum > maximum {
			return fmt.Errorf("schema integer bounds at %s must be ordered", displayPath(path))
		}
	}
	if typeName == "integer" || typeName == "number" {
		for _, keyword := range numericSchemaKeywords {
			if err := validateNumericLiteral(node[keyword], node, typeName, path, keyword); err != nil {
				return err
			}
		}
	}
	literalSchema, literalType := literalContext(node, typeName)
	for _, keyword := range []string{"default", "const"} {
		if err := validateNumericLiteral(node[keyword], literalSchema, literalType, path, keyword); err != nil {
			return err
		}
	}
	for _, keyword := range []string{"enum", "examples"} {
		raw, ok := node[keyword]
		if !ok {
			continue
		}
		values, ok := raw.([]any)
		if !ok {
			return fmt.Errorf("schema %s at %s must be an array", keyword, displayPath(path))
		}
		for _, value := range values {
			if err := validateNumericLiteral(value, literalSchema, literalType, path, keyword); err != nil {
				return err
			}
		}
	}
	return nil
}

func literalContext(node map[string]any, typeName string) (map[string]any, string) {
	if typeName != "" {
		return node, typeName
	}
	branches, ok := node["anyOf"].([]any)
	if !ok {
		return node, ""
	}
	var selected map[string]any
	for _, raw := range branches {
		branch, ok := raw.(map[string]any)
		if !ok || branch["type"] == "null" {
			continue
		}
		if selected != nil {
			return node, ""
		}
		selected = branch
	}
	if selected == nil {
		return node, ""
	}
	typeName, _ = selected["type"].(string)
	return selected, typeName
}

func validateNumericLiteral(raw any, schema map[string]any, typeName, path, keyword string) error {
	if raw == nil {
		return nil
	}
	switch typeName {
	case "integer":
		number, ok := raw.(json.Number)
		if !ok {
			return fmt.Errorf("schema %s at %s must be an integer", keyword, displayPath(path))
		}
		if _, ok := exactJSONSafeIntegerLiteral(number.String()); !ok {
			return fmt.Errorf("schema %s at %s must be in the exact JSON integer domain", keyword, displayPath(path))
		}
	case "number":
		number, ok := raw.(json.Number)
		if !ok {
			return fmt.Errorf("schema %s at %s must be a number", keyword, displayPath(path))
		}
		value, err := strconv.ParseFloat(number.String(), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("schema %s at %s must be a finite IEEE 754 number", keyword, displayPath(path))
		}
	case "":
		number, ok := raw.(json.Number)
		if !ok {
			return nil
		}
		value, err := strconv.ParseFloat(number.String(), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("schema %s at %s must be a finite IEEE 754 number", keyword, displayPath(path))
		}
		if value == math.Trunc(value) {
			if _, ok := exactJSONSafeIntegerLiteral(number.String()); !ok {
				return fmt.Errorf("schema %s at %s must be in the exact JSON integer domain when its type is omitted", keyword, displayPath(path))
			}
		}
	case "object":
		value, ok := raw.(map[string]any)
		if !ok {
			return nil
		}
		properties, _ := schema["properties"].(map[string]any)
		for name, childValue := range value {
			child, ok := properties[name].(map[string]any)
			if !ok {
				continue
			}
			childSchema, childType := literalContext(child, schemaType(child))
			if err := validateNumericLiteral(childValue, childSchema, childType, joinPath(path, name), keyword); err != nil {
				return err
			}
		}
	case "array":
		values, ok := raw.([]any)
		if !ok {
			return nil
		}
		items, ok := schema["items"].(map[string]any)
		if !ok {
			return nil
		}
		itemSchema, itemType := literalContext(items, schemaType(items))
		for _, value := range values {
			if err := validateNumericLiteral(value, itemSchema, itemType, joinPath(path, "*"), keyword); err != nil {
				return err
			}
		}
	}
	return nil
}

func schemaType(node map[string]any) string {
	typeName, _ := node["type"].(string)
	return typeName
}

func exactSchemaInteger(node map[string]any, keyword, path string) (int64, bool, error) {
	raw, ok := node[keyword]
	if !ok {
		return 0, false, nil
	}
	number, ok := raw.(json.Number)
	if !ok {
		return 0, false, fmt.Errorf("schema %s at %s must be an integer", keyword, displayPath(path))
	}
	value, ok := exactJSONSafeIntegerLiteral(number.String())
	if !ok {
		return 0, false, fmt.Errorf("schema %s at %s must be in the exact JSON integer domain", keyword, displayPath(path))
	}
	return value, true, nil
}

func (inv *capabilityInventory) walkUI(node map[string]any, path string) error {
	if rawType, ok := node["type"]; ok {
		elementType, ok := rawType.(string)
		if !ok {
			return fmt.Errorf("UI element type at %s must be a string", path)
		}
		inv.elements[elementType] = struct{}{}
	}
	if rawOptions, ok := node["options"]; ok {
		options, ok := rawOptions.(map[string]any)
		if !ok {
			return fmt.Errorf("UI options at %s must be an object", path)
		}
		if rawControl, ok := options["control"]; ok {
			control, ok := rawControl.(string)
			if !ok {
				return fmt.Errorf("UI control at %s must be a string", path)
			}
			inv.controls[control] = struct{}{}
		}
	}
	if rawRule, ok := node["rule"]; ok {
		rule, ok := rawRule.(map[string]any)
		if !ok {
			return fmt.Errorf("UI rule at %s must be an object", path)
		}
		if rawCondition, ok := rule["condition"]; ok {
			condition, ok := rawCondition.(map[string]any)
			if !ok {
				return fmt.Errorf("UI rule condition at %s must be an object", path)
			}
			if err := inv.walkCondition(condition, path+"/rule/condition"); err != nil {
				return err
			}
		}
	}
	if rawElements, ok := node["elements"]; ok {
		elements, ok := rawElements.([]any)
		if !ok {
			return fmt.Errorf("UI elements at %s must be an array", path)
		}
		for i, rawElement := range elements {
			element, ok := rawElement.(map[string]any)
			if !ok {
				return fmt.Errorf("UI element %d at %s must be an object", i, path)
			}
			if err := inv.walkUI(element, fmt.Sprintf("%s/elements/%d", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (inv *capabilityInventory) walkCondition(condition map[string]any, path string) error {
	if rawSchema, ok := condition["schema"]; ok {
		schema, ok := rawSchema.(map[string]any)
		if !ok {
			return fmt.Errorf("UI rule condition schema at %s must be an object", path)
		}
		if err := inv.walkSchema(schema, path+"/schema", false); err != nil {
			return err
		}
	}
	if rawConditions, ok := condition["conditions"]; ok {
		conditions, ok := rawConditions.([]any)
		if !ok {
			return fmt.Errorf("UI rule conditions at %s must be an array", path)
		}
		for i, rawChild := range conditions {
			child, ok := rawChild.(map[string]any)
			if !ok {
				return fmt.Errorf("UI rule condition %d at %s must be an object", i, path)
			}
			if err := inv.walkCondition(child, fmt.Sprintf("%s/conditions/%d", path, i)); err != nil {
				return err
			}
		}
	}
	if rawChild, ok := condition["condition"]; ok {
		child, ok := rawChild.(map[string]any)
		if !ok {
			return fmt.Errorf("UI rule condition at %s must be an object", path)
		}
		return inv.walkCondition(child, path+"/condition")
	}
	return nil
}

func validateCanonicalList(name string, values []string) error {
	for i, value := range values {
		if i > 0 && values[i-1] == value {
			return fmt.Errorf("consumer_requirements.%s contains duplicate %q", name, value)
		}
		if i > 0 && values[i-1] > value {
			return fmt.Errorf("consumer_requirements.%s must be sorted", name)
		}
	}
	return nil
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	sort.Strings(keys)
	return keys
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func joinPath(parent, child string) string {
	escaped := strings.ReplaceAll(strings.ReplaceAll(child, "~", "~0"), "/", "~1")
	return parent + "/" + escaped
}

func displayPath(path string) string {
	if path == "" {
		return "/"
	}
	return path
}
