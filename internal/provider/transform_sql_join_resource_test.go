package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTransformSqlJoinResource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
				ResourceName:      "streamkap_transform_sql_join.test",
				ImportState:       true,
				ImportStateVerify: true,
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
  transforms_input_serialization_format  = "AVRO"
  transforms_output_serialization_format = "AVRO"
}
`
}
