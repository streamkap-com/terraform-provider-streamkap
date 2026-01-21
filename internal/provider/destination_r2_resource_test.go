package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationR2Account = os.Getenv("TF_VAR_destination_r2_account")
var destinationR2AccessKeyID = os.Getenv("TF_VAR_destination_r2_access_key_id")
var destinationR2SecretAccessKey = os.Getenv("TF_VAR_destination_r2_secret_access_key")
var destinationR2BucketName = os.Getenv("TF_VAR_destination_r2_bucket_name")

func TestAccDestinationR2Resource(t *testing.T) {
	if destinationR2Account == "" || destinationR2AccessKeyID == "" || destinationR2SecretAccessKey == "" || destinationR2BucketName == "" {
		t.Skip("Skipping TestAccDestinationR2Resource: TF_VAR_destination_r2_account, TF_VAR_destination_r2_access_key_id, TF_VAR_destination_r2_secret_access_key, or TF_VAR_destination_r2_bucket_name not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_r2_account" {
	type        = string
	description = "Cloudflare R2 account ID"
}
variable "destination_r2_access_key_id" {
	type        = string
	description = "R2 access key ID"
}
variable "destination_r2_secret_access_key" {
	type        = string
	sensitive   = true
	description = "R2 secret access key"
}
variable "destination_r2_bucket_name" {
	type        = string
	description = "R2 bucket name"
}
resource "streamkap_destination_r2" "test" {
	name                  = "tf-acc-test-destination-r2"
	r2_account            = var.destination_r2_account
	aws_access_key_id     = var.destination_r2_access_key_id
	aws_secret_access_key = var.destination_r2_secret_access_key
	aws_s3_bucket_name    = var.destination_r2_bucket_name
	format                = "JSON Array"
	file_compression_type = "gzip"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_r2.test", "name", "tf-acc-test-destination-r2"),
					resource.TestCheckResourceAttr("streamkap_destination_r2.test", "r2_account", destinationR2Account),
					resource.TestCheckResourceAttr("streamkap_destination_r2.test", "aws_access_key_id", destinationR2AccessKeyID),
					resource.TestCheckResourceAttr("streamkap_destination_r2.test", "aws_s3_bucket_name", destinationR2BucketName),
					resource.TestCheckResourceAttr("streamkap_destination_r2.test", "format", "JSON Array"),
					resource.TestCheckResourceAttr("streamkap_destination_r2.test", "file_compression_type", "gzip"),
					resource.TestCheckResourceAttrSet("streamkap_destination_r2.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_r2.test", "connector", "r2"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_r2.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_r2_account" {
	type        = string
	description = "Cloudflare R2 account ID"
}
variable "destination_r2_access_key_id" {
	type        = string
	description = "R2 access key ID"
}
variable "destination_r2_secret_access_key" {
	type        = string
	sensitive   = true
	description = "R2 secret access key"
}
variable "destination_r2_bucket_name" {
	type        = string
	description = "R2 bucket name"
}
resource "streamkap_destination_r2" "test" {
	name                  = "tf-acc-test-destination-r2-updated"
	r2_account            = var.destination_r2_account
	aws_access_key_id     = var.destination_r2_access_key_id
	aws_secret_access_key = var.destination_r2_secret_access_key
	aws_s3_bucket_name    = var.destination_r2_bucket_name
	format                = "Parquet"
	file_compression_type = "none"
	file_name_prefix      = "updated/"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_r2.test", "name", "tf-acc-test-destination-r2-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_r2.test", "format", "Parquet"),
					resource.TestCheckResourceAttr("streamkap_destination_r2.test", "file_compression_type", "none"),
					resource.TestCheckResourceAttr("streamkap_destination_r2.test", "file_name_prefix", "updated/"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
