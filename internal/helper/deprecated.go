package helper

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DeprecatedAlias represents a mapping from old attribute name to new attribute name
type DeprecatedAlias struct {
	OldName string
	NewName string
}

// PostgreSQLDeprecatedAliases contains all deprecated attribute mappings for PostgreSQL source
var PostgreSQLDeprecatedAliases = []DeprecatedAlias{
	{OldName: "insert_static_key_field_1", NewName: "transforms_insert_static_key1_static_field"},
	{OldName: "insert_static_key_value_1", NewName: "transforms_insert_static_key1_static_value"},
	{OldName: "insert_static_value_field_1", NewName: "transforms_insert_static_value1_static_field"},
	{OldName: "insert_static_value_1", NewName: "transforms_insert_static_value1_static_value"},
	{OldName: "insert_static_key_field_2", NewName: "transforms_insert_static_key2_static_field"},
	{OldName: "insert_static_key_value_2", NewName: "transforms_insert_static_key2_static_value"},
	{OldName: "insert_static_value_field_2", NewName: "transforms_insert_static_value2_static_field"},
	{OldName: "insert_static_value_2", NewName: "transforms_insert_static_value2_static_value"},
	{OldName: "predicates_istopictoenrich_pattern", NewName: "predicates_is_topic_to_enrich_pattern"},
}

// SnowflakeDeprecatedAliases contains all deprecated attribute mappings for Snowflake destination
var SnowflakeDeprecatedAliases = []DeprecatedAlias{
	{OldName: "auto_schema_creation", NewName: "create_schema_auto"},
}

// CreateDeprecatedStringAttribute creates a deprecated string attribute that warns users
func CreateDeprecatedStringAttribute(newName string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:            true,
		Computed:            true,
		DeprecationMessage:  "This attribute is deprecated. Use '" + newName + "' instead.",
		Description:         "DEPRECATED: Use '" + newName + "' instead.",
		MarkdownDescription: "**DEPRECATED:** Use `" + newName + "` instead.",
		Default:             stringdefault.StaticString(""),
	}
}

// MigrateDeprecatedValues copies values from deprecated attributes to new attributes if set
// Call this in Create/Update before processing the model
func MigrateDeprecatedValues(ctx context.Context, config map[string]any, aliases []DeprecatedAlias) map[string]any {
	for _, alias := range aliases {
		if oldVal, exists := config[alias.OldName]; exists && oldVal != nil && oldVal != "" {
			tflog.Warn(ctx, "Deprecated attribute used",
				map[string]any{
					"deprecated": alias.OldName,
					"use":        alias.NewName,
				})
			// Only copy if new value is not set
			if newVal, newExists := config[alias.NewName]; !newExists || newVal == nil || newVal == "" {
				config[alias.NewName] = oldVal
			}
			// Remove old key to avoid API confusion
			delete(config, alias.OldName)
		}
	}
	return config
}
