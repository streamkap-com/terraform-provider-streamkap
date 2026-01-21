// Package main provides the tfgen CLI tool for generating Terraform provider
// schemas from backend configuration files.
//
// # Generator Module (generator.go)
//
// This file implements the code generator that transforms parsed ConfigEntry
// data into Go source files containing Terraform schema definitions.
//
// # Output Format
//
// The generator produces Go files in internal/generated/ with:
//   - Model struct with tfsdk tags for state management
//   - Schema function returning schema.Schema with attributes
//   - FieldMappings map for API field name resolution
//
// # Configuration Files
//
// Two JSON configuration files customize the generation:
//
// overrides.json: Defines special handling for complex types that cannot be
// auto-generated, such as map_string and map_nested fields (e.g., Snowflake
// properties, ClickHouse custom_types).
//
// deprecations.json: Defines deprecated attribute aliases that maintain backward
// compatibility with v2.1.18 while mapping to new attribute names.
//
// # Key Components
//
// Generator: Orchestrates the code generation process, loading overrides and
// deprecations, then iterating through connectors to generate schema files.
//
// TemplateData: Holds all data needed by the Go template, including package
// name, entity type, connector code, and processed field definitions.
//
// FieldData: Represents a single Terraform attribute with all schema properties
// (required, optional, computed, sensitive, default, validators).
//
// # Template Processing
//
// The schemaTemplate constant defines the Go template used for code generation.
// It produces properly formatted Go code with:
//   - "DO NOT EDIT" header comment
//   - Required imports
//   - Model struct with all fields
//   - Schema function with attribute definitions
//   - Field mappings for API translation
//
// # Sensitive Field Handling
//
// Fields marked with encrypt=true in backend config or using password/file
// control types generate Sensitive=true in the schema, preventing values
// from appearing in logs or CLI output.
//
// # Validator Generation
//
// The generator creates validators for:
//   - Enum fields (oneOfValidator from raw_values)
//   - Slider fields (rangeValidator from min/max)
//   - Port fields (Between validator for 1-65535)
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

// Generator generates Terraform schema code from ConnectorConfig.
type Generator struct {
	outputDir    string
	entityType   string // "source", "destination", "transform"
	overrides    *OverrideConfig
	deprecations *DeprecationConfig
}

// DeprecationConfig holds all deprecated field definitions.
type DeprecationConfig struct {
	DeprecatedFields []DeprecatedField `json:"deprecated_fields"`
}

// DeprecatedField represents a deprecated field alias.
type DeprecatedField struct {
	Connector      string `json:"connector"`
	EntityType     string `json:"entity_type"`
	DeprecatedAttr string `json:"deprecated_attr"` // The old/deprecated attribute name
	NewAttr        string `json:"new_attr"`        // The new attribute name it maps to
	Type           string `json:"type"`            // "string", "int64", "bool"
}

// OverrideConfig holds all field overrides for map types and other special cases.
type OverrideConfig struct {
	FieldOverrides []FieldOverride `json:"field_overrides"`
}

// FieldOverride represents an override configuration for a specific field.
type FieldOverride struct {
	Connector         string                 `json:"connector"`
	EntityType        string                 `json:"entity_type"`
	APIFieldName      string                 `json:"api_field_name"`
	TerraformAttrName string                 `json:"terraform_attr_name"`
	Type              string                 `json:"type"` // "map_string", "map_nested"
	Optional          bool                   `json:"optional"`
	Description       string                 `json:"description"`
	NestedModelName   string                 `json:"nested_model_name,omitempty"`
	NestedFields      []NestedFieldOverride  `json:"nested_fields,omitempty"`
}

// NestedFieldOverride represents a field within a nested map model.
type NestedFieldOverride struct {
	Name              string            `json:"name"`
	TerraformAttrName string            `json:"terraform_attr_name"`
	Type              string            `json:"type"` // "string", "int64", "bool"
	Optional          bool              `json:"optional"`
	Required          bool              `json:"required"`
	Validators        []ValidatorConfig `json:"validators,omitempty"`
}

// ValidatorConfig represents a validator configuration.
type ValidatorConfig struct {
	Type  string `json:"type"`  // "int64_at_least", etc.
	Value int64  `json:"value"` // validator value
}

// LoadOverrides loads the override configuration from a JSON file.
func LoadOverrides(path string) (*OverrideConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &OverrideConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read overrides file %s: %w", path, err)
	}

	var config OverrideConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse overrides file %s: %w", path, err)
	}

	return &config, nil
}

