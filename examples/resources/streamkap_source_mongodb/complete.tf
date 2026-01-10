# Complete MongoDB Atlas CDC source configuration
# This example shows all available configuration options for capturing changes
# from MongoDB collections using change streams

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
  description = "The connection string of the MongoDB database"
}

resource "streamkap_source_mongodb" "example-source-mongodb" {
  # Display name for this source in Streamkap UI
  name = "example-source-mongodb"

  # MongoDB connection string (includes credentials)
  # Format: mongodb+srv://user:password@cluster.mongodb.net/
  mongodb_connection_string = var.source_mongodb_connection_string

  # Database and collection selection
  database_include_list   = "Test"                          # Databases to sync
  collection_include_list = "Test.test_data4,Test.test_data2" # Collections to capture (db.collection format)

  # Signal collection for incremental snapshots (required)
  # This database must contain a 'streamkap_signal' collection for snapshot coordination
  signal_data_collection_schema_or_database = "Test"

  # Data encoding options
  transforms_unwrap_array_encoding    = "array_string" # Options: array, array_string
  transforms_unwrap_document_encoding = "document"     # Options: document, string

  # SSH tunnel settings (optional, for secure connections)
  ssh_enabled = false
  # When ssh_enabled = true, also configure:
  # ssh_host = "bastion.example.com"
  # ssh_port = 22
  # ssh_user = "streamkap"
}

output "example-source-mongodb" {
  value = streamkap_source_mongodb.example-source-mongodb.id
}
