# Complete S3 source configuration
# This example shows all available configuration options for capturing data
# from S3 bucket files

terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

variable "source_s3_access_key_id" {
  type        = string
  description = "AWS access key ID"
}

variable "source_s3_secret_access_key" {
  type        = string
  sensitive   = true
  description = "AWS secret access key"
}

resource "streamkap_source_s3" "example-source-s3" {
  # Display name for this source in Streamkap UI
  name = "example-source-s3"

  # File format
  format = "json" # Options: json, csv, avro

  # Topic naming
  topic_postfix = "events" # Topic will be source_[UUID]_events

  # AWS credentials
  aws_access_key_id     = var.source_s3_access_key_id
  aws_secret_access_key = var.source_s3_secret_access_key

  # S3 bucket settings
  aws_s3_region       = "us-west-2" # Options: us-east-1, us-west-2, eu-west-1, etc.
  aws_s3_bucket_name  = "my-data-bucket"
  aws_s3_object_prefix = "data/incoming/"

  # Scan settings
  fs_scan_interval_ms = 10000 # Interval in ms (100-100000)

  # Cleanup policy after processing
  fs_cleanup_policy_class = "io.streamthoughts.kafka.connect.filepulse.fs.clean.LogCleanupPolicy"
  # Options:
  # - io.streamthoughts.kafka.connect.filepulse.fs.clean.LogCleanupPolicy (log only)
  # - io.streamthoughts.kafka.connect.filepulse.fs.clean.DeleteCleanupPolicy (delete after processing)

  # Task parallelism
  tasks_max = 5 # Between 1-10
}

output "example-source-s3" {
  value = streamkap_source_s3.example-source-s3.id
}
