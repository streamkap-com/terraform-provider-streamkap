// Package source provides Terraform resources for source connectors.
package source

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/streamkap-com/terraform-provider-streamkap/internal/generated"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/resource/connector"
)

// sourceKafkadirectModelWithDeprecated embeds the generated model and adds
// the deprecated kafka_format alias so reflection-based marshaling can find it.
type sourceKafkadirectModelWithDeprecated struct {
	generated.SourceKafkadirectModel
	KafkaFormatOld types.String `tfsdk:"kafka_format"`
}

// kafkadirectFieldMappings extends the generated field mappings with deprecated aliases.
var kafkadirectFieldMappings = func() map[string]string {
	mappings := make(map[string]string)
	for k, v := range generated.SourceKafkadirectFieldMappings {
		mappings[k] = v
	}
	mappings["kafka_format"] = "format"
	return mappings
}()

// KafkaDirectConfig implements the ConnectorConfig interface for KafkaDirect sources.
type KafkaDirectConfig struct{}

// Ensure KafkaDirectConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*KafkaDirectConfig)(nil)

// GetSchema returns the Terraform schema for KafkaDirect source.
func (c *KafkaDirectConfig) GetSchema() schema.Schema {
	s := generated.SourceKafkadirectSchema()
	s.Attributes["kafka_format"] = schema.StringAttribute{
		Optional:           true,
		Computed:           true,
		DeprecationMessage: "Use 'format' instead.",
		Description:        "DEPRECATED: Use 'format' instead.",
		Validators: []validator.String{
			stringvalidator.ConflictsWith(path.MatchRoot("format")),
		},
	}
	return s
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *KafkaDirectConfig) GetFieldMappings() map[string]string {
	return kafkadirectFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *KafkaDirectConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for KafkaDirect.
func (c *KafkaDirectConfig) GetConnectorCode() string {
	return "kafkadirect"
}

// GetResourceName returns the Terraform resource name suffix.
func (c *KafkaDirectConfig) GetResourceName() string {
	return "source_kafkadirect"
}

// NewModelInstance returns a new instance of the KafkaDirect model with deprecated fields.
func (c *KafkaDirectConfig) NewModelInstance() any {
	return &sourceKafkadirectModelWithDeprecated{}
}

// NewKafkaDirectResource creates a new KafkaDirect source resource.
func NewKafkaDirectResource() resource.Resource {
	return connector.NewBaseConnectorResource(&KafkaDirectConfig{})
}
