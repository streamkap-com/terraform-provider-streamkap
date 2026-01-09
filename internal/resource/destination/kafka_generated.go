// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// KafkaConfig implements the ConnectorConfig interface for Kafka destinations.
type KafkaConfig struct{}

// Ensure KafkaConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*KafkaConfig)(nil)

// GetSchema returns the Terraform schema for Kafka destination.
func (c *KafkaConfig) GetSchema() schema.Schema {
	return generated.DestinationKafkaSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *KafkaConfig) GetFieldMappings() map[string]string {
	return generated.DestinationKafkaFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *KafkaConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Kafka.
func (c *KafkaConfig) GetConnectorCode() string {
	return "kafka"
}

// GetResourceName returns the Terraform resource name.
func (c *KafkaConfig) GetResourceName() string {
	return "streamkap_destination_kafka"
}

// NewModelInstance returns a new instance of the Kafka model.
func (c *KafkaConfig) NewModelInstance() any {
	return &generated.DestinationKafkaModel{}
}

// NewKafkaResource creates a new Kafka destination resource.
func NewKafkaResource() resource.Resource {
	return connector.NewBaseConnectorResource(&KafkaConfig{})
}
