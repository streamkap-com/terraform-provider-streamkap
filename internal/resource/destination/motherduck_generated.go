// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// MotherduckConfig implements the ConnectorConfig interface for Motherduck destinations.
type MotherduckConfig struct{}

// Ensure MotherduckConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*MotherduckConfig)(nil)

// GetSchema returns the Terraform schema for Motherduck destination.
func (c *MotherduckConfig) GetSchema() schema.Schema {
	return generated.DestinationMotherduckSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *MotherduckConfig) GetFieldMappings() map[string]string {
	return generated.DestinationMotherduckFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *MotherduckConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Motherduck.
func (c *MotherduckConfig) GetConnectorCode() string {
	return "motherduck"
}

// GetResourceName returns the Terraform resource name.
func (c *MotherduckConfig) GetResourceName() string {
	return "destination_motherduck"
}

// NewModelInstance returns a new instance of the Motherduck model.
func (c *MotherduckConfig) NewModelInstance() any {
	return &generated.DestinationMotherduckModel{}
}

// NewMotherduckResource creates a new Motherduck destination resource.
func NewMotherduckResource() resource.Resource {
	return connector.NewBaseConnectorResource(&MotherduckConfig{})
}
