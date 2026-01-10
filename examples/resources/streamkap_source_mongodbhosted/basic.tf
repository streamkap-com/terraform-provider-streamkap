# Minimal MongoDB Hosted source configuration
# This example captures changes from self-hosted MongoDB collections

resource "streamkap_source_mongodbhosted" "example" {
  name = "my-mongodb-hosted-source"

  # Connection string with credentials
  mongodb_connection_string = var.connection_string

  # Databases and collections to capture
  database_include_list   = "mydb"
  collection_include_list = "mydb.orders,mydb.customers"

  # Signal collection for incremental snapshots (required)
  signal_data_collection_schema_or_database = "streamkap"
}

variable "connection_string" {
  description = "MongoDB connection string"
  type        = string
  sensitive   = true
}
