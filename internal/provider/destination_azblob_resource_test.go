package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationAzblobConnectionString = os.Getenv("TF_VAR_destination_azblob_connection_string")
var destinationAzblobContainerName = os.Getenv("TF_VAR_destination_azblob_container_name")

func TestAccDestinationAzblobResource(t *testing.T) {
	if destinationAzblobConnectionString == "" || destinationAzblobContainerName == "" {
		t.Skip("Skipping TestAccDestinationAzblobResource: TF_VAR_destination_azblob_connection_string or TF_VAR_destination_azblob_container_name not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
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
	name                      = "tf-acc-test-destination-azblob"
	azblob_connection_string  = var.destination_azblob_connection_string
	azblob_container_name     = var.destination_azblob_container_name
	format                    = "json"
	flush_size                = 1000
	file_size                 = 65536
	rotate_interval_ms        = -1
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "name", "tf-acc-test-destination-azblob"),
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
				ResourceName:      "streamkap_destination_azblob.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
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
	name                      = "tf-acc-test-destination-azblob-updated"
	azblob_connection_string  = var.destination_azblob_connection_string
	azblob_container_name     = var.destination_azblob_container_name
	format                    = "parquet"
	flush_size                = 2000
	file_size                 = 131072
	rotate_interval_ms        = 60000
	topics_dir                = "streamkap/data"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_azblob.test", "name", "tf-acc-test-destination-azblob-updated"),
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
