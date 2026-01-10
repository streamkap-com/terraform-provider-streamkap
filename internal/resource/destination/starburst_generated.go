// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// StarburstConfig implements the ConnectorConfig interface for Starburst destinations.
type StarburstConfig struct{}

// Ensure StarburstConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*StarburstConfig)(nil)

// GetSchema returns the Terraform schema for Starburst destination.
func (c *StarburstConfig) GetSchema() schema.Schema {
	return generated.DestinationStarburstSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *StarburstConfig) GetFieldMappings() map[string]string {
	return generated.DestinationStarburstFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *StarburstConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Starburst.
func (c *StarburstConfig) GetConnectorCode() string {
	return "starburst"
}

// GetResourceName returns the Terraform resource name.
func (c *StarburstConfig) GetResourceName() string {
	return "destination_starburst"
}

// NewModelInstance returns a new instance of the Starburst model.
func (c *StarburstConfig) NewModelInstance() any {
	return &generated.DestinationStarburstModel{}
}

// NewStarburstResource creates a new Starburst destination resource.
func NewStarburstResource() resource.Resource {
	return connector.NewBaseConnectorResource(&StarburstConfig{})
}
