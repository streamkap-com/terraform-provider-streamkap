# Minimal BigQuery destination configuration

resource "streamkap_destination_bigquery" "example" {
  name           = "my-bigquery-dest"
  bigquery_json  = var.bigquery_json
  bigquery_region = var.bigquery_region
}

variable "bigquery_json" {
  description = "GCP service account credentials JSON"
  type        = string
  sensitive   = true
}

variable "bigquery_region" {
  description = "BigQuery region"
  type        = string
  default     = "us-central1"
}
