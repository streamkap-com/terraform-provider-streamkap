# Minimal CockroachDB destination configuration

resource "streamkap_destination_cockroachdb" "example" {
  name                = "my-cockroachdb-dest"
  database_hostname   = var.cockroachdb_hostname
  connection_username = var.cockroachdb_username
  connection_password = var.cockroachdb_password
}

variable "cockroachdb_hostname" {
  description = "CockroachDB server hostname"
  type        = string
}

variable "cockroachdb_username" {
  description = "Username to access CockroachDB"
  type        = string
}

variable "cockroachdb_password" {
  description = "Password to access CockroachDB"
  type        = string
  sensitive   = true
}
