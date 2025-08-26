package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Define environment variables for S3 configuration
var s3AwsAccessKey = os.Getenv("TF_VAR_s3_aws_access_key")
var s3AwsSecretKey = os.Getenv("TF_VAR_s3_aws_secret_key")

func TestAccDestinationS3Resource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create and Read Testing
			{
				Config: providerConfig + `
variable "s3_aws_access_key" {
  type        = string
  description = "The AWS Access Key ID used to connect to S3"
}
variable "s3_aws_secret_key" {
  type        = string
  sensitive   = true
  description = "The AWS Secret Access Key used to connect to S3"
}

resource "streamkap_destination_s3" "test" {
  name           = "test-destination-s3"
  aws_access_key = var.s3_aws_access_key
  aws_secret_key = var.s3_aws_secret_key
  aws_region     = "us-west-2"
  bucket_name    = "bucketname"
  format         = "JSON Array"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "name", "test-destination-s3"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_access_key", s3AwsAccessKey),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_secret_key", s3AwsSecretKey),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_region", "us-west-2"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "bucket_name", "bucketname"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "format", "JSON Array"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "filename_template", "{{topic}}-{{partition}}-{{start_offset}}"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "filename_prefix", ""),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "compression_type", "gzip"),
				),
			},
			// Step 2: ImportState Testing
			{
				ResourceName:      "streamkap_destination_s3.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update and Read Testing
			{
				Config: providerConfig + `
variable "s3_aws_access_key" {
  type        = string
  description = "The AWS Access Key ID used to connect to S3"
}
variable "s3_aws_secret_key" {
  type        = string
  sensitive   = true
  description = "The AWS Secret Access Key used to connect to S3"
}

resource "streamkap_destination_s3" "test" {
  name           = "example-destination-s3-updated"
  aws_access_key = var.s3_aws_access_key
  aws_secret_key = var.s3_aws_secret_key
  aws_region     = "us-west-1"
  bucket_name    = "bucketname-updated"
  compression_type         = "none"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "name", "example-destination-s3-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_access_key", icebergAwsAccessKey),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_secret_key", icebergAwsSecretKey),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_region", "us-west-1"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "bucket_name", "bucketname-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "format", "JSON Array"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "filename_template", "{{topic}}-{{partition}}-{{start_offset}}"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "filename_prefix", ""),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "compression_type", "none"),
				),
			},
			// Delete testing is automatically handled by the test framework
		},
	})
}
