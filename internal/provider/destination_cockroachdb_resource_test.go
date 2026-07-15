package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationCockroachdbHostname = os.Getenv("TF_VAR_destination_cockroachdb_hostname")
var destinationCockroachdbPassword = os.Getenv("TF_VAR_destination_cockroachdb_password")

func TestAccDestinationCockroachdbResource(t *testing.T) {
	if destinationCockroachdbHostname == "" || destinationCockroachdbPassword == "" {
		t.Skip("Skipping TestAccDestinationCockroachdbResource: TF_VAR_destination_cockroachdb_hostname or TF_VAR_destination_cockroachdb_password not set")
	}

	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "destination_cockroachdb_hostname" {
	type        = string
	description = "CockroachDB hostname"
}
variable "destination_cockroachdb_password" {
	type        = string
	sensitive   = true
	description = "CockroachDB password"
}
resource "streamkap_destination_cockroachdb" "test" {
	name                 = %q
	database_hostname    = var.destination_cockroachdb_hostname
	database_port        = 26257
	database_database    = "defaultdb"
	connection_username  = "wasi"
	connection_password  = var.destination_cockroachdb_password
	table_name_prefix    = "public"
	schema_evolution     = "basic"
	insert_mode          = "insert"
	delete_enabled       = false
	primary_key_mode     = "record_key"
	tasks_max            = 5
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "database_hostname", destinationCockroachdbHostname),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "database_port", "26257"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "database_database", "defaultdb"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "connection_username", "wasi"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "connection_password", destinationCockroachdbPassword),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "table_name_prefix", "public"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "insert_mode", "insert"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "delete_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "primary_key_mode", "record_key"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "tasks_max", "5"),
					resource.TestCheckResourceAttrSet("streamkap_destination_cockroachdb.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "connector", "cockroachdb"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_destination_cockroachdb.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "destination_cockroachdb_hostname" {
	type        = string
	description = "CockroachDB hostname"
}
variable "destination_cockroachdb_password" {
	type        = string
	sensitive   = true
	description = "CockroachDB password"
}
resource "streamkap_destination_cockroachdb" "test" {
	name                 = %q
	database_hostname    = var.destination_cockroachdb_hostname
	database_port        = 26257
	database_database    = "defaultdb"
	connection_username  = "wasi"
	connection_password  = var.destination_cockroachdb_password
	table_name_prefix    = "streamkap"
	schema_evolution     = "none"
	insert_mode          = "upsert"
	delete_enabled       = true
	primary_key_mode     = "record_value"
	primary_key_fields   = "id"
	tasks_max            = 3
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "table_name_prefix", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "schema_evolution", "none"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "insert_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "delete_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "primary_key_mode", "record_value"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "primary_key_fields", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_cockroachdb.test", "tasks_max", "3"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
