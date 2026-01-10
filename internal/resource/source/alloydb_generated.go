// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// AlloyDBConfig implements the ConnectorConfig interface for AlloyDB sources.
type AlloyDBConfig struct{}

// Ensure AlloyDBConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*AlloyDBConfig)(nil)

// GetSchema returns the Terraform schema for AlloyDB source.
func (c *AlloyDBConfig) GetSchema() schema.Schema {
	return generated.SourceAlloydbSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *AlloyDBConfig) GetFieldMappings() map[string]string {
	return generated.SourceAlloydbFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *AlloyDBConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for AlloyDB.
func (c *AlloyDBConfig) GetConnectorCode() string {
	return "alloydb"
}

// GetResourceName returns the Terraform resource name.
func (c *AlloyDBConfig) GetResourceName() string {
	return "source_alloydb"
}

// NewModelInstance returns a new instance of the AlloyDB model.
func (c *AlloyDBConfig) NewModelInstance() any {
	return &generated.SourceAlloydbModel{}
}

// NewAlloyDBResource creates a new AlloyDB source resource.
func NewAlloyDBResource() resource.Resource {
	return connector.NewBaseConnectorResource(&AlloyDBConfig{})
}
