// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// HTTPSinkConfig implements the ConnectorConfig interface for HTTP Sink destinations.
type HTTPSinkConfig struct{}

// Ensure HTTPSinkConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*HTTPSinkConfig)(nil)

// GetSchema returns the Terraform schema for HTTP Sink destination.
func (c *HTTPSinkConfig) GetSchema() schema.Schema {
	return generated.DestinationHttpsinkSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *HTTPSinkConfig) GetFieldMappings() map[string]string {
	return generated.DestinationHttpsinkFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *HTTPSinkConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for HTTP Sink.
func (c *HTTPSinkConfig) GetConnectorCode() string {
	return "httpsink"
}

// GetResourceName returns the Terraform resource name.
func (c *HTTPSinkConfig) GetResourceName() string {
	return "destination_httpsink"
}

// NewModelInstance returns a new instance of the HTTP Sink model.
func (c *HTTPSinkConfig) NewModelInstance() any {
	return &generated.DestinationHttpsinkModel{}
}

// NewHTTPSinkResource creates a new HTTP Sink destination resource.
func NewHTTPSinkResource() resource.Resource {
	return connector.NewBaseConnectorResource(&HTTPSinkConfig{})
}
