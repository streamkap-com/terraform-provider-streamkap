// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// KafkaDirectConfig implements the ConnectorConfig interface for KafkaDirect sources.
type KafkaDirectConfig struct{}

// Ensure KafkaDirectConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*KafkaDirectConfig)(nil)

// GetSchema returns the Terraform schema for KafkaDirect source.
func (c *KafkaDirectConfig) GetSchema() schema.Schema {
	return generated.SourceKafkadirectSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *KafkaDirectConfig) GetFieldMappings() map[string]string {
	return generated.SourceKafkadirectFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *KafkaDirectConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for KafkaDirect.
func (c *KafkaDirectConfig) GetConnectorCode() string {
	return "kafkadirect"
}

// GetResourceName returns the Terraform resource name suffix.
func (c *KafkaDirectConfig) GetResourceName() string {
	return "source_kafkadirect"
}

// NewModelInstance returns a new instance of the KafkaDirect model.
func (c *KafkaDirectConfig) NewModelInstance() any {
	return &generated.SourceKafkadirectModel{}
}

// NewKafkaDirectResource creates a new KafkaDirect source resource.
func NewKafkaDirectResource() resource.Resource {
	return connector.NewBaseConnectorResource(&KafkaDirectConfig{})
}
