// Package transform provides Terraform resources for transform connectors.
package transform

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
)

// EnrichAsyncConfig implements the TransformConfig interface for enrich_async transforms.
type EnrichAsyncConfig struct{}

// Ensure EnrichAsyncConfig implements TransformConfig.
var _ TransformConfig = (*EnrichAsyncConfig)(nil)

// GetSchema returns the Terraform schema for enrich_async transform.
func (c *EnrichAsyncConfig) GetSchema() schema.Schema {
	return generated.TransformEnrichAsyncSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *EnrichAsyncConfig) GetFieldMappings() map[string]string {
	return generated.TransformEnrichAsyncFieldMappings
}

// GetTransformType returns the transform type code for enrich_async.
func (c *EnrichAsyncConfig) GetTransformType() string {
	return "enrich_async"
}

// GetResourceName returns the Terraform resource name.
func (c *EnrichAsyncConfig) GetResourceName() string {
	return "transform_enrich_async"
}

// NewModelInstance returns a new instance of the enrich_async model.
func (c *EnrichAsyncConfig) NewModelInstance() any {
	return &generated.TransformEnrichAsyncModel{}
}

// NewEnrichAsyncResource creates a new enrich_async transform resource.
func NewEnrichAsyncResource() resource.Resource {
	return NewBaseTransformResource(&EnrichAsyncConfig{})
}
