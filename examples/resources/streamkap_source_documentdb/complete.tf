# Complete DocumentDB CDC source configuration
# This example shows all available configuration options for capturing changes
# from AWS DocumentDB collections

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

variable "source_documentdb_connection_string" {
  type        = string
  sensitive   = true
  description = "DocumentDB connection string with credentials"
}

resource "streamkap_source_documentdb" "example-source-documentdb" {
  # Display name for this source in Streamkap UI
  name = "example-source-documentdb"

  # Connection string (includes credentials)
  mongodb_connection_string = var.source_documentdb_connection_string

  # Database and collection selection
  database_include_list   = "ecommerce,analytics"
  collection_include_list = "ecommerce.orders,ecommerce.customers,analytics.events"

  # Signal collection for incremental snapshots
  signal_data_collection_schema_or_database = "streamkap"

  # Array encoding (for mixed-type arrays)
  transforms_unwrap_array_encoding = "array_string" # Options: array, array_string

  # Document encoding (nested documents)
  transforms_unwrap_document_encoding = "document" # Options: document, string

  # SSH tunnel settings (optional)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = "22"
  # ssh_user = "streamkap"
}

output "example-source-documentdb" {
  value = streamkap_source_documentdb.example-source-documentdb.id
}
