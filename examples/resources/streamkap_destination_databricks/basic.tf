# Minimal Databricks destination configuration

resource "streamkap_destination_databricks" "example" {
  name             = "my-databricks-dest"
  connection_url   = var.databricks_jdbc_url
  databricks_token = var.databricks_token
}

variable "databricks_jdbc_url" {
  description = "Databricks JDBC connection URL"
  type        = string
}

variable "databricks_token" {
  description = "Databricks personal access token"
  type        = string
  sensitive   = true
}
