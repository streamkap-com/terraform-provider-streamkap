package datasource

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/streamkap-com/terraform-provider-streamkap/internal/api"
)

// TopicSerializationModel is the shared nested model for a topic's serialization
// block, used by both the streamkap_topic and streamkap_topics data sources.
type TopicSerializationModel struct {
	KeyFormat             types.String `tfsdk:"key_format"`
	ValueFormat           types.String `tfsdk:"value_format"`
	KeyConverter          types.String `tfsdk:"key_converter"`
	ValueConverter        types.String `tfsdk:"value_converter"`
	SchemaRegistryEnabled types.Bool   `tfsdk:"schema_registry_enabled"`
}

// topicSerializationAttribute returns the schema for the serialization nested
// block. Both topic data sources share it so the two schemas can't drift.
func topicSerializationAttribute() schema.SingleNestedAttribute {
	const formats = "Valid values: avro, json, json_schema, protobuf, string, bytearray, unknown."
	const formatsMD = "Valid values: `avro`, `json`, `json_schema`, `protobuf`, `string`, `bytearray`, `unknown`."
	return schema.SingleNestedAttribute{
		Computed:            true,
		Description:         "Serialization format information for the topic, inherited from its producer.",
		MarkdownDescription: "Serialization format information for the topic, inherited from its producer (source/transform).",
		Attributes: map[string]schema.Attribute{
			"key_format": schema.StringAttribute{
				Computed:            true,
				Description:         "Serialization format for message keys. " + formats,
				MarkdownDescription: "Serialization format for message keys. " + formatsMD,
			},
			"value_format": schema.StringAttribute{
				Computed:            true,
				Description:         "Serialization format for message values. " + formats,
				MarkdownDescription: "Serialization format for message values. " + formatsMD,
			},
			"key_converter": schema.StringAttribute{
				Computed:            true,
				Description:         "Kafka Connect converter class used for message keys.",
				MarkdownDescription: "Kafka Connect converter class used for message keys.",
			},
			"value_converter": schema.StringAttribute{
				Computed:            true,
				Description:         "Kafka Connect converter class used for message values.",
				MarkdownDescription: "Kafka Connect converter class used for message values.",
			},
			"schema_registry_enabled": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether Schema Registry is used for this topic's serialization.",
				MarkdownDescription: "Whether Schema Registry is used for this topic's serialization.",
			},
		},
	}
}

// flattenTopicSerialization maps the API serialization object to its data-source
// model, returning nil (a null block) when the API omits serialization.
func flattenTopicSerialization(s *api.TopicSerialization) *TopicSerializationModel {
	if s == nil {
		return nil
	}
	return &TopicSerializationModel{
		KeyFormat:             types.StringValue(s.KeyFormat),
		ValueFormat:           types.StringValue(s.ValueFormat),
		KeyConverter:          types.StringPointerValue(s.KeyConverter),
		ValueConverter:        types.StringPointerValue(s.ValueConverter),
		SchemaRegistryEnabled: types.BoolValue(s.SchemaRegistryEnabled),
	}
}