// LoadDeprecations loads the deprecation configuration from a JSON file.
func LoadDeprecations(path string) (*DeprecationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &DeprecationConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read deprecations file %s: %w", path, err)
	}

	var config DeprecationConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse deprecations file %s: %w", path, err)
	}

	return &config, nil
}

// NewGenerator creates a new Generator instance.
func NewGenerator(outputDir, entityType string) *Generator {
	return &Generator{
		outputDir:  outputDir,
		entityType: entityType,
	}
}

// NewGeneratorWithOverrides creates a new Generator instance with override configuration.
func NewGeneratorWithOverrides(outputDir, entityType string, overrides *OverrideConfig) *Generator {
	return &Generator{
		outputDir:  outputDir,
		entityType: entityType,
		overrides:  overrides,
	}
}

// NewGeneratorWithConfig creates a new Generator instance with both overrides and deprecations.
func NewGeneratorWithConfig(outputDir, entityType string, overrides *OverrideConfig, deprecations *DeprecationConfig) *Generator {
	return &Generator{
		outputDir:    outputDir,
		entityType:   entityType,
		overrides:    overrides,
		deprecations: deprecations,
	}
}

// getOverridesForConnector returns all field overrides for a specific connector.
func (g *Generator) getOverridesForConnector(connectorCode string) []FieldOverride {
	if g.overrides == nil {
		return nil
	}

	// Map entity type to match the JSON format (with 's' suffix)
	entityTypePlural := g.entityType + "s"

	var result []FieldOverride
	for _, override := range g.overrides.FieldOverrides {
		if override.Connector == connectorCode && override.EntityType == entityTypePlural {
			result = append(result, override)
		}
	}
	return result
}

// getDeprecationsForConnector returns all deprecated field definitions for a specific connector.
func (g *Generator) getDeprecationsForConnector(connectorCode string) []DeprecatedField {
	if g.deprecations == nil {
		return nil
	}

	// Map entity type to match the JSON format (with 's' suffix)
	entityTypePlural := g.entityType + "s"

	var result []DeprecatedField
	for _, dep := range g.deprecations.DeprecatedFields {
		if dep.Connector == connectorCode && dep.EntityType == entityTypePlural {
			result = append(result, dep)
		}
	}
	return result
}

// Generate creates Terraform schema files from a ConnectorConfig.
func (g *Generator) Generate(config *ConnectorConfig, connectorCode string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", g.outputDir, err)
	}

	// Prepare template data
	data := g.prepareTemplateData(config, connectorCode)

	// Generate the file
	outputPath := filepath.Join(g.outputDir, fmt.Sprintf("%s_%s.go", g.entityType, connectorCode))
	if err := g.generateFile(outputPath, data); err != nil {
		return fmt.Errorf("failed to generate %s: %w", outputPath, err)
	}

	return nil
}

// TemplateData holds all data needed for template rendering.
type TemplateData struct {
	PackageName       string
	EntityType        string // "source", "destination", "transform"
	EntityTypeCap     string // "Source", "Destination", "Transform"
	ConnectorCode     string // e.g., "postgresql"
	ConnectorCodeCap  string // e.g., "Postgresql"
	DisplayName       string // e.g., "PostgreSQL"
	ModelName         string // e.g., "SourcePostgresqlModel"
	SchemaFuncName    string // e.g., "SourcePostgresqlSchema"
	FieldMappingsName string // e.g., "SourcePostgresqlFieldMappings"
	Fields            []FieldData
	MapFields         []MapFieldData      // Map type fields from overrides
	NestedModels      []NestedModelData   // Nested model definitions for map fields
	DeprecatedFields  []DeprecatedFieldData // Deprecated field aliases (model-only, no schema)
	Imports           []string
}

// DeprecatedFieldData holds data for generating deprecated field aliases in the model.
// These fields are added to the Model struct only, not to the schema or field mappings.
// The schema and field mappings for deprecated fields are added by the wrapper files.
type DeprecatedFieldData struct {
	GoFieldName string // e.g., "InsertStaticKeyField1"
	GoType      string // e.g., "types.String"
	TfsdkTag    string // e.g., "insert_static_key_field_1"
}

