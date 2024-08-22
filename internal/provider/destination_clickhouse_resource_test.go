package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationClickhouseHostname = os.Getenv("TF_VAR_destination_clickhouse_hostname")
var destinationClickhouseConnectionUsername = os.Getenv("TF_VAR_destination_clickhouse_connection_username")
var destinationClickhouseConnectionPassword = os.Getenv("TF_VAR_destination_clickhouse_connection_password")

func TestAccDestinationClickHouseResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_clickhouse_hostname" {
	type        = string
	description = "The hostname of the Clickhouse server"
}

variable "destination_clickhouse_connection_username" {
	type        = string
	description = "The username to connect to the Clickhouse server"
}

variable "destination_clickhouse_connection_password" {
	type        = string
	description = "The password to connect to the Clickhouse server"
}

resource "streamkap_destination_clickhouse" "test" {
	name                = "test-destination-clickhouse"
	ingestion_mode      = "append"
	tasks_max           = 5
	hostname            = var.destination_clickhouse_hostname
	connection_username = var.destination_clickhouse_connection_username
	connection_password = var.destination_clickhouse_connection_password
	port                = 8443
	database            = "demo"
	ssl                 = true
	topics_config_map = {
		"public.users" = {
			delete_sql_execute = "SELECT 1;"
		}
	}
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "name", "test-destination-clickhouse"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "ingestion_mode", "append"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "tasks_max", "5"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "hostname", destinationClickhouseHostname),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connection_username", destinationClickhouseConnectionUsername),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connection_password", destinationClickhouseConnectionPassword),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "port", "8443"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "database", "demo"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "ssl", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "topics_config_map.public.users.delete_sql_execute", "SELECT 1;"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_clickhouse.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_clickhouse_hostname" {
	type        = string
	description = "The hostname of the Clickhouse server"
}

variable "destination_clickhouse_connection_username" {
	type        = string
	description = "The username to connect to the Clickhouse server"
}

variable "destination_clickhouse_connection_password" {
	type        = string
	description = "The password to connect to the Clickhouse server"
}

resource "streamkap_destination_clickhouse" "test" {
	name                = "test-destination-clickhouse-updated"
	ingestion_mode      = "append"
	tasks_max           = 1
	hostname            = var.destination_clickhouse_hostname
	connection_username = var.destination_clickhouse_connection_username
	connection_password = var.destination_clickhouse_connection_password
	port                = 8443
	database            = "demo"
	ssl                 = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "name", "test-destination-clickhouse-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "ingestion_mode", "append"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "tasks_max", "1"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "hostname", destinationClickhouseHostname),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connection_username", destinationClickhouseConnectionUsername),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "connection_password", destinationClickhouseConnectionPassword),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "port", "8443"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "database", "demo"),
					resource.TestCheckResourceAttr("streamkap_destination_clickhouse.test", "ssl", "true"),
					resource.TestCheckNoResourceAttr("streamkap_destination_clickhouse.test", "topics_config_map"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
