package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformEnrichResource_basic(t *testing.T) {
	name := acctestName(t, "main")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformEnrichResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_enrich.test", "name", name),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich.test", "transform_type"),
				),
			},
			{
				ResourceName:            "streamkap_transform_enrich.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json", "deploy", "replay_window"},
			},
		},
	})
}

func TestAccTransformEnrichResource_withImplementation(t *testing.T) {
	name := acctestName(t, "impl")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformEnrichWithImplementationConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_enrich.test_impl", "name", name),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich.test_impl", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_enrich.test_impl", "implementation_json"),
				),
			},
			{
				ResourceName:            "streamkap_transform_enrich.test_impl",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json", "deploy", "replay_window"},
			},
		},
	})
}

func testAccTransformEnrichResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_enrich" "test" {
  name                                   = %q
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
}
`, name)
}

func testAccTransformEnrichWithImplementationConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_enrich" "test_impl" {
  name                                   = %q
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
`, name)
}
