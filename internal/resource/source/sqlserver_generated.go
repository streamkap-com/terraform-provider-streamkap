// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// SQLServerConfig implements the ConnectorConfig interface for SQL Server sources.
type SQLServerConfig struct{}

// Ensure SQLServerConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*SQLServerConfig)(nil)

// GetSchema returns the Terraform schema for SQL Server source.
func (c *SQLServerConfig) GetSchema() schema.Schema {
	return generated.SourceSqlserverawsSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *SQLServerConfig) GetFieldMappings() map[string]string {
	return generated.SourceSqlserverawsFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *SQLServerConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for SQL Server.
func (c *SQLServerConfig) GetConnectorCode() string {
	return "sqlserveraws"
}

// GetResourceName returns the Terraform resource name.
func (c *SQLServerConfig) GetResourceName() string {
	return "source_sqlserver"
}

// NewModelInstance returns a new instance of the SQL Server model.
func (c *SQLServerConfig) NewModelInstance() any {
	return &generated.SourceSqlserverawsModel{}
}

// NewSQLServerResource creates a new SQL Server source resource.
func NewSQLServerResource() resource.Resource {
	return connector.NewBaseConnectorResource(&SQLServerConfig{})
}
