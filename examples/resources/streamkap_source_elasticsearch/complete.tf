# Complete Elasticsearch source configuration
# This example shows all available configuration options for capturing data
# from Elasticsearch indices

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

variable "source_es_host" {
  type        = string
  description = "The hostname of the Elasticsearch cluster"
}

variable "source_es_password" {
  type        = string
  sensitive   = true
  description = "The password for Elasticsearch authentication"
}

resource "streamkap_source_elasticsearch" "example-source-elasticsearch" {
  # Display name for this source in Streamkap UI
  name = "example-source-elasticsearch"

  # Connection settings
  es_host   = var.source_es_host # Can use semicolon for multiple hosts
  es_scheme = "https"            # Options: http, https
  es_port   = "443"              # Default: 443 for HTTPS, 9200 for HTTP

  # Authentication
  http_auth      = "Basic" # Options: None, Basic
  http_auth_user     = "elastic"
  http_auth_password = var.source_es_password

  # Indices to capture
  endpoint_include_list = "orders,customers,products"

  # Datetime field for incremental sync
  datetime_field_name  = "updated_at"
  datetime_field_value = "2024-01-01T00:00:00Z" # Starting point (optional)

  # Task parallelism
  tasks_max = 5 # Between 1-10
}

output "example-source-elasticsearch" {
  value = streamkap_source_elasticsearch.example-source-elasticsearch.id
}