// MapFieldData holds data for map type fields.
type MapFieldData struct {
	GoFieldName         string // e.g., "AutoQADedupeTableMapping"
	GoType              string // e.g., "map[string]types.String" or "map[string]ModelName"
	TfsdkTag            string // e.g., "auto_qa_dedupe_table_mapping"
	TfAttrName          string // e.g., "auto_qa_dedupe_table_mapping"
	Description         string
	MarkdownDescription string
	Optional            bool
	IsNested            bool   // true if MapNestedAttribute, false if MapAttribute
	NestedModelName     string // Name of the nested model type (for nested maps)
	NestedAttributes    []NestedAttributeData // Attributes for nested model
	APIFieldName        string // e.g., "auto.qa.dedupe.table.mapping"
}

// NestedAttributeData holds data for attributes within a nested map.
type NestedAttributeData struct {
	TfAttrName     string
	SchemaAttrType string // "schema.StringAttribute", "schema.Int64Attribute", etc.
	Optional       bool
	Required       bool
	HasValidators  bool
	Validators     string
}

// NestedModelData holds data for generating nested model type definitions.
type NestedModelData struct {
	ModelName string
	Fields    []NestedModelField
}

// NestedModelField holds data for a field within a nested model.
type NestedModelField struct {
	GoFieldName string
	GoType      string
	TfsdkTag    string
}

// FieldData holds data for a single field in the model and schema.
type FieldData struct {
	// Model fields
	GoFieldName string // e.g., "DatabaseHostname"
	GoType      string // e.g., "types.String"
	TfsdkTag    string // e.g., "database_hostname"

	// Schema fields
	TfAttrName     string // e.g., "database_hostname"
	SchemaAttrType string // e.g., "schema.StringAttribute"
	Required       bool
	Optional       bool
	Computed       bool
	Sensitive      bool
	Description         string // Plain text description for CLI
	MarkdownDescription string // Rich markdown description for docs/AI
	HasDefault          bool
	DefaultFunc    string // e.g., "stringdefault.StaticString(\"5432\")"
	HasValidators  bool
	Validators     string // e.g., "stringvalidator.OneOf(\"Yes\", \"No\")"
	NeedsPlanMod    bool // needs UseStateForUnknown plan modifier
	RequiresReplace bool // needs RequiresReplace plan modifier (set_once)
	IsListType      bool // needs ElementType for ListAttribute

	// Field mapping
	APIFieldName string // e.g., "database.hostname.user.defined"
}

