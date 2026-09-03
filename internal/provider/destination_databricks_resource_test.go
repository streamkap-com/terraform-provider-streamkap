package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDestinationDatabricksResource(t *testing.T) {
	// Define environment variables for Databricks configuration
	var destinationDatabricksConnectionUrl = os.Getenv("TF_VAR_destination_databricks_connection_url")
	var destinationDatabricksToken = os.Getenv("TF_VAR_destination_databricks_token")
	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create and Read Testing
			{
				Config: providerConfig + fmt.Sprintf(`
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
	name                 = %q
	connection_url       = var.destination_databricks_connection_url
	databricks_token     = var.destination_databricks_token
    table_name_prefix    = "streamkap"
	ingestion_mode       = "upsert"
	partition_mode       = "by_topic"
	hard_delete          = true
	tasks_max            = 3
	schema_evolution     = "basic"
	databricks_stage_path = "/Volumes/main/streamkap/stage/"
	transforms_mask_field_fields_include_list       = "public.users.email"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "REDACT"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_replace_null_with_default = false
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "transforms_mask_field_fields_include_list", "public.users.email"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "transforms_mask_field_mask_function", "REDACT"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "transforms_mask_field_mask_fixed_value", "***"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "transforms_mask_field_replace_null_with_default", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "connection_url", destinationDatabricksConnectionUrl),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "databricks_token", destinationDatabricksToken),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "ingestion_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "partition_mode", "by_topic"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "hard_delete", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "tasks_max", "3"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "table_name_prefix", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "databricks_stage_path", "/Volumes/main/streamkap/stage/"),
					resource.TestCheckResourceAttrSet("streamkap_destination_databricks.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "connector", "databricks"),
				),
			},
			// Step 2: ImportState Testing
			{
				ResourceName:            "streamkap_destination_databricks.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Step 3: Update and Read Testing
			{
				Config: providerConfig + fmt.Sprintf(`
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
	name                 = %q
	connection_url       = var.destination_databricks_connection_url
	databricks_token     = var.destination_databricks_token
	table_name_prefix    = "streamkap"
	ingestion_mode       = "append"
	partition_mode       = "by_topic"
	hard_delete          = false
	tasks_max            = 5
	schema_evolution     = "none"
	databricks_stage_path = ""
	transforms_mask_field_fields_include_list       = "public.users.email,public.users.phone"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "FIXED"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_mask_fixed_value          = "MASKED"
	transforms_mask_field_replace_null_with_default = true
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "transforms_mask_field_fields_include_list", "public.users.email,public.users.phone"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "transforms_mask_field_mask_function", "FIXED"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "transforms_mask_field_mask_fixed_value", "MASKED"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "transforms_mask_field_replace_null_with_default", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "connection_url", destinationDatabricksConnectionUrl),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "databricks_token", destinationDatabricksToken),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "ingestion_mode", "append"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "partition_mode", "by_topic"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "hard_delete", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "tasks_max", "5"),
					resource.TestCheckResourceAttr("streamkap_destination_databricks.test", "databricks_stage_path", ""),
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
