package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationOracleHostname = os.Getenv("TF_VAR_destination_oracle_hostname")
var destinationOracleUsername = os.Getenv("TF_VAR_destination_oracle_username")
var destinationOraclePassword = os.Getenv("TF_VAR_destination_oracle_password")
var _ = os.Getenv("TF_VAR_destination_oracle_database") // used via TF_VAR in HCL config

func TestAccDestinationOracleResource(t *testing.T) {
	if destinationOracleHostname == "" || destinationOracleUsername == "" || destinationOraclePassword == "" {
		t.Skip("Skipping TestAccDestinationOracleResource: TF_VAR_destination_oracle_hostname, TF_VAR_destination_oracle_username, or TF_VAR_destination_oracle_password not set")
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
variable "destination_oracle_hostname" {
	type        = string
	description = "Oracle hostname"
}
variable "destination_oracle_username" {
	type        = string
	description = "Oracle username"
}
variable "destination_oracle_password" {
	type        = string
	sensitive   = true
	description = "Oracle password"
}
variable "destination_oracle_database" {
	type        = string
	description = "Oracle database name"
	default     = ""
}
resource "streamkap_destination_oracle" "test" {
	name                = %q
	database_hostname   = var.destination_oracle_hostname
	database_port       = 1521
	database_database   = var.destination_oracle_database
	connection_username = var.destination_oracle_username
	connection_password = var.destination_oracle_password
	schema_evolution    = "basic"
	insert_mode         = "insert"
	delete_enabled      = false
	primary_key_mode    = "record_key"
	tasks_max           = 5
	transforms_mask_field_fields_include_list       = "public.users.email"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "REDACT"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_replace_null_with_default = false
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "transforms_mask_field_fields_include_list", "public.users.email"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "transforms_mask_field_mask_function", "REDACT"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "transforms_mask_field_mask_fixed_value", "***"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "transforms_mask_field_replace_null_with_default", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "database_hostname", destinationOracleHostname),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "database_port", "1521"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "connection_username", destinationOracleUsername),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "insert_mode", "insert"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "delete_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "primary_key_mode", "record_key"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "tasks_max", "5"),
					resource.TestCheckResourceAttrSet("streamkap_destination_oracle.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "connector", "oracle"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_destination_oracle.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "destination_oracle_hostname" {
	type        = string
	description = "Oracle hostname"
}
variable "destination_oracle_username" {
	type        = string
	description = "Oracle username"
}
variable "destination_oracle_password" {
	type        = string
	sensitive   = true
	description = "Oracle password"
}
variable "destination_oracle_database" {
	type        = string
	description = "Oracle database name"
	default     = ""
}
resource "streamkap_destination_oracle" "test" {
	name                = %q
	database_hostname   = var.destination_oracle_hostname
	database_port       = 1521
	database_database   = var.destination_oracle_database
	connection_username = var.destination_oracle_username
	connection_password = var.destination_oracle_password
	schema_evolution    = "none"
	insert_mode         = "upsert"
	delete_enabled      = true
	primary_key_mode    = "record_value"
	tasks_max           = 10
	transforms_mask_field_fields_include_list       = "public.users.email,public.users.phone"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "FIXED"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_mask_fixed_value          = "MASKED"
	transforms_mask_field_replace_null_with_default = true
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "transforms_mask_field_fields_include_list", "public.users.email,public.users.phone"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "transforms_mask_field_mask_function", "FIXED"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "transforms_mask_field_mask_fixed_value", "MASKED"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "transforms_mask_field_replace_null_with_default", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "schema_evolution", "none"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "insert_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "delete_enabled", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "primary_key_mode", "record_value"),
					resource.TestCheckResourceAttr("streamkap_destination_oracle.test", "tasks_max", "10"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
