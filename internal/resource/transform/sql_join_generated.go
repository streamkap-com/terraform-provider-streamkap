// Package transform provides Terraform resources for transform connectors.
package transform

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
)

// SqlJoinConfig implements the TransformConfig interface for sql_join transforms.
type SqlJoinConfig struct{}

// Ensure SqlJoinConfig implements TransformConfig.
var _ TransformConfig = (*SqlJoinConfig)(nil)

// GetSchema returns the Terraform schema for sql_join transform.
func (c *SqlJoinConfig) GetSchema() schema.Schema {
	return generated.TransformSQLJoinSchema()
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *SqlJoinConfig) GetFieldMappings() map[string]string {
	return generated.TransformSQLJoinFieldMappings
}

// GetTransformType returns the transform type code for sql_join.
func (c *SqlJoinConfig) GetTransformType() string {
	return "sql_join"
}

// GetResourceName returns the Terraform resource name.
func (c *SqlJoinConfig) GetResourceName() string {
	return "transform_sql_join"
}

// NewModelInstance returns a new instance of the sql_join model.
func (c *SqlJoinConfig) NewModelInstance() any {
	return &generated.TransformSQLJoinModel{}
}

// NewSqlJoinResource creates a new sql_join transform resource.
func NewSqlJoinResource() resource.Resource {
	return NewBaseTransformResource(&SqlJoinConfig{})
}
