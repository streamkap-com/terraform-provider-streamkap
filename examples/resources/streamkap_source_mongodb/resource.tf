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
  description = "The connection string of the MongoDB database"
}

resource "streamkap_source_mongodb" "example-source-mongodb" {
  name                                      = "example-source-mongodb"
  mongodb_connection_string                 = var.source_mongodb_connection_string
  array_encoding                            = "array_string"
  nested_document_encoding                  = "document"
  database_include_list                     = "Test"
  collection_include_list                   = "Test.test_data4,Test.test_data2"
  signal_data_collection_schema_or_database = "Test"
  ssh_enabled                               = false
}

output "example-source-mongodb" {
  value = streamkap_source_mongodb.example-source-mongodb.id
}