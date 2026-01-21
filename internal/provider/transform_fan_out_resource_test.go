package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformFanOutResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
				ResourceName:      "streamkap_transform_fan_out.test",
				ImportState:       true,
				ImportStateVerify: true,
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
