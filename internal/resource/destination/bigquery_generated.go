// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// BigQueryConfig implements the ConnectorConfig interface for BigQuery destinations.
type BigQueryConfig struct{}

// Ensure BigQueryConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*BigQueryConfig)(nil)

// GetSchema returns the Terraform schema for BigQuery destination.
func (c *BigQueryConfig) GetSchema() schema.Schema {
	return generated.DestinationBigquerySchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *BigQueryConfig) GetFieldMappings() map[string]string {
	return generated.DestinationBigqueryFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *BigQueryConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for BigQuery.
func (c *BigQueryConfig) GetConnectorCode() string {
	return "bigquery"
}

// GetResourceName returns the Terraform resource name.
func (c *BigQueryConfig) GetResourceName() string {
	return "destination_bigquery"
}

// NewModelInstance returns a new instance of the BigQuery model.
func (c *BigQueryConfig) NewModelInstance() any {
	return &generated.DestinationBigqueryModel{}
}

// NewBigQueryResource creates a new BigQuery destination resource.
func NewBigQueryResource() resource.Resource {
	return connector.NewBaseConnectorResource(&BigQueryConfig{})
}
