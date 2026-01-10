# Minimal Oracle destination configuration

resource "streamkap_destination_oracle" "example" {
  name                = "my-oracle-dest"
  database_hostname   = var.oracle_hostname
  connection_username = var.oracle_username
  connection_password = var.oracle_password
}

variable "oracle_hostname" {
  description = "Oracle server hostname"
  type        = string
}

variable "oracle_username" {
  description = "Username to access Oracle"
  type        = string
}

variable "oracle_password" {
  description = "Password to access Oracle"
  type        = string
  sensitive   = true
}
