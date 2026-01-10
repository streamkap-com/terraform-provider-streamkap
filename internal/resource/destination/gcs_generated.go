// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// GCSConfig implements the ConnectorConfig interface for GCS destinations.
type GCSConfig struct{}

// Ensure GCSConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*GCSConfig)(nil)

// GetSchema returns the Terraform schema for GCS destination.
func (c *GCSConfig) GetSchema() schema.Schema {
	return generated.DestinationGcsSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *GCSConfig) GetFieldMappings() map[string]string {
	return generated.DestinationGcsFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *GCSConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for GCS.
func (c *GCSConfig) GetConnectorCode() string {
	return "gcs"
}

// GetResourceName returns the Terraform resource name.
func (c *GCSConfig) GetResourceName() string {
	return "destination_gcs"
}

// NewModelInstance returns a new instance of the GCS model.
func (c *GCSConfig) NewModelInstance() any {
	return &generated.DestinationGcsModel{}
}

// NewGCSResource creates a new GCS destination resource.
func NewGCSResource() resource.Resource {
	return connector.NewBaseConnectorResource(&GCSConfig{})
}
