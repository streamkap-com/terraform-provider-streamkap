package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDestinationGcsResource(t *testing.T) {
	var destinationGcsCredentialsJson = os.Getenv("TF_VAR_destination_gcs_credentials_json")
	var destinationGcsBucketName = os.Getenv("TF_VAR_destination_gcs_bucket_name")
	if destinationGcsCredentialsJson == "" || destinationGcsBucketName == "" {
		t.Skip("Skipping TestAccDestinationGcsResource: TF_VAR_destination_gcs_credentials_json or TF_VAR_destination_gcs_bucket_name not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_gcs_credentials_json" {
	type        = string
	sensitive   = true
	description = "GCS service account credentials JSON"
}
variable "destination_gcs_bucket_name" {
	type        = string
	description = "GCS bucket name"
}
resource "streamkap_destination_gcs" "test" {
	name                  = "tf-acc-test-destination-gcs"
	gcs_credentials_json  = var.destination_gcs_credentials_json
	gcs_bucket_name       = var.destination_gcs_bucket_name
	format                = "CSV"
	file_compression_type = "gzip"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_gcs.test", "name", "tf-acc-test-destination-gcs"),
					resource.TestCheckResourceAttr("streamkap_destination_gcs.test", "gcs_credentials_json", destinationGcsCredentialsJson),
					resource.TestCheckResourceAttr("streamkap_destination_gcs.test", "gcs_bucket_name", destinationGcsBucketName),
					resource.TestCheckResourceAttr("streamkap_destination_gcs.test", "format", "CSV"),
					resource.TestCheckResourceAttr("streamkap_destination_gcs.test", "file_compression_type", "gzip"),
					resource.TestCheckResourceAttrSet("streamkap_destination_gcs.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_gcs.test", "connector", "gcs"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_gcs.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_gcs_credentials_json" {
	type        = string
	sensitive   = true
	description = "GCS service account credentials JSON"
}
variable "destination_gcs_bucket_name" {
	type        = string
	description = "GCS bucket name"
}
resource "streamkap_destination_gcs" "test" {
	name                  = "tf-acc-test-destination-gcs-updated"
	gcs_credentials_json  = var.destination_gcs_credentials_json
	gcs_bucket_name       = var.destination_gcs_bucket_name
	format                = "Parquet"
	file_compression_type = "snappy"
	file_name_prefix      = "streamkap/data/"
	file_name_template    = "{{topic}}-{{partition}}-{{start_offset}}"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_gcs.test", "name", "tf-acc-test-destination-gcs-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_gcs.test", "format", "Parquet"),
					resource.TestCheckResourceAttr("streamkap_destination_gcs.test", "file_compression_type", "snappy"),
					resource.TestCheckResourceAttr("streamkap_destination_gcs.test", "file_name_prefix", "streamkap/data/"),
					resource.TestCheckResourceAttr("streamkap_destination_gcs.test", "file_name_template", "{{topic}}-{{partition}}-{{start_offset}}"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
