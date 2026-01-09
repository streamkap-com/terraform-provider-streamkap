// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// PostgreSQLConfig implements the ConnectorConfig interface for PostgreSQL destinations.
type PostgreSQLConfig struct{}

// Ensure PostgreSQLConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*PostgreSQLConfig)(nil)

// GetSchema returns the Terraform schema for PostgreSQL destination.
func (c *PostgreSQLConfig) GetSchema() schema.Schema {
	return generated.DestinationPostgresqlSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *PostgreSQLConfig) GetFieldMappings() map[string]string {
	return generated.DestinationPostgresqlFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *PostgreSQLConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for PostgreSQL.
func (c *PostgreSQLConfig) GetConnectorCode() string {
	return "postgresql"
}

// GetResourceName returns the Terraform resource name.
func (c *PostgreSQLConfig) GetResourceName() string {
	return "destination_postgresql"
}

// NewModelInstance returns a new instance of the PostgreSQL model.
func (c *PostgreSQLConfig) NewModelInstance() any {
	return &generated.DestinationPostgresqlModel{}
}

// NewPostgreSQLResource creates a new PostgreSQL destination resource.
func NewPostgreSQLResource() resource.Resource {
	return connector.NewBaseConnectorResource(&PostgreSQLConfig{})
}
