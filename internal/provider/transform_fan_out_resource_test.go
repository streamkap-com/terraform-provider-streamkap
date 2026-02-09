package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformFanOutResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: testAccTransformFanOutResourceConfig("tf-acc-test-transform-fan-out"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "name", "tf-acc-test-transform-fan-out"),
					resource.TestCheckResourceAttrSet("streamkap_transform_fan_out.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_fan_out.test", "transform_type"),
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "transforms_language", "JavaScript"),
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "transforms_input_topic_pattern", "test-input-topic"),
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "transforms_output_topic_pattern", "test-output-topic"),
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "transforms_input_serialization_format", "Avro"),
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "transforms_output_serialization_format", "Avro"),
				),
			},
			// ImportState
			{
				ResourceName:            "streamkap_transform_fan_out.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json"},
			},
			// Update
			{
				Config: testAccTransformFanOutResourceConfigUpdated("tf-acc-test-transform-fan-out-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "name", "tf-acc-test-transform-fan-out-updated"),
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "transforms_input_serialization_format", "Json"),
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "transforms_output_serialization_format", "Json"),
				),
			},
		},
	})
}

func TestAccTransformFanOutResource_withImplementation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Create with implementation_json
			{
				Config: testAccTransformFanOutWithImplementationConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test_impl", "name", "tf-acc-test-transform-fan-out-impl"),
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test_impl", "transforms_language", "JavaScript"),
					resource.TestCheckResourceAttrSet("streamkap_transform_fan_out.test_impl", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_fan_out.test_impl", "implementation_json"),
				),
			},
			// ImportState
			{
				ResourceName:            "streamkap_transform_fan_out.test_impl",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json"},
			},
		},
	})
}

func testAccTransformFanOutResourceConfig(name string) string {
	return `
resource "streamkap_transform_fan_out" "test" {
  name                                   = "` + name + `"
  transforms_language                    = "JavaScript"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
}
`
}

func testAccTransformFanOutResourceConfigUpdated(name string) string {
	return `
resource "streamkap_transform_fan_out" "test" {
  name                                   = "` + name + `"
  transforms_language                    = "JavaScript"
  transforms_input_topic_pattern         = "test-input-topic-updated"
  transforms_output_topic_pattern        = "test-output-topic-updated"
  transforms_input_serialization_format  = "Json"
  transforms_output_serialization_format = "Json"
}
`
}

func testAccTransformFanOutWithImplementationConfig() string {
	return `
resource "streamkap_transform_fan_out" "test_impl" {
  name                                   = "tf-acc-test-transform-fan-out-impl"
  transforms_language                    = "JavaScript"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"

  implementation_json = jsonencode({
    language        = "JAVASCRIPT"
    value_transform = "function _streamkap_transform(inputObj, keyObj, topicName, timestamp, commonObject) { var outputObj = inputObj; delete outputObj['entity_type']; return outputObj; }"
    topic_transform = "function _streamkap_transform_topic(valueObject, keyObject, topic, timestamp, commonObject, valueSchema, keySchema) { return topic + '_' + valueObject.entity_type; }"
  })
}
`
}
