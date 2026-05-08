// Package transform provides Terraform resources for transform connectors.
package transform

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
)

// ToastHandlingConfig implements the TransformConfig interface for toast_handling transforms.
type ToastHandlingConfig struct{}

var _ TransformConfig = (*ToastHandlingConfig)(nil)

func (c *ToastHandlingConfig) GetSchema() schema.Schema {
	return generated.TransformToastHandlingSchema()
}

func (c *ToastHandlingConfig) GetFieldMappings() map[string]string {
	return generated.TransformToastHandlingFieldMappings
}

func (c *ToastHandlingConfig) GetTransformType() string {
	return "toast_handling"
}

func (c *ToastHandlingConfig) GetResourceName() string {
	return "transform_toast_handling"
}

func (c *ToastHandlingConfig) NewModelInstance() any {
	return &generated.TransformToastHandlingModel{}
}

// SupportsPreviewDeploy returns true because toast_handling is Flink-based.
func (c *ToastHandlingConfig) SupportsPreviewDeploy() bool {
	return true
}

// NewToastHandlingResource creates a new toast_handling transform resource.
func NewToastHandlingResource() resource.Resource {
	return NewBaseTransformResource(&ToastHandlingConfig{})
}
