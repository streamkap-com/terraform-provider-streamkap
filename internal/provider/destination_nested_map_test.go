package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The two override-driven map attributes are Optional without Computed, so the
// provider must keep null (omitted), {} (explicitly cleared) and a populated
// map distinct on both the write and the read side. Any collapse shows up as
// "Provider produced inconsistent result after apply" or a non-empty plan after
// apply, both of which the test framework fails on.

func TestAccDestinationClickHouseResource_topicsConfigMapRoundTrip(t *testing.T) {
	name := acctestName(t, "map-roundtrip")
	config := func(mapBlock string) string {
		return providerConfig + fmt.Sprintf(`
variable "destination_clickhouse_hostname" { type = string }
variable "destination_clickhouse_connection_username" { type = string }
variable "destination_clickhouse_connection_password" {
	type      = string
	sensitive = true
}
resource "streamkap_destination_clickhouse" "test" {
	name                = %q
	hostname            = var.destination_clickhouse_hostname
	connection_username = var.destination_clickhouse_connection_username
	connection_password = var.destination_clickhouse_connection_password
	port                = 8123
	database            = "default"
	ssl                 = false
	ingestion_mode      = "upsert"
	%s
}
`, name, mapBlock)
	}
	addr := "streamkap_destination_clickhouse.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			{ // omitted → null
				Config: config(""),
				Check:  resource.TestCheckNoResourceAttr(addr, "topics_config_map.%"),
			},
			{ // explicit {} → empty map, not null
				Config: config("topics_config_map = {}"),
				Check:  resource.TestCheckResourceAttr(addr, "topics_config_map.%", "0"),
			},
			{ // populated
				Config: config(`topics_config_map = {
		"public.orders" = { delete_sql_execute = "ALTER TABLE orders DELETE WHERE id = {id}" }
	}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "topics_config_map.%", "1"),
					resource.TestCheckResourceAttr(addr, "topics_config_map.public.orders.delete_sql_execute", "ALTER TABLE orders DELETE WHERE id = {id}"),
				),
			},
			{ // back to {} clears the mapping on the backend
				Config: config("topics_config_map = {}"),
				Check:  resource.TestCheckResourceAttr(addr, "topics_config_map.%", "0"),
			},
			{ // back to omitted
				Config: config(""),
				Check:  resource.TestCheckNoResourceAttr(addr, "topics_config_map.%"),
			},
		},
	})
}

func TestAccDestinationSnowflakeResource_dedupeMappingRoundTrip(t *testing.T) {
	name := acctestName(t, "map-roundtrip")
	config := func(mapBlock string) string {
		return providerConfig + fmt.Sprintf(`
variable "destination_snowflake_url_name" { type = string }
variable "destination_snowflake_private_key" {
	type      = string
	sensitive = true
}
variable "destination_snowflake_key_passphrase" {
	type      = string
	sensitive = true
}
resource "streamkap_destination_snowflake" "test" {
	name                             = %q
	snowflake_url_name               = var.destination_snowflake_url_name
	snowflake_user_name              = "STREAMKAP_USER_JUNIT"
	snowflake_private_key            = var.destination_snowflake_private_key
	snowflake_private_key_passphrase = var.destination_snowflake_key_passphrase
	sfwarehouse                      = "STREAMKAP_WH"
	snowflake_database_name          = "JUNIT"
	snowflake_schema_name            = "JUNIT"
	snowflake_role_name              = "STREAMKAP_ROLE_JUNIT"
	ingestion_mode                   = "upsert"
	%s
}
`, name, mapBlock)
	}
	addr := "streamkap_destination_snowflake.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			{
				Config: config(""),
				Check:  resource.TestCheckNoResourceAttr(addr, "auto_qa_dedupe_table_mapping.%"),
			},
			{
				Config: config("auto_qa_dedupe_table_mapping = {}"),
				Check:  resource.TestCheckResourceAttr(addr, "auto_qa_dedupe_table_mapping.%", "0"),
			},
			{
				Config: config(`auto_qa_dedupe_table_mapping = { users = "JUNIT.USERS" }`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(addr, "auto_qa_dedupe_table_mapping.%", "1"),
					resource.TestCheckResourceAttr(addr, "auto_qa_dedupe_table_mapping.users", "JUNIT.USERS"),
				),
			},
			{
				Config: config("auto_qa_dedupe_table_mapping = {}"),
				Check:  resource.TestCheckResourceAttr(addr, "auto_qa_dedupe_table_mapping.%", "0"),
			},
			{
				Config: config(""),
				Check:  resource.TestCheckNoResourceAttr(addr, "auto_qa_dedupe_table_mapping.%"),
			},
		},
	})
}
