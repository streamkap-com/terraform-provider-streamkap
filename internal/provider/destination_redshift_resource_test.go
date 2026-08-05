package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationRedshiftDomain = os.Getenv("TF_VAR_destination_redshift_domain")
var destinationRedshiftUsername = os.Getenv("TF_VAR_destination_redshift_username")
var destinationRedshiftPassword = os.Getenv("TF_VAR_destination_redshift_password")
var _ = os.Getenv("TF_VAR_destination_redshift_database") // used via TF_VAR in HCL config

func TestAccDestinationRedshiftResource(t *testing.T) {
	if destinationRedshiftDomain == "" || destinationRedshiftUsername == "" || destinationRedshiftPassword == "" {
		t.Skip("Skipping TestAccDestinationRedshiftResource: TF_VAR_destination_redshift_domain, TF_VAR_destination_redshift_username, or TF_VAR_destination_redshift_password not set")
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
variable "destination_redshift_domain" {
	type        = string
	description = "Redshift cluster domain"
}
variable "destination_redshift_username" {
	type        = string
	description = "Redshift username"
}
variable "destination_redshift_password" {
	type        = string
	sensitive   = true
	description = "Redshift password"
}
variable "destination_redshift_database" {
	type        = string
	description = "Redshift database name"
	default     = ""
}
resource "streamkap_destination_redshift" "test" {
	name                 = %q
	aws_redshift_domain  = var.destination_redshift_domain
	aws_redshift_port    = 5439
	aws_redshift_database = var.destination_redshift_database
	connection_username  = var.destination_redshift_username
	connection_password  = var.destination_redshift_password
	primary_key_fields   = "id"
	schema_evolution     = "basic"
	table_name_prefix    = "streamkap"
	tasks_max            = 5
	transforms_mask_field_fields_include_list       = "public.users.email"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "REDACT"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_replace_null_with_default = false
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "transforms_mask_field_fields_include_list", "public.users.email"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "transforms_mask_field_mask_function", "REDACT"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "transforms_mask_field_mask_fixed_value", "***"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "transforms_mask_field_replace_null_with_default", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "aws_redshift_domain", destinationRedshiftDomain),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "aws_redshift_port", "5439"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "connection_username", destinationRedshiftUsername),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "primary_key_fields", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "table_name_prefix", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "tasks_max", "5"),
					resource.TestCheckResourceAttrSet("streamkap_destination_redshift.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "connector", "redshift"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_destination_redshift.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "destination_redshift_domain" {
	type        = string
	description = "Redshift cluster domain"
}
variable "destination_redshift_username" {
	type        = string
	description = "Redshift username"
}
variable "destination_redshift_password" {
	type        = string
	sensitive   = true
	description = "Redshift password"
}
variable "destination_redshift_database" {
	type        = string
	description = "Redshift database name"
	default     = ""
}
resource "streamkap_destination_redshift" "test" {
	name                 = %q
	aws_redshift_domain  = var.destination_redshift_domain
	aws_redshift_port    = 5439
	aws_redshift_database = var.destination_redshift_database
	connection_username  = var.destination_redshift_username
	connection_password  = var.destination_redshift_password
	primary_key_fields   = "id,created_at"
	schema_evolution     = "none"
	table_name_prefix    = "streamkap_updated"
	tasks_max            = 10
	transforms_mask_field_fields_include_list       = "public.users.email,public.users.phone"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "FIXED"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_mask_fixed_value          = "MASKED"
	transforms_mask_field_replace_null_with_default = true
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "transforms_mask_field_fields_include_list", "public.users.email,public.users.phone"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "transforms_mask_field_mask_function", "FIXED"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "transforms_mask_field_mask_fixed_value", "MASKED"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "transforms_mask_field_replace_null_with_default", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "primary_key_fields", "id,created_at"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "schema_evolution", "none"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "table_name_prefix", "streamkap_updated"),
					resource.TestCheckResourceAttr("streamkap_destination_redshift.test", "tasks_max", "10"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
