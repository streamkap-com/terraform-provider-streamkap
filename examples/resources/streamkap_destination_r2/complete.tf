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

variable "destination_r2_secret_access_key" {
  type        = string
  sensitive   = true
  description = "Cloudflare R2 Secret Access Key"
}

# Complete R2 (Cloudflare) destination configuration with all options
resource "streamkap_destination_r2" "example" {
  name = "example-destination-r2"

  # Connection settings (required)
  r2_account            = "your-cloudflare-account-id"
  aws_access_key_id     = "your-r2-access-key-id"
  aws_secret_access_key = var.destination_r2_secret_access_key
  aws_s3_bucket_name    = "my-r2-bucket"

  # File format settings
  format = "JSON Array"                         # Valid values: JSON Lines, JSON Array, Parquet. Default: JSON Array

  # File naming
  file_name_template = "{{topic}}-{{partition}}-{{start_offset}}" # Default filename template
  file_name_prefix   = "data/output/"           # Directory prefix for files

  # Compression
  file_compression_type = "gzip"                # Valid values: none, gzip, snappy, zstd. Default: gzip

  # Output field selection
  format_output_fields = ["key", "value", "timestamp", "offset", "headers"]
}

output "example_destination_r2" {
  value = streamkap_destination_r2.example.id
}
