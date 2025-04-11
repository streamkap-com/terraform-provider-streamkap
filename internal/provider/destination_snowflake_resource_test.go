package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationSnowflakeURLName = os.Getenv("TF_VAR_destination_snowflake_url_name")
var destinationSnowflakePrivateKey = os.Getenv("TF_VAR_destination_snowflake_private_key")
var destinationSnowflakeKeyPassphrase = os.Getenv("TF_VAR_destination_snowflake_key_passphrase")
var destinationSnowflakePrivateKeyNoCrypt = os.Getenv("TF_VAR_destination_snowflake_private_key_nocrypt")

func TestAccDestinationSnowflakeResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create and Read testing with passphrase
			{
				Config: providerConfig + `
variable "destination_snowflake_url_name" {
	type        = string
	description = "The URL name of the Snowflake database"
}
variable "destination_snowflake_private_key" {
	type        = string
	sensitive   = true
	description = "The private key of the Snowflake database"
}
variable "destination_snowflake_key_passphrase" {
	type        = string
	sensitive   = true
	description = "The passphrase of the private key of the Snowflake database"
}
resource "streamkap_destination_snowflake" "test" {
	name                             = "test-destination-snowflake"
	snowflake_url_name               = var.destination_snowflake_url_name
	snowflake_user_name              = "STREAMKAP_USER_JUNIT"
	snowflake_private_key            = var.destination_snowflake_private_key
	snowflake_private_key_passphrase = var.destination_snowflake_key_passphrase
	sfwarehouse                      = "STREAMKAP_WH"
	snowflake_database_name          = "JUNIT"
	snowflake_schema_name            = "JUNIT"
	snowflake_role_name              = "STREAMKAP_ROLE_JUNIT"
	ingestion_mode                   = "upsert"
	hard_delete                      = true
	use_hybrid_tables                = false
	apply_dynamic_table_script       = false
	dynamic_table_target_lag         = 60
	cleanup_task_schedule            = 120
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "name", "test-destination-snowflake"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_url_name", destinationSnowflakeURLName),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_user_name", "STREAMKAP_USER_JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key", destinationSnowflakePrivateKey),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key_passphrase", destinationSnowflakeKeyPassphrase),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "sfwarehouse", "STREAMKAP_WH"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_database_name", "JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_schema_name", "JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_role_name", "STREAMKAP_ROLE_JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "ingestion_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "hard_delete", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "use_hybrid_tables", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "apply_dynamic_table_script", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "dynamic_table_target_lag", "60"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "cleanup_task_schedule", "120"),
					resource.TestCheckResourceAttrSet("streamkap_destination_snowflake.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "connector", "snowflake"),
				),
			},
			// Step 2: ImportState testing
			{
				ResourceName:      "streamkap_destination_snowflake.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update and Read testing with passphrase (change name and ingestion mode)
			{
				Config: providerConfig + `
variable "destination_snowflake_url_name" {
	type        = string
	description = "The URL name of the Snowflake database"
}
variable "destination_snowflake_private_key" {
	type        = string
	sensitive   = true
	description = "The private key of the Snowflake database"
}
variable "destination_snowflake_key_passphrase" {
	type        = string
	sensitive   = true
	description = "The passphrase of the private key of the Snowflake database"
}
resource "streamkap_destination_snowflake" "test" {
	name                             = "test-destination-snowflake-updated"
	snowflake_url_name               = var.destination_snowflake_url_name
	snowflake_user_name              = "STREAMKAP_USER_JUNIT"
	snowflake_private_key            = var.destination_snowflake_private_key
	snowflake_private_key_passphrase = var.destination_snowflake_key_passphrase
	sfwarehouse                      = "STREAMKAP_WH"
	snowflake_database_name          = "JUNIT"
	snowflake_schema_name            = "JUNIT"
	snowflake_role_name              = "STREAMKAP_ROLE_JUNIT"
	ingestion_mode                   = "append"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "name", "test-destination-snowflake-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_url_name", destinationSnowflakeURLName),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_user_name", "STREAMKAP_USER_JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key", destinationSnowflakePrivateKey),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key_passphrase", destinationSnowflakeKeyPassphrase),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "sfwarehouse", "STREAMKAP_WH"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_database_name", "JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_schema_name", "JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_role_name", "STREAMKAP_ROLE_JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "ingestion_mode", "append"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "hard_delete", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "use_hybrid_tables", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "apply_dynamic_table_script", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "dynamic_table_target_lag", "15"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "cleanup_task_schedule", "60"),
					resource.TestCheckResourceAttrSet("streamkap_destination_snowflake.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "connector", "snowflake"),
				),
			},
