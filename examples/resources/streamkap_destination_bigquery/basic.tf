# Minimal BigQuery destination configuration

resource "streamkap_destination_bigquery" "example" {
  name              = "my-bigquery-dest"
  bigquery_json     = var.bigquery_json
  table_name_prefix = var.dataset_name
  bigquery_region   = var.bigquery_region
}

variable "bigquery_json" {
  description = "GCP service account credentials JSON"
  type        = string
  sensitive   = true
}

variable "dataset_name" {
  description = "BigQuery dataset name (used as table prefix)"
  type        = string
}

variable "bigquery_region" {
  description = "BigQuery region"
  type        = string
  default     = "us-central1"
}
