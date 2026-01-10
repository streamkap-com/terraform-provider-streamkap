// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// DB2Config implements the ConnectorConfig interface for DB2 sources.
type DB2Config struct{}

// Ensure DB2Config implements ConnectorConfig.
var _ connector.ConnectorConfig = (*DB2Config)(nil)

// GetSchema returns the Terraform schema for DB2 source.
func (c *DB2Config) GetSchema() schema.Schema {
	return generated.SourceDb2Schema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *DB2Config) GetFieldMappings() map[string]string {
	return generated.SourceDb2FieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *DB2Config) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for DB2.
func (c *DB2Config) GetConnectorCode() string {
	return "db2"
}

// GetResourceName returns the Terraform resource name.
func (c *DB2Config) GetResourceName() string {
	return "source_db2"
}

// NewModelInstance returns a new instance of the DB2 model.
func (c *DB2Config) NewModelInstance() any {
	return &generated.SourceDb2Model{}
}

// NewDB2Resource creates a new DB2 source resource.
func NewDB2Resource() resource.Resource {
	return connector.NewBaseConnectorResource(&DB2Config{})
}
