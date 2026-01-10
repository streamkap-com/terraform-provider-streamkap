# Minimal SQL Server CDC source configuration
# This example captures changes from SQL Server tables using CDC

resource "streamkap_source_sqlserver" "example" {
  name = "my-sqlserver-source"

  # Connection details
  database_hostname = "sqlserver.example.com"
  database_port     = 1433
  database_user     = "streamkap_user"
  database_password = var.db_password # Use variables for secrets

  # Database and tables to capture
  database_names      = "mydb"
  schema_include_list = "dbo"
  table_include_list  = "dbo.Orders,dbo.Customers"
}

variable "db_password" {
  description = "SQL Server database password"
  type        = string
  sensitive   = true
}
