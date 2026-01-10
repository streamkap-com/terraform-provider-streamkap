// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// DocumentDBConfig implements the ConnectorConfig interface for DocumentDB sources.
type DocumentDBConfig struct{}

// Ensure DocumentDBConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*DocumentDBConfig)(nil)

// GetSchema returns the Terraform schema for DocumentDB source.
func (c *DocumentDBConfig) GetSchema() schema.Schema {
	return generated.SourceDocumentdbSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *DocumentDBConfig) GetFieldMappings() map[string]string {
	return generated.SourceDocumentdbFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *DocumentDBConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for DocumentDB.
func (c *DocumentDBConfig) GetConnectorCode() string {
	return "documentdb"
}

// GetResourceName returns the Terraform resource name.
func (c *DocumentDBConfig) GetResourceName() string {
	return "source_documentdb"
}

// NewModelInstance returns a new instance of the DocumentDB model.
func (c *DocumentDBConfig) NewModelInstance() any {
	return &generated.SourceDocumentdbModel{}
}

// NewDocumentDBResource creates a new DocumentDB source resource.
func NewDocumentDBResource() resource.Resource {
	return connector.NewBaseConnectorResource(&DocumentDBConfig{})
}
