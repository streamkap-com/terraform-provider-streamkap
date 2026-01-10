// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// RedisConfig implements the ConnectorConfig interface for Redis sources.
type RedisConfig struct{}

// Ensure RedisConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*RedisConfig)(nil)

// GetSchema returns the Terraform schema for Redis source.
func (c *RedisConfig) GetSchema() schema.Schema {
	return generated.SourceRedisSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *RedisConfig) GetFieldMappings() map[string]string {
	return generated.SourceRedisFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *RedisConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for Redis.
func (c *RedisConfig) GetConnectorCode() string {
	return "redis"
}

// GetResourceName returns the Terraform resource name.
func (c *RedisConfig) GetResourceName() string {
	return "source_redis"
}

// NewModelInstance returns a new instance of the Redis model.
func (c *RedisConfig) NewModelInstance() any {
	return &generated.SourceRedisModel{}
}

// NewRedisResource creates a new Redis source resource.
func NewRedisResource() resource.Resource {
	return connector.NewBaseConnectorResource(&RedisConfig{})
}
