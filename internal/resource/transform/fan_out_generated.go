// Package transform provides Terraform resources for transform connectors.
package transform

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
)

// FanOutConfig implements the TransformConfig interface for fan_out transforms.
type FanOutConfig struct{}

// Ensure FanOutConfig implements TransformConfig.
var _ TransformConfig = (*FanOutConfig)(nil)

// GetSchema returns the Terraform schema for fan_out transform.
func (c *FanOutConfig) GetSchema() schema.Schema {
	return generated.TransformFanOutSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *FanOutConfig) GetFieldMappings() map[string]string {
	return generated.TransformFanOutFieldMappings
}

// GetTransformType returns the transform type code for fan_out.
func (c *FanOutConfig) GetTransformType() string {
	return "fan_out"
}

// GetResourceName returns the Terraform resource name.
func (c *FanOutConfig) GetResourceName() string {
	return "transform_fan_out"
}

// NewModelInstance returns a new instance of the fan_out model.
func (c *FanOutConfig) NewModelInstance() any {
	return &generated.TransformFanOutModel{}
}

// NewFanOutResource creates a new fan_out transform resource.
func NewFanOutResource() resource.Resource {
	return NewBaseTransformResource(&FanOutConfig{})
}
