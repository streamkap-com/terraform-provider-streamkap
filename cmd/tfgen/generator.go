// Package main provides the tfgen CLI tool for generating Terraform provider
// schemas from backend configuration files.
package main

import (
	"bytes"
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
	outputDir  string
	entityType string // "source", "destination", "transform"
}

// NewGenerator creates a new Generator instance.
func NewGenerator(outputDir, entityType string) *Generator {
	return &Generator{
		outputDir:  outputDir,
		entityType: entityType,
	}
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
	Imports           []string
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
		"github.com/hashicorp/terraform-plugin-framework/resource/schema": true,
		"github.com/hashicorp/terraform-plugin-framework/types":           true,
	}

	// Add common fields first (id, name, connector/transform_type)
	data.Fields = append(data.Fields, g.commonFields()...)

	// Add user-defined fields from config
	for _, entry := range config.UserDefinedEntries() {
		// Skip transforms.name for transforms - it's handled by the common Name field
		if g.entityType == "transform" && entry.Name == "transforms.name" {
			continue
		}

		field := g.entryToFieldData(&entry)
		data.Fields = append(data.Fields, field)

		// Track required imports based on field properties
		if field.HasDefault {
			switch entry.TerraformType() {
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
			switch entry.TerraformType() {
			case TerraformTypeString:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"] = true
			case TerraformTypeInt64:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"] = true
			case TerraformTypeBool:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"] = true
			}
		}
		if field.RequiresReplace {
			switch entry.TerraformType() {
			case TerraformTypeString:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"] = true
			case TerraformTypeInt64:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"] = true
			case TerraformTypeBool:
				imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"] = true
			}
		}
		if field.HasValidators {
			switch entry.TerraformType() {
			case TerraformTypeString:
				imports["github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"] = true
				imports["github.com/hashicorp/terraform-plugin-framework/schema/validator"] = true
			case TerraformTypeInt64:
				imports["github.com/hashicorp/terraform-plugin-framework-validators/int64validator"] = true
				imports["github.com/hashicorp/terraform-plugin-framework/schema/validator"] = true
			}
		}
	}

	// Add plan modifier imports for common fields (id, connector)
	imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"] = true
	imports["github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"] = true

	// Convert imports map to sorted slice
	data.Imports = make([]string, 0, len(imports))
	for imp := range imports {
		data.Imports = append(data.Imports, imp)
	}
	sort.Strings(data.Imports)

	return data
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

// entryToFieldData converts a ConfigEntry to FieldData.
func (g *Generator) entryToFieldData(entry *ConfigEntry) FieldData {
	tfAttrName := entry.TerraformAttributeName()
	goFieldName := toPascalCase(tfAttrName)

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

	// Use display name as description if description is empty
	if field.Description == "" {
		field.Description = entry.DisplayName
	}

	// Initialize MarkdownDescription from Description
	field.MarkdownDescription = field.Description

	// Determine schema attribute type
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

	// Determine Required/Optional/Computed
	if entry.IsRequired() && !entry.HasDefault() {
		field.Required = true
	} else if entry.HasDefault() {
		field.Optional = true
		field.Computed = true
		field.DefaultFunc = g.defaultFunc(entry)
		// Only set HasDefault if we actually have a default function
		// (list types don't have simple defaults like strings/ints/bools)
		if field.DefaultFunc != "" {
			field.HasDefault = true
			field.NeedsPlanMod = false // Fields with defaults don't need UseStateForUnknown

			// Capture default value and enhance descriptions
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
func (g *Generator) defaultFunc(entry *ConfigEntry) string {
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
		if upper == "ID" || upper == "SSH" || upper == "SSL" || upper == "DB" || upper == "URL" || upper == "API" || upper == "AWS" || upper == "ARN" {
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

// {{ .ModelName }} is the Terraform model for the {{ .ConnectorCode }} {{ .EntityType }}.
type {{ .ModelName }} struct {
{{- range .Fields }}
	{{ .GoFieldName }} {{ .GoType }} ` + "`" + `tfsdk:"{{ .TfsdkTag }}"` + "`" + `
{{- end }}
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
				PlanModifiers: []planmodifier.{{ if eq .SchemaAttrType "schema.StringAttribute" }}String{{ else if eq .SchemaAttrType "schema.Int64Attribute" }}Int64{{ else if eq .SchemaAttrType "schema.BoolAttribute" }}Bool{{ end }}{
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
}
`
