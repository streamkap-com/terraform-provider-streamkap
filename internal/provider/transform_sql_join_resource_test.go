package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformSqlJoinResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformSqlJoinResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_sql_join.test", "name", "tf-acc-test-transform-sql-join"),
					resource.TestCheckResourceAttrSet("streamkap_transform_sql_join.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_sql_join.test", "transform_type"),
				),
			},
			{
				ResourceName:            "streamkap_transform_sql_join.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json"},
			},
		},
	})
}

func TestAccTransformSqlJoinResource_withImplementation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformSqlJoinWithImplementationConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_sql_join.test_impl", "name", "tf-acc-test-transform-sql-join-impl"),
					resource.TestCheckResourceAttrSet("streamkap_transform_sql_join.test_impl", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_sql_join.test_impl", "implementation_json"),
				),
			},
			{
				ResourceName:            "streamkap_transform_sql_join.test_impl",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json"},
			},
		},
	})
}

func testAccTransformSqlJoinResourceConfig() string {
	return `
resource "streamkap_transform_sql_join" "test" {
  name                                   = "tf-acc-test-transform-sql-join"
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
}
`
}

func testAccTransformSqlJoinWithImplementationConfig() string {
	return `
resource "streamkap_transform_sql_join" "test_impl" {
  name                                   = "tf-acc-test-transform-sql-join-impl"
  transforms_input_topic_pattern         = "test-input-.*"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"

  implementation_json = jsonencode({
    inputTables = [
      {
        name              = "orders"
        topicMatcherRegex = ".*orders$"
        createTableSQL    = "CREATE TABLE orders (order_id STRING PRIMARY KEY, amount DECIMAL)"
      },
      {
        name              = "customers"
        topicMatcherRegex = ".*customers$"
        createTableSQL    = "CREATE TABLE customers (customer_id STRING PRIMARY KEY, name STRING)"
      }
    ]
    joinSQL    = "SELECT o.order_id, c.name FROM orders o JOIN customers c ON o.customer_id = c.customer_id"
    keyFields  = ["order_id"]
    stateTtlMs = "86400000"
  })
}
`
}
