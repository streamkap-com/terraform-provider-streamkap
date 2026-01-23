package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationDb2Hostname = os.Getenv("TF_VAR_destination_db2_hostname")
var destinationDb2Username = os.Getenv("TF_VAR_destination_db2_username")
var destinationDb2Password = os.Getenv("TF_VAR_destination_db2_password")
var _ = os.Getenv("TF_VAR_destination_db2_database") // used via TF_VAR in HCL config

func TestAccDestinationDb2Resource(t *testing.T) {
	if destinationDb2Hostname == "" || destinationDb2Username == "" || destinationDb2Password == "" {
		t.Skip("Skipping TestAccDestinationDb2Resource: TF_VAR_destination_db2_hostname, TF_VAR_destination_db2_username, or TF_VAR_destination_db2_password not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_db2_hostname" {
	type        = string
	description = "DB2 hostname"
}
variable "destination_db2_username" {
	type        = string
	description = "DB2 username"
}
variable "destination_db2_password" {
	type        = string
	sensitive   = true
	description = "DB2 password"
}
variable "destination_db2_database" {
	type        = string
	description = "DB2 database name"
	default     = ""
}
resource "streamkap_destination_db2" "test" {
	name                = "tf-acc-test-destination-db2"
	database_hostname   = var.destination_db2_hostname
	database_port       = 50000
	database_database   = var.destination_db2_database
	connection_username = var.destination_db2_username
	connection_password = var.destination_db2_password
	schema_evolution    = "basic"
	insert_mode         = "insert"
	delete_enabled      = false
	primary_key_mode    = "record_key"
	tasks_max           = 1
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "name", "tf-acc-test-destination-db2"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "database_hostname", destinationDb2Hostname),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "database_port", "50000"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "connection_username", destinationDb2Username),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "insert_mode", "insert"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "delete_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "primary_key_mode", "record_key"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "tasks_max", "1"),
					resource.TestCheckResourceAttrSet("streamkap_destination_db2.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "connector", "db2"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_db2.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_db2_hostname" {
	type        = string
	description = "DB2 hostname"
}
variable "destination_db2_username" {
	type        = string
	description = "DB2 username"
}
variable "destination_db2_password" {
	type        = string
	sensitive   = true
	description = "DB2 password"
}
variable "destination_db2_database" {
	type        = string
	description = "DB2 database name"
	default     = ""
}
resource "streamkap_destination_db2" "test" {
	name                = "tf-acc-test-destination-db2-updated"
	database_hostname   = var.destination_db2_hostname
	database_port       = 50000
	database_database   = var.destination_db2_database
	connection_username = var.destination_db2_username
	connection_password = var.destination_db2_password
	schema_evolution    = "none"
	insert_mode         = "upsert"
	delete_enabled      = true
	primary_key_mode    = "record_value"
	tasks_max           = 2
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "name", "tf-acc-test-destination-db2-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "schema_evolution", "none"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "insert_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "delete_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "primary_key_mode", "record_value"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "tasks_max", "2"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
