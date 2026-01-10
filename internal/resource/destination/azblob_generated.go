// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// AzBlobConfig implements the ConnectorConfig interface for Azure Blob Storage destinations.
type AzBlobConfig struct{}

// Ensure AzBlobConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*AzBlobConfig)(nil)

// GetSchema returns the Terraform schema for Azure Blob Storage destination.
func (c *AzBlobConfig) GetSchema() schema.Schema {
	return generated.DestinationAzblobSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *AzBlobConfig) GetFieldMappings() map[string]string {
	return generated.DestinationAzblobFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *AzBlobConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Azure Blob Storage.
func (c *AzBlobConfig) GetConnectorCode() string {
	return "azblob"
}

// GetResourceName returns the Terraform resource name.
func (c *AzBlobConfig) GetResourceName() string {
	return "destination_azblob"
}

// NewModelInstance returns a new instance of the Azure Blob Storage model.
func (c *AzBlobConfig) NewModelInstance() any {
	return &generated.DestinationAzblobModel{}
}

// NewAzBlobResource creates a new Azure Blob Storage destination resource.
func NewAzBlobResource() resource.Resource {
	return connector.NewBaseConnectorResource(&AzBlobConfig{})
}
