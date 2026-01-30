# Minimal S3 source configuration
# This example captures data from S3 bucket files

resource "streamkap_source_s3" "example" {
  name = "my-s3-source"

  # AWS credentials
  aws_access_key_id     = var.aws_access_key_id
  aws_secret_access_key = var.aws_secret_access_key

  # S3 bucket settings
  aws_s3_region      = "us-west-2"
  aws_s3_bucket_name = "my-data-bucket"
}

variable "aws_access_key_id" {
  description = "AWS access key ID"
  type        = string
}

variable "aws_secret_access_key" {
  description = "AWS secret access key"
  type        = string
  sensitive   = true
}
