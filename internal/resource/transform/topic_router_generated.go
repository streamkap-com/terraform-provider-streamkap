// Package transform provides Terraform resources for transform connectors.
package transform

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
)

// TopicRouterConfig implements the TransformConfig interface for topic_router transforms.
type TopicRouterConfig struct{}

// Ensure TopicRouterConfig implements TransformConfig.
var _ TransformConfig = (*TopicRouterConfig)(nil)

// GetSchema returns the Terraform schema for topic_router transform.
func (c *TopicRouterConfig) GetSchema() schema.Schema {
	return generated.TransformTopicRouterSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *TopicRouterConfig) GetFieldMappings() map[string]string {
	return generated.TransformTopicRouterFieldMappings
}

// GetTransformType returns the transform type code for topic_router.
func (c *TopicRouterConfig) GetTransformType() string {
	return "topic_router"
}

// GetResourceName returns the Terraform resource name.
func (c *TopicRouterConfig) GetResourceName() string {
	return "transform_topic_router"
}

// NewModelInstance returns a new instance of the topic_router model.
func (c *TopicRouterConfig) NewModelInstance() any {
	return &generated.TransformTopicRouterModel{}
}

// SupportsPreviewDeploy returns false because topic_router is KC-based, not Flink-based.
func (c *TopicRouterConfig) SupportsPreviewDeploy() bool {
	return false
}

// NewTopicRouterResource creates a new topic_router transform resource.
func NewTopicRouterResource() resource.Resource {
	return NewBaseTransformResource(&TopicRouterConfig{})
}
