package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformMapFilterResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformMapFilterResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "name", "tf-acc-test-transform-map-filter"),
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "transforms_input_serialization_format", "AVRO"),
					resource.TestCheckResourceAttr("streamkap_transform_map_filter.test", "transforms_output_serialization_format", "AVRO"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_map_filter.test", "transform_type"),
				),
			},
			{
				ResourceName:      "streamkap_transform_map_filter.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccTransformMapFilterResourceConfig() string {
	return `
resource "streamkap_transform_map_filter" "test" {
  name                                   = "tf-acc-test-transform-map-filter"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "AVRO"
  transforms_output_serialization_format = "AVRO"
  transforms_language                    = "PYTHON"
}
`
}
