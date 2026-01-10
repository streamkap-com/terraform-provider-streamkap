# Complete DynamoDB CDC source configuration
# This example shows all available configuration options for capturing changes
# from DynamoDB tables using DynamoDB Streams

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
  sensitive   = true
  description = "AWS Access Key ID"
}

variable "source_dynamodb_aws_secret_key" {
  type        = string
  sensitive   = true
  description = "AWS Secret Key"
}

resource "streamkap_source_dynamodb" "example-source-dynamodb" {
  # Display name for this source in Streamkap UI
  name = "example-source-dynamodb"

  # AWS credentials and region
  aws_region        = var.source_dynamodb_aws_region
  aws_access_key_id = var.source_dynamodb_aws_access_key_id
  aws_secret_key    = var.source_dynamodb_aws_secret_key

  # S3 bucket for snapshot exports (required)
  # This bucket is used to export table data for initial snapshots
  s3_export_bucket_name = "tst-s3-export-snapshot"

  # Tables to capture (comma-separated)
  table_include_list = "DynamoDBToClickHouseDemo"

  # Optional: Custom DynamoDB endpoint (for local testing or VPC endpoints)
  # dynamodb_service_endpoint = "http://localhost:8000"

  # Performance tuning
  tasks_max                      = 3     # Maximum number of parallel tasks (1-40)
  batch_size                     = 1024  # Records per batch
  poll_timeout_ms                = 1000  # Poll timeout in milliseconds
  incremental_snapshot_chunk_size  = 32768 # Chunk size for incremental snapshots
  incremental_snapshot_max_threads = 8     # Max threads for incremental snapshots
  full_export_expiration_time_ms   = 86400000 # Export expiration (24 hours)
  signal_kafka_poll_timeout_ms     = 1000  # Kafka signal poll timeout

  # Parallel snapshot configuration
  # snapshot_parallel_time_offset = 0 # Set > 0 to run snapshot in parallel with CDC

  # Data encoding options
  array_encoding_json  = true # Encode arrays as JSON strings
  struct_encoding_json = true # Encode structs/maps as JSON strings
}

output "example-source-dynamodb" {
  value = streamkap_source_dynamodb.example-source-dynamodb.id
}