// 			// Step 4: Update to remove passphrase (passphrase is None)
// 			{
// 				Config: providerConfig + `
// variable "destination_snowflake_url_name" {
// 	type        = string
// 	description = "The URL name of the Snowflake database"
// }
// variable "destination_snowflake_private_key_nocrypt" {
// 	type        = string
// 	sensitive   = true
// 	description = "The private key of the Snowflake database without passphrase"
// }
// resource "streamkap_destination_snowflake" "test" {
// 	name                             = "test-destination-snowflake-no-passphrase"
// 	snowflake_url_name               = var.destination_snowflake_url_name
// 	snowflake_user_name              = "STREAMKAP_USER_JUNIT_NOCRYPT"
// 	snowflake_private_key            = var.destination_snowflake_private_key_nocrypt
// 	# snowflake_private_key_passphrase is omitted (None)
// 	sfwarehouse                      = "STREAMKAP_WH"
// 	snowflake_database_name          = "JUNIT"
// 	snowflake_schema_name            = "JUNIT"
// 	snowflake_role_name              = "STREAMKAP_ROLE_JUNIT"
// 	ingestion_mode                   = "append"
// }
// `,
// 				Check: resource.ComposeAggregateTestCheckFunc(
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "name", "test-destination-snowflake-no-passphrase"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_url_name", destinationSnowflakeURLName),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_user_name", "STREAMKAP_USER_JUNIT_NOCRYPT"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key", destinationSnowflakePrivateKeyNoCrypt),
// 					resource.TestCheckNoResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key_passphrase"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "sfwarehouse", "STREAMKAP_WH"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_database_name", "JUNIT"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_schema_name", "JUNIT"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_role_name", "STREAMKAP_ROLE_JUNIT"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "ingestion_mode", "append"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "hard_delete", "false"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "schema_evolution", "basic"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "use_hybrid_tables", "false"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "apply_dynamic_table_script", "false"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "dynamic_table_target_lag", "15"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "cleanup_task_schedule", "60"),
// 					resource.TestCheckResourceAttrSet("streamkap_destination_snowflake.test", "id"),
// 					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "connector", "snowflake"),
// 				),
// 			},
			// Step 5: Update to add passphrase back (passphrase is not None)
			{
				Config: providerConfig + `
variable "destination_snowflake_url_name" {
	type        = string
	description = "The URL name of the Snowflake database"
}
variable "destination_snowflake_private_key" {
	type        = string
	sensitive   = true
	description = "The private key of the Snowflake database"
}
variable "destination_snowflake_key_passphrase" {
	type        = string
	sensitive   = true
	description = "The passphrase of the private key of the Snowflake database"
}
resource "streamkap_destination_snowflake" "test" {
	name                             = "test-destination-snowflake-with-passphrase"
	snowflake_url_name               = var.destination_snowflake_url_name
	snowflake_user_name              = "STREAMKAP_USER_JUNIT"
	snowflake_private_key            = var.destination_snowflake_private_key
	snowflake_private_key_passphrase = var.destination_snowflake_key_passphrase
	sfwarehouse                      = "STREAMKAP_WH"
	snowflake_database_name          = "JUNIT"
	snowflake_schema_name            = "JUNIT"
	snowflake_role_name              = "STREAMKAP_ROLE_JUNIT"
	ingestion_mode                   = "append"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "name", "test-destination-snowflake-with-passphrase"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_url_name", destinationSnowflakeURLName),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_user_name", "STREAMKAP_USER_JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key", destinationSnowflakePrivateKey),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_private_key_passphrase", destinationSnowflakeKeyPassphrase),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "sfwarehouse", "STREAMKAP_WH"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_database_name", "JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_schema_name", "JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "snowflake_role_name", "STREAMKAP_ROLE_JUNIT"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "ingestion_mode", "append"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "hard_delete", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "use_hybrid_tables", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "apply_dynamic_table_script", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "dynamic_table_target_lag", "15"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "cleanup_task_schedule", "60"),
					resource.TestCheckResourceAttrSet("streamkap_destination_snowflake.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_snowflake.test", "connector", "snowflake"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
