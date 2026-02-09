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

func TestAccTransformRollupResource_withImplementation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			// Create with implementation_json
			{
				Config: testAccTransformRollupWithImplementationConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_rollup.test_impl", "name", "tf-acc-test-transform-rollup-impl"),
					resource.TestCheckResourceAttr("streamkap_transform_rollup.test_impl", "transforms_language", "SQL"),
					resource.TestCheckResourceAttrSet("streamkap_transform_rollup.test_impl", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_rollup.test_impl", "implementation_json"),
				),
			},
			// ImportState
			{
				ResourceName:            "streamkap_transform_rollup.test_impl",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json"},
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

func testAccTransformRollupWithImplementationConfig() string {
	return `
resource "streamkap_transform_rollup" "test_impl" {
  name                                   = "tf-acc-test-transform-rollup-impl"
  transforms_language                    = "SQL"
  transforms_input_topic_pattern         = "test-input-.*"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"

  implementation_json = jsonencode({
    language = "SQL"
    inputTables = [
      {
        name              = "orders"
        topicMatcherRegex = ".*orders$$"
        createTableSQL    = "CREATE TABLE orders (product_id STRING, quantity INT, amount DECIMAL(10,2))"
      }
    ]
    rollupSQL  = "SELECT product_id, SUM(quantity) as total_quantity, SUM(amount) as total_amount, COUNT(*) as order_count FROM orders GROUP BY product_id"
    keyFields  = ["product_id"]
  })
}
`
}
