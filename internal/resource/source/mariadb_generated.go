// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// MariaDBConfig implements the ConnectorConfig interface for MariaDB sources.
type MariaDBConfig struct{}

// Ensure MariaDBConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*MariaDBConfig)(nil)

// GetSchema returns the Terraform schema for MariaDB source.
func (c *MariaDBConfig) GetSchema() schema.Schema {
	return generated.SourceMariadbSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *MariaDBConfig) GetFieldMappings() map[string]string {
	return generated.SourceMariadbFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *MariaDBConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for MariaDB.
func (c *MariaDBConfig) GetConnectorCode() string {
	return "mariadb"
}

// GetResourceName returns the Terraform resource name.
func (c *MariaDBConfig) GetResourceName() string {
	return "source_mariadb"
}

// NewModelInstance returns a new instance of the MariaDB model.
func (c *MariaDBConfig) NewModelInstance() any {
	return &generated.SourceMariadbModel{}
}

// NewMariaDBResource creates a new MariaDB source resource.
func NewMariaDBResource() resource.Resource {
	return connector.NewBaseConnectorResource(&MariaDBConfig{})
}
