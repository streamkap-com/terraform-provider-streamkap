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
  name                                 = "test-destination-iceberg"
  iceberg_catalog_type                 = "rest"
  iceberg_catalog_name                 = "iceberg_catalog_name"
  iceberg_catalog_uri                  = "iceberg_catalog_uri"
  iceberg_catalog_s3_access_key_id     = var.iceberg_aws_access_key
  iceberg_catalog_s3_secret_access_key = var.iceberg_aws_secret_key
  iceberg_catalog_client_region        = "us-west-2"
  iceberg_catalog_warehouse            = "s3://my-bucket/iceberg-warehouse"
  table_name_prefix                    = "analytics"
  insert_mode                          = "insert"
  iceberg_tables_default_id_columns    = "id,created_at"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "name", "test-destination-iceberg"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_type", "rest"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_name", "iceberg_catalog_name"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_uri", "iceberg_catalog_uri"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_s3_access_key_id", icebergAwsAccessKey),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_s3_secret_access_key", icebergAwsSecretKey),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_client_region", "us-west-2"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_warehouse", "s3://my-bucket/iceberg-warehouse"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "table_name_prefix", "analytics"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "insert_mode", "insert"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_tables_default_id_columns", "id,created_at"),
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
  name                                 = "test-destination-iceberg-updated"
  iceberg_catalog_type                 = "hive"
  iceberg_catalog_name                 = "iceberg_catalog_name_updated"
  iceberg_catalog_uri                  = "iceberg_catalog_uri_updated"
  iceberg_catalog_s3_access_key_id     = var.iceberg_aws_access_key
  iceberg_catalog_s3_secret_access_key = var.iceberg_aws_secret_key
  iceberg_catalog_client_region        = "us-west-1"
  iceberg_catalog_warehouse            = "s3://my-bucket/iceberg-warehouse-updated"
  table_name_prefix                    = "analytics_updated"
  insert_mode                          = "upsert"
  iceberg_tables_default_id_columns    = "id"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "name", "test-destination-iceberg-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_type", "hive"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_name", "iceberg_catalog_name_updated"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_uri", "iceberg_catalog_uri_updated"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_s3_access_key_id", icebergAwsAccessKey),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_s3_secret_access_key", icebergAwsSecretKey),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_client_region", "us-west-1"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_catalog_warehouse", "s3://my-bucket/iceberg-warehouse-updated"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "table_name_prefix", "analytics_updated"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "insert_mode", "upsert"),
					resource.TestCheckResourceAttr("streamkap_destination_iceberg.test", "iceberg_tables_default_id_columns", "id"),
				),
			},
			// Delete testing is automatically handled by the test framework
		},
	})
}
