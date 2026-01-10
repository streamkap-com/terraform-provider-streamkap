# Minimal MongoDB Atlas CDC source configuration
# This example captures changes from MongoDB collections using change streams

resource "streamkap_source_mongodb" "example" {
  name = "my-mongodb-source"

  # Connection string (includes authentication)
  mongodb_connection_string = var.mongodb_connection_string

  # Databases and collections to capture
  database_include_list   = "mydb"
  collection_include_list = "mydb.orders,mydb.customers"

  # Signal collection for incremental snapshots (required)
  signal_data_collection_schema_or_database = "streamkap"
}

variable "mongodb_connection_string" {
  description = "MongoDB connection string (mongodb+srv://user:pass@cluster.mongodb.net)"
  type        = string
  sensitive   = true
}
