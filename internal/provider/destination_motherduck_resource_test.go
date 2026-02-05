package provider

import (
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

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
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
	name               = "tf-acc-test-destination-motherduck"
	motherduck_token   = var.destination_motherduck_token
	motherduck_catalog = var.destination_motherduck_catalog
	ingestion_mode     = "upsert"
	schema_evolution   = "basic"
	table_name_prefix  = "streamkap"
	hard_delete        = false
	tasks_max          = 5
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "name", "tf-acc-test-destination-motherduck"),
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
				ResourceName:      "streamkap_destination_motherduck.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
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
	name               = "tf-acc-test-destination-motherduck-updated"
	motherduck_token   = var.destination_motherduck_token
	motherduck_catalog = var.destination_motherduck_catalog
	ingestion_mode     = "append"
	schema_evolution   = "none"
	table_name_prefix  = "streamkap_updated"
	hard_delete        = true
	tasks_max          = 10
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_motherduck.test", "name", "tf-acc-test-destination-motherduck-updated"),
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
