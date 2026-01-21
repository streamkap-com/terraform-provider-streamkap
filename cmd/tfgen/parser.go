// Package main provides the tfgen CLI tool for generating Terraform provider
// schemas from Streamkap backend configuration.latest.json files.
//
// # Parser Module (parser.go)
//
// This file implements the configuration parser that reads backend JSON schemas
// and extracts field metadata for Terraform schema generation.
//
// # Input Format
//
// The parser reads configuration.latest.json files from the backend repository.
// Each file defines a connector's configuration fields:
//
//	{
//	  "display_name": "PostgreSQL",
//	  "config": [
//	    {
//	      "name": "database.hostname.user.defined",
//	      "user_defined": true,
//	      "required": true,
//	      "value": {"control": "text", "type": "raw"}
//	    }
//	  ]
//	}
//
// # Key Data Structures
//
// ConnectorConfig: Top-level structure containing display name and config array.
//
// ConfigEntry: Individual configuration field with name, user_defined flag,
// required status, display settings, and value metadata.
//
// ValueObject: Field value metadata including control type (text, select, toggle,
// slider, password), default values, validation constraints (min/max/step for
// sliders), and raw_values for select options.
//
// Condition: Conditional visibility rules (EQ, NE, IN operators) for fields
// that depend on other field values.
//
// # Type Mapping
//
// The TerraformType method maps backend controls to Terraform types:
//   - text, select, textarea, password, file → types.String
//   - number, slider → types.Int64
//   - toggle, checkbox → types.Bool
//   - multi_select → types.List[types.String]
//
// # User-Defined Field Filtering
//
// Only fields with user_defined=true are exposed in Terraform schemas.
// Dynamic/computed fields (user_defined=false) are handled by the backend
// and excluded from user input.
//
// # Attribute Naming
//
// TerraformAttributeName converts backend names to Terraform conventions:
//   - Strips ".user.defined" suffix
//   - Replaces dots and hyphens with underscores
//   - Example: "database.hostname.user.defined" → "database_hostname"
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ConnectorConfig represents the top-level structure of a configuration.latest.json file.
type ConnectorConfig struct {
	DisplayName          string        `json:"display_name"`
	Description          string        `json:"description,omitempty"`
	SchemaLevels         []string      `json:"schema_levels,omitempty"`
	DebeziumConnectorName string       `json:"debezium_connector_name,omitempty"`
	Serialisation        string        `json:"serialisation,omitempty"`
	Metrics              []Metric      `json:"metrics,omitempty"`
	Config               []ConfigEntry `json:"config"`
}

// Metric represents a metrics definition (primarily for sources).
type Metric struct {
	Attribute              string   `json:"attribute"`
	Context                string   `json:"context,omitempty"`
	Category               string   `json:"category,omitempty"`
	TimeSeriesAggregation  string   `json:"time_series_aggregation,omitempty"`
	AllTimeAggregation     string   `json:"all_time_aggregation,omitempty"`
	PartAggregation        string   `json:"part_aggregation,omitempty"`
	InitialValue           any      `json:"initial_value,omitempty"`
	ValueColumn            string   `json:"value_column,omitempty"`
	InitialLabels          []string `json:"initial_labels,omitempty"`
	ClickhouseMetricName   string   `json:"clickhouse_metric_name,omitempty"`
}

// ConfigEntry represents a single configuration entry in the config array.
type ConfigEntry struct {
	Name                 string       `json:"name"`
	Description          string       `json:"description,omitempty"`
	UserDefined          bool         `json:"user_defined"`
	Required             *bool        `json:"required,omitempty"`
	OrderOfDisplay       *int         `json:"order_of_display,omitempty"`
	DisplayName          string       `json:"display_name,omitempty"`
	Value                ValueObject  `json:"value"`
	Tab                  string       `json:"tab,omitempty"`
	KafkaConfig          any          `json:"kafka_config,omitempty"` // Can be bool or object
	Encrypt              bool         `json:"encrypt,omitempty"`
	DisplayAdvanced      bool         `json:"display_advanced,omitempty"`
	Conditions           []Condition  `json:"conditions,omitempty"`
	SchemaLevel          string       `json:"schema_level,omitempty"`
	SchemaNameFormat     string       `json:"schema_name_format,omitempty"`
	SSHUpdateDeterminant bool         `json:"ssh_update_determinant,omitempty"`
	ShowLast             *int         `json:"show_last,omitempty"`
	NotClonable          bool         `json:"not_clonable,omitempty"`
	Global               bool         `json:"global,omitempty"`
	SetOnce              bool         `json:"set_once,omitempty"`
	IsOverwrite          bool         `json:"is_overwrite,omitempty"`
	IsDeleted            bool         `json:"is_deleted,omitempty"`
}

