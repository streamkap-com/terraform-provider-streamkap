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

func TestAccTransformEnrichResource_withImplementation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformEnrichWithImplementationConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_enrich.test_impl", "name", "tf-acc-test-transform-enrich-impl"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich.test_impl", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich.test_impl", "implementation_json"),
				),
			},
			{
				ResourceName:            "streamkap_transform_enrich.test_impl",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json"},
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

func testAccTransformEnrichWithImplementationConfig() string {
	return `
resource "streamkap_transform_enrich" "test_impl" {
  name                                   = "tf-acc-test-transform-enrich-impl"
  transforms_input_topic_pattern         = "test-input-.*"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"

  implementation_json = jsonencode({
    mainTable = {
      name              = "orders"
      topicMatcherRegex = ".*orders$"
      createTableSQL    = "CREATE TABLE orders (order_id STRING PRIMARY KEY, location_id STRING)"
    }
    lookupTable = {
      name              = "locations"
      topicMatcherRegex = ".*locations$"
      createTableSQL    = "CREATE TABLE locations (location_id STRING PRIMARY KEY, city STRING)"
    }
    enrichSQL = "SELECT o.order_id, l.city FROM orders o JOIN locations l ON o.location_id = l.location_id"
  })
}
`
}
