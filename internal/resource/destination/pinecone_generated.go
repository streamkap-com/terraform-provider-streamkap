// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// PineconeDestConfig implements the ConnectorConfig interface for Pinecone destinations.
type PineconeDestConfig struct{}

// Ensure PineconeDestConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*PineconeDestConfig)(nil)

// GetSchema returns the Terraform schema for Pinecone destination.
func (c *PineconeDestConfig) GetSchema() schema.Schema {
	return generated.DestinationPineconeSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *PineconeDestConfig) GetFieldMappings() map[string]string {
	return generated.DestinationPineconeFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *PineconeDestConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Pinecone.
func (c *PineconeDestConfig) GetConnectorCode() string {
	return "pinecone"
}

// GetResourceName returns the Terraform resource name.
func (c *PineconeDestConfig) GetResourceName() string {
	return "destination_pinecone"
}

// NewModelInstance returns a new instance of the Pinecone model.
func (c *PineconeDestConfig) NewModelInstance() any {
	return &generated.DestinationPineconeModel{}
}

// NewPineconeDestResource creates a new Pinecone destination resource.
func NewPineconeDestResource() resource.Resource {
	return connector.NewBaseConnectorResource(&PineconeDestConfig{})
}
