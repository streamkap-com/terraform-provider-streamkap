// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// MySQLConfig implements the ConnectorConfig interface for MySQL sources.
type MySQLConfig struct{}

// Ensure MySQLConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*MySQLConfig)(nil)

// GetSchema returns the Terraform schema for MySQL source.
func (c *MySQLConfig) GetSchema() schema.Schema {
	return generated.SourceMysqlSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *MySQLConfig) GetFieldMappings() map[string]string {
	return generated.SourceMysqlFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *MySQLConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for MySQL.
func (c *MySQLConfig) GetConnectorCode() string {
	return "mysql"
}

// GetResourceName returns the Terraform resource name.
func (c *MySQLConfig) GetResourceName() string {
	return "streamkap_source_mysql"
}

// NewModelInstance returns a new instance of the MySQL model.
func (c *MySQLConfig) NewModelInstance() any {
	return &generated.SourceMysqlModel{}
}

// NewMySQLResource creates a new MySQL source resource.
func NewMySQLResource() resource.Resource {
	return connector.NewBaseConnectorResource(&MySQLConfig{})
}
