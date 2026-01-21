package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformEnrichAsyncResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read
			{
				Config: testAccTransformEnrichAsyncResourceConfig("tf-acc-test-transform-enrich-async"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "name", "tf-acc-test-transform-enrich-async"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich_async.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich_async.test", "transform_type"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_language", "JavaScript"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_async_timeout_ms", "1000"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_async_capacity", "10"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_input_topic_pattern", "test-input-topic"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_output_topic_pattern", "test-output-topic"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_input_serialization_format", "Avro"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_output_serialization_format", "Avro"),
				),
			},
			// ImportState
			{
				ResourceName:      "streamkap_transform_enrich_async.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update
			{
				Config: testAccTransformEnrichAsyncResourceConfigUpdated("tf-acc-test-transform-enrich-async-updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "name", "tf-acc-test-transform-enrich-async-updated"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_language", "Python"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_async_timeout_ms", "2000"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_async_capacity", "20"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_input_serialization_format", "Json"),
					resource.TestCheckResourceAttr("streamkap_transform_enrich_async.test", "transforms_output_serialization_format", "Json"),
				),
			},
		},
	})
}

func testAccTransformEnrichAsyncResourceConfig(name string) string {
	return `
resource "streamkap_transform_enrich_async" "test" {
  name                                   = "` + name + `"
  transforms_language                    = "JavaScript"
  transforms_async_timeout_ms            = 1000
  transforms_async_capacity              = 10
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
}
`
}

func testAccTransformEnrichAsyncResourceConfigUpdated(name string) string {
	return `
resource "streamkap_transform_enrich_async" "test" {
  name                                   = "` + name + `"
  transforms_language                    = "Python"
  transforms_async_timeout_ms            = 2000
  transforms_async_capacity              = 20
  transforms_input_topic_pattern         = "test-input-topic-updated"
  transforms_output_topic_pattern        = "test-output-topic-updated"
  transforms_input_serialization_format  = "Json"
  transforms_output_serialization_format = "Json"
}
`
}
