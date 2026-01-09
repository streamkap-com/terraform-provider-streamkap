// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// ClickHouseConfig implements the ConnectorConfig interface for ClickHouse destinations.
type ClickHouseConfig struct{}

// Ensure ClickHouseConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*ClickHouseConfig)(nil)

// GetSchema returns the Terraform schema for ClickHouse destination.
func (c *ClickHouseConfig) GetSchema() schema.Schema {
	return generated.DestinationClickhouseSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *ClickHouseConfig) GetFieldMappings() map[string]string {
	return generated.DestinationClickhouseFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *ClickHouseConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for ClickHouse.
func (c *ClickHouseConfig) GetConnectorCode() string {
	return "clickhouse"
}

// GetResourceName returns the Terraform resource name.
func (c *ClickHouseConfig) GetResourceName() string {
	return "destination_clickhouse"
}

// NewModelInstance returns a new instance of the ClickHouse model.
func (c *ClickHouseConfig) NewModelInstance() any {
	return &generated.DestinationClickhouseModel{}
}

// NewClickHouseResource creates a new ClickHouse destination resource.
func NewClickHouseResource() resource.Resource {
	return connector.NewBaseConnectorResource(&ClickHouseConfig{})
}