// ValueObject represents the value field in a config entry.
type ValueObject struct {
	Control       string   `json:"control,omitempty"`
	Type          string   `json:"type,omitempty"`          // "raw" or "dynamic"
	Default       any      `json:"default,omitempty"`       // Can be string, int, bool, etc.
	RawValue      any      `json:"raw_value,omitempty"`     // Static value when type=raw, user_defined=false
	RawValues     []any    `json:"raw_values,omitempty"`    // Options for select controls (can be strings or bools)
	FunctionName  string   `json:"function_name,omitempty"` // Dynamic resolution function
	Dependencies  []string `json:"dependencies,omitempty"`  // Dependencies for dynamic resolution
	Max           *float64 `json:"max,omitempty"`           // Slider: maximum value
	Min           *float64 `json:"min,omitempty"`           // Slider: minimum value
	Step          *float64 `json:"step,omitempty"`          // Slider: step increment
	Rows          *int     `json:"rows,omitempty"`          // Textarea: display rows
	Placeholder   string   `json:"placeholder,omitempty"`   // Input placeholder text
	Readonly      bool     `json:"readonly,omitempty"`      // Read-only field
	Multiline     bool     `json:"multiline,omitempty"`     // Textarea: allow multiline input
	EarlyResolved string   `json:"early_resolved,omitempty"` // Early resolution function
	Validation    any      `json:"validation,omitempty"`    // Custom validation rules (flexible type)
}

// Condition represents conditional visibility based on other field values.
type Condition struct {
	Operator string `json:"operator"` // "EQ", "NE", "IN"
	Config   string `json:"config"`   // The config field to check
	Value    any    `json:"value"`    // The value to compare against
}

// TerraformType represents the Terraform type for a config entry.
type TerraformType string

const (
	TerraformTypeString TerraformType = "types.String"
	TerraformTypeInt64  TerraformType = "types.Int64"
	TerraformTypeBool   TerraformType = "types.Bool"
	TerraformTypeList   TerraformType = "types.List[types.String]"
)

// ParseConnectorConfig reads and parses a configuration.latest.json file.
func ParseConnectorConfig(filepath string) (*ConnectorConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", filepath, err)
	}

	var config ConnectorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", filepath, err)
	}

	return &config, nil
}

// IsUserDefined returns true if this config entry should become a Terraform attribute.
// Only fields with user_defined=true should be exposed in the Terraform schema.
func (e *ConfigEntry) IsUserDefined() bool {
	return e.UserDefined
}

// IsSensitive returns true if this config entry should be marked as sensitive in Terraform.
// Fields with encrypt=true or control=password are considered sensitive.
func (e *ConfigEntry) IsSensitive() bool {
	return e.Encrypt || e.Value.Control == "password"
}

// IsRequired returns true if this config entry is required.
func (e *ConfigEntry) IsRequired() bool {
	if e.Required != nil {
		return *e.Required
	}
	return false
}

// IsSetOnce returns true if this config entry can only be set on resource creation.
// Such fields should use RequiresReplace() plan modifier in Terraform.
func (e *ConfigEntry) IsSetOnce() bool {
	return e.SetOnce
}

// IsReadOnly returns true if this config entry is read-only.
func (e *ConfigEntry) IsReadOnly() bool {
	return e.Value.Readonly
}

// IsDynamic returns true if this config entry's value is dynamically computed by the backend.
func (e *ConfigEntry) IsDynamic() bool {
	return e.Value.Type == "dynamic"
}

// IsConditional returns true if this config entry has conditional visibility.
func (e *ConfigEntry) IsConditional() bool {
	return len(e.Conditions) > 0
}

// HasDefault returns true if this config entry has a default value.
func (e *ConfigEntry) HasDefault() bool {
	return e.Value.Default != nil
}

// GetDefault returns the default value for this config entry.
// Returns nil if no default is set.
func (e *ConfigEntry) GetDefault() any {
	return e.Value.Default
}

