# Minimal GCS destination configuration

resource "streamkap_destination_gcs" "example" {
  name                 = "my-gcs-dest"
  gcs_credentials_json = var.gcs_credentials_json
  gcs_bucket_name      = var.gcs_bucket_name
}

variable "gcs_credentials_json" {
  description = "GCP service account credentials JSON"
  type        = string
  sensitive   = true
}

variable "gcs_bucket_name" {
  description = "GCS bucket name"
  type        = string
}
