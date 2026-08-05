package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDestinationAzblobResource(t *testing.T) {
	var destinationAzblobConnectionString = os.Getenv("TF_VAR_destination_azblob_connection_string")
	var destinationAzblobContainerName = os.Getenv("TF_VAR_destination_azblob_container_name")
	if destinationAzblobConnectionString == "" || destinationAzblobContainerName == "" {
		t.Skip("Skipping TestAccDestinationAzblobResource: TF_VAR_destination_azblob_connection_string or TF_VAR_destination_azblob_container_name not set")
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
variable "destination_azblob_connection_string" {
	type        = string
	sensitive   = true
	description = "Azure Blob Storage connection string"
}
variable "destination_azblob_container_name" {
	type        = string
	description = "Azure Blob Storage container name"
}
resource "streamkap_destination_azblob" "test" {
	name                      = %q
	azblob_connection_string  = var.destination_azblob_connection_string
	azblob_container_name     = var.destination_azblob_container_name
	format                    = "json"
	flush_size                = 1000
	file_size                 = 65536
	rotate_interval_ms        = -1
	transforms_mask_field_fields_include_list       = "public.users.email"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "REDACT"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_replace_null_with_default = false
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "transforms_mask_field_fields_include_list", "public.users.email"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "transforms_mask_field_mask_function", "REDACT"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "transforms_mask_field_mask_fixed_value", "***"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "transforms_mask_field_replace_null_with_default", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "azblob_connection_string", destinationAzblobConnectionString),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "azblob_container_name", destinationAzblobContainerName),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "format", "json"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "flush_size", "1000"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "file_size", "65536"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "rotate_interval_ms", "-1"),
					resource.TestCheckResourceAttrSet("streamkap_destination_azblob.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "connector", "azblob"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_destination_azblob.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "destination_azblob_connection_string" {
	type        = string
	sensitive   = true
	description = "Azure Blob Storage connection string"
}
variable "destination_azblob_container_name" {
	type        = string
	description = "Azure Blob Storage container name"
}
resource "streamkap_destination_azblob" "test" {
	name                      = %q
	azblob_connection_string  = var.destination_azblob_connection_string
	azblob_container_name     = var.destination_azblob_container_name
	format                    = "parquet"
	flush_size                = 2000
	file_size                 = 131072
	rotate_interval_ms        = 60000
	topics_dir                = "streamkap/data"
	transforms_mask_field_fields_include_list       = "public.users.email,public.users.phone"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "FIXED"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_mask_fixed_value          = "MASKED"
	transforms_mask_field_replace_null_with_default = true
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "transforms_mask_field_fields_include_list", "public.users.email,public.users.phone"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "transforms_mask_field_mask_function", "FIXED"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "transforms_mask_field_mask_fixed_value", "MASKED"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "transforms_mask_field_replace_null_with_default", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "format", "parquet"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "flush_size", "2000"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "file_size", "131072"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "rotate_interval_ms", "60000"),
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "topics_dir", "streamkap/data"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
