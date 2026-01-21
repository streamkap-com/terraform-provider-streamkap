package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var destinationStarburstAccessKeyID = os.Getenv("TF_VAR_destination_starburst_access_key_id")
var destinationStarburstSecretAccessKey = os.Getenv("TF_VAR_destination_starburst_secret_access_key")
var destinationStarburstBucketName = os.Getenv("TF_VAR_destination_starburst_bucket_name")

func TestAccDestinationStarburstResource(t *testing.T) {
	if destinationStarburstAccessKeyID == "" || destinationStarburstSecretAccessKey == "" || destinationStarburstBucketName == "" {
		t.Skip("Skipping TestAccDestinationStarburstResource: TF_VAR_destination_starburst_access_key_id, TF_VAR_destination_starburst_secret_access_key, or TF_VAR_destination_starburst_bucket_name not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "destination_starburst_access_key_id" {
	type        = string
	description = "AWS Access Key ID for Starburst"
}
variable "destination_starburst_secret_access_key" {
	type        = string
	sensitive   = true
	description = "AWS Secret Access Key for Starburst"
}
variable "destination_starburst_bucket_name" {
	type        = string
	description = "S3 bucket name for Starburst"
}
resource "streamkap_destination_starburst" "test" {
	name                  = "tf-acc-test-destination-starburst"
	aws_access_key_id     = var.destination_starburst_access_key_id
	aws_secret_access_key = var.destination_starburst_secret_access_key
	aws_s3_region         = "us-west-2"
	aws_s3_bucket_name    = var.destination_starburst_bucket_name
	format                = "CSV"
	file_name_template    = "{{topic}}-{{partition}}-{{start_offset}}"
	file_name_prefix      = "streamkap/"
	file_compression_type = "gzip"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "name", "tf-acc-test-destination-starburst"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "aws_access_key_id", destinationStarburstAccessKeyID),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "aws_s3_region", "us-west-2"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "aws_s3_bucket_name", destinationStarburstBucketName),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "format", "CSV"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "file_name_template", "{{topic}}-{{partition}}-{{start_offset}}"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "file_name_prefix", "streamkap/"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "file_compression_type", "gzip"),
					resource.TestCheckResourceAttrSet("streamkap_destination_starburst.test", "id"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "connector", "starburst"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_destination_starburst.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "destination_starburst_access_key_id" {
	type        = string
	description = "AWS Access Key ID for Starburst"
}
variable "destination_starburst_secret_access_key" {
	type        = string
	sensitive   = true
	description = "AWS Secret Access Key for Starburst"
}
variable "destination_starburst_bucket_name" {
	type        = string
	description = "S3 bucket name for Starburst"
}
resource "streamkap_destination_starburst" "test" {
	name                  = "tf-acc-test-destination-starburst-updated"
	aws_access_key_id     = var.destination_starburst_access_key_id
	aws_secret_access_key = var.destination_starburst_secret_access_key
	aws_s3_region         = "us-east-1"
	aws_s3_bucket_name    = var.destination_starburst_bucket_name
	format                = "Parquet"
	file_name_template    = "{{topic}}/{{partition}}/{{start_offset}}"
	file_name_prefix      = "streamkap-updated/"
	file_compression_type = "snappy"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "name", "tf-acc-test-destination-starburst-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "aws_s3_region", "us-east-1"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "format", "Parquet"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "file_name_template", "{{topic}}/{{partition}}/{{start_offset}}"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "file_name_prefix", "streamkap-updated/"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "file_compression_type", "snappy"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
