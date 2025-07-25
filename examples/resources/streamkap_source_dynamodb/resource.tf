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

resource "streamkap_source_dynamodb" "example-source-dynamodb" {
  name                             = "example-source-dynamodb"
  aws_region                       = var.source_dynamodb_aws_region
  aws_access_key_id                = var.source_dynamodb_aws_access_key_id
  aws_secret_key                   = var.source_dynamodb_aws_secret_key
  s3_export_bucket_name            = "tst-s3-export-snapshot"
  table_include_list_user_defined  = "DynamoDBToClickHouseDemo"
  batch_size                       = 1024
  poll_timeout_ms                  = 1000
  incremental_snapshot_chunk_size  = 32768
  incremental_snapshot_max_threads = 8
  full_export_expiration_time_ms   = 86400000
  signal_kafka_poll_timeout_ms     = 1000
  array_encoding_json              = true
  struct_encoding_json             = true
}

output "example-source-dynamodb" {
  value = streamkap_source_dynamodb.example-source-dynamodb.id
}