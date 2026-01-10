# Minimal DB2 destination configuration

resource "streamkap_destination_db2" "example" {
  name                = "my-db2-dest"
  database_hostname   = var.db2_hostname
  connection_username = var.db2_username
  connection_password = var.db2_password
}

variable "db2_hostname" {
  description = "DB2 server hostname"
  type        = string
}

variable "db2_username" {
  description = "Username to access DB2"
  type        = string
}

variable "db2_password" {
  description = "Password to access DB2"
  type        = string
  sensitive   = true
}
