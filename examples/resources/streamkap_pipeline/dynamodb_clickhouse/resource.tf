terraform {
  required_providers {
    streamkap = {
      source = "streamkap-com/streamkap"
    }
  }
  required_version = ">= 2.0.0"
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
  s3_export_bucket_name            = "streamkap-export"
  table_include_list_user_defined  = "warehouse-test-2"
  batch_size                       = 1024
  poll_timeout_ms                  = 1000
  incremental_snapshot_chunk_size  = 32768
  incremental_snapshot_max_threads = 8
  incremental_snapshot_interval_ms = 8
  full_export_expiration_time_ms   = 86400000
  signal_kafka_poll_timeout_ms     = 1000
}

variable "destination_clickhouse_hostname" {
  type        = string
  description = "The hostname of the Clickhouse server"
}

variable "destination_clickhouse_connection_username" {
  type        = string
  description = "The username to connect to the Clickhouse server"
}

variable "destination_clickhouse_connection_password" {
  type        = string
  sensitive   = true
  description = "The password to connect to the Clickhouse server"
}

resource "streamkap_destination_clickhouse" "example-destination-clickhouse" {
  name                = "example-destination-clickhouse"
  ingestion_mode      = "append"
  tasks_max           = 5
  hostname            = var.destination_clickhouse_hostname
  connection_username = var.destination_clickhouse_connection_username
  connection_password = var.destination_clickhouse_connection_password
  port                = 8443
  database            = "demo"
  ssl                 = true
}

resource "streamkap_pipeline" "example-pipeline" {
  name                = "example-pipeline"
  snapshot_new_tables = true
  source = {
    id        = streamkap_source_dynamodb.example-source-dynamodb.id
    name      = streamkap_source_dynamodb.example-source-dynamodb.name
    connector = streamkap_source_dynamodb.example-source-dynamodb.connector
    topics = [
      "default.warehouse-test-2",
    ]
  }
  destination = {
    id        = streamkap_destination_clickhouse.example-destination-clickhouse.id
    name      = streamkap_destination_clickhouse.example-destination-clickhouse.name
    connector = streamkap_destination_clickhouse.example-destination-clickhouse.connector
  }
}

output "example-pipeline" {
  value = streamkap_pipeline.example-pipeline.id
}