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

// sourceMongodbModelWithDeprecated embeds the generated model and adds
// deprecated field aliases so reflection-based marshaling can find them.
type sourceMongodbModelWithDeprecated struct {
	generated.SourceMongodbModel
	InsertStaticKeyField1               types.String `tfsdk:"insert_static_key_field_1"`
	InsertStaticKeyValue1               types.String `tfsdk:"insert_static_key_value_1"`
	InsertStaticValueField1             types.String `tfsdk:"insert_static_value_field_1"`
	InsertStaticValue1                  types.String `tfsdk:"insert_static_value_1"`
	InsertStaticKeyField2               types.String `tfsdk:"insert_static_key_field_2"`
	InsertStaticKeyValue2               types.String `tfsdk:"insert_static_key_value_2"`
	InsertStaticValueField2             types.String `tfsdk:"insert_static_value_field_2"`
	InsertStaticValue2                  types.String `tfsdk:"insert_static_value_2"`
	PredicatesIsTopicToEnrichPatternOld types.String `tfsdk:"predicates_istopictoenrich_pattern"`
	ArrayEncodingOld                    types.String `tfsdk:"array_encoding"`
	NestedDocumentEncodingOld           types.String `tfsdk:"nested_document_encoding"`
}

// mongodbFieldMappings extends the generated field mappings with deprecated aliases.
var mongodbFieldMappings = func() map[string]string {
	mappings := make(map[string]string)
	for k, v := range generated.SourceMongodbFieldMappings {
		mappings[k] = v
	}
	// Deprecated aliases - map to same API fields
	mappings["insert_static_key_field_1"] = "transforms.InsertStaticKey1.static.field"
	mappings["insert_static_key_value_1"] = "transforms.InsertStaticKey1.static.value"
	mappings["insert_static_value_field_1"] = "transforms.InsertStaticValue1.static.field"
	mappings["insert_static_value_1"] = "transforms.InsertStaticValue1.static.value"
	mappings["insert_static_key_field_2"] = "transforms.InsertStaticKey2.static.field"
	mappings["insert_static_key_value_2"] = "transforms.InsertStaticKey2.static.value"
	mappings["insert_static_value_field_2"] = "transforms.InsertStaticValue2.static.field"
	mappings["insert_static_value_2"] = "transforms.InsertStaticValue2.static.value"
	mappings["predicates_istopictoenrich_pattern"] = "predicates.IsTopicToEnrich.pattern"
	mappings["array_encoding"] = "transforms.unwrap.array.encoding"
	mappings["nested_document_encoding"] = "transforms.unwrap.document.encoding"
	return mappings
}()

// MongoDBConfig implements the ConnectorConfig interface for MongoDB sources.
type MongoDBConfig struct{}

// Ensure MongoDBConfig implements ConnectorConfig.
var _ connector.ConnectorConfig = (*MongoDBConfig)(nil)

// GetSchema returns the Terraform schema for MongoDB source.
func (c *MongoDBConfig) GetSchema() schema.Schema {
	s := generated.SourceMongodbSchema()
	// Deprecated aliases — map to the same API fields as the new names.
	// Computed so the provider can populate them from the API on Read.
	// ConflictsWith prevents setting both old and new names simultaneously.
	deprecatedAliases := []struct {
		oldName, newName string
	}{
		{"insert_static_key_field_1", "transforms_insert_static_key1_static_field"},
		{"insert_static_key_value_1", "transforms_insert_static_key1_static_value"},
		{"insert_static_value_field_1", "transforms_insert_static_value1_static_field"},
		{"insert_static_value_1", "transforms_insert_static_value1_static_value"},
		{"insert_static_key_field_2", "transforms_insert_static_key2_static_field"},
		{"insert_static_key_value_2", "transforms_insert_static_key2_static_value"},
		{"insert_static_value_field_2", "transforms_insert_static_value2_static_field"},
		{"insert_static_value_2", "transforms_insert_static_value2_static_value"},
		{"predicates_istopictoenrich_pattern", "predicates_is_topic_to_enrich_pattern"},
		{"array_encoding", "transforms_unwrap_array_encoding"},
		{"nested_document_encoding", "transforms_unwrap_document_encoding"},
	}
	for _, alias := range deprecatedAliases {
		s.Attributes[alias.oldName] = schema.StringAttribute{
			Optional:           true,
			Computed:           true,
			DeprecationMessage: "Use '" + alias.newName + "' instead.",
			Description:        "DEPRECATED: Use '" + alias.newName + "' instead.",
			Validators: []validator.String{
				stringvalidator.ConflictsWith(path.MatchRoot(alias.newName)),
			},
		}
	}
	return s
}

// GetFieldMappings returns the field mappings from Terraform attributes to API fields.
func (c *MongoDBConfig) GetFieldMappings() map[string]string {
	return mongodbFieldMappings
}

// GetConnectorType returns the connector type (source).
func (c *MongoDBConfig) GetConnectorType() connector.ConnectorType {
	return connector.ConnectorTypeSource
}

// GetConnectorCode returns the connector code for MongoDB.
func (c *MongoDBConfig) GetConnectorCode() string {
	return "mongodb"
}

// GetResourceName returns the Terraform resource name.
func (c *MongoDBConfig) GetResourceName() string {
	return "source_mongodb"
}

// NewModelInstance returns a new instance of the MongoDB model with deprecated fields.
func (c *MongoDBConfig) NewModelInstance() any {
	return &sourceMongodbModelWithDeprecated{}
}

// NewMongoDBResource creates a new MongoDB source resource.
func NewMongoDBResource() resource.Resource {
	return connector.NewBaseConnectorResource(&MongoDBConfig{})
}
