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

variable "destination_bigquery_json" {
  type        = string
  sensitive   = true
  description = "The BigQuery service account JSON credentials"
}

# Complete BigQuery destination configuration with all options
resource "streamkap_destination_bigquery" "example" {
  name = "example-destination-bigquery"

  # Connection settings (required)
  bigquery_json     = var.destination_bigquery_json
  table_name_prefix = "streamkap_dataset"       # Destination Dataset name

  # Region configuration
  bigquery_region = "us-central1"               # Default: us-central1
  # Valid values: us-east5, us-central1, us-west4, us-west2, northamerica-northeast1,
  # us-east4, us-west1, us-west3, southamerica-east1, southamerica-west1, us-east1,
  # northamerica-northeast2, asia-south2, asia-east2, asia-southeast2, australia-southeast2,
  # asia-south1, asia-northeast2, asia-northeast3, asia-southeast1, australia-southeast1,
  # asia-east1, asia-northeast1, europe-west1, europe-north1, europe-west3, europe-west2,
  # europe-southwest1, europe-west8, europe-west4, europe-west9, europe-central2, europe-west6, EU, US

  # Partitioning and clustering
  custom_bigquery_cluster_field   = "cluster_field"   # Custom cluster field name
  custom_bigquery_partition_field = "partition_field" # Custom partition field name
  bigquery_time_based_partition   = false             # Time-based partitioning. Default: false

  # Performance settings
  tasks_max = 5                                 # Max active tasks (1-10). Default: 5
}

output "example_destination_bigquery" {
  value = streamkap_destination_bigquery.example.id
}
