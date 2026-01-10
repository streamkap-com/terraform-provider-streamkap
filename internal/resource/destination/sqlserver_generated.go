// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// SQLServerDestConfig implements the ConnectorConfig interface for SQL Server destinations.
type SQLServerDestConfig struct{}

// Ensure SQLServerDestConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*SQLServerDestConfig)(nil)

// GetSchema returns the Terraform schema for SQL Server destination.
func (c *SQLServerDestConfig) GetSchema() schema.Schema {
	return generated.DestinationSqlserverSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *SQLServerDestConfig) GetFieldMappings() map[string]string {
	return generated.DestinationSqlserverFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *SQLServerDestConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for SQL Server.
func (c *SQLServerDestConfig) GetConnectorCode() string {
	return "sqlserver"
}

// GetResourceName returns the Terraform resource name.
func (c *SQLServerDestConfig) GetResourceName() string {
	return "destination_sqlserver"
}

// NewModelInstance returns a new instance of the SQL Server model.
func (c *SQLServerDestConfig) NewModelInstance() any {
	return &generated.DestinationSqlserverModel{}
}

// NewSQLServerDestResource creates a new SQL Server destination resource.
func NewSQLServerDestResource() resource.Resource {
	return connector.NewBaseConnectorResource(&SQLServerDestConfig{})
}
