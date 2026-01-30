// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

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
	// Add deprecated aliases - these map to the same API fields as the new names
	s.Attributes["insert_static_key_field_1"] = schema.StringAttribute{
		Optional:           true,
		Computed:           true,
		DeprecationMessage: "Use 'transforms_insert_static_key1_static_field' instead.",
		Description:        "DEPRECATED: Use 'transforms_insert_static_key1_static_field' instead.",
	}
	s.Attributes["insert_static_key_value_1"] = schema.StringAttribute{
		Optional:           true,
		Computed:           true,
		DeprecationMessage: "Use 'transforms_insert_static_key1_static_value' instead.",
		Description:        "DEPRECATED: Use 'transforms_insert_static_key1_static_value' instead.",
	}
	s.Attributes["insert_static_value_field_1"] = schema.StringAttribute{
		Optional:           true,
		Computed:           true,
		DeprecationMessage: "Use 'transforms_insert_static_value1_static_field' instead.",
		Description:        "DEPRECATED: Use 'transforms_insert_static_value1_static_field' instead.",
	}
	s.Attributes["insert_static_value_1"] = schema.StringAttribute{
		Optional:           true,
		Computed:           true,
		DeprecationMessage: "Use 'transforms_insert_static_value1_static_value' instead.",
		Description:        "DEPRECATED: Use 'transforms_insert_static_value1_static_value' instead.",
	}
	s.Attributes["insert_static_key_field_2"] = schema.StringAttribute{
		Optional:           true,
		Computed:           true,
		DeprecationMessage: "Use 'transforms_insert_static_key2_static_field' instead.",
		Description:        "DEPRECATED: Use 'transforms_insert_static_key2_static_field' instead.",
	}
	s.Attributes["insert_static_key_value_2"] = schema.StringAttribute{
		Optional:           true,
		Computed:           true,
		DeprecationMessage: "Use 'transforms_insert_static_key2_static_value' instead.",
		Description:        "DEPRECATED: Use 'transforms_insert_static_key2_static_value' instead.",
	}
	s.Attributes["insert_static_value_field_2"] = schema.StringAttribute{
		Optional:           true,
		Computed:           true,
		DeprecationMessage: "Use 'transforms_insert_static_value2_static_field' instead.",
		Description:        "DEPRECATED: Use 'transforms_insert_static_value2_static_field' instead.",
	}
	s.Attributes["insert_static_value_2"] = schema.StringAttribute{
		Optional:           true,
		Computed:           true,
		DeprecationMessage: "Use 'transforms_insert_static_value2_static_value' instead.",
		Description:        "DEPRECATED: Use 'transforms_insert_static_value2_static_value' instead.",
	}
	s.Attributes["predicates_istopictoenrich_pattern"] = schema.StringAttribute{
		Optional:           true,
		Computed:           true,
		DeprecationMessage: "Use 'predicates_is_topic_to_enrich_pattern' instead.",
		Description:        "DEPRECATED: Use 'predicates_is_topic_to_enrich_pattern' instead.",
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

// NewModelInstance returns a new instance of the PostgreSQL model.
func (c *PostgreSQLConfig) NewModelInstance() any {
	return &generated.SourcePostgresqlModel{}
}

// NewPostgreSQLResource creates a new PostgreSQL source resource.
func NewPostgreSQLResource() resource.Resource {
	return connector.NewBaseConnectorResource(&PostgreSQLConfig{})
}
