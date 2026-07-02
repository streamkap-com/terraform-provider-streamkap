# Minimal BigQuery destination configuration

resource "streamkap_destination_bigquery" "example" {
  name            = "my-bigquery-dest"
  keyfile         = var.bigquery_keyfile
  default_dataset = var.bigquery_dataset
}

variable "bigquery_keyfile" {
  description = "BigQuery service-account JSON key file contents"
  type        = string
  sensitive   = true
}

variable "bigquery_dataset" {
  description = "Destination BigQuery dataset name (must already exist)"
  type        = string
}
