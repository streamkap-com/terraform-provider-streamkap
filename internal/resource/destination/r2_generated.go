// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// R2Config implements the ConnectorConfig interface for R2 destinations.
type R2Config struct{}

// Ensure R2Config implements ConnectorConfig.
var _ connector.ConnectorConfig = (*R2Config)(nil)

// GetSchema returns the Terraform schema for R2 destination.
func (c *R2Config) GetSchema() schema.Schema {
	return generated.DestinationR2Schema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *R2Config) GetFieldMappings() map[string]string {
	return generated.DestinationR2FieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *R2Config) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for R2.
func (c *R2Config) GetConnectorCode() string {
	return "r2"
}

// GetResourceName returns the Terraform resource name.
func (c *R2Config) GetResourceName() string {
	return "destination_r2"
}

// NewModelInstance returns a new instance of the R2 model.
func (c *R2Config) NewModelInstance() any {
	return &generated.DestinationR2Model{}
}

// NewR2Resource creates a new R2 destination resource.
func NewR2Resource() resource.Resource {
	return connector.NewBaseConnectorResource(&R2Config{})
}
