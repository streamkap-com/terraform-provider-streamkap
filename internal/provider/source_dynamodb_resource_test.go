package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceDynamoDBAWSRegion = os.Getenv("TF_VAR_source_dynamodb_aws_region")
var sourceDynamoDBAWSAcessKeyID = os.Getenv("TF_VAR_source_dynamodb_aws_access_key_id")
var sourceDynamoDBAWSSecretKey = os.Getenv("TF_VAR_source_dynamodb_aws_secret_key")

func TestAccSourceDynamoDBResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_dynamodb_aws_region" {
	type        = string
	description = "AWS Region"
}

variable "source_dynamodb_aws_access_key_id" {
	type        = string
	description = "AWS Access Key ID"
}

variable "source_dynamodb_aws_secret_key" {
	type        = string
	sensitive   = true
	description = "AWS Secret Key"
}

resource "streamkap_source_dynamodb" "test" {
	name                             = "test-source-dynamodb"
	aws_region                       = var.source_dynamodb_aws_region
	aws_access_key_id                = var.source_dynamodb_aws_access_key_id
	aws_secret_key                   = var.source_dynamodb_aws_secret_key
	s3_export_bucket_name            = "streamkap-export"
	table_include_list_user_defined  = "warehouse-test-2"
	batch_size                       = 1024
	poll_timeout_ms                  = 1000
	incremental_snapshot_chunk_size  = 32768
	incremental_snapshot_max_threads = 8
	full_export_expiration_time_ms   = 86400000
	signal_kafka_poll_timeout_ms     = 1000
	array_encoding_json              = true
	struct_encoding_json             = true
}
`,

				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "name", "test-source-dynamodb"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "aws_region", sourceDynamoDBAWSRegion),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "aws_access_key_id", sourceDynamoDBAWSAcessKeyID),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "aws_secret_key", sourceDynamoDBAWSSecretKey),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "s3_export_bucket_name", "streamkap-export"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "table_include_list_user_defined", "warehouse-test-2"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "batch_size", "1024"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "poll_timeout_ms", "1000"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "incremental_snapshot_chunk_size", "32768"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "incremental_snapshot_max_threads", "8"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "full_export_expiration_time_ms", "86400000"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "signal_kafka_poll_timeout_ms", "1000"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "array_encoding_json", "true"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "struct_encoding_json", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_dynamodb.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_dynamodb_aws_region" {
	type        = string
	description = "AWS Region"
}

variable "source_dynamodb_aws_access_key_id" {
	type        = string
	description = "AWS Access Key ID"
}

variable "source_dynamodb_aws_secret_key" {
	type        = string
	sensitive   = true
	description = "AWS Secret Key"
}

resource "streamkap_source_dynamodb" "test" {
	name                             = "test-source-dynamodb-updated"
	aws_region                       = var.source_dynamodb_aws_region
	aws_access_key_id                = var.source_dynamodb_aws_access_key_id
	aws_secret_key                   = var.source_dynamodb_aws_secret_key
	s3_export_bucket_name            = "streamkap-export"
	table_include_list_user_defined  = "warehouse-test-2"
	batch_size                       = 1024
	poll_timeout_ms                  = 2000
	incremental_snapshot_chunk_size  = 32768
	incremental_snapshot_max_threads = 8
	full_export_expiration_time_ms   = 86400000
	signal_kafka_poll_timeout_ms     = 2000
	array_encoding_json              = true
	struct_encoding_json             = true
}
`,

				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify if attributes are propagated correctly
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "name", "test-source-dynamodb-updated"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "aws_region", sourceDynamoDBAWSRegion),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "aws_access_key_id", sourceDynamoDBAWSAcessKeyID),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "aws_secret_key", sourceDynamoDBAWSSecretKey),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "s3_export_bucket_name", "streamkap-export"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "table_include_list_user_defined", "warehouse-test-2"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "batch_size", "1024"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "poll_timeout_ms", "2000"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "incremental_snapshot_chunk_size", "32768"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "incremental_snapshot_max_threads", "8"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "incremental_snapshot_interval_ms", "12"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "full_export_expiration_time_ms", "86400000"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "signal_kafka_poll_timeout_ms", "2000"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "array_encoding_json", "true"),
					resource.TestCheckResourceAttr("streamkap_source_dynamodb.test", "struct_encoding_json", "true"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
