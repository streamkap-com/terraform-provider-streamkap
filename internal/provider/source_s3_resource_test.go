package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var sourceS3AccessKeyID = os.Getenv("TF_VAR_source_s3_access_key_id")
var sourceS3SecretAccessKey = os.Getenv("TF_VAR_source_s3_secret_access_key")
var sourceS3BucketName = os.Getenv("TF_VAR_source_s3_bucket_name")

func TestAccSourceS3Resource(t *testing.T) {
	if sourceS3AccessKeyID == "" || sourceS3SecretAccessKey == "" || sourceS3BucketName == "" {
		t.Skip("Skipping TestAccSourceS3Resource: TF_VAR_source_s3_access_key_id, TF_VAR_source_s3_secret_access_key, or TF_VAR_source_s3_bucket_name not set")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
variable "source_s3_access_key_id" {
	type        = string
	description = "The AWS Access Key ID for S3"
}
variable "source_s3_secret_access_key" {
	type        = string
	sensitive   = true
	description = "The AWS Secret Access Key for S3"
}
variable "source_s3_bucket_name" {
	type        = string
	description = "The S3 bucket name"
}
resource "streamkap_source_s3" "test" {
	name                   = "tf-acc-test-source-s3"
	format                 = "json"
	topic_postfix          = "default"
	aws_access_key_id      = var.source_s3_access_key_id
	aws_secret_access_key  = var.source_s3_secret_access_key
	aws_s3_region          = "us-west-2"
	aws_s3_bucket_name     = var.source_s3_bucket_name
	aws_s3_object_prefix   = "file-pulse/"
	fs_scan_interval_ms    = 10000
	fs_cleanup_policy_class = "io.streamthoughts.kafka.connect.filepulse.fs.clean.LogCleanupPolicy"
	tasks_max              = 5
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "name", "tf-acc-test-source-s3"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "format", "json"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "topic_postfix", "default"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "aws_access_key_id", sourceS3AccessKeyID),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "aws_secret_access_key", sourceS3SecretAccessKey),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "aws_s3_region", "us-west-2"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "aws_s3_bucket_name", sourceS3BucketName),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "aws_s3_object_prefix", "file-pulse/"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "fs_scan_interval_ms", "10000"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "fs_cleanup_policy_class", "io.streamthoughts.kafka.connect.filepulse.fs.clean.LogCleanupPolicy"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "tasks_max", "5"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "streamkap_source_s3.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: providerConfig + `
variable "source_s3_access_key_id" {
	type        = string
	description = "The AWS Access Key ID for S3"
}
variable "source_s3_secret_access_key" {
	type        = string
	sensitive   = true
	description = "The AWS Secret Access Key for S3"
}
variable "source_s3_bucket_name" {
	type        = string
	description = "The S3 bucket name"
}
resource "streamkap_source_s3" "test" {
	name                   = "tf-acc-test-source-s3-updated"
	format                 = "csv"
	topic_postfix          = "updated"
	aws_access_key_id      = var.source_s3_access_key_id
	aws_secret_access_key  = var.source_s3_secret_access_key
	aws_s3_region          = "us-east-1"
	aws_s3_bucket_name     = var.source_s3_bucket_name
	aws_s3_object_prefix   = "data/"
	fs_scan_interval_ms    = 20000
	fs_cleanup_policy_class = "io.streamthoughts.kafka.connect.filepulse.fs.clean.DeleteCleanupPolicy"
	tasks_max              = 3
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "name", "tf-acc-test-source-s3-updated"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "format", "csv"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "topic_postfix", "updated"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "aws_s3_region", "us-east-1"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "aws_s3_object_prefix", "data/"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "fs_scan_interval_ms", "20000"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "fs_cleanup_policy_class", "io.streamthoughts.kafka.connect.filepulse.fs.clean.DeleteCleanupPolicy"),
					resource.TestCheckResourceAttr("streamkap_source_s3.test", "tasks_max", "3"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
