# Minimal SQL Server destination configuration

resource "streamkap_destination_sqlserver" "example" {
  name                = "my-sqlserver-dest"
  database_hostname   = var.sqlserver_hostname
  connection_username = var.sqlserver_username
  connection_password = var.sqlserver_password
}

variable "sqlserver_hostname" {
  description = "SQL Server hostname"
  type        = string
}

variable "sqlserver_username" {
  description = "Username to access SQL Server"
  type        = string
}

variable "sqlserver_password" {
  description = "Password to access SQL Server"
  type        = string
  sensitive   = true
}
