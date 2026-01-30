// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// DatabricksConfig implements the ConnectorConfig interface for Databricks destinations.
type DatabricksConfig struct{}

// Ensure DatabricksConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*DatabricksConfig)(nil)

// GetSchema returns the Terraform schema for Databricks destination.
func (c *DatabricksConfig) GetSchema() schema.Schema {
	return generated.DestinationDatabricksSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *DatabricksConfig) GetFieldMappings() map[string]string {
	return generated.DestinationDatabricksFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *DatabricksConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Databricks.
func (c *DatabricksConfig) GetConnectorCode() string {
	return "databricks"
}

// GetResourceName returns the Terraform resource name.
func (c *DatabricksConfig) GetResourceName() string {
	return "destination_databricks"
}

// NewModelInstance returns a new instance of the Databricks model.
func (c *DatabricksConfig) NewModelInstance() any {
	return &generated.DestinationDatabricksModel{}
}

// NewDatabricksResource creates a new Databricks destination resource.
func NewDatabricksResource() resource.Resource {
	return connector.NewBaseConnectorResource(&DatabricksConfig{})
}
