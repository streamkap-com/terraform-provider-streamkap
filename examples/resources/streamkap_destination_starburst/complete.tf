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

variable "destination_starburst_secret_access_key" {
  type        = string
  sensitive   = true
  description = "AWS Secret Access Key for Starburst S3 access"
}

# Complete Starburst destination configuration with all options
resource "streamkap_destination_starburst" "example" {
  name = "example-destination-starburst"

  # AWS S3 settings
  aws_access_key_id     = "your-access-key-id"
  aws_secret_access_key = var.destination_starburst_secret_access_key
  aws_s3_region         = "us-west-2"           # Default: us-west-2
  # Valid values: ap-south-1, eu-west-2, eu-west-1, ap-northeast-2, ap-northeast-1,
  # ca-central-1, sa-east-1, cn-north-1, us-gov-west-1, ap-southeast-1, ap-southeast-2,
  # eu-central-1, us-east-1, us-east-2, us-west-1, us-west-2

  aws_s3_bucket_name = "my-starburst-bucket"

  # File format settings
  format = "Parquet"                            # Valid values: CSV, JSON Lines, JSON Array, Parquet. Default: CSV

  # File naming
  file_name_template = "{{topic}}-{{partition}}-{{start_offset}}" # Default filename template
  file_name_prefix   = "starburst/data/"        # Directory prefix for files

  # Compression
  file_compression_type = "gzip"                # Valid values: none, gzip, snappy, zstd. Default: gzip

  # Output field selection
  format_output_fields = ["key", "value", "timestamp", "offset", "headers"]
}

output "example_destination_starburst" {
  value = streamkap_destination_starburst.example.id
}
