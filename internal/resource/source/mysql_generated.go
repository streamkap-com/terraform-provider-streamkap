// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// sourceMysqlModelWithDeprecated embeds the generated model and adds
// deprecated field aliases so reflection-based marshaling can find them.
type sourceMysqlModelWithDeprecated struct {
	generated.SourceMysqlModel
	InsertStaticKeyField1               types.String `tfsdk:"insert_static_key_field_1"`
	InsertStaticKeyValue1               types.String `tfsdk:"insert_static_key_value_1"`
	InsertStaticValueField1             types.String `tfsdk:"insert_static_value_field_1"`
	InsertStaticValue1                  types.String `tfsdk:"insert_static_value_1"`
	InsertStaticKeyField2               types.String `tfsdk:"insert_static_key_field_2"`
	InsertStaticKeyValue2               types.String `tfsdk:"insert_static_key_value_2"`
	InsertStaticValueField2             types.String `tfsdk:"insert_static_value_field_2"`
	InsertStaticValue2                  types.String `tfsdk:"insert_static_value_2"`
	PredicatesIsTopicToEnrichPatternOld types.String `tfsdk:"predicates_istopictoenrich_pattern"`
	DatabaseConnectionTimezoneOld       types.String `tfsdk:"database_connection_timezone"`
}

// mysqlFieldMappings extends the generated field mappings with deprecated aliases.
var mysqlFieldMappings = func() map[string]string {
	mappings := make(map[string]string)
	for k, v := range generated.SourceMysqlFieldMappings {
		mappings[k] = v
	}
	// Deprecated aliases - map to same API fields
	mappings["insert_static_key_field_1"] = "transforms.InsertStaticKey1.static.field"
	mappings["insert_static_key_value_1"] = "transforms.InsertStaticKey1.static.value"
	mappings["insert_static_value_field_1"] = "transforms.InsertStaticValue1.static.field"
	mappings["insert_static_value_1"] = "transforms.InsertStaticValue1.static.value"
	mappings["insert_static_key_field_2"] = "transforms.InsertStaticKey2.static.field"
	mappings["insert_static_key_value_2"] = "transforms.InsertStaticKey2.static.value"
	mappings["insert_static_value_field_2"] = "transforms.InsertStaticValue2.static.field"
	mappings["insert_static_value_2"] = "transforms.InsertStaticValue2.static.value"
	mappings["predicates_istopictoenrich_pattern"] = "predicates.IsTopicToEnrich.pattern"
	mappings["database_connection_timezone"] = "database.connectionTimeZone"
	return mappings
}()

// MySQLConfig implements the ConnectorConfig interface for MySQL sources.
type MySQLConfig struct{}

// Ensure MySQLConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*MySQLConfig)(nil)

// GetSchema returns the Terraform schema for MySQL source.
func (c *MySQLConfig) GetSchema() schema.Schema {
	s := generated.SourceMysqlSchema()
	// Deprecated aliases — map to the same API fields as the new names.
	// Computed so the provider can populate them from the API on Read.
	// ConflictsWith prevents setting both old and new names simultaneously.
	deprecatedAliases := []struct {
		oldName, newName string
	}{
		{"insert_static_key_field_1", "transforms_insert_static_key1_static_field"},
		{"insert_static_key_value_1", "transforms_insert_static_key1_static_value"},
		{"insert_static_value_field_1", "transforms_insert_static_value1_static_field"},
		{"insert_static_value_1", "transforms_insert_static_value1_static_value"},
		{"insert_static_key_field_2", "transforms_insert_static_key2_static_field"},
		{"insert_static_key_value_2", "transforms_insert_static_key2_static_value"},
		{"insert_static_value_field_2", "transforms_insert_static_value2_static_field"},
		{"insert_static_value_2", "transforms_insert_static_value2_static_value"},
		{"predicates_istopictoenrich_pattern", "predicates_is_topic_to_enrich_pattern"},
		{"database_connection_timezone", "database_connection_time_zone"},
	}
	for _, alias := range deprecatedAliases {
		s.Attributes[alias.oldName] = schema.StringAttribute{
			Optional:           true,
			Computed:           true,
			DeprecationMessage: "Use '" + alias.newName + "' instead.",
			Description:        "DEPRECATED: Use '" + alias.newName + "' instead.",
			Validators: []validator.String{
				stringvalidator.ConflictsWith(path.MatchRoot(alias.newName)),
			},
		}
	}
	return s
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *MySQLConfig) GetFieldMappings() map[string]string {
	return mysqlFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *MySQLConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for MySQL.
func (c *MySQLConfig) GetConnectorCode() string {
	return "mysql"
}

// GetResourceName returns the Terraform resource name.
func (c *MySQLConfig) GetResourceName() string {
	return "source_mysql"
}

// NewModelInstance returns a new instance of the MySQL model with deprecated fields.
func (c *MySQLConfig) NewModelInstance() any {
	return &sourceMysqlModelWithDeprecated{}
}

// NewMySQLResource creates a new MySQL source resource.
func NewMySQLResource() resource.Resource {
	return connector.NewBaseConnectorResource(&MySQLConfig{})
}
