// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// RedshiftConfig implements the ConnectorConfig interface for Redshift destinations.
type RedshiftConfig struct{}

// Ensure RedshiftConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*RedshiftConfig)(nil)

// GetSchema returns the Terraform schema for Redshift destination.
func (c *RedshiftConfig) GetSchema() schema.Schema {
	return generated.DestinationRedshiftSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *RedshiftConfig) GetFieldMappings() map[string]string {
	return generated.DestinationRedshiftFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *RedshiftConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Redshift.
func (c *RedshiftConfig) GetConnectorCode() string {
	return "redshift"
}

// GetResourceName returns the Terraform resource name.
func (c *RedshiftConfig) GetResourceName() string {
	return "destination_redshift"
}

// NewModelInstance returns a new instance of the Redshift model.
func (c *RedshiftConfig) NewModelInstance() any {
	return &generated.DestinationRedshiftModel{}
}

// NewRedshiftResource creates a new Redshift destination resource.
func NewRedshiftResource() resource.Resource {
	return connector.NewBaseConnectorResource(&RedshiftConfig{})
}
