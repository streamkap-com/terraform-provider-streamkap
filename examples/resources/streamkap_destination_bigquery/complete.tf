terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.5.0"
}

provider "streamkap" {}

variable "destination_bigquery_keyfile" {
  type        = string
  sensitive   = true
  description = "BigQuery service-account JSON key file contents"
}

# Complete BigQuery destination configuration with all options
resource "streamkap_destination_bigquery" "example" {
  name = "example-destination-bigquery"

  # Connection (required)
  keyfile         = var.destination_bigquery_keyfile # Service-account JSON key (sensitive)
  default_dataset = "streamkap_dataset"              # Destination dataset (must already exist)

  # Partitioning and clustering
  time_partitioning_type           = "DAY"             # Partition granularity. Default: DAY. Valid: DAY, HOUR, MONTH, YEAR, NONE
  custom_partition_field           = "partition_field" # Optional record field to partition by. Blank = ingestion-time
  custom_clustering_fields         = "field1,field2"   # Optional comma-separated fields to cluster by (max 4)
  custom_partition_expiration_days = "30"              # Optional. Drop partitions older than N days. Blank = keep all. Ignored when NONE

  # Schema evolution
  auto_create_tables                        = true # Auto-create tables for new topics. Default: true
  allow_new_big_query_fields                = true # Add new columns as the schema evolves. Default: true
  allow_big_query_required_field_relaxation = true # Relax REQUIRED -> NULLABLE as the schema evolves. Default: true

  # Performance settings
  tasks_max = 5 # Max active tasks (1-10). Default: 5
}

output "example_destination_bigquery" {
  value = streamkap_destination_bigquery.example.id
}