// GetDefaultString returns the default value as a string, or empty string if not set.
func (e *ConfigEntry) GetDefaultString() string {
	if e.Value.Default == nil {
		return ""
	}
	switch v := e.Value.Default.(type) {
	case string:
		return v
	case float64:
		// JSON numbers are decoded as float64
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetDefaultInt64 returns the default value as int64, or 0 if not set or not a number.
func (e *ConfigEntry) GetDefaultInt64() int64 {
	if e.Value.Default == nil {
		return 0
	}
	switch v := e.Value.Default.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}

// GetDefaultInt64FromString returns the default value as int64, parsing from string if necessary.
// This is useful for fields like ports where the backend stores the value as a string but
// we want to use Int64 in Terraform for better UX.
func (e *ConfigEntry) GetDefaultInt64FromString() int64 {
	if e.Value.Default == nil {
		return 0
	}
	switch v := e.Value.Default.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case string:
		// Parse string as int64
		if val, err := strconv.ParseInt(v, 10, 64); err == nil {
			return val
		}
		return 0
	default:
		return 0
	}
}

// GetDefaultBool returns the default value as bool, or false if not set or not a boolean.
func (e *ConfigEntry) GetDefaultBool() bool {
	if e.Value.Default == nil {
		return false
	}
	switch v := e.Value.Default.(type) {
	case bool:
		return v
	default:
		return false
	}
}

// TerraformType returns the appropriate Terraform type for this config entry.
// This maps control types to Terraform types as specified in the audit document.
func (e *ConfigEntry) TerraformType() TerraformType {
	switch e.Value.Control {
	case "string", "password", "textarea", "json", "datetime":
		return TerraformTypeString
	case "number":
		// Number is often stored as string in the API, but we use Int64 for type safety
		return TerraformTypeInt64
	case "boolean", "toggle":
		return TerraformTypeBool
	case "one-select":
		return TerraformTypeString
	case "multi-select":
		return TerraformTypeList
	case "slider":
		return TerraformTypeInt64
	default:
		// Default to string for unknown control types
		return TerraformTypeString
	}
}

// TerraformAttributeName converts the backend config name to a Terraform-friendly attribute name.
// - Replaces "." with "_"
// - Replaces "-" with "_" (hyphens not allowed in TF attribute names)
// - Removes ".user.defined" suffix
// - Converts to snake_case
func (e *ConfigEntry) TerraformAttributeName() string {
	name := e.Name

	// Remove common suffixes that indicate user-facing fields
	name = strings.TrimSuffix(name, ".user.defined")
	name = strings.TrimSuffix(name, ".user.displayed")

	// Replace dots and hyphens with underscores
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")

	// Convert camelCase to snake_case
	name = camelToSnake(name)

	return strings.ToLower(name)
}

// camelToSnake converts a camelCase or PascalCase string to snake_case.
// It also handles strings that already contain underscores.
func camelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		// If it's uppercase and not the first character
		if r >= 'A' && r <= 'Z' {
			// Check if we need to insert underscore
			if i > 0 {
				prev := rune(s[i-1])
				// Don't add underscore if previous char is underscore or uppercase
				// (to handle sequences like "SSLMode" -> "ssl_mode" not "s_s_l_mode")
				if prev != '_' && (prev < 'A' || prev > 'Z') {
					result.WriteRune('_')
				} else if prev >= 'A' && prev <= 'Z' && i+1 < len(s) {
					// Check next char - if lowercase, we need underscore before current
					// e.g., "SSLMode" -> the 'M' needs underscore before it
					next := rune(s[i+1])
					if next >= 'a' && next <= 'z' {
						result.WriteRune('_')
					}
				}
			}
			result.WriteRune(r)
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// GetRawValues returns the list of allowed values for one-select or multi-select controls.
// Converts any boolean or numeric values to their string representation.
func (e *ConfigEntry) GetRawValues() []string {
	result := make([]string, 0, len(e.Value.RawValues))
	for _, v := range e.Value.RawValues {
		switch val := v.(type) {
		case string:
			result = append(result, val)
		case bool:
			result = append(result, fmt.Sprintf("%t", val))
		case float64:
			if val == float64(int64(val)) {
				result = append(result, fmt.Sprintf("%d", int64(val)))
			} else {
				result = append(result, fmt.Sprintf("%v", val))
			}
		default:
			result = append(result, fmt.Sprintf("%v", v))
		}
	}
	return result
}

// GetSliderMin returns the minimum value for a slider control, or 0 if not set.
func (e *ConfigEntry) GetSliderMin() int64 {
	if e.Value.Min != nil {
		return int64(*e.Value.Min)
	}
	return 0
}

// GetSliderMax returns the maximum value for a slider control, or 0 if not set.
func (e *ConfigEntry) GetSliderMax() int64 {
	if e.Value.Max != nil {
		return int64(*e.Value.Max)
	}
	return 0
}

// GetSliderStep returns the step increment for a slider control, or 1 if not set.
func (e *ConfigEntry) GetSliderStep() int64 {
	if e.Value.Step != nil {
		return int64(*e.Value.Step)
	}
	return 1
}

// UserDefinedEntries returns only the config entries that should become Terraform attributes.
// These are entries where user_defined=true.
func (c *ConnectorConfig) UserDefinedEntries() []ConfigEntry {
	var entries []ConfigEntry
	for _, entry := range c.Config {
		if entry.IsUserDefined() {
			entries = append(entries, entry)
		}
	}
	return entries
}

// GetEntryByName returns the config entry with the given name, or nil if not found.
func (c *ConnectorConfig) GetEntryByName(name string) *ConfigEntry {
	for i := range c.Config {
		if c.Config[i].Name == name {
			return &c.Config[i]
		}
	}
	return nil
}

// ConditionOperator returns a human-readable description of the condition operator.
func (cond *Condition) ConditionOperator() string {
	switch cond.Operator {
	case "EQ":
		return "equals"
	case "NE":
		return "not equals"
	case "IN":
		return "in"
	default:
		return cond.Operator
	}
}

// GetConditionValueString returns the condition value as a string.
func (cond *Condition) GetConditionValueString() string {
	if cond.Value == nil {
		return ""
	}
	switch v := cond.Value.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetConditionValueBool returns the condition value as a bool, or false if not a bool.
func (cond *Condition) GetConditionValueBool() bool {
	if v, ok := cond.Value.(bool); ok {
		return v
	}
	return false
}
