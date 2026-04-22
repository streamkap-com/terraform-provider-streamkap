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

// sourcePostgresqlModelWithDeprecated embeds the generated model and adds
// deprecated field aliases so reflection-based marshaling can find them.
type sourcePostgresqlModelWithDeprecated struct {
	generated.SourcePostgresqlModel
	InsertStaticKeyField1               types.String `tfsdk:"insert_static_key_field_1"`
	InsertStaticKeyValue1               types.String `tfsdk:"insert_static_key_value_1"`
	InsertStaticValueField1             types.String `tfsdk:"insert_static_value_field_1"`
	InsertStaticValue1                  types.String `tfsdk:"insert_static_value_1"`
	InsertStaticKeyField2               types.String `tfsdk:"insert_static_key_field_2"`
	InsertStaticKeyValue2               types.String `tfsdk:"insert_static_key_value_2"`
	InsertStaticValueField2             types.String `tfsdk:"insert_static_value_field_2"`
	InsertStaticValue2                  types.String `tfsdk:"insert_static_value_2"`
	PredicatesIsTopicToEnrichPatternOld types.String `tfsdk:"predicates_istopictoenrich_pattern"`
}

// postgresqlFieldMappings extends the generated field mappings with deprecated aliases.
var postgresqlFieldMappings = func() map[string]string {
	mappings := make(map[string]string)
	for k, v := range generated.SourcePostgresqlFieldMappings {
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
	return mappings
}()

// PostgreSQLConfig implements the ConnectorConfig interface for PostgreSQL sources.
type PostgreSQLConfig struct{}

// Ensure PostgreSQLConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*PostgreSQLConfig)(nil)

// GetSchema returns the Terraform schema for PostgreSQL source.
func (c *PostgreSQLConfig) GetSchema() schema.Schema {
	s := generated.SourcePostgresqlSchema()
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
func (c *PostgreSQLConfig) GetFieldMappings() map[string]string {
	return postgresqlFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *PostgreSQLConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for PostgreSQL.
func (c *PostgreSQLConfig) GetConnectorCode() string {
	return "postgresql"
}

// GetResourceName returns the Terraform resource name.
func (c *PostgreSQLConfig) GetResourceName() string {
	return "source_postgresql"
}

// NewModelInstance returns a new instance of the PostgreSQL model with deprecated fields.
func (c *PostgreSQLConfig) NewModelInstance() any {
	return &sourcePostgresqlModelWithDeprecated{}
}

// NewPostgreSQLResource creates a new PostgreSQL source resource.
func NewPostgreSQLResource() resource.Resource {
	return connector.NewBaseConnectorResource(&PostgreSQLConfig{})
}
