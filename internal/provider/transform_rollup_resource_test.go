package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformRollupResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: testAccTransformRollupResourceConfig("tf-acc-test-transform-rollup"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_rollup.test", "name", "tf-acc-test-transform-rollup"),
					resource.TestCheckResourceAttrSet("streamkap_transform_rollup.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_rollup.test", "transform_type"),
					resource.TestCheckResourceAttr("streamkap_transform_rollup.test", "transforms_language", "SQL"),
					resource.TestCheckResourceAttr("streamkap_transform_rollup.test", "transforms_input_topic_pattern", "test-input-topic"),
					resource.TestCheckResourceAttr("streamkap_transform_rollup.test", "transforms_output_topic_pattern", "test-output-topic"),
					resource.TestCheckResourceAttr("streamkap_transform_rollup.test", "transforms_input_serialization_format", "Avro"),
					resource.TestCheckResourceAttr("streamkap_transform_rollup.test", "transforms_output_serialization_format", "Avro"),
				),
			},
			// ImportState
			{
				ResourceName:            "streamkap_transform_rollup.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json"},
			},
			// Update
			{
				Config: testAccTransformRollupResourceConfigUpdated("tf-acc-test-transform-rollup-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_rollup.test", "name", "tf-acc-test-transform-rollup-updated"),
					resource.TestCheckResourceAttr("streamkap_transform_rollup.test", "transforms_input_serialization_format", "Json"),
					resource.TestCheckResourceAttr("streamkap_transform_rollup.test", "transforms_output_serialization_format", "Json"),
				),
			},
		},
	})
}

func testAccTransformRollupResourceConfig(name string) string {
	return `
resource "streamkap_transform_rollup" "test" {
  name                                   = "` + name + `"
  transforms_language                    = "SQL"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
}
`
}

func testAccTransformRollupResourceConfigUpdated(name string) string {
	return `
resource "streamkap_transform_rollup" "test" {
  name                                   = "` + name + `"
  transforms_language                    = "SQL"
  transforms_input_topic_pattern         = "test-input-topic-updated"
  transforms_output_topic_pattern        = "test-output-topic-updated"
  transforms_input_serialization_format  = "Json"
  transforms_output_serialization_format = "Json"
}
`
}