// prepareTemplateData creates template data from the config.
func (g *Generator) prepareTemplateData(config *ConnectorConfig, connectorCode string) *TemplateData {
	entityTypeCap := capitalizeFirst(g.entityType)
	connectorCodeCap := toPascalCase(connectorCode)

	data := &TemplateData{
		PackageName:       "generated",
		EntityType:        g.entityType,
		EntityTypeCap:     entityTypeCap,
		ConnectorCode:     connectorCode,
		ConnectorCodeCap:  connectorCodeCap,
		DisplayName:       config.DisplayName,
		ModelName:         entityTypeCap + connectorCodeCap + "Model",
		SchemaFuncName:    entityTypeCap + connectorCodeCap + "Schema",
		FieldMappingsName: entityTypeCap + connectorCodeCap + "FieldMappings",
		Fields:            []FieldData{},
	}

	// Track which imports we need
	imports := map[string]bool{
		"github.com/hashicorp/terraform-plugin-framework/resource/schema":             true,
		"github.com/hashicorp/terraform-plugin-framework/types":                        true,
		"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts": true,
	}

	// Add common fields first (id, name, connector/transform_type)
	data.Fields = append(data.Fields, g.commonFields()...)

	// Get overrides for this connector to skip fields that have custom handling
	overrides := g.getOverridesForConnector(connectorCode)
	overrideAPIFields := make(map[string]bool)
	for _, override := range overrides {
		overrideAPIFields[override.APIFieldName] = true
	}

	// Add user-defined fields from config
	for _, entry := range config.UserDefinedEntries() {
		// Skip transforms.name for transforms - it's handled by the common Name field
		if g.entityType == "transform" && entry.Name == "transforms.name" {
			continue
		}

		// Skip fields that have custom overrides (they'll be added separately)
		if overrideAPIFields[entry.Name] {
			continue
		}

		field := g.entryToFieldData(&entry)
		data.Fields = append(data.Fields, field)

		// Determine the effective type (port fields force Int64)
		effectiveType := entry.TerraformType()
		if isPortField(field.TfAttrName) {
			effectiveType = TerraformTypeInt64
		}

		// Track required imports based on field properties
		if field.HasDefault {
			switch effectiveType {
			case TerraformTypeString:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"] = true
			case TerraformTypeInt64:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"] = true
			case TerraformTypeBool:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"] = true
			}
		}
		if field.NeedsPlanMod || field.RequiresReplace {
			imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"] = true
		}
		if field.NeedsPlanMod {
			switch effectiveType {
			case TerraformTypeString:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"] = true
			case TerraformTypeInt64:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"] = true
			case TerraformTypeBool:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"] = true
			}
		}
		if field.RequiresReplace {
			switch effectiveType {
			case TerraformTypeString:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"] = true
			case TerraformTypeInt64:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"] = true
			case TerraformTypeBool:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"] = true
			}
		}
		if field.HasValidators {
			switch effectiveType {
			case TerraformTypeString:
				imports["github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"] = true
				imports["github.com/hashicorp/terraform-plugin-framework/schema/validator"] = true
			case TerraformTypeInt64:
				imports["github.com/hashicorp/terraform-plugin-framework-validators/int64validator"] = true
				imports["github.com/hashicorp/terraform-plugin-framework/schema/validator"] = true
			}
		}
	}

	// Process map field overrides (overrides already loaded above)
	for _, override := range overrides {
		mapField := g.overrideToMapFieldData(&override)
		data.MapFields = append(data.MapFields, mapField)

		// Add nested model if this is a nested map
		if override.Type == "map_nested" && len(override.NestedFields) > 0 {
			nestedModel := g.overrideToNestedModelData(&override)
			data.NestedModels = append(data.NestedModels, nestedModel)

			// Add validator imports for nested fields
			for _, nf := range override.NestedFields {
				if len(nf.Validators) > 0 {
					switch nf.Type {
					case "int64":
						imports["github.com/hashicorp/terraform-plugin-framework-validators/int64validator"] = true
						imports["github.com/hashicorp/terraform-plugin-framework/schema/validator"] = true
					case "string":
						imports["github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"] = true
						imports["github.com/hashicorp/terraform-plugin-framework/schema/validator"] = true
					}
				}
			}
		}
	}

	// Add plan modifier imports for common fields (id, connector)
	imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"] = true
	imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"] = true

	// Process deprecated field aliases (add to model only, not schema or field mappings)
	deprecations := g.getDeprecationsForConnector(connectorCode)
	for _, dep := range deprecations {
		depField := g.deprecationToFieldData(&dep)
		data.DeprecatedFields = append(data.DeprecatedFields, depField)
	}

	// Convert imports map to sorted slice
	data.Imports = make([]string, 0, len(imports))
	for imp := range imports {
		data.Imports = append(data.Imports, imp)
	}
	sort.Strings(data.Imports)

	return data
}

// deprecationToFieldData converts a DeprecatedField to DeprecatedFieldData.
func (g *Generator) deprecationToFieldData(dep *DeprecatedField) DeprecatedFieldData {
	goFieldName := toPascalCase(dep.DeprecatedAttr)

	var goType string
	switch dep.Type {
	case "string":
		goType = "types.String"
	case "int64":
		goType = "types.Int64"
	case "bool":
		goType = "types.Bool"
	default:
		goType = "types.String" // default to string
	}

	return DeprecatedFieldData{
		GoFieldName: goFieldName,
		GoType:      goType,
		TfsdkTag:    dep.DeprecatedAttr,
	}
}

// overrideToMapFieldData converts a FieldOverride to MapFieldData.
func (g *Generator) overrideToMapFieldData(override *FieldOverride) MapFieldData {
	goFieldName := toPascalCase(override.TerraformAttrName)

	field := MapFieldData{
		GoFieldName:         goFieldName,
		TfsdkTag:            override.TerraformAttrName,
		TfAttrName:          override.TerraformAttrName,
		Description:         override.Description,
		MarkdownDescription: override.Description,
		Optional:            override.Optional,
		APIFieldName:        override.APIFieldName,
	}

	switch override.Type {
	case "map_string":
		field.GoType = "map[string]types.String"
		field.IsNested = false
	case "map_nested":
		// Use lowercase first letter for nested model type to match main branch convention
		modelName := lowercaseFirst(override.NestedModelName)
		field.GoType = fmt.Sprintf("map[string]%s", modelName)
		field.IsNested = true
		field.NestedModelName = modelName

		// Convert nested fields to NestedAttributeData
		for _, nf := range override.NestedFields {
			attr := NestedAttributeData{
				TfAttrName: nf.TerraformAttrName,
				Optional:   nf.Optional,
				Required:   nf.Required,
			}

			// Determine schema attribute type
			switch nf.Type {
			case "string":
				attr.SchemaAttrType = "schema.StringAttribute"
			case "int64":
				attr.SchemaAttrType = "schema.Int64Attribute"
			case "bool":
				attr.SchemaAttrType = "schema.BoolAttribute"
			}

			// Add validators if present
			if len(nf.Validators) > 0 {
				attr.HasValidators = true
				var validators []string
				for _, v := range nf.Validators {
					switch v.Type {
					case "int64_at_least":
						validators = append(validators, fmt.Sprintf("int64validator.AtLeast(%d)", v.Value))
					}
				}
				attr.Validators = strings.Join(validators, ", ")
			}

			field.NestedAttributes = append(field.NestedAttributes, attr)
		}
	}

	return field
}

