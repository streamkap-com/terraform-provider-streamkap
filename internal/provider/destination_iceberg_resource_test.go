package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Define environment variables for Iceberg configuration
var icebergAwsAccessKey = os.Getenv("TF_VAR_iceberg_aws_access_key")
var icebergAwsSecretKey = os.Getenv("TF_VAR_iceberg_aws_secret_key")

func TestAccDestinationIcebergResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create and Read Testing
			{
				Config: providerConfig + `
variable "iceberg_aws_access_key" {
  type        = string
  description = "The AWS Access Key ID used to connect to Iceberg. Required for rest and hive."
}
variable "iceberg_aws_secret_key" {
  type        = string
  sensitive   = true
  description = "The AWS Secret Access Key used to connect to Iceberg. Required for rest and hive."
}

resource "streamkap_destination_iceberg" "test" {
  name           = "test-destination-iceberg"
  catalog_type   = "rest"
  catalog_name   = "iceberg_catalog_name"
  catalog_uri    = "iceberg_catalog_uri"
  aws_access_key = var.iceberg_aws_access_key
  aws_secret_key = var.iceberg_aws_secret_key
  aws_region     = "us-west-2"
  bucket_path    = "iceberg_bucket_path"
  schema         = "iceberg_schema"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "name", "test-destination-iceberg"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "catalog_type", "rest"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "catalog_name", "iceberg_catalog_name"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "catalog_uri", "iceberg_catalog_uri"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "aws_access_key", icebergAwsAccessKey),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "aws_secret_key", icebergAwsSecretKey),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "aws_region", "us-west-2"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "bucket_path", "iceberg_bucket_path"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "schema", "iceberg_schema"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "insert_mode", "insert"),
				),
			},
			// Step 2: ImportState Testing
			{
				ResourceName:      "streamkap_destination_iceberg.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 3: Update and Read Testing
			{
				Config: providerConfig + `
variable "iceberg_aws_access_key" {
  type        = string
  description = "The AWS Access Key ID used to connect to Iceberg. Required for rest and hive."
}
variable "iceberg_aws_secret_key" {
  type        = string
  sensitive   = true
  description = "The AWS Secret Access Key used to connect to Iceberg. Required for rest and hive."
}

resource "streamkap_destination_iceberg" "test" {
  name           = "test-destination-iceberg-updated"
  catalog_type   = "hive"
  catalog_name   = "iceberg_catalog_name-updated"
  catalog_uri    = "iceberg_catalog_uri"
  aws_access_key = var.iceberg_aws_access_key
  aws_secret_key = var.iceberg_aws_secret_key
  aws_region     = "us-west-1"
  bucket_path    = "iceberg_bucket_path-updated"
  schema         = "iceberg_schema"
  insert_mode    = "upsert"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "name", "test-destination-iceberg-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "catalog_type", "hive"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "catalog_name", "iceberg_catalog_name-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "catalog_uri", "iceberg_catalog_uri"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "aws_access_key", icebergAwsAccessKey),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "aws_secret_key", icebergAwsSecretKey),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "aws_region", "us-west-1"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "bucket_path", "iceberg_bucket_path-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "schema", "iceberg_schema"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "insert_mode", "upsert"),
				),
			},
			// Delete testing is automatically handled by the test framework
		},
	})
}
