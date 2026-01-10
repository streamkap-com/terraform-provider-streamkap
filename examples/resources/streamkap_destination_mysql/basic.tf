# Minimal MySQL destination configuration

resource "streamkap_destination_mysql" "example" {
  name                = "my-mysql-dest"
  database_hostname   = var.mysql_hostname
  connection_username = var.mysql_username
  connection_password = var.mysql_password
}

variable "mysql_hostname" {
  description = "MySQL server hostname"
  type        = string
}

variable "mysql_username" {
  description = "Username to access MySQL"
  type        = string
}

variable "mysql_password" {
  description = "Password to access MySQL"
  type        = string
  sensitive   = true
}