// overrideToNestedModelData converts a FieldOverride to NestedModelData.
func (g *Generator) overrideToNestedModelData(override *FieldOverride) NestedModelData {
	// Use lowercase first letter for model name to match main branch convention
	modelName := lowercaseFirst(override.NestedModelName)

	model := NestedModelData{
		ModelName: modelName,
	}

	for _, nf := range override.NestedFields {
		goFieldName := toPascalCase(nf.TerraformAttrName)
		var goType string
		switch nf.Type {
		case "string":
			goType = "types.String"
		case "int64":
			goType = "types.Int64"
		case "bool":
			goType = "types.Bool"
		}

		model.Fields = append(model.Fields, NestedModelField{
			GoFieldName: goFieldName,
			GoType:      goType,
			TfsdkTag:    nf.TerraformAttrName,
		})
	}

	return model
}

// lowercaseFirst returns the string with the first letter lowercased.
func lowercaseFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// commonFields returns the common fields present in all resources.
func (g *Generator) commonFields() []FieldData {
	fields := []FieldData{
		{
			GoFieldName:         "ID",
			GoType:              "types.String",
			TfsdkTag:            "id",
			TfAttrName:          "id",
			SchemaAttrType:      "schema.StringAttribute",
			Computed:            true,
			Description:         fmt.Sprintf("Unique identifier for the %s", g.entityType),
			MarkdownDescription: fmt.Sprintf("Unique identifier for the %s", g.entityType),
			NeedsPlanMod:        true,
			APIFieldName:        "", // ID is handled separately
		},
		{
			GoFieldName:         "Name",
			GoType:              "types.String",
			TfsdkTag:            "name",
			TfAttrName:          "name",
			SchemaAttrType:      "schema.StringAttribute",
			Required:            true,
			Description:         fmt.Sprintf("Name of the %s", g.entityType),
			MarkdownDescription: fmt.Sprintf("Name of the %s", g.entityType),
			APIFieldName:        "", // Name is handled separately
		},
	}

	// For transforms, use "transform_type" instead of "connector"
	if g.entityType == "transform" {
		fields = append(fields, FieldData{
			GoFieldName:         "TransformType",
			GoType:              "types.String",
			TfsdkTag:            "transform_type",
			TfAttrName:          "transform_type",
			SchemaAttrType:      "schema.StringAttribute",
			Computed:            true,
			Description:         "Transform type",
			MarkdownDescription: "Transform type",
			NeedsPlanMod:        true,
			APIFieldName:        "", // TransformType is handled separately
		})
	} else {
		fields = append(fields, FieldData{
			GoFieldName:         "Connector",
			GoType:              "types.String",
			TfsdkTag:            "connector",
			TfAttrName:          "connector",
			SchemaAttrType:      "schema.StringAttribute",
			Computed:            true,
			Description:         "Connector type",
			MarkdownDescription: "Connector type",
			NeedsPlanMod:        true,
			APIFieldName:        "", // Connector is handled separately
		})
	}

	return fields
}

// isPortField returns true if the field name indicates it's a port field.
// Port fields should use Int64 type for better UX.
func isPortField(tfAttrName string) bool {
	return tfAttrName == "port" ||
		strings.HasSuffix(tfAttrName, "_port")
}

