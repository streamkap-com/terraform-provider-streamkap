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

variable "s3_aws_access_key_id" {
  type        = string
  description = "The AWS Access Key ID used to connect to S3"
}
variable "s3_aws_secret_access_key" {
  type        = string
  sensitive   = true
  description = "The AWS Secret Access Key used to connect to S3"
}

# Complete S3 destination configuration with all options
resource "streamkap_destination_s3" "example" {
  name = "example-destination-s3"

  # AWS credentials
  aws_access_key_id     = var.s3_aws_access_key_id
  aws_secret_access_key = var.s3_aws_secret_access_key

  # S3 bucket configuration
  # Valid regions: ap-south-1, eu-west-2, eu-west-1, ap-northeast-2, ap-northeast-1,
  #                ca-central-1, sa-east-1, cn-north-1, us-gov-west-1, ap-southeast-1,
  #                ap-southeast-2, eu-central-1, us-east-1, us-east-2, us-west-1, us-west-2
  aws_s3_region      = "us-west-2"
  aws_s3_bucket_name = "my-streamkap-bucket"

  # File format options: JSON Lines, JSON Array, Parquet
  format = "JSON Array"

  # File naming configuration
  file_name_template = "{{topic}}-{{partition}}-{{start_offset}}"
  file_name_prefix   = "data/"

  # Compression options: none, gzip, snappy, zstd
  file_compression_type = "gzip"

  # Output fields to include: key, offset, timestamp, value, headers
  format_output_fields = ["value", "key"]
}

output "s3_destination_id" {
  value = streamkap_destination_s3.example.id
}
