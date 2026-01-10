// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// DB2DestConfig implements the ConnectorConfig interface for DB2 destinations.
type DB2DestConfig struct{}

// Ensure DB2DestConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*DB2DestConfig)(nil)

// GetSchema returns the Terraform schema for DB2 destination.
func (c *DB2DestConfig) GetSchema() schema.Schema {
	return generated.DestinationDb2Schema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *DB2DestConfig) GetFieldMappings() map[string]string {
	return generated.DestinationDb2FieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *DB2DestConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for DB2.
func (c *DB2DestConfig) GetConnectorCode() string {
	return "db2"
}

// GetResourceName returns the Terraform resource name.
func (c *DB2DestConfig) GetResourceName() string {
	return "destination_db2"
}

// NewModelInstance returns a new instance of the DB2 model.
func (c *DB2DestConfig) NewModelInstance() any {
	return &generated.DestinationDb2Model{}
}

// NewDB2DestResource creates a new DB2 destination resource.
func NewDB2DestResource() resource.Resource {
	return connector.NewBaseConnectorResource(&DB2DestConfig{})
}