// entryToFieldData converts a ConfigEntry to FieldData.
func (g *Generator) entryToFieldData(entry *ConfigEntry) FieldData {
	tfAttrName := entry.TerraformAttributeName()
	goFieldName := toPascalCase(tfAttrName)

	// Check if this is a port field that should be Int64
	forceInt64 := isPortField(tfAttrName)

	field := FieldData{
		GoFieldName:    goFieldName,
		GoType:         string(entry.TerraformType()),
		TfsdkTag:       tfAttrName,
		TfAttrName:     tfAttrName,
		Description:    entry.Description,
		Sensitive:      entry.IsSensitive(),
		APIFieldName:   entry.Name,
		RequiresReplace: entry.IsSetOnce(),
	}

	// Override type for port fields
	if forceInt64 {
		field.GoType = string(TerraformTypeInt64)
	}

	// Use display name as description if description is empty
	if field.Description == "" {
		field.Description = entry.DisplayName
	}

	// Initialize MarkdownDescription from Description
	field.MarkdownDescription = field.Description

	// Determine schema attribute type (respecting port field override)
	if forceInt64 {
		field.SchemaAttrType = "schema.Int64Attribute"
	} else {
		switch entry.TerraformType() {
		case TerraformTypeString:
			field.SchemaAttrType = "schema.StringAttribute"
		case TerraformTypeInt64:
			field.SchemaAttrType = "schema.Int64Attribute"
		case TerraformTypeBool:
			field.SchemaAttrType = "schema.BoolAttribute"
		case TerraformTypeList:
			field.SchemaAttrType = "schema.ListAttribute"
			field.GoType = "types.List"
			field.IsListType = true // Flag for template to add ElementType
		}
	}

	// Determine Required/Optional/Computed
	if entry.IsRequired() && !entry.HasDefault() {
		field.Required = true
	} else if entry.HasDefault() {
		field.Optional = true
		field.Computed = true
		field.DefaultFunc = g.defaultFunc(entry, forceInt64)
		// Only set HasDefault if we actually have a default function
		// (list types don't have simple defaults like strings/ints/bools)
		if field.DefaultFunc != "" {
			field.HasDefault = true
			field.NeedsPlanMod = false // Fields with defaults don't need UseStateForUnknown

			// Capture default value and enhance descriptions
			if forceInt64 {
				// Port fields: parse string default as int64
				defaultVal := entry.GetDefaultInt64FromString()
				field.Description = field.Description + fmt.Sprintf(" Defaults to %d.", defaultVal)
				field.MarkdownDescription = field.MarkdownDescription + fmt.Sprintf(" Defaults to `%d`.", defaultVal)
			} else {
				switch entry.TerraformType() {
				case TerraformTypeString:
					defaultVal := entry.GetDefaultString()
					field.Description = field.Description + fmt.Sprintf(" Defaults to %q.", defaultVal)
					field.MarkdownDescription = field.MarkdownDescription + fmt.Sprintf(" Defaults to `%s`.", defaultVal)
				case TerraformTypeInt64:
					defaultVal := entry.GetDefaultInt64()
					field.Description = field.Description + fmt.Sprintf(" Defaults to %d.", defaultVal)
					field.MarkdownDescription = field.MarkdownDescription + fmt.Sprintf(" Defaults to `%d`.", defaultVal)
				case TerraformTypeBool:
					defaultVal := entry.GetDefaultBool()
					field.Description = field.Description + fmt.Sprintf(" Defaults to %t.", defaultVal)
					field.MarkdownDescription = field.MarkdownDescription + fmt.Sprintf(" Defaults to `%t`.", defaultVal)
				}
			}
		}
	} else {
		field.Optional = true
	}

	// Handle validators for one-select fields
	if entry.Value.Control == "one-select" && len(entry.GetRawValues()) > 0 {
		field.HasValidators = true
		field.Validators = g.oneOfValidator(entry)

		// Enhance descriptions with valid values
		values := entry.GetRawValues()
		valuesStr := strings.Join(values, ", ")
		field.Description = field.Description + " Valid values: " + valuesStr + "."

		// Build markdown list for MarkdownDescription
		var mdValues []string
		for _, v := range values {
			mdValues = append(mdValues, fmt.Sprintf("`%s`", v))
		}
		field.MarkdownDescription = field.MarkdownDescription + " Valid values: " + strings.Join(mdValues, ", ") + "."
	}

	// Handle slider validators (int64 range)
	if entry.Value.Control == "slider" && entry.Value.Min != nil && entry.Value.Max != nil {
		field.HasValidators = true
		field.Validators = g.rangeValidator(entry)
	}

	// Add security note for sensitive fields
	if field.Sensitive {
		field.Description = field.Description + " This value is sensitive and will not appear in logs or CLI output."
		field.MarkdownDescription = field.MarkdownDescription + "\n\n**Security:** This value is marked sensitive and will not appear in CLI output or logs."
	}

	return field
}

