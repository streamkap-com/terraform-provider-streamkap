package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationMysqlHostname = os.Getenv("TF_VAR_destination_mysql_hostname")
var destinationMysqlUsername = os.Getenv("TF_VAR_destination_mysql_username")
var destinationMysqlPassword = os.Getenv("TF_VAR_destination_mysql_password")
var _ = os.Getenv("TF_VAR_destination_mysql_database") // used via TF_VAR in HCL config

func TestAccDestinationMysqlResource(t *testing.T) {
	if destinationMysqlHostname == "" || destinationMysqlUsername == "" || destinationMysqlPassword == "" {
		t.Skip("Skipping TestAccDestinationMysqlResource: TF_VAR_destination_mysql_hostname, TF_VAR_destination_mysql_username, or TF_VAR_destination_mysql_password not set")
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
variable "destination_mysql_hostname" {
	type        = string
	description = "MySQL hostname"
}
variable "destination_mysql_username" {
	type        = string
	description = "MySQL username"
}
variable "destination_mysql_password" {
	type        = string
	sensitive   = true
	description = "MySQL password"
}
variable "destination_mysql_database" {
	type        = string
	description = "MySQL database name"
	default     = ""
}
resource "streamkap_destination_mysql" "test" {
	name                = %q
	database_hostname   = var.destination_mysql_hostname
	database_port       = 3306
	database_database   = var.destination_mysql_database
	connection_username = var.destination_mysql_username
	connection_password = var.destination_mysql_password
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
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "transforms_mask_field_fields_include_list", "public.users.email"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "transforms_mask_field_mask_function", "REDACT"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "transforms_mask_field_mask_fixed_value", "***"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "transforms_mask_field_replace_null_with_default", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "database_hostname", destinationMysqlHostname),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "database_port", "3306"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "connection_username", destinationMysqlUsername),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "insert_mode", "insert"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "delete_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "primary_key_mode", "record_key"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "tasks_max", "5"),
					resource.TestCheckResourceAttrSet("streamkap_destination_mysql.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "connector", "mysql"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_destination_mysql.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "destination_mysql_hostname" {
	type        = string
	description = "MySQL hostname"
}
variable "destination_mysql_username" {
	type        = string
	description = "MySQL username"
}
variable "destination_mysql_password" {
	type        = string
	sensitive   = true
	description = "MySQL password"
}
variable "destination_mysql_database" {
	type        = string
	description = "MySQL database name"
	default     = ""
}
resource "streamkap_destination_mysql" "test" {
	name                = %q
	database_hostname   = var.destination_mysql_hostname
	database_port       = 3306
	database_database   = var.destination_mysql_database
	connection_username = var.destination_mysql_username
	connection_password = var.destination_mysql_password
	schema_evolution    = "basic"
	insert_mode         = "upsert"
	delete_enabled      = false
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
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "transforms_mask_field_fields_include_list", "public.users.email,public.users.phone"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "transforms_mask_field_mask_function", "FIXED"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "transforms_mask_field_mask_fixed_value", "MASKED"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "transforms_mask_field_replace_null_with_default", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "insert_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "delete_enabled", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "primary_key_mode", "record_value"),
					resource.TestCheckResourceAttr("streamkap_destination_mysql.test", "tasks_max", "10"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
