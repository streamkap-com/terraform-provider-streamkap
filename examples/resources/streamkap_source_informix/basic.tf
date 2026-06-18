# Minimal Informix CDC source configuration.
# Captures changes from IBM Informix tables.
resource "streamkap_source_informix" "example" {
  name = "my-informix-source"

  # Connection details
  database_hostname = "informix.example.com"
  database_port     = 9088
  database_user     = "streamkap_user"
  database_password = var.db_password
  database_dbname   = "stores_demo"

  # Schemas and tables to capture
  schema_include_list = "informix"
  table_include_list  = "informix.orders,informix.customer"
}

variable "db_password" {
  description = "Informix database password"
  type        = string
  sensitive   = true
}
