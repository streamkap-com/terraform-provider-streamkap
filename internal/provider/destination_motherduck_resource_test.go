package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationMotherduckToken = os.Getenv("TF_VAR_destination_motherduck_token")
var destinationMotherduckCatalog = os.Getenv("TF_VAR_destination_motherduck_catalog")

func TestAccDestinationMotherduckResource(t *testing.T) {
	if destinationMotherduckToken == "" || destinationMotherduckCatalog == "" {
		t.Skip("Skipping TestAccDestinationMotherduckResource: TF_VAR_destination_motherduck_token or TF_VAR_destination_motherduck_catalog not set")
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
variable "destination_motherduck_token" {
	type        = string
	sensitive   = true
	description = "Motherduck token"
}
variable "destination_motherduck_catalog" {
	type        = string
	description = "Motherduck catalog/database name"
}
resource "streamkap_destination_motherduck" "test" {
	name               = %q
	motherduck_token   = var.destination_motherduck_token
	motherduck_catalog = var.destination_motherduck_catalog
	ingestion_mode     = "upsert"
	schema_evolution   = "basic"
	table_name_prefix  = "streamkap"
	hard_delete        = false
	tasks_max          = 5
	transforms_mask_field_fields_include_list       = "public.users.email"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "REDACT"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_replace_null_with_default = false
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "transforms_mask_field_fields_include_list", "public.users.email"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "transforms_mask_field_mask_function", "REDACT"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "transforms_mask_field_mask_fixed_value", "***"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "transforms_mask_field_replace_null_with_default", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "motherduck_catalog", destinationMotherduckCatalog),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "ingestion_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "schema_evolution", "basic"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "table_name_prefix", "streamkap"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "hard_delete", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "tasks_max", "5"),
					resource.TestCheckResourceAttrSet("streamkap_destination_motherduck.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "connector", "motherduck"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "streamkap_destination_motherduck.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "destination_motherduck_token" {
	type        = string
	sensitive   = true
	description = "Motherduck token"
}
variable "destination_motherduck_catalog" {
	type        = string
	description = "Motherduck catalog/database name"
}
resource "streamkap_destination_motherduck" "test" {
	name               = %q
	motherduck_token   = var.destination_motherduck_token
	motherduck_catalog = var.destination_motherduck_catalog
	ingestion_mode     = "append"
	schema_evolution   = "none"
	table_name_prefix  = "streamkap_updated"
	hard_delete        = true
	tasks_max          = 10
	transforms_mask_field_fields_include_list       = "public.users.email,public.users.phone"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "FIXED"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_mask_fixed_value          = "MASKED"
	transforms_mask_field_replace_null_with_default = true
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "transforms_mask_field_fields_include_list", "public.users.email,public.users.phone"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "transforms_mask_field_mask_function", "FIXED"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "transforms_mask_field_mask_fixed_value", "MASKED"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "transforms_mask_field_replace_null_with_default", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "ingestion_mode", "append"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "schema_evolution", "none"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "table_name_prefix", "streamkap_updated"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "hard_delete", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "tasks_max", "10"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
