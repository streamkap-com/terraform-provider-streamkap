package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDestinationDb2Resource(t *testing.T) {
	var destinationDb2Hostname = os.Getenv("TF_VAR_destination_db2_hostname")
	var destinationDb2Username = os.Getenv("TF_VAR_destination_db2_username")
	var destinationDb2Password = os.Getenv("TF_VAR_destination_db2_password")
	var _ = os.Getenv("TF_VAR_destination_db2_database") // used via TF_VAR in HCL config
	if destinationDb2Hostname == "" || destinationDb2Username == "" || destinationDb2Password == "" {
		t.Skip("Skipping TestAccDestinationDb2Resource: TF_VAR_destination_db2_hostname, TF_VAR_destination_db2_username, or TF_VAR_destination_db2_password not set")
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
	name                = %q
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
	transforms_mask_field_fields_include_list       = "public.users.email"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "REDACT"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_replace_null_with_default = false
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "transforms_mask_field_fields_include_list", "public.users.email"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "transforms_mask_field_mask_function", "REDACT"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "transforms_mask_field_mask_fixed_value", "***"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "transforms_mask_field_replace_null_with_default", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "name", name),
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
				ResourceName:            "streamkap_destination_db2.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
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
	name                = %q
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
	transforms_mask_field_fields_include_list       = "public.users.email,public.users.phone"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "FIXED"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_mask_fixed_value          = "MASKED"
	transforms_mask_field_replace_null_with_default = true
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "transforms_mask_field_fields_include_list", "public.users.email,public.users.phone"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "transforms_mask_field_mask_function", "FIXED"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "transforms_mask_field_mask_fixed_value", "MASKED"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "transforms_mask_field_replace_null_with_default", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_db2.test", "name", nameUpdated),
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
