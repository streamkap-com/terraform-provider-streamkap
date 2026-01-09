// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// PostgreSQLConfig implements the ConnectorConfig interface for PostgreSQL sources.
type PostgreSQLConfig struct{}

// Ensure PostgreSQLConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*PostgreSQLConfig)(nil)

// GetSchema returns the Terraform schema for PostgreSQL source.
func (c *PostgreSQLConfig) GetSchema() schema.Schema {
	return generated.SourcePostgresqlSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *PostgreSQLConfig) GetFieldMappings() map[string]string {
	return generated.SourcePostgresqlFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *PostgreSQLConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for PostgreSQL.
func (c *PostgreSQLConfig) GetConnectorCode() string {
	return "postgresql"
}

// GetResourceName returns the Terraform resource name.
func (c *PostgreSQLConfig) GetResourceName() string {
	return "streamkap_source_postgresql"
}

// NewModelInstance returns a new instance of the PostgreSQL model.
func (c *PostgreSQLConfig) NewModelInstance() any {
	return &generated.SourcePostgresqlModel{}
}

// NewPostgreSQLResource creates a new PostgreSQL source resource.
func NewPostgreSQLResource() resource.Resource {
	return connector.NewBaseConnectorResource(&PostgreSQLConfig{})
}
