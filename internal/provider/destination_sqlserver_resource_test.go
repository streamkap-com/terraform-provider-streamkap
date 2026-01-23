package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationSqlserverHostname = os.Getenv("TF_VAR_destination_sqlserver_hostname")
var destinationSqlserverUsername = os.Getenv("TF_VAR_destination_sqlserver_username")
var destinationSqlserverPassword = os.Getenv("TF_VAR_destination_sqlserver_password")
var _ = os.Getenv("TF_VAR_destination_sqlserver_database") // used via TF_VAR in HCL config

func TestAccDestinationSqlserverResource(t *testing.T) {
	if destinationSqlserverHostname == "" || destinationSqlserverUsername == "" || destinationSqlserverPassword == "" {
		t.Skip("Skipping TestAccDestinationSqlserverResource: TF_VAR_destination_sqlserver_hostname, TF_VAR_destination_sqlserver_username, or TF_VAR_destination_sqlserver_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_sqlserver_hostname" {
	type        = string
	description = "SQL Server hostname"
}
variable "destination_sqlserver_username" {
	type        = string
	description = "SQL Server username"
}
variable "destination_sqlserver_password" {
	type        = string
	sensitive   = true
	description = "SQL Server password"
}
variable "destination_sqlserver_database" {
	type        = string
	description = "SQL Server database name"
	default     = ""
}
resource "streamkap_destination_sqlserver" "test" {
	name                = "tf-acc-test-destination-sqlserver"
	database_hostname   = var.destination_sqlserver_hostname
	database_port       = 1433
	database_database   = var.destination_sqlserver_database
	connection_username = var.destination_sqlserver_username
	connection_password = var.destination_sqlserver_password
	table_name_prefix   = "dbo"
	schema_evolution    = "basic"
	insert_mode         = "insert"
	delete_enabled      = false
	primary_key_mode    = "record_key"
	tasks_max           = 5
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "name", "tf-acc-test-destination-sqlserver"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "database_hostname", destinationSqlserverHostname),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "database_port", "1433"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "connection_username", destinationSqlserverUsername),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "table_name_prefix", "dbo"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "insert_mode", "insert"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "delete_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "primary_key_mode", "record_key"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "tasks_max", "5"),
					resource.TestCheckResourceAttrSet("streamkap_destination_sqlserver.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "connector", "sqlserver"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_sqlserver.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_sqlserver_hostname" {
	type        = string
	description = "SQL Server hostname"
}
variable "destination_sqlserver_username" {
	type        = string
	description = "SQL Server username"
}
variable "destination_sqlserver_password" {
	type        = string
	sensitive   = true
	description = "SQL Server password"
}
variable "destination_sqlserver_database" {
	type        = string
	description = "SQL Server database name"
	default     = ""
}
resource "streamkap_destination_sqlserver" "test" {
	name                = "tf-acc-test-destination-sqlserver-updated"
	database_hostname   = var.destination_sqlserver_hostname
	database_port       = 1433
	database_database   = var.destination_sqlserver_database
	connection_username = var.destination_sqlserver_username
	connection_password = var.destination_sqlserver_password
	table_name_prefix   = "streamkap"
	schema_evolution    = "none"
	insert_mode         = "upsert"
	delete_enabled      = true
	primary_key_mode    = "record_value"
	tasks_max           = 10
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "name", "tf-acc-test-destination-sqlserver-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "table_name_prefix", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "schema_evolution", "none"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "insert_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "delete_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "primary_key_mode", "record_value"),
					resource.TestCheckResourceAttr("streamkap_destination_sqlserver.test", "tasks_max", "10"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
