// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// KafkaDirectDestConfig implements the ConnectorConfig interface for Kafka Direct destinations.
type KafkaDirectDestConfig struct{}

// Ensure KafkaDirectDestConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*KafkaDirectDestConfig)(nil)

// GetSchema returns the Terraform schema for Kafka Direct destination.
func (c *KafkaDirectDestConfig) GetSchema() schema.Schema {
	return generated.DestinationKafkadirectSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *KafkaDirectDestConfig) GetFieldMappings() map[string]string {
	return generated.DestinationKafkadirectFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *KafkaDirectDestConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Kafka Direct.
func (c *KafkaDirectDestConfig) GetConnectorCode() string {
	return "kafkadirect"
}

// GetResourceName returns the Terraform resource name.
func (c *KafkaDirectDestConfig) GetResourceName() string {
	return "destination_kafkadirect"
}

// NewModelInstance returns a new instance of the Kafka Direct model.
func (c *KafkaDirectDestConfig) NewModelInstance() any {
	return &generated.DestinationKafkadirectModel{}
}

// NewKafkaDirectDestResource creates a new Kafka Direct destination resource.
func NewKafkaDirectDestResource() resource.Resource {
	return connector.NewBaseConnectorResource(&KafkaDirectDestConfig{})
}
