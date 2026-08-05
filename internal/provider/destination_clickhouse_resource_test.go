package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDestinationClickHouseResource(t *testing.T) {
	// Define environment variables for ClickHouse configuration
	var destinationClickHouseHostname = os.Getenv("TF_VAR_destination_clickhouse_hostname")
	var destinationClickHouseUsername = os.Getenv("TF_VAR_destination_clickhouse_connection_username")
	var destinationClickHousePassword = os.Getenv("TF_VAR_destination_clickhouse_connection_password")
	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create and Read Testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "destination_clickhouse_hostname" {
	type        = string
	description = "The hostname of the ClickHouse database"
}
variable "destination_clickhouse_connection_username" {
	type        = string
	description = "The username for the ClickHouse database"
}
variable "destination_clickhouse_connection_password" {
	type        = string
	sensitive   = true
	description = "The password for the ClickHouse database"
}
resource "streamkap_destination_clickhouse" "test" {
	name                 = %q
	hostname             = var.destination_clickhouse_hostname
	connection_username  = var.destination_clickhouse_connection_username
	connection_password  = var.destination_clickhouse_connection_password
	ingestion_mode       = "upsert"
	hard_delete          = true
	tasks_max            = 3
	port                 = 8123
	database             = "default"
	ssl                  = false
	schema_evolution     = "basic"
	transforms_mask_field_fields_include_list       = "public.users.email"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "REDACT"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_replace_null_with_default = false
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "transforms_mask_field_fields_include_list", "public.users.email"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "transforms_mask_field_mask_function", "REDACT"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "transforms_mask_field_mask_fixed_value", "***"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "transforms_mask_field_replace_null_with_default", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "hostname", destinationClickHouseHostname),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connection_username", destinationClickHouseUsername),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connection_password", destinationClickHousePassword),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "ingestion_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "hard_delete", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "tasks_max", "3"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "port", "8123"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "database", "default"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "ssl", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttrSet("streamkap_destination_clickhouse.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connector", "clickhouse"),
				),
			},
			// Step 2: ImportState Testing
			{
				ResourceName:            "streamkap_destination_clickhouse.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Step 3: Update and Read Testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "destination_clickhouse_hostname" {
	type        = string
	description = "The hostname of the ClickHouse database"
}
variable "destination_clickhouse_connection_username" {
	type        = string
	description = "The username for the ClickHouse database"
}
variable "destination_clickhouse_connection_password" {
	type        = string
	sensitive   = true
	description = "The password for the ClickHouse database"
}
resource "streamkap_destination_clickhouse" "test" {
	name                 = %q
	hostname             = var.destination_clickhouse_hostname
	connection_username  = var.destination_clickhouse_connection_username
	connection_password  = var.destination_clickhouse_connection_password
	ingestion_mode       = "append"
	hard_delete          = false
	tasks_max            = 5
	port                 = 8123
	database             = "default"
	ssl                  = false
	schema_evolution     = "none"
	transforms_mask_field_fields_include_list       = "public.users.email,public.users.phone"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "FIXED"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_mask_fixed_value          = "MASKED"
	transforms_mask_field_replace_null_with_default = true
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "transforms_mask_field_fields_include_list", "public.users.email,public.users.phone"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "transforms_mask_field_mask_function", "FIXED"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "transforms_mask_field_mask_fixed_value", "MASKED"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "transforms_mask_field_replace_null_with_default", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "hostname", destinationClickHouseHostname),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connection_username", destinationClickHouseUsername),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connection_password", destinationClickHousePassword),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "ingestion_mode", "append"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "hard_delete", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "tasks_max", "5"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "port", "8123"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "database", "default"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "ssl", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "schema_evolution", "none"),
					resource.TestCheckResourceAttrSet("streamkap_destination_clickhouse.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connector", "clickhouse"),
				),
			},
			// Delete testing is automatically handled by the test framework
		},
	})
}
