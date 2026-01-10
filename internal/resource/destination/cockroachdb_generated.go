// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// CockroachDBConfig implements the ConnectorConfig interface for CockroachDB destinations.
type CockroachDBConfig struct{}

// Ensure CockroachDBConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*CockroachDBConfig)(nil)

// GetSchema returns the Terraform schema for CockroachDB destination.
func (c *CockroachDBConfig) GetSchema() schema.Schema {
	return generated.DestinationCockroachdbSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *CockroachDBConfig) GetFieldMappings() map[string]string {
	return generated.DestinationCockroachdbFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *CockroachDBConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for CockroachDB.
func (c *CockroachDBConfig) GetConnectorCode() string {
	return "cockroachdb"
}

// GetResourceName returns the Terraform resource name.
func (c *CockroachDBConfig) GetResourceName() string {
	return "destination_cockroachdb"
}

// NewModelInstance returns a new instance of the CockroachDB model.
func (c *CockroachDBConfig) NewModelInstance() any {
	return &generated.DestinationCockroachdbModel{}
}

// NewCockroachDBResource creates a new CockroachDB destination resource.
func NewCockroachDBResource() resource.Resource {
	return connector.NewBaseConnectorResource(&CockroachDBConfig{})
}
