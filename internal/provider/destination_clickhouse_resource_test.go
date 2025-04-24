package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Define environment variables for ClickHouse configuration
var destinationClickHouseHostname = os.Getenv("TF_VAR_destination_clickhouse_hostname")
var destinationClickHouseUsername = os.Getenv("TF_VAR_destination_clickhouse_connection_username")
var destinationClickHousePassword = os.Getenv("TF_VAR_destination_clickhouse_connection_password")

func TestAccDestinationClickHouseResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create and Read Testing
			{
				Config: providerConfig + `
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
	name                 = "test-destination-clickhouse"
	hostname             = var.destination_clickhouse_hostname
	connection_username  = var.destination_clickhouse_connection_username
	connection_password  = var.destination_clickhouse_connection_password
	ingestion_mode       = "upsert"
	hard_delete          = true
	tasks_max            = 3
	port                 = 8443
	database             = "demo"
	ssl                  = true
	topics_config_map = {
		"streamkap.customer" = {
			delete_sql_execute = "DELETE FROM table1 WHERE id = ?"
		}
		"streamkap.customer2" = {
			delete_sql_execute = "DELETE FROM table2 WHERE id = ?"
		}
	}
	schema_evolution     = "basic"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "name", "test-destination-clickhouse"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "hostname", destinationClickHouseHostname),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connection_username", destinationClickHouseUsername),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connection_password", destinationClickHousePassword),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "ingestion_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "hard_delete", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "tasks_max", "3"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "port", "8443"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "database", "demo"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "ssl", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "topics_config_map.streamkap.customer.delete_sql_execute", "DELETE FROM table1 WHERE id = ?"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "topics_config_map.streamkap.customer2.delete_sql_execute", "DELETE FROM table2 WHERE id = ?"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttrSet("streamkap_destination_clickhouse.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connector", "clickhouse"),
				),
			},
			// Step 2: ImportState Testing
			{
				ResourceName:      "streamkap_destination_clickhouse.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update and Read Testing
			{
				Config: providerConfig + `
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
	name                 = "test-destination-clickhouse-updated"
	hostname             = var.destination_clickhouse_hostname
	connection_username  = var.destination_clickhouse_connection_username
	connection_password  = var.destination_clickhouse_connection_password
	ingestion_mode       = "append"
	hard_delete          = false
	tasks_max            = 5
	port                 = 8443
	database             = "demo"
	ssl                  = false
	topics_config_map = {
		"streamkap.customer" = {
			delete_sql_execute = "DELETE FROM table1 WHERE id = ? AND name = ?"
		}
		"topic3" = {
			delete_sql_execute = "DELETE FROM table3 WHERE id = ?"
		}
	}
	schema_evolution     = "none"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "name", "test-destination-clickhouse-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "hostname", destinationClickHouseHostname),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connection_username", destinationClickHouseUsername),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connection_password", destinationClickHousePassword),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "ingestion_mode", "append"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "hard_delete", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "tasks_max", "5"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "port", "8443"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "database", "demo"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "ssl", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "topics_config_map.streamkap.customer.delete_sql_execute", "DELETE FROM table1 WHERE id = ? AND name = ?"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "topics_config_map.topic3.delete_sql_execute", "DELETE FROM table3 WHERE id = ?"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "schema_evolution", "none"),
					resource.TestCheckResourceAttrSet("streamkap_destination_clickhouse.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connector", "clickhouse"),
				),
			},
			// Delete testing is automatically handled by the test framework
		},
	})
}
