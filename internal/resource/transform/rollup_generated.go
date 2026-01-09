// Package transform provides Terraform resources for transform connectors.
package transform

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
)

// RollupConfig implements the TransformConfig interface for rollup transforms.
type RollupConfig struct{}

// Ensure RollupConfig implements TransformConfig.
var _ TransformConfig = (*RollupConfig)(nil)

// GetSchema returns the Terraform schema for rollup transform.
func (c *RollupConfig) GetSchema() schema.Schema {
	return generated.TransformRollupSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *RollupConfig) GetFieldMappings() map[string]string {
	return generated.TransformRollupFieldMappings
}

// GetTransformType returns the transform type code for rollup.
func (c *RollupConfig) GetTransformType() string {
	return "rollup"
}

// GetResourceName returns the Terraform resource name.
func (c *RollupConfig) GetResourceName() string {
	return "transform_rollup"
}

// NewModelInstance returns a new instance of the rollup model.
func (c *RollupConfig) NewModelInstance() any {
	return &generated.TransformRollupModel{}
}

// NewRollupResource creates a new rollup transform resource.
func NewRollupResource() resource.Resource {
	return NewBaseTransformResource(&RollupConfig{})
}
