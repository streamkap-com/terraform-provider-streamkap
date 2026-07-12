package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformSqlJoinResource_basic(t *testing.T) {
	name := acctestName(t, "main")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformSqlJoinResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_sql_join.test", "name", name),
					resource.TestCheckResourceAttrSet("streamkap_transform_sql_join.test", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_sql_join.test", "transform_type"),
				),
			},
			{
				ResourceName:            "streamkap_transform_sql_join.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json", "deploy", "replay_window"},
			},
		},
	})
}

func TestAccTransformSqlJoinResource_withImplementation(t *testing.T) {
	name := acctestName(t, "impl")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTransformDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccTransformSqlJoinWithImplementationConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_transform_sql_join.test_impl", "name", name),
					resource.TestCheckResourceAttrSet("streamkap_transform_sql_join.test_impl", "id"),
					resource.TestCheckResourceAttrSet("streamkap_transform_sql_join.test_impl", "implementation_json"),
				),
			},
			{
				ResourceName:            "streamkap_transform_sql_join.test_impl",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"implementation_json", "deploy", "replay_window"},
			},
		},
	})
}

func testAccTransformSqlJoinResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_sql_join" "test" {
  name                                   = %q
  transforms_input_topic_pattern         = "test-input-topic"
  transforms_output_topic_pattern        = "test-output-topic"
  transforms_input_serialization_format  = "Avro"
  transforms_output_serialization_format = "Avro"
}
`, name)
}

func testAccTransformSqlJoinWithImplementationConfig(name string) string {
	return fmt.Sprintf(`
resource "streamkap_transform_sql_join" "test_impl" {
  name                                   = %q
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
`, name)
}
