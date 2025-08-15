package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Define environment variables for Postgresql configuration
var destinationPostgresqlHostname = os.Getenv("TF_VAR_destination_postgresql_hostname")
var destinationPostgresqlPassword = os.Getenv("TF_VAR_destination_postgresql_password")

func TestAccDestinationPostgresqlResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create and Read Testing
			{
				Config: providerConfig + `
variable "destination_postgresql_hostname" {
	type        = string
	description = "The hostname of the PostgreSQL database"
}
variable "destination_postgresql_password" {
	type        = string
	sensitive   = true
	description = "The password of the PostgreSQL database"
}

resource "streamkap_destination_postgresql" "test" {
	name                 = "test-destination-postgresql"
	database_hostname    = var.destination_postgresql_hostname
	database_port        = 5432
	database_dbname      = "postgres"
	database_username    = "postgresql"
	database_password    = var.destination_postgresql_password
	database_schema_name = "streamkap"
	schema_evolution     = "basic"
	insert_mode          = "insert"
	hard_delete          = false
	ssh_enabled          = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "name", "test-destination-postgresql"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "database_hostname", destinationPostgresqlHostname),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "database_port", "5432"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "database_dbname", "postgres"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "database_username", "postgresql"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "database_password", destinationPostgresqlPassword),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "database_schema_name", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "insert_mode", "insert"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "hard_delete", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "ssh_enabled", "false"),

					resource.TestCheckResourceAttrSet("streamkap_destination_postgresql.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "connector", "postgresql"),
				),
			},
			// Step 2: ImportState Testing
			{
				ResourceName:      "streamkap_destination_postgresql.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update and Read Testing
			{
				Config: providerConfig + `
variable "destination_postgresql_hostname" {
	type        = string
	description = "The hostname of the PostgreSQL database"
}
variable "destination_postgresql_password" {
	type        = string
	sensitive   = true
	description = "The password of the PostgreSQL database"
}

resource "streamkap_destination_postgresql" "test" {
	name                 = "test-destination-postgresql-updated"
	database_hostname    = var.destination_postgresql_hostname
	database_port        = 5432
	database_dbname      = "postgres"
	database_username    = "postgresql"
	database_password    = var.destination_postgresql_password
	database_schema_name = "streamkap"
	schema_evolution     = "none"
	insert_mode          = "upsert"
	hard_delete          = true
	ssh_enabled          = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "name", "test-destination-postgresql-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "database_hostname", destinationPostgresqlHostname),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "database_port", "5432"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "database_dbname", "postgres"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "database_username", "postgresql"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "database_password", destinationPostgresqlPassword),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "database_schema_name", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "schema_evolution", "none"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "insert_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "hard_delete", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "ssh_enabled", "false"),

					resource.TestCheckResourceAttrSet("streamkap_destination_postgresql.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_postgresql.test", "connector", "postgresql"),
				),
			},
			// Delete testing is automatically handled by the test framework
		},
	})
}
