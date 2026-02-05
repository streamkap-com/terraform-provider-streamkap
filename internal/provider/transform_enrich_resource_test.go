package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformEnrichResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformEnrichResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_enrich.test", "name", "tf-acc-test-transform-enrich"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich.test", "transform_type"),
				),
			},
			{
				ResourceName:      "streamkap_transform_enrich.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccTransformEnrichResourceConfig() string {
	return `
resource "streamkap_transform_enrich" "test" {
  name                                   = "tf-acc-test-transform-enrich"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
}
`
}
