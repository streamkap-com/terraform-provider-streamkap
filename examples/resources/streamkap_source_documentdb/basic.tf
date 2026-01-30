# Minimal DocumentDB CDC source configuration
# This example captures changes from AWS DocumentDB collections

resource "streamkap_source_documentdb" "example" {
  name = "my-documentdb-source"

  # Connection string with credentials
  mongodb_connection_string = var.connection_string

  # Databases and collections to capture
  database_include_list   = "mydb"
  collection_include_list = "mydb.orders,mydb.customers"

  # Signal collection for incremental snapshots (required)
  signal_data_collection_schema_or_database = "streamkap"
}

variable "connection_string" {
  description = "DocumentDB connection string"
  type        = string
  sensitive   = true
}
