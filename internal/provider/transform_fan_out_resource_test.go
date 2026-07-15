package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformFanOutResource_basic(t *testing.T) {
	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: testAccTransformFanOutResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "name", name),
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
				ImportStateVerifyIgnore: []string{"implementation_json", "deploy", "replay_window"},
			},
			// Update
			{
				Config: testAccTransformFanOutResourceConfigUpdated(nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "transforms_input_serialization_format", "Json"),
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test", "transforms_output_serialization_format", "Json"),
				),
			},
		},
	})
}

func TestAccTransformFanOutResource_withImplementation(t *testing.T) {
	name := acctestName(t, "impl")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Create with implementation_json
			{
				Config: testAccTransformFanOutWithImplementationConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_fan_out.test_impl", "name", name),
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
				ImportStateVerifyIgnore: []string{"implementation_json", "deploy", "replay_window"},
			},
		},
	})
}

func testAccTransformFanOutResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_fan_out" "test" {
  name                                   = %q
  transforms_language                    = "JavaScript"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
}
`, name)
}

func testAccTransformFanOutResourceConfigUpdated(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_fan_out" "test" {
  name                                   = %q
  transforms_language                    = "JavaScript"
  transforms_input_topic_pattern         = "test-input-topic-updated"
  transforms_output_topic_pattern        = "test-output-topic-updated"
  transforms_input_serialization_format  = "Json"
  transforms_output_serialization_format = "Json"
}
`, name)
}

func testAccTransformFanOutWithImplementationConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_fan_out" "test_impl" {
  name                                   = %q
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
`, name)
}
