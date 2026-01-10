# Minimal ClickHouse destination configuration

resource "streamkap_destination_clickhouse" "example" {
  name                = "my-clickhouse-dest"
  hostname            = var.clickhouse_hostname
  connection_username = var.clickhouse_username
  connection_password = var.clickhouse_password
  database            = "default"
}

variable "clickhouse_hostname" {
  description = "ClickHouse server hostname or IP address"
  type        = string
}

variable "clickhouse_username" {
  description = "Username to access ClickHouse"
  type        = string
}

variable "clickhouse_password" {
  description = "Password to access ClickHouse"
  type        = string
  sensitive   = true
}