// defaultFunc generates the default function call for a config entry.
// If forceInt64 is true, treats the default as int64 even if backend stores it as string.
func (g *Generator) defaultFunc(entry *ConfigEntry, forceInt64 bool) string {
	if forceInt64 {
		return fmt.Sprintf("int64default.StaticInt64(%d)", entry.GetDefaultInt64FromString())
	}
	switch entry.TerraformType() {
	case TerraformTypeString:
		return fmt.Sprintf("stringdefault.StaticString(%q)", entry.GetDefaultString())
	case TerraformTypeInt64:
		return fmt.Sprintf("int64default.StaticInt64(%d)", entry.GetDefaultInt64())
	case TerraformTypeBool:
		return fmt.Sprintf("booldefault.StaticBool(%t)", entry.GetDefaultBool())
	default:
		return ""
	}
}

// oneOfValidator generates a OneOf validator for select fields.
func (g *Generator) oneOfValidator(entry *ConfigEntry) string {
	values := entry.GetRawValues()
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = fmt.Sprintf("%q", v)
	}
	return fmt.Sprintf("stringvalidator.OneOf(%s)", strings.Join(quoted, ", "))
}

// rangeValidator generates a Between validator for slider fields.
func (g *Generator) rangeValidator(entry *ConfigEntry) string {
	min := entry.GetSliderMin()
	max := entry.GetSliderMax()
	return fmt.Sprintf("int64validator.Between(%d, %d)", min, max)
}

// generateFile renders the template and writes to the output file.
func (g *Generator) generateFile(outputPath string, data *TemplateData) error {
	tmpl, err := template.New("schema").Funcs(template.FuncMap{
		"isCommonField": func(name string) bool {
			return name == "id" || name == "name" || name == "connector" || name == "transform_type"
		},
	}).Parse(schemaTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Format the generated code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted for debugging
		if writeErr := os.WriteFile(outputPath+".unformatted", buf.Bytes(), 0644); writeErr != nil {
			return fmt.Errorf("failed to format generated code: %w (and failed to write unformatted: %v)", err, writeErr)
		}
		return fmt.Errorf("failed to format generated code (see %s.unformatted): %w", outputPath, err)
	}

	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	return nil
}

// capitalizeFirst capitalizes the first letter of a string.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// toPascalCase converts a snake_case string to PascalCase.
// Also handles hyphens by replacing them with underscores first.
func toPascalCase(s string) string {
	// Replace hyphens with underscores for consistent splitting
	s = strings.ReplaceAll(s, "-", "_")
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		// Handle common abbreviations
		upper := strings.ToUpper(part)
		if upper == "ID" || upper == "SSH" || upper == "SSL" || upper == "DB" || upper == "URL" || upper == "API" || upper == "AWS" || upper == "ARN" || upper == "SQL" || upper == "QA" {
			parts[i] = upper
		} else {
			parts[i] = capitalizeFirst(part)
		}
	}
	return strings.Join(parts, "")
}

