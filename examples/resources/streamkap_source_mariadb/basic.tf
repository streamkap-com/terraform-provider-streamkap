# Minimal MariaDB CDC source configuration
# This example captures changes from MariaDB tables using binary log replication

resource "streamkap_source_mariadb" "example" {
  name = "my-mariadb-source"

  # Connection details
  database_hostname = "mariadb.example.com"
  database_port     = "3306"
  database_user     = "streamkap_user"
  database_password = var.db_password

  # Databases and tables to capture
  database_include_list = "mydb"
  table_include_list    = "mydb.orders,mydb.customers"

  # Heartbeat table for monitoring (required)
  heartbeat_data_collection_schema_or_database = "streamkap"
}

variable "db_password" {
  description = "MariaDB database password"
  type        = string
  sensitive   = true
}
