// Package destination provides Terraform resources for destination connectors.
package destination

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// RedisDestConfig implements the ConnectorConfig interface for Redis destinations.
type RedisDestConfig struct{}

// Ensure RedisDestConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*RedisDestConfig)(nil)

// GetSchema returns the Terraform schema for Redis destination.
func (c *RedisDestConfig) GetSchema() schema.Schema {
	return generated.DestinationRedisSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *RedisDestConfig) GetFieldMappings() map[string]string {
	return generated.DestinationRedisFieldMappings
}

// GetConnectorType returns the connector type (destination).
func (c *RedisDestConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeDestination
}

// GetConnectorCode returns the connector code for Redis.
func (c *RedisDestConfig) GetConnectorCode() string {
	return "redis"
}

// GetResourceName returns the Terraform resource name.
func (c *RedisDestConfig) GetResourceName() string {
	return "destination_redis"
}

// NewModelInstance returns a new instance of the Redis model.
func (c *RedisDestConfig) NewModelInstance() any {
	return &generated.DestinationRedisModel{}
}

// NewRedisDestResource creates a new Redis destination resource.
func NewRedisDestResource() resource.Resource {
	return connector.NewBaseConnectorResource(&RedisDestConfig{})
}
