// Package transform provides Terraform resources for transform connectors.
package transform

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
)

// MapFilterConfig implements the TransformConfig interface for map_filter transforms.
type MapFilterConfig struct{}

// Ensure MapFilterConfig implements TransformConfig.
var _ TransformConfig = (*MapFilterConfig)(nil)

// GetSchema returns the Terraform schema for map_filter transform.
func (c *MapFilterConfig) GetSchema() schema.Schema {
	return generated.TransformMapFilterSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *MapFilterConfig) GetFieldMappings() map[string]string {
	return generated.TransformMapFilterFieldMappings
}

// GetTransformType returns the transform type code for map_filter.
func (c *MapFilterConfig) GetTransformType() string {
	return "map_filter"
}

// GetResourceName returns the Terraform resource name.
func (c *MapFilterConfig) GetResourceName() string {
	return "transform_map_filter"
}

// NewModelInstance returns a new instance of the map_filter model.
func (c *MapFilterConfig) NewModelInstance() any {
	return &generated.TransformMapFilterModel{}
}

// NewMapFilterResource creates a new map_filter transform resource.
func NewMapFilterResource() resource.Resource {
	return NewBaseTransformResource(&MapFilterConfig{})
}
