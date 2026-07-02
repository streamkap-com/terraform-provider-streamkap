// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// InformixConfig implements the ConnectorConfig interface for Informix sources.
type InformixConfig struct{}

// Ensure InformixConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*InformixConfig)(nil)

// GetSchema returns the Terraform schema for Informix source.
func (c *InformixConfig) GetSchema() schema.Schema {
	return generated.SourceInformixSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *InformixConfig) GetFieldMappings() map[string]string {
	return generated.SourceInformixFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *InformixConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for Informix.
func (c *InformixConfig) GetConnectorCode() string {
	return "informix"
}

// GetResourceName returns the Terraform resource name.
func (c *InformixConfig) GetResourceName() string {
	return "source_informix"
}

// NewModelInstance returns a new instance of the Informix model.
func (c *InformixConfig) NewModelInstance() any {
	return &generated.SourceInformixModel{}
}

// NewInformixResource creates a new Informix source resource.
func NewInformixResource() resource.Resource {
	return connector.NewBaseConnectorResource(&InformixConfig{})
}
