package smt

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func validateSecretMetadata(raw json.RawMessage, spec *TypeSpec) error {
	var declared struct {
		SecretPaths json.RawMessage `json:"secret_paths"`
		RowKeys     json.RawMessage `json:"row_keys"`
	}
	if err := json.Unmarshal(raw, &declared); err != nil {
		return fmt.Errorf("secret metadata: %w", err)
	}
	if len(declared.SecretPaths) == 0 || string(declared.SecretPaths) == "null" {
		return fmt.Errorf("secret_paths must be an array")
	}
	var secretPaths []string
	if err := json.Unmarshal(declared.SecretPaths, &secretPaths); err != nil {
		return fmt.Errorf("secret_paths must be an array of strings: %w", err)
	}
	if len(declared.RowKeys) == 0 || string(declared.RowKeys) == "null" {
		return fmt.Errorf("row_keys must be an object")
	}
	var rowKeys map[string]string
	if err := json.Unmarshal(declared.RowKeys, &rowKeys); err != nil {
		return fmt.Errorf("row_keys must be an object of string values: %w", err)
	}

	derivedPaths, err := collectWriteOnlyLeaves(spec.DataSchema, "")
	if err != nil {
		return err
	}
	sort.Strings(derivedPaths)
	if err := validateSecretPaths(secretPaths, derivedPaths); err != nil {
		return err
	}
	return validateRowKeys(spec.DataSchema, derivedPaths, rowKeys)
}

func collectWriteOnlyLeaves(node *JSONSchema, schemaPath string) ([]string, error) {
	if node == nil {
		return nil, &UnsupportedConstructError{Path: displayPath(schemaPath), Construct: "missing data_schema", Detail: "secret metadata cannot be derived"}
	}
	b := &builder{}
	kind, _, err := b.scalarType(node, displayPath(schemaPath))
	if err != nil {
		return nil, err
	}
	structural := structuralSchema(node)
	if node.WriteOnly || structural.WriteOnly {
		if kind == "object" || kind == "array" {
			return nil, &UnsupportedConstructError{Path: displayPath(schemaPath), Construct: "writeOnly", Detail: "only scalar leaves can be secrets"}
		}
		return []string{displayPath(schemaPath)}, nil
	}
	switch kind {
	case "object":
		names := make([]string, 0, len(structural.Properties))
		for name := range structural.Properties {
			names = append(names, name)
		}
		sort.Strings(names)
		var paths []string
		for _, name := range names {
			childPaths, err := collectWriteOnlyLeaves(structural.Properties[name], schemaPath+"/"+EscapeToken(name))
			if err != nil {
				return nil, err
			}
			paths = append(paths, childPaths...)
		}
		return paths, nil
	case "array":
		var item JSONSchema
		if err := json.Unmarshal(structural.Items, &item); err != nil {
			return nil, &UnsupportedConstructError{Path: displayPath(schemaPath), Construct: "items", Detail: err.Error()}
		}
		return collectWriteOnlyLeaves(&item, schemaPath+"/*")
	default:
		return nil, nil
	}
}

func validateSecretPaths(declared, derived []string) error {
	seen := make(map[string]bool, len(declared))
	for _, path := range declared {
		if seen[path] {
			return fmt.Errorf("secret_paths contains duplicate %q", path)
		}
		seen[path] = true
	}
	derivedSet := make(map[string]bool, len(derived))
	for _, path := range derived {
		derivedSet[path] = true
	}
	for _, path := range declared {
		if !derivedSet[path] {
			return fmt.Errorf("secret_paths contains %q, which is not a writeOnly leaf", path)
		}
	}
	for _, path := range derived {
		if !seen[path] {
			return fmt.Errorf("secret_paths is missing writeOnly leaf %q", path)
		}
	}
	return nil
}

func validateRowKeys(root *JSONSchema, secretPaths []string, declared map[string]string) error {
	requiredPaths := map[string]bool{}
	for _, secretPath := range secretPaths {
		parts := strings.Split(strings.TrimPrefix(secretPath, "/"), "/")
		for i, part := range parts {
			if part == "*" {
				requiredPaths["/"+strings.Join(parts[:i], "/")] = true
			}
		}
	}
	for path := range declared {
		if !requiredPaths[path] {
			return fmt.Errorf("row_keys contains %q, which is not a secret-bearing object list", path)
		}
	}
	paths := make([]string, 0, len(requiredPaths))
	for path := range requiredPaths {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		row, err := objectListRow(root, path)
		if err != nil {
			return err
		}
		expectedKey := ""
		for _, candidate := range []string{"key", "id", "name"} {
			if containsString(row.Required, candidate) {
				expectedKey = candidate
				break
			}
		}
		if expectedKey == "" {
			return fmt.Errorf("row_keys %q needs a declared required key, id or name property", path)
		}
		key, ok := declared[path]
		if !ok {
			return fmt.Errorf("row_keys is missing %q for required identity property %q", path, expectedKey)
		}
		if key != expectedKey {
			return fmt.Errorf("row_keys %q must name required identity property %q, got %q", path, expectedKey, key)
		}
		property, ok := row.Properties[key]
		if !ok {
			return fmt.Errorf("row_keys %q names unknown property %q", path, key)
		}
		if !containsString(row.Required, key) {
			return fmt.Errorf("row_keys %q property %q must be required", path, key)
		}
		kind, nullable, err := (&builder{}).scalarType(property, path+"/*/"+key)
		if err != nil {
			return err
		}
		if kind != "string" || nullable {
			return fmt.Errorf("row_keys %q property %q must be a non-null string", path, key)
		}
		if property.WriteOnly || structuralSchema(property).WriteOnly {
			return fmt.Errorf("row_keys %q property %q must not be writeOnly", path, key)
		}
		if len(property.Default) > 0 {
			return fmt.Errorf("row_keys %q property %q must be immutable and have no default", path, key)
		}
	}
	return nil
}

func objectListRow(root *JSONSchema, listPath string) (*JSONSchema, error) {
	node := root
	parts := strings.Split(strings.TrimPrefix(listPath, "/"), "/")
	for _, part := range parts {
		structural := structuralSchema(node)
		if part == "*" {
			var item JSONSchema
			if err := json.Unmarshal(structural.Items, &item); err != nil {
				return nil, fmt.Errorf("row_keys %q does not resolve to an object list", listPath)
			}
			node = &item
			continue
		}
		property, ok := structural.Properties[UnescapeToken(part)]
		if !ok {
			return nil, fmt.Errorf("row_keys %q does not resolve to an object list", listPath)
		}
		node = property
	}
	structural := structuralSchema(node)
	var row JSONSchema
	if err := json.Unmarshal(structural.Items, &row); err != nil {
		return nil, fmt.Errorf("row_keys %q does not resolve to an object list", listPath)
	}
	return structuralSchema(&row), nil
}

func structuralSchema(node *JSONSchema) *JSONSchema {
	if len(node.AnyOf) == 0 {
		return node
	}
	for _, branch := range node.AnyOf {
		if kind, _ := branch.Type.(string); kind != "null" {
			return branch
		}
	}
	return node
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
