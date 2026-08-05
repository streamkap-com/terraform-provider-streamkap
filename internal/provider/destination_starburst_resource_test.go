package provider

import (
	"fmt"
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

	name := acctestName(t, "main")
	nameUpdated := acctestName(t, "updated")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckDestinationDestroy,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
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
	name                  = %q
	aws_access_key_id     = var.destination_starburst_access_key_id
	aws_secret_access_key = var.destination_starburst_secret_access_key
	aws_s3_region         = "us-west-2"
	aws_s3_bucket_name    = var.destination_starburst_bucket_name
	format                = "CSV"
	file_name_template    = "{{topic}}-{{partition}}-{{start_offset}}"
	file_name_prefix      = "streamkap/"
	file_compression_type = "gzip"
	transforms_mask_field_fields_include_list       = "public.users.email"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "REDACT"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_replace_null_with_default = false
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "transforms_mask_field_fields_include_list", "public.users.email"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "transforms_mask_field_mask_function", "REDACT"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "transforms_mask_field_mask_fixed_value", "***"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "transforms_mask_field_replace_null_with_default", "false"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "name", name),
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
				ResourceName:            "streamkap_destination_starburst.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"connector_status"},
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
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
	name                  = %q
	aws_access_key_id     = var.destination_starburst_access_key_id
	aws_secret_access_key = var.destination_starburst_secret_access_key
	aws_s3_region         = "us-east-1"
	aws_s3_bucket_name    = var.destination_starburst_bucket_name
	format                = "Parquet"
	file_name_template    = "{{topic}}/{{partition}}/{{start_offset}}"
	file_name_prefix      = "streamkap-updated/"
	file_compression_type = "snappy"
	transforms_mask_field_fields_include_list       = "public.users.email,public.users.phone"
	transforms_mask_field_fields_exclude_list       = "public.users.id"
	transforms_mask_field_mask_function             = "FIXED"
	transforms_mask_field_mask_char                 = "#"
	transforms_mask_field_mask_fixed_value          = "MASKED"
	transforms_mask_field_replace_null_with_default = true
}
`, nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "transforms_mask_field_fields_include_list", "public.users.email,public.users.phone"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "transforms_mask_field_fields_exclude_list", "public.users.id"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "transforms_mask_field_mask_function", "FIXED"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "transforms_mask_field_mask_char", "#"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "transforms_mask_field_mask_fixed_value", "MASKED"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "transforms_mask_field_replace_null_with_default", "true"),
					resource.TestCheckResourceAttr("streamkap_destination_starburst.test", "name", nameUpdated),
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
