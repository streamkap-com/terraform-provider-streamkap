// Package transform provides Terraform resources for transform connectors.
package transform

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
)

// UnNestingConfig implements the TransformConfig interface for un_nesting transforms.
type UnNestingConfig struct{}

var _ TransformConfig = (*UnNestingConfig)(nil)

func (c *UnNestingConfig) GetSchema() schema.Schema {
	return generated.TransformUnNestingSchema()
}

func (c *UnNestingConfig) GetFieldMappings() map[string]string {
	return generated.TransformUnNestingFieldMappings
}

func (c *UnNestingConfig) GetTransformType() string {
	return "un_nesting"
}

func (c *UnNestingConfig) GetResourceName() string {
	return "transform_un_nesting"
}

func (c *UnNestingConfig) NewModelInstance() any {
	return &generated.TransformUnNestingModel{}
}

// SupportsPreviewDeploy returns true because un_nesting is Flink-based.
func (c *UnNestingConfig) SupportsPreviewDeploy() bool {
	return true
}

// NewUnNestingResource creates a new un_nesting transform resource.
func NewUnNestingResource() resource.Resource {
	return NewBaseTransformResource(&UnNestingConfig{})
}
