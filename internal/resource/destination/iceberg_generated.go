// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// IcebergConfig implements the ConnectorConfig interface for Iceberg destinations.
type IcebergConfig struct{}

// Ensure IcebergConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*IcebergConfig)(nil)

// GetSchema returns the Terraform schema for Iceberg destination.
func (c *IcebergConfig) GetSchema() schema.Schema {
	return generated.DestinationIcebergSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *IcebergConfig) GetFieldMappings() map[string]string {
	return generated.DestinationIcebergFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *IcebergConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Iceberg.
func (c *IcebergConfig) GetConnectorCode() string {
	return "iceberg"
}

// GetResourceName returns the Terraform resource name.
func (c *IcebergConfig) GetResourceName() string {
	return "destination_iceberg"
}

// NewModelInstance returns a new instance of the Iceberg model.
func (c *IcebergConfig) NewModelInstance() any {
	return &generated.DestinationIcebergModel{}
}

// NewIcebergResource creates a new Iceberg destination resource.
func NewIcebergResource() resource.Resource {
	return connector.NewBaseConnectorResource(&IcebergConfig{})
}
