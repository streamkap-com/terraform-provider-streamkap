package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Define environment variables for Databricks configuration
var destinationDatabricksConnectionUrl = os.Getenv("TF_VAR_destination_databricks_connection_url")
var destinationDatabricksToken = os.Getenv("TF_VAR_destination_databricks_token")

func TestAccDestinationDatabricksResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create and Read Testing
			{
				Config: providerConfig + `
variable "destination_databricks_connection_url" {
	type        = string
	description = "The connection url of the Databricks database"
}
variable "destination_databricks_token" {
	type        = string
	sensitive   = true
	description = "The token for the Databricks database"
}
resource "streamkap_destination_databricks" "test" {
	name                 = "test-destination-databricks"
	connection_url       = var.destination_databricks_connection_url
	databricks_token     = var.destination_databricks_token
    table_name_prefix    = "streamkap"
	ingestion_mode       = "upsert"
	partition_mode       = "by_topic"
	hard_delete          = true
	tasks_max            = 3
	schema_evolution     = "basic"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "name", "test-destination-databricks"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "connection_url", destinationDatabricksConnectionUrl),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "databricks_token", destinationDatabricksToken),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "ingestion_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "partition_mode", "by_topic"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "hard_delete", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "tasks_max", "3"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "table_name_prefix", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttrSet("streamkap_destination_databricks.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "connector", "databricks"),
				),
			},
			// Step 2: ImportState Testing
			{
				ResourceName:      "streamkap_destination_databricks.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update and Read Testing
			{
				Config: providerConfig + `
variable "destination_databricks_connection_url" {
	type        = string
	description = "The connection url of the Databricks database"
}
variable "destination_databricks_token" {
	type        = string
	sensitive   = true
	description = "The token for the Databricks database"
}
resource "streamkap_destination_databricks" "test" {
	name                 = "test-destination-databricks-updated"
	connection_url       = var.destination_databricks_connection_url
	databricks_token     = var.destination_databricks_token
	table_name_prefix    = "streamkap"
	ingestion_mode       = "append"
	partition_mode       = "by_topic"
	hard_delete          = false
	tasks_max            = 5
	schema_evolution     = "none"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "name", "test-destination-databricks-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "connection_url", destinationDatabricksConnectionUrl),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "databricks_token", destinationDatabricksToken),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "ingestion_mode", "append"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "partition_mode", "by_topic"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "hard_delete", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "tasks_max", "5"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "table_name_prefix", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "schema_evolution", "none"),
					resource.TestCheckResourceAttrSet("streamkap_destination_databricks.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "connector", "databricks"),
				),
			},
			// Delete testing is automatically handled by the test framework
		},
	})
}
