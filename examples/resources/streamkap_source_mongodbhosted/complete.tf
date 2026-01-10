# Complete MongoDB Hosted source configuration
# This example shows all available configuration options for capturing changes
# from self-hosted MongoDB collections

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

variable "source_mongodb_connection_string" {
  type        = string
  sensitive   = true
  description = "MongoDB connection string with credentials"
}

resource "streamkap_source_mongodbhosted" "example-source-mongodbhosted" {
  # Display name for this source in Streamkap UI
  name = "example-source-mongodbhosted"

  # Connection string (includes credentials)
  mongodb_connection_string = var.source_mongodb_connection_string

  # Database and collection selection
  database_include_list   = "ecommerce,analytics"
  collection_include_list = "ecommerce.orders,ecommerce.customers,analytics.events"

  # Signal collection for incremental snapshots
  signal_data_collection_schema_or_database = "streamkap"

  # Array encoding (for mixed-type arrays)
  transforms_unwrap_array_encoding = "array_string" # Options: array, array_string

  # Document encoding (nested documents)
  transforms_unwrap_document_encoding = "document" # Options: document, string

  # Static fields for message enrichment (optional)
  # transforms_insert_static_key1_static_field   = "tenant"
  # transforms_insert_static_key1_static_value   = "acme"
  # transforms_insert_static_value1_static_field = "environment"
  # transforms_insert_static_value1_static_value = "production"

  # Topic enrichment pattern (optional)
  predicates_is_topic_to_enrich_pattern = "$^"

  # SSH tunnel settings (optional)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = "22"
  # ssh_user = "streamkap"
}

output "example-source-mongodbhosted" {
  value = streamkap_source_mongodbhosted.example-source-mongodbhosted.id
}
