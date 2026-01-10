# Minimal PostgreSQL destination configuration

resource "streamkap_destination_postgresql" "example" {
  name                = "my-postgresql-dest"
  database_hostname   = var.postgresql_hostname
  connection_username = var.postgresql_username
  connection_password = var.postgresql_password
  table_name_prefix   = "streamkap"
}

variable "postgresql_hostname" {
  description = "PostgreSQL server hostname"
  type        = string
}

variable "postgresql_username" {
  description = "Username to access PostgreSQL"
  type        = string
}

variable "postgresql_password" {
  description = "Password to access PostgreSQL"
  type        = string
  sensitive   = true
}