// schemaTemplate is the Go template for generating schema files.
const schemaTemplate = `// Code generated by tfgen. DO NOT EDIT.

package {{ .PackageName }}

import (
{{- range .Imports }}
	"{{ . }}"
{{- end }}
)

{{- /* Generate nested model types for map fields */ -}}
{{- range .NestedModels }}

type {{ .ModelName }} struct {
{{- range .Fields }}
	{{ .GoFieldName }} {{ .GoType }} ` + "`" + `tfsdk:"{{ .TfsdkTag }}"` + "`" + `
{{- end }}
}
{{- end }}

// {{ .ModelName }} is the Terraform model for the {{ .ConnectorCode }} {{ .EntityType }}.
type {{ .ModelName }} struct {
{{- range .Fields }}
	{{ .GoFieldName }} {{ .GoType }} ` + "`" + `tfsdk:"{{ .TfsdkTag }}"` + "`" + `
{{- end }}
{{- range .MapFields }}
	{{ .GoFieldName }} {{ .GoType }} ` + "`" + `tfsdk:"{{ .TfsdkTag }}"` + "`" + `
{{- end }}
{{- if .DeprecatedFields }}
	// Deprecated fields - kept for backward compatibility
{{- range .DeprecatedFields }}
	{{ .GoFieldName }} {{ .GoType }} ` + "`" + `tfsdk:"{{ .TfsdkTag }}"` + "`" + `
{{- end }}
{{- end }}
	Timeouts timeouts.Value ` + "`" + `tfsdk:"timeouts"` + "`" + `
}

// {{ .SchemaFuncName }} returns the Terraform schema for the {{ .ConnectorCode }} {{ .EntityType }}.
func {{ .SchemaFuncName }}() schema.Schema {
	return schema.Schema{
		Description:         "Manages a {{ .DisplayName }} {{ .EntityType }} connector.",
		MarkdownDescription: "Manages a **{{ .DisplayName }} {{ .EntityType }} connector**.\n\n" +
			"This resource creates and manages a {{ .DisplayName }} {{ .EntityType }} for Streamkap data pipelines.\n\n" +
			"[Documentation](https://docs.streamkap.com/streamkap-provider-for-terraform)",
		Attributes: map[string]schema.Attribute{
{{- range .Fields }}
			"{{ .TfAttrName }}": {{ .SchemaAttrType }}{
{{- if .Required }}
				Required:            true,
{{- end }}
{{- if .Optional }}
				Optional:            true,
{{- end }}
{{- if .Computed }}
				Computed:            true,
{{- end }}
{{- if .IsListType }}
				ElementType:         types.StringType,
{{- end }}
{{- if .Sensitive }}
				Sensitive:           true,
{{- end }}
{{- if .Description }}
				Description:         {{ printf "%q" .Description }},
				MarkdownDescription: {{ printf "%q" .MarkdownDescription }},
{{- end }}
{{- if .HasDefault }}
				Default:             {{ .DefaultFunc }},
{{- end }}
{{- if .HasValidators }}
				Validators: []validator.{{ if eq .SchemaAttrType "schema.StringAttribute" }}String{{ else if eq .SchemaAttrType "schema.Int64Attribute" }}Int64{{ else if eq .SchemaAttrType "schema.BoolAttribute" }}Bool{{ end }}{
					{{ .Validators }},
				},
{{- end }}
{{- if or .NeedsPlanMod .RequiresReplace }}
				PlanModifiers: []planmodifier.{{ if eq .SchemaAttrType "schema.StringAttribute" }}String{{ else if eq .SchemaAttrType "schema.Int64Attribute" }}Int64{{ else if eq .SchemaAttrType "schema.BoolAttribute" }}boolplanmodifier{{ end }}{
{{- if .NeedsPlanMod }}
					{{ if eq .SchemaAttrType "schema.StringAttribute" }}stringplanmodifier{{ else if eq .SchemaAttrType "schema.Int64Attribute" }}int64planmodifier{{ else if eq .SchemaAttrType "schema.BoolAttribute" }}boolplanmodifier{{ end }}.UseStateForUnknown(),
{{- end }}
{{- if .RequiresReplace }}
					{{ if eq .SchemaAttrType "schema.StringAttribute" }}stringplanmodifier{{ else if eq .SchemaAttrType "schema.Int64Attribute" }}int64planmodifier{{ else if eq .SchemaAttrType "schema.BoolAttribute" }}boolplanmodifier{{ end }}.RequiresReplace(),
{{- end }}
				},
{{- end }}
			},
{{- end }}
{{- /* Generate map field schema attributes */ -}}
{{- range .MapFields }}
{{- if .IsNested }}
			"{{ .TfAttrName }}": schema.MapNestedAttribute{
				Optional: {{ .Optional }},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
{{- range .NestedAttributes }}
						"{{ .TfAttrName }}": {{ .SchemaAttrType }}{
{{- if .Optional }}
							Optional: true,
{{- end }}
{{- if .Required }}
							Required: true,
{{- end }}
{{- if .HasValidators }}
							Validators: []validator.{{ if eq .SchemaAttrType "schema.StringAttribute" }}String{{ else if eq .SchemaAttrType "schema.Int64Attribute" }}Int64{{ else if eq .SchemaAttrType "schema.BoolAttribute" }}Bool{{ end }}{
								{{ .Validators }},
							},
{{- end }}
						},
{{- end }}
					},
				},
				Description:         {{ printf "%q" .Description }},
				MarkdownDescription: {{ printf "%q" .MarkdownDescription }},
			},
{{- else }}
			"{{ .TfAttrName }}": schema.MapAttribute{
				Optional:            {{ .Optional }},
				ElementType:         types.StringType,
				Description:         {{ printf "%q" .Description }},
				MarkdownDescription: {{ printf "%q" .MarkdownDescription }},
			},
{{- end }}
{{- end }}
		},
	}
}

// {{ .FieldMappingsName }} maps Terraform attribute names to API field names.
var {{ .FieldMappingsName }} = map[string]string{
{{- range .Fields }}
{{- if .APIFieldName }}
	"{{ .TfAttrName }}": "{{ .APIFieldName }}",
{{- end }}
{{- end }}
{{- range .MapFields }}
{{- if .APIFieldName }}
	"{{ .TfAttrName }}": "{{ .APIFieldName }}",
{{- end }}
{{- end }}
}
`
