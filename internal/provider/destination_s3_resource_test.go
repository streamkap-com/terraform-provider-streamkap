package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Define environment variables for S3 configuration
var s3AwsAccessId = os.Getenv("TF_VAR_s3_aws_access_key_id")
var s3AwsSecretKey = os.Getenv("TF_VAR_s3_aws_secret_access_key")

func TestAccDestinationS3Resource(t *testing.T) {
	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			// Step 1: Create and Read Testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "s3_aws_access_key_id" {
  type        = string
  description = "The AWS Access Key ID used to connect to S3"
}
variable "s3_aws_secret_access_key" {
  type        = string
  sensitive   = true
  description = "The AWS Secret Access Key used to connect to S3"
}

resource "streamkap_destination_s3" "test" {
  name           = %q
  aws_access_key_id = var.s3_aws_access_key_id
  aws_secret_access_key = var.s3_aws_secret_access_key
  aws_s3_region     = "us-west-2"
  aws_s3_bucket_name    = "bucketname"
  format         = "JSON Array"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "name", name),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_access_key_id", s3AwsAccessId),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_secret_access_key", s3AwsSecretKey),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_s3_region", "us-west-2"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_s3_bucket_name", "bucketname"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "format", "JSON Array"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "file_name_template", "{{topic}}-{{partition}}-{{start_offset}}"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "file_compression_type", "gzip"),
				),
			},
			// Step 2: ImportState Testing
			{
				ResourceName:            "streamkap_destination_s3.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Step 3: Update and Read Testing
			{
				Config: providerConfig + fmt.Sprintf(`
variable "s3_aws_access_key_id" {
  type        = string
  description = "The AWS Access Key ID used to connect to S3"
}
variable "s3_aws_secret_access_key" {
  type        = string
  sensitive   = true
  description = "The AWS Secret Access Key used to connect to S3"
}

resource "streamkap_destination_s3" "test" {
  name           = %q
  aws_access_key_id = var.s3_aws_access_key_id
  aws_secret_access_key = var.s3_aws_secret_access_key
  aws_s3_region     = "us-west-2"
  aws_s3_bucket_name    = "bucketname-updated"
  file_compression_type         = "none"
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "name", nameUpdated),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_access_key_id", s3AwsAccessId),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_secret_access_key", s3AwsSecretKey),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_s3_region", "us-west-2"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "aws_s3_bucket_name", "bucketname-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "format", "JSON Array"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "file_name_template", "{{topic}}-{{partition}}-{{start_offset}}"),
					resource.TestCheckResourceAttr("streamkap_destination_s3.test", "file_compression_type", "none"),
				),
			},
			// Delete testing is automatically handled by the test framework
		},
	})
}
