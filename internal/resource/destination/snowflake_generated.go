// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// SnowflakeConfig implements the ConnectorConfig interface for Snowflake destinations.
type SnowflakeConfig struct{}

// Ensure SnowflakeConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*SnowflakeConfig)(nil)

// GetSchema returns the Terraform schema for Snowflake destination.
func (c *SnowflakeConfig) GetSchema() schema.Schema {
	return generated.DestinationSnowflakeSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *SnowflakeConfig) GetFieldMappings() map[string]string {
	return generated.DestinationSnowflakeFieldMappings
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
	return "streamkap_destination_snowflake"
}

// NewModelInstance returns a new instance of the Snowflake model.
func (c *SnowflakeConfig) NewModelInstance() any {
	return &generated.DestinationSnowflakeModel{}
}

// NewSnowflakeResource creates a new Snowflake destination resource.
func NewSnowflakeResource() resource.Resource {
	return connector.NewBaseConnectorResource(&SnowflakeConfig{})
}
