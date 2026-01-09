// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// snowflakeFieldMappings extends the generated field mappings with deprecated aliases.
var snowflakeFieldMappings = func() map[string]string {
	mappings := make(map[string]string)
	for k, v := range generated.DestinationSnowflakeFieldMappings {
		mappings[k] = v
	}
	// Deprecated alias - maps to same API field
	mappings["auto_schema_creation"] = "create.schema.auto"
	return mappings
}()

// SnowflakeConfig implements the ConnectorConfig interface for Snowflake destinations.
type SnowflakeConfig struct{}

// Ensure SnowflakeConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*SnowflakeConfig)(nil)

// GetSchema returns the Terraform schema for Snowflake destination.
func (c *SnowflakeConfig) GetSchema() schema.Schema {
	s := generated.DestinationSnowflakeSchema()
	// Add deprecated alias - maps to the same API field as create_schema_auto
	s.Attributes["auto_schema_creation"] = schema.BoolAttribute{
		Optional:           true,
		Computed:           true,
		DeprecationMessage: "Use 'create_schema_auto' instead.",
		Description:        "DEPRECATED: Use 'create_schema_auto' instead.",
	}
	return s
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *SnowflakeConfig) GetFieldMappings() map[string]string {
	return snowflakeFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *SnowflakeConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Snowflake.
func (c *SnowflakeConfig) GetConnectorCode() string {
	return "snowflake"
}

// GetResourceName returns the Terraform resource name.
func (c *SnowflakeConfig) GetResourceName() string {
	return "destination_snowflake"
}

// NewModelInstance returns a new instance of the Snowflake model.
func (c *SnowflakeConfig) NewModelInstance() any {
	return &generated.DestinationSnowflakeModel{}
}

// NewSnowflakeResource creates a new Snowflake destination resource.
func NewSnowflakeResource() resource.Resource {
	return connector.NewBaseConnectorResource(&SnowflakeConfig{})
}
