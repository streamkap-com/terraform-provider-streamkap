// Package transform provides Terraform resources for transform connectors.
package transform

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
)

// EnrichConfig implements the TransformConfig interface for enrich transforms.
type EnrichConfig struct{}

// Ensure EnrichConfig implements TransformConfig.
var _ TransformConfig = (*EnrichConfig)(nil)

// GetSchema returns the Terraform schema for enrich transform.
func (c *EnrichConfig) GetSchema() schema.Schema {
	return generated.TransformEnrichSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *EnrichConfig) GetFieldMappings() map[string]string {
	return generated.TransformEnrichFieldMappings
}

// GetTransformType returns the transform type code for enrich.
func (c *EnrichConfig) GetTransformType() string {
	return "enrich"
}

// GetResourceName returns the Terraform resource name.
func (c *EnrichConfig) GetResourceName() string {
	return "transform_enrich"
}

// NewModelInstance returns a new instance of the enrich model.
func (c *EnrichConfig) NewModelInstance() any {
	return &generated.TransformEnrichModel{}
}

// NewEnrichResource creates a new enrich transform resource.
func NewEnrichResource() resource.Resource {
	return NewBaseTransformResource(&EnrichConfig{})
}
