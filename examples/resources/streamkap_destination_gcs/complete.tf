terraform {
  required_providers {
    streamkap = {
      source  = "streamkap-com/streamkap"
      version = ">= 2.0.0"
    }
  }
  required_version = ">= 1.0.0"
}

provider "streamkap" {}

variable "destination_gcs_credentials_json" {
  type        = string
  sensitive   = true
  description = "GCP service account credentials JSON"
}

# Complete GCS destination configuration with all options
resource "streamkap_destination_gcs" "example" {
  name = "example-destination-gcs"

  # Connection settings
  gcs_credentials_json = var.destination_gcs_credentials_json
  gcs_bucket_name      = "my-streamkap-bucket"

  # File format settings
  format = "CSV"                                # Valid values: CSV, JSON Lines, JSON Array, Parquet. Default: CSV

  # File naming
  file_name_template = "{{topic}}-{{partition}}-{{start_offset}}" # Default filename template
  file_name_prefix   = "data/output/"           # Directory prefix for files

  # Compression
  file_compression_type = "gzip"                # Valid values: none, gzip, snappy, zstd. Default: gzip

  # Output field selection
  format_output_fields = ["key", "value", "timestamp", "offset", "headers"]
}

output "example_destination_gcs" {
  value = streamkap_destination_gcs.example.id
}
