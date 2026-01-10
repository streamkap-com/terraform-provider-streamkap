// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// MySQLDestConfig implements the ConnectorConfig interface for MySQL destinations.
type MySQLDestConfig struct{}

// Ensure MySQLDestConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*MySQLDestConfig)(nil)

// GetSchema returns the Terraform schema for MySQL destination.
func (c *MySQLDestConfig) GetSchema() schema.Schema {
	return generated.DestinationMysqlSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *MySQLDestConfig) GetFieldMappings() map[string]string {
	return generated.DestinationMysqlFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *MySQLDestConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for MySQL.
func (c *MySQLDestConfig) GetConnectorCode() string {
	return "mysql"
}

// GetResourceName returns the Terraform resource name.
func (c *MySQLDestConfig) GetResourceName() string {
	return "destination_mysql"
}

// NewModelInstance returns a new instance of the MySQL model.
func (c *MySQLDestConfig) NewModelInstance() any {
	return &generated.DestinationMysqlModel{}
}

// NewMySQLDestResource creates a new MySQL destination resource.
func NewMySQLDestResource() resource.Resource {
	return connector.NewBaseConnectorResource(&MySQLDestConfig{})
}
