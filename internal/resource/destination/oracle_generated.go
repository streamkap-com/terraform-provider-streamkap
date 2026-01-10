// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// OracleDestConfig implements the ConnectorConfig interface for Oracle destinations.
type OracleDestConfig struct{}

// Ensure OracleDestConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*OracleDestConfig)(nil)

// GetSchema returns the Terraform schema for Oracle destination.
func (c *OracleDestConfig) GetSchema() schema.Schema {
	return generated.DestinationOracleSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *OracleDestConfig) GetFieldMappings() map[string]string {
	return generated.DestinationOracleFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *OracleDestConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Oracle.
func (c *OracleDestConfig) GetConnectorCode() string {
	return "oracle"
}

// GetResourceName returns the Terraform resource name.
func (c *OracleDestConfig) GetResourceName() string {
	return "destination_oracle"
}

// NewModelInstance returns a new instance of the Oracle model.
func (c *OracleDestConfig) NewModelInstance() any {
	return &generated.DestinationOracleModel{}
}

// NewOracleDestResource creates a new Oracle destination resource.
func NewOracleDestResource() resource.Resource {
	return connector.NewBaseConnectorResource(&OracleDestConfig{})
}
